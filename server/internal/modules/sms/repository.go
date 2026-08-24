package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dugble/dugble/server/internal/adapters/postgres"
	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	platformwebhook "github.com/dugble/dugble/server/internal/platform/webhook"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var ErrMessageNotFound = errors.New("sms message not found")
var ErrMessageNotSchedulable = errors.New("sms message is not a pending scheduled message")

type Repository struct {
	db      *pgxpool.Pool
	dbtx    dbsqlc.DBTX
	queries *dbsqlc.Queries
	tx      pgx.Tx
	emitter webhookEmitter
}

type webhookEmitter interface {
	EmitTx(context.Context, pgx.Tx, platformwebhook.Event) (uuid.UUID, int64, error)
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, dbtx: db, queries: dbsqlc.New(db)}
}

func NewRepositoryWithWebhookEmitter(db *pgxpool.Pool, emitter webhookEmitter) *Repository {
	repository := NewRepository(db)
	repository.emitter = emitter
	return repository
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin sms transaction: %w", err)
	}
	return tx, nil
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: r.db, dbtx: tx, queries: r.queries.WithTx(tx), tx: tx, emitter: r.emitter}
}

type createMessageParams struct {
	TeamID             uuid.UUID
	SenderID           *uuid.UUID
	To                 string
	From               string
	Body               string
	Status             string
	Segments           int32
	Metadata           json.RawMessage
	ScheduledAt        *time.Time
	DestinationCountry string
}

func (r *Repository) Create(ctx context.Context, params createMessageParams) (Message, error) {
	row, err := r.queries.CreateSMSMessage(ctx, dbsqlc.CreateSMSMessageParams{
		TeamID:             params.TeamID,
		SenderID:           params.SenderID,
		ToNumber:           params.To,
		FromName:           params.From,
		Body:               params.Body,
		Status:             params.Status,
		Segments:           params.Segments,
		Metadata:           ensureMetadata(params.Metadata),
		ScheduledAt:        pgconv.NullableTimestamptz(params.ScheduledAt),
		DestinationCountry: params.DestinationCountry,
	})
	if err != nil {
		return Message{}, fmt.Errorf("create sms message: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID, limit, offset int32) ([]Message, error) {
	rows, err := r.queries.ListSMSMessages(ctx, dbsqlc.ListSMSMessagesParams{TeamID: teamID, LimitCount: limit, OffsetCount: offset})
	if err != nil {
		return nil, fmt.Errorf("list sms messages: %w", err)
	}
	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, messageFromSQLC(row))
	}
	return messages, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (Message, error) {
	row, err := r.queries.GetSMSMessage(ctx, dbsqlc.GetSMSMessageParams{ID: id, TeamID: teamID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrMessageNotFound
		}
		return Message{}, fmt.Errorf("get sms message: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) ListEvents(ctx context.Context, teamID, messageID uuid.UUID, limit int32) ([]Event, error) {
	rows, err := r.queries.ListSMSMessageEvents(ctx, dbsqlc.ListSMSMessageEventsParams{
		MessageID: messageID, TeamID: teamID, LimitCount: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list SMS events: %w", err)
	}
	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, Event{
			ID: row.ID, Type: row.Type, OccurredAt: pgconv.TimestamptzToTime(row.OccurredAt),
			Provider: row.Provider, Code: row.Code, Message: row.Message,
		})
	}
	return events, nil
}

func (r *Repository) CancelTx(ctx context.Context, tx pgx.Tx, id, teamID uuid.UUID) (Message, error) {
	if err := lockScheduledSMS(ctx, tx, id, teamID); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE sms_messages SET status = $3, updated_at = now() WHERE id = $1 AND team_id = $2`, id, teamID, StatusCanceled); err != nil {
		return Message{}, fmt.Errorf("cancel SMS message: %w", err)
	}
	return r.WithTx(tx).Get(ctx, id, teamID)
}

func (r *Repository) RescheduleTx(ctx context.Context, tx pgx.Tx, id, teamID uuid.UUID, scheduledAt time.Time) (Message, error) {
	if err := lockScheduledSMS(ctx, tx, id, teamID); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE sms_messages SET scheduled_at = $3, updated_at = now() WHERE id = $1 AND team_id = $2`, id, teamID, scheduledAt); err != nil {
		return Message{}, fmt.Errorf("reschedule SMS message: %w", err)
	}
	return r.WithTx(tx).Get(ctx, id, teamID)
}

func lockScheduledSMS(ctx context.Context, tx pgx.Tx, id, teamID uuid.UUID) error {
	var status string
	var scheduledAt *time.Time
	err := tx.QueryRow(ctx, `SELECT status, scheduled_at FROM sms_messages WHERE id = $1 AND team_id = $2 FOR UPDATE`, id, teamID).Scan(&status, &scheduledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMessageNotFound
	}
	if err != nil {
		return fmt.Errorf("lock scheduled SMS message: %w", err)
	}
	if status != StatusQueued || scheduledAt == nil || !scheduledAt.After(time.Now().UTC().Add(scheduleMutationCutoff)) {
		return ErrMessageNotSchedulable
	}
	return nil
}

func (r *Repository) MarkProcessing(ctx context.Context, id uuid.UUID, teamID uuid.UUID) (Message, error) {
	row, err := r.queries.MarkSMSMessageProcessing(ctx, dbsqlc.MarkSMSMessageProcessingParams{ID: id, TeamID: teamID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrMessageNotFound
		}
		return Message{}, fmt.Errorf("mark sms message processing: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) MarkDeliveryUnknown(ctx context.Context, id uuid.UUID, teamID uuid.UUID, message string) (Message, error) {
	row, err := r.queries.MarkSMSMessageDeliveryUnknown(ctx, dbsqlc.MarkSMSMessageDeliveryUnknownParams{
		ID: id, TeamID: teamID, ErrorMessage: &message,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrMessageNotFound
		}
		return Message{}, fmt.Errorf("mark sms message delivery unknown: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) MarkSubmitted(ctx context.Context, id uuid.UUID, teamID uuid.UUID, providerID string, providerMessageID string, status string) (Message, error) {
	if r.tx == nil && r.emitter != nil {
		return withSMSLifecycleTx(ctx, r, func(repository *Repository) (Message, error) {
			return repository.MarkSubmitted(ctx, id, teamID, providerID, providerMessageID, status)
		})
	}
	row, err := r.queries.MarkSMSMessageSubmitted(ctx, dbsqlc.MarkSMSMessageSubmittedParams{
		ID:                id,
		TeamID:            teamID,
		ProviderID:        &providerID,
		ProviderMessageID: &providerMessageID,
		Status:            status,
	})
	if err != nil {
		return Message{}, fmt.Errorf("mark sms message submitted: %w", err)
	}
	message := messageFromSQLC(row)
	if err := r.emitLifecycle(ctx, message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, teamID uuid.UUID, message string) (Message, error) {
	if r.tx == nil && r.emitter != nil {
		return withSMSLifecycleTx(ctx, r, func(repository *Repository) (Message, error) {
			return repository.MarkFailed(ctx, id, teamID, message)
		})
	}
	row, err := r.queries.MarkSMSMessageFailed(ctx, dbsqlc.MarkSMSMessageFailedParams{ID: id, TeamID: teamID, ErrorMessage: &message})
	if err != nil {
		return Message{}, fmt.Errorf("mark sms message failed: %w", err)
	}
	updated := messageFromSQLC(row)
	if err := r.emitLifecycle(ctx, updated); err != nil {
		return Message{}, err
	}
	return updated, nil
}

func (r *Repository) ApplyFeedback(
	ctx context.Context,
	id uuid.UUID,
	teamID uuid.UUID,
	status string,
	errorMessage string,
	occurredAt time.Time,
) (Message, error) {
	if r.tx == nil && r.emitter != nil {
		return withSMSLifecycleTx(ctx, r, func(repository *Repository) (Message, error) {
			return repository.ApplyFeedback(ctx, id, teamID, status, errorMessage, occurredAt)
		})
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	} else {
		occurredAt = occurredAt.UTC()
	}
	var currentStatus string
	if err := r.dbtx.QueryRow(ctx, `
		SELECT status
		FROM sms_messages
		WHERE id = $1 AND team_id = $2
		FOR UPDATE
	`, id, teamID).Scan(&currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrMessageNotFound
		}
		return Message{}, fmt.Errorf("lock SMS message for feedback: %w", err)
	}
	nextStatus := nextSMSFeedbackStatus(currentStatus, status)
	if nextStatus == currentStatus {
		return r.Get(ctx, id, teamID)
	}
	if _, err := r.dbtx.Exec(ctx, `
		UPDATE sms_messages
		SET status = $3,
			submitted_at = CASE
				WHEN $3 IN ('submitted', 'sent', 'delivered')
				THEN COALESCE(submitted_at, $5)
				ELSE submitted_at
			END,
			delivered_at = CASE
				WHEN $3 = 'delivered' THEN COALESCE(delivered_at, $5)
				ELSE delivered_at
			END,
			error_message = CASE
				WHEN $3 IN ('undelivered', 'rejected', 'failed', 'expired', 'unknown')
				THEN NULLIF(trim($4), '')
				WHEN $3 IN ('submitted', 'sent', 'delivered') THEN NULL
				ELSE error_message
			END,
			updated_at = now()
		WHERE id = $1 AND team_id = $2
	`, id, teamID, nextStatus, errorMessage, occurredAt); err != nil {
		return Message{}, fmt.Errorf("apply SMS feedback: %w", err)
	}
	message, err := r.Get(ctx, id, teamID)
	if err != nil {
		return Message{}, err
	}
	if err := r.emitLifecycle(ctx, message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func nextSMSFeedbackStatus(current, next string) string {
	current = strings.ToLower(strings.TrimSpace(current))
	next = strings.ToLower(strings.TrimSpace(next))
	if current == next || smsFeedbackTerminalStatus(current) {
		return current
	}
	if !knownSMSMessageStatus(next) {
		return current
	}
	if next == StatusUnknown || current == StatusUnknown || smsFeedbackTerminalStatus(next) {
		return next
	}
	currentRank, currentProgress := smsFeedbackProgressRank(current)
	nextRank, nextProgress := smsFeedbackProgressRank(next)
	if currentProgress && nextProgress && nextRank > currentRank {
		return next
	}
	return current
}

func smsFeedbackTerminalStatus(status string) bool {
	switch status {
	case StatusDelivered, StatusUndelivered, StatusRejected, StatusFailed, StatusExpired, StatusCanceled:
		return true
	default:
		return false
	}
}

func knownSMSMessageStatus(status string) bool {
	switch status {
	case StatusQueued, StatusProcessing, StatusSubmitted, StatusSent, StatusDelivered,
		StatusUndelivered, StatusRejected, StatusFailed, StatusExpired, StatusUnknown, StatusCanceled:
		return true
	default:
		return false
	}
}

func smsFeedbackProgressRank(status string) (int, bool) {
	switch status {
	case StatusQueued:
		return 0, true
	case StatusProcessing:
		return 1, true
	case StatusSubmitted:
		return 2, true
	case StatusSent:
		return 3, true
	default:
		return 0, false
	}
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, teamID uuid.UUID, status string) (Message, error) {
	if r.tx == nil && r.emitter != nil {
		return withSMSLifecycleTx(ctx, r, func(repository *Repository) (Message, error) {
			return repository.UpdateStatus(ctx, id, teamID, status)
		})
	}
	row, err := r.queries.UpdateSMSMessageStatus(ctx, dbsqlc.UpdateSMSMessageStatusParams{ID: id, TeamID: teamID, Status: status})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// A concurrent status update may have already advanced or completed the
			// message. Return that authoritative state without emitting a duplicate
			// lifecycle event for the rejected transition.
			return r.Get(ctx, id, teamID)
		}
		return Message{}, fmt.Errorf("update sms message status: %w", err)
	}
	message := messageFromSQLC(row)
	if err := r.emitLifecycle(ctx, message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func withSMSLifecycleTx(ctx context.Context, repository *Repository, operation func(*Repository) (Message, error)) (Message, error) {
	return postgres.InTransactionResult(ctx, repository.db, func(tx pgx.Tx) (Message, error) {
		return operation(repository.WithTx(tx))
	})
}

func (r *Repository) emitLifecycle(ctx context.Context, message Message) error {
	if r.emitter == nil {
		return nil
	}
	if r.tx == nil {
		return errors.New("SMS lifecycle webhook transaction is required")
	}
	event, ok, err := smsLifecycleEvent(message)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if _, _, err := r.emitter.EmitTx(ctx, r.tx, event); err != nil {
		return fmt.Errorf("emit SMS lifecycle webhook: %w", err)
	}
	return nil
}

func smsLifecycleEvent(message Message) (platformwebhook.Event, bool, error) {
	eventTypes := map[string]string{
		StatusSubmitted:   platformwebhook.EventSMSSubmitted,
		StatusSent:        platformwebhook.EventSMSSent,
		StatusDelivered:   platformwebhook.EventSMSDelivered,
		StatusUndelivered: platformwebhook.EventSMSUndelivered,
		StatusFailed:      platformwebhook.EventSMSFailed,
	}
	eventType, ok := eventTypes[message.Status]
	if !ok {
		return platformwebhook.Event{}, false, nil
	}
	messageID, err := uuid.Parse(message.ID)
	if err != nil {
		return platformwebhook.Event{}, false, fmt.Errorf("parse SMS message id for webhook: %w", err)
	}
	teamID, err := uuid.Parse(message.TeamID)
	if err != nil {
		return platformwebhook.Event{}, false, fmt.Errorf("parse SMS team id for webhook: %w", err)
	}
	payload, err := json.Marshal(message.Response())
	if err != nil {
		return platformwebhook.Event{}, false, fmt.Errorf("encode SMS webhook payload: %w", err)
	}
	return platformwebhook.Event{
		ID: uuid.New(), TeamID: teamID, Type: eventType, ObjectType: "sms", ObjectID: &messageID,
		Payload: payload, OccurredAt: message.UpdatedAt,
	}, true, nil
}

func (r *Repository) FindApprovedSender(ctx context.Context, teamID uuid.UUID, name string) (*uuid.UUID, error) {
	id, err := r.queries.FindApprovedSMSSender(ctx, dbsqlc.FindApprovedSMSSenderParams{TeamID: teamID, Name: name})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find approved sender id: %w", err)
	}
	return &id, nil
}

func messageFromSQLC(row dbsqlc.SmsMessage) Message {
	message := Message{
		ID:                 row.ID.String(),
		TeamID:             row.TeamID.String(),
		To:                 row.ToNumber,
		From:               row.FromName,
		Body:               row.Body,
		Status:             row.Status,
		ProviderID:         row.ProviderID,
		ProviderMessageID:  row.ProviderMessageID,
		Segments:           row.Segments,
		ErrorMessage:       row.ErrorMessage,
		Metadata:           ensureMetadata(row.Metadata),
		ScheduledAt:        pgconv.TimestamptzToTimePtr(row.ScheduledAt),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
		DestinationCountry: row.DestinationCountry,
	}
	if row.SenderID != nil {
		value := row.SenderID.String()
		message.SenderID = &value
	}
	if row.SubmittedAt.Valid {
		message.SubmittedAt = &row.SubmittedAt.Time
	}
	if row.DeliveredAt.Valid {
		message.DeliveredAt = &row.DeliveredAt.Time
	}
	return message
}

func ensureMetadata(metadata json.RawMessage) json.RawMessage {
	if len(metadata) == 0 {
		return json.RawMessage(`{}`)
	}
	return metadata
}

func (r *Repository) GetAnalytics(ctx context.Context, teamID uuid.UUID) (AnalyticsResponse, error) {
	windows := make([]AnalyticsWindow, 0, 3)
	for _, days := range []int32{7, 30, 90} {
		points, err := r.smsAnalyticsSeries(ctx, teamID, days)
		if err != nil {
			return AnalyticsResponse{}, err
		}
		windows = append(windows, AnalyticsWindow{Days: days, Rates: smsRates(points), Series: points})
	}
	countries, err := r.smsDeliveryByCountry(ctx, teamID)
	if err != nil {
		return AnalyticsResponse{}, err
	}
	return AnalyticsResponse{Object: "sms.analytics", Windows: windows, DeliveryByCountry: countries}, nil
}

func (r *Repository) smsAnalyticsSeries(ctx context.Context, teamID uuid.UUID, days int32) ([]AnalyticsPoint, error) {
	rows, err := r.db.Query(ctx, `
		WITH dates AS (
			SELECT generate_series(date_trunc('day', now()) - (($2::int - 1) * interval '1 day'), date_trunc('day', now()), interval '1 day') AS bucket
		), counts AS (
			SELECT date_trunc('day', created_at) AS bucket,
			       count(*)::bigint AS total,
			       count(*) FILTER (WHERE status = 'delivered')::bigint AS delivered,
			       count(*) FILTER (WHERE status IN ('undelivered', 'rejected', 'failed', 'expired'))::bigint AS failed
			FROM sms_messages
			WHERE team_id = $1 AND created_at >= date_trunc('day', now()) - (($2::int - 1) * interval '1 day')
			GROUP BY 1
		)
		SELECT to_char(dates.bucket, 'YYYY-MM-DD'), COALESCE(counts.total, 0), COALESCE(counts.delivered, 0), COALESCE(counts.failed, 0)
		FROM dates LEFT JOIN counts ON counts.bucket = dates.bucket ORDER BY dates.bucket
	`, teamID, days)
	if err != nil {
		return nil, fmt.Errorf("get SMS analytics series: %w", err)
	}
	defer rows.Close()
	points := make([]AnalyticsPoint, 0, days)
	for rows.Next() {
		var p AnalyticsPoint
		if err := rows.Scan(&p.Date, &p.Total, &p.Delivered, &p.Failed); err != nil {
			return nil, fmt.Errorf("scan SMS analytics series: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SMS analytics series: %w", err)
	}
	return points, nil
}

func (r *Repository) smsDeliveryByCountry(ctx context.Context, teamID uuid.UUID) ([]CountryAnalytics, error) {
	rows, err := r.db.Query(ctx, `
		SELECT destination_country::text,
		       count(*)::bigint,
		       count(*) FILTER (WHERE status = 'delivered')::bigint,
		       count(*) FILTER (WHERE status IN ('undelivered', 'rejected', 'failed', 'expired'))::bigint
		FROM sms_messages
		WHERE team_id = $1 AND created_at >= now() - interval '90 days'
		GROUP BY destination_country ORDER BY count(*) DESC, destination_country LIMIT 25
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("get SMS delivery by country: %w", err)
	}
	defer rows.Close()
	countries := []CountryAnalytics{}
	for rows.Next() {
		var c CountryAnalytics
		if err := rows.Scan(&c.Country, &c.Total, &c.Delivered, &c.Failed); err != nil {
			return nil, fmt.Errorf("scan SMS delivery by country: %w", err)
		}
		countries = append(countries, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SMS delivery by country: %w", err)
	}
	return countries, nil
}

func smsRates(points []AnalyticsPoint) []AnalyticsRate {
	var total, delivered, failed int64
	for _, point := range points {
		total += point.Total
		delivered += point.Delivered
		failed += point.Failed
	}
	return []AnalyticsRate{{Name: "delivery_rate", Value: percentage(delivered, total)}, {Name: "failure_rate", Value: percentage(failed, total)}}
}

func percentage(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total)
}
