package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	platformevent "github.com/dugble/dugble/server/internal/platform/event"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var (
	ErrNotFound = errors.New("broadcast not found")
	ErrConflict = errors.New("broadcast conflict")
)

type eventEmitter interface {
	EmitTx(context.Context, pgx.Tx, platformevent.Envelope) (platformevent.Result, error)
}

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
	emitter eventEmitter
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, queries: dbsqlc.New(db)}
}

func NewRepositoryWithEventEmitter(db *pgxpool.Pool, emitter eventEmitter) *Repository {
	return &Repository{db: db, queries: dbsqlc.New(db), emitter: emitter}
}

func (r *Repository) Create(ctx context.Context, teamID, segmentID uuid.UUID, topicID *uuid.UUID, req CreateRequest) (Broadcast, error) {
	bindings, err := encodeBindings(req.VariableBindings)
	if err != nil {
		return Broadcast{}, err
	}
	row, err := r.queries.CreateBroadcast(ctx, dbsqlc.CreateBroadcastParams{
		TeamID: teamID, Name: req.Name, SegmentID: segmentID, TopicID: topicID,
		FromEmail: req.FromEmail, FromName: req.FromName, ReplyToEmail: req.ReplyToEmail,
		Subject: req.Subject, PreviewText: req.PreviewText, HtmlBody: req.HTML, TextBody: req.Text,
		VariableBindings: bindings,
	})
	if err != nil {
		return Broadcast{}, fmt.Errorf("create broadcast: %w", err)
	}
	return broadcastFromSQLC(row)
}

func (r *Repository) Duplicate(ctx context.Context, teamID, sourceID uuid.UUID, name string) (Broadcast, error) {
	row, err := r.queries.DuplicateBroadcast(ctx, dbsqlc.DuplicateBroadcastParams{Name: name, SourceID: sourceID, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrNotFound
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("duplicate broadcast: %w", err)
	}
	return broadcastFromSQLC(row)
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID, limit, offset int32) ([]Broadcast, error) {
	rows, err := r.queries.ListBroadcasts(ctx, dbsqlc.ListBroadcastsParams{TeamID: teamID, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list broadcasts: %w", err)
	}
	values := make([]Broadcast, 0, len(rows))
	for _, row := range rows {
		value, mapErr := broadcastFromSQLC(row)
		if mapErr != nil {
			return nil, mapErr
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *Repository) Get(ctx context.Context, teamID, id uuid.UUID) (Broadcast, error) {
	row, err := r.queries.GetBroadcast(ctx, dbsqlc.GetBroadcastParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrNotFound
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("get broadcast: %w", err)
	}
	return broadcastFromSQLC(row)
}

func (r *Repository) Update(ctx context.Context, teamID, id, segmentID uuid.UUID, topicID *uuid.UUID, revision int64, current Broadcast) (Broadcast, error) {
	bindings, err := encodeBindings(current.VariableBindings)
	if err != nil {
		return Broadcast{}, err
	}
	row, err := r.queries.UpdateBroadcastDraftOrScheduled(ctx, dbsqlc.UpdateBroadcastDraftOrScheduledParams{
		Name: current.Name, SegmentID: segmentID, TopicID: topicID,
		FromEmail: stringPointer(current.FromEmail), FromName: current.FromName, ReplyToEmail: current.ReplyToEmail,
		Subject: current.Subject, PreviewText: current.PreviewText, HtmlBody: current.HTML, TextBody: current.Text,
		VariableBindings: bindings, ID: id, TeamID: teamID, Revision: revision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrConflict
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("update broadcast: %w", err)
	}
	return broadcastFromSQLC(row)
}

func (r *Repository) Send(ctx context.Context, teamID, id uuid.UUID, scheduledAt *time.Time) (Broadcast, error) {
	if r == nil || r.db == nil || r.queries == nil {
		return Broadcast{}, errors.New("broadcast repository is not configured")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Broadcast{}, fmt.Errorf("begin broadcast send: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)

	currentRow, err := queries.GetBroadcast(ctx, dbsqlc.GetBroadcastParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrNotFound
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("load broadcast before send: %w", err)
	}
	if scheduledAt == nil && currentRow.Status != StatusDraft {
		return Broadcast{}, ErrConflict
	}
	if scheduledAt != nil && currentRow.Status != StatusDraft && currentRow.Status != StatusScheduled {
		return Broadcast{}, ErrConflict
	}

	var row dbsqlc.Broadcast
	eventType := platformevent.TypeBroadcastQueued
	reason := "immediate_send"
	fromStatus := currentRow.Status
	if scheduledAt == nil {
		row, err = queries.QueueBroadcast(ctx, dbsqlc.QueueBroadcastParams{ID: id, TeamID: teamID})
	} else {
		row, err = queries.ScheduleBroadcast(ctx, dbsqlc.ScheduleBroadcastParams{
			ScheduledAt: pgconv.TimestamptzFromTime(*scheduledAt), ID: id, TeamID: teamID,
		})
		eventType = platformevent.TypeBroadcastScheduled
		reason = "scheduled_send"
		if fromStatus == StatusScheduled {
			reason = "rescheduled"
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrConflict
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("send broadcast: %w", err)
	}
	value, err := broadcastFromSQLC(row)
	if err != nil {
		return Broadcast{}, err
	}
	if err := emitBroadcastEvent(ctx, tx, r.emitter, eventType, value, fromStatus, reason, nil); err != nil {
		return Broadcast{}, fmt.Errorf("emit broadcast lifecycle event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Broadcast{}, fmt.Errorf("commit broadcast send: %w", err)
	}
	return value, nil
}

func (r *Repository) QueueScheduled(ctx context.Context, teamID, id uuid.UUID) (Broadcast, error) {
	return r.transition(ctx, platformevent.TypeBroadcastQueued, StatusScheduled, "schedule_due", nil, func(q *dbsqlc.Queries) (dbsqlc.Broadcast, error) {
		return q.QueueScheduledBroadcast(ctx, dbsqlc.QueueScheduledBroadcastParams{ID: id, TeamID: teamID})
	})
}

func (r *Repository) QueueNextDueScheduled(ctx context.Context) (Broadcast, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Broadcast{}, false, fmt.Errorf("begin due broadcast claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := r.queries.WithTx(tx).QueueNextDueBroadcast(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, false, nil
	}
	if err != nil {
		return Broadcast{}, false, fmt.Errorf("claim due broadcast: %w", err)
	}
	value, err := broadcastFromSQLC(row)
	if err != nil {
		return Broadcast{}, false, err
	}
	if err := emitBroadcastEvent(ctx, tx, r.emitter, platformevent.TypeBroadcastQueued, value, StatusScheduled, "schedule_due", nil); err != nil {
		return Broadcast{}, false, fmt.Errorf("emit due broadcast queued event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Broadcast{}, false, fmt.Errorf("commit due broadcast claim: %w", err)
	}
	return value, true, nil
}

func (r *Repository) MarkSent(ctx context.Context, teamID, id uuid.UUID) (Broadcast, error) {
	return r.transition(ctx, platformevent.TypeBroadcastSent, StatusQueued, "recipient_fanout_completed", nil, func(q *dbsqlc.Queries) (dbsqlc.Broadcast, error) {
		return q.MarkBroadcastSent(ctx, dbsqlc.MarkBroadcastSentParams{ID: id, TeamID: teamID})
	})
}

func (r *Repository) MarkFailed(ctx context.Context, teamID, id uuid.UUID, phase, code, message string, retryable bool) (Broadcast, error) {
	failure := map[string]any{"phase": phase, "code": code, "message": message, "retryable": retryable}
	return r.transition(ctx, platformevent.TypeBroadcastFailed, StatusQueued, "orchestration_failed", failure, func(q *dbsqlc.Queries) (dbsqlc.Broadcast, error) {
		return q.MarkBroadcastFailed(ctx, dbsqlc.MarkBroadcastFailedParams{ID: id, TeamID: teamID})
	})
}

func (r *Repository) Cancel(ctx context.Context, teamID, id uuid.UUID) (Broadcast, error) {
	current, err := r.Get(ctx, teamID, id)
	if err != nil {
		return Broadcast{}, err
	}
	switch current.Status {
	case StatusScheduled:
		return r.transition(ctx, platformevent.TypeBroadcastCanceled, StatusScheduled, "schedule_canceled", nil, func(q *dbsqlc.Queries) (dbsqlc.Broadcast, error) {
			return q.CancelScheduledBroadcast(ctx, dbsqlc.CancelScheduledBroadcastParams{ID: id, TeamID: teamID})
		})
	case StatusQueued:
		return r.transition(ctx, platformevent.TypeBroadcastCanceled, StatusQueued, "execution_canceled", nil, func(q *dbsqlc.Queries) (dbsqlc.Broadcast, error) {
			return q.CancelQueuedBroadcast(ctx, dbsqlc.CancelQueuedBroadcastParams{ID: id, TeamID: teamID})
		})
	default:
		return Broadcast{}, ErrConflict
	}
}

func (r *Repository) transition(ctx context.Context, eventType platformevent.Type, from, reason string, failure map[string]any, update func(*dbsqlc.Queries) (dbsqlc.Broadcast, error)) (Broadcast, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Broadcast{}, fmt.Errorf("begin broadcast transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := update(r.queries.WithTx(tx))
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrConflict
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("update broadcast transition: %w", err)
	}
	value, err := broadcastFromSQLC(row)
	if err != nil {
		return Broadcast{}, err
	}
	if err := emitBroadcastEvent(ctx, tx, r.emitter, eventType, value, from, reason, failure); err != nil {
		return Broadcast{}, fmt.Errorf("emit broadcast lifecycle event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Broadcast{}, fmt.Errorf("commit broadcast transition: %w", err)
	}
	return value, nil
}

func (r *Repository) Delete(ctx context.Context, teamID, id uuid.UUID) (Broadcast, error) {
	row, err := r.queries.SoftDeleteBroadcast(ctx, dbsqlc.SoftDeleteBroadcastParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrConflict
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("delete broadcast: %w", err)
	}
	return broadcastFromSQLC(row)
}

func (r *Repository) ListRecipients(ctx context.Context, teamID, broadcastID uuid.UUID, limit, offset int32) ([]Recipient, error) {
	rows, err := r.queries.ListBroadcastRecipients(ctx, dbsqlc.ListBroadcastRecipientsParams{TeamID: teamID, BroadcastID: broadcastID, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list broadcast recipients: %w", err)
	}
	values := make([]Recipient, 0, len(rows))
	for _, row := range rows {
		value, mapErr := recipientFromSQLC(row)
		if mapErr != nil {
			return nil, mapErr
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *Repository) GetExclusionSummary(ctx context.Context, teamID, broadcastID uuid.UUID) (ExclusionSummary, error) {
	rows, err := r.queries.GetBroadcastExclusionSummary(ctx, dbsqlc.GetBroadcastExclusionSummaryParams{TeamID: teamID, BroadcastID: broadcastID})
	if err != nil {
		return ExclusionSummary{}, fmt.Errorf("query broadcast exclusion summary: %w", err)
	}
	summary := ExclusionSummary{Object: "broadcast.exclusion_summary", BroadcastID: broadcastID.String(), Reasons: make(map[string]int64)}
	for _, row := range rows {
		summary.Reasons[row.Reason] = row.RecipientCount
		summary.Total += row.RecipientCount
	}
	return summary, nil
}

func (r *Repository) GetAnalytics(ctx context.Context, teamID, broadcastID uuid.UUID) (Analytics, error) {
	row, err := r.queries.GetBroadcastAnalytics(ctx, dbsqlc.GetBroadcastAnalyticsParams{BroadcastID: broadcastID, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Analytics{}, ErrNotFound
	}
	if err != nil {
		return Analytics{}, fmt.Errorf("get broadcast analytics: %w", err)
	}
	return Analytics{
		Object: "broadcast.analytics", BroadcastID: broadcastID.String(), Audience: row.AudienceCount,
		Eligible: row.EligibleCount, Excluded: row.Excluded, Queued: row.Queued, Delivered: row.Delivered,
		Bounced: row.Bounced, Complained: row.Complained, Failed: row.Failed, Opened: row.Opened, Clicked: row.Clicked,
	}, nil
}

func (r *Repository) BeginFanoutTx(ctx context.Context) (pgx.Tx, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("broadcast repository is not configured")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin broadcast fanout transaction: %w", err)
	}
	return tx, nil
}

func (r *Repository) ClaimNextRecipientForFanoutTx(ctx context.Context, tx pgx.Tx) (FanoutRecipient, bool, error) {
	if r == nil || r.queries == nil || tx == nil {
		return FanoutRecipient{}, false, errors.New("broadcast fanout repository is not configured")
	}
	row, err := r.queries.WithTx(tx).ClaimNextBroadcastRecipientForFanout(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return FanoutRecipient{}, false, nil
	}
	if err != nil {
		return FanoutRecipient{}, false, fmt.Errorf("claim broadcast recipient for fanout: %w", err)
	}
	if row.TemplateID == nil || row.TemplateVersionID == nil {
		return FanoutRecipient{}, false, errors.New("broadcast fanout still requires legacy template compatibility data")
	}
	snapshot, err := decodeObject(row.ContactSnapshot, "broadcast recipient snapshot")
	if err != nil {
		return FanoutRecipient{}, false, err
	}
	bindings, err := decodeObject(row.VariableBindings, "broadcast variable bindings")
	if err != nil {
		return FanoutRecipient{}, false, err
	}
	return FanoutRecipient{
		ID: row.ID, TeamID: row.TeamID, BroadcastID: row.BroadcastID, ContactID: row.ContactID,
		Email: row.Email, FirstName: row.FirstName, LastName: row.LastName, ContactSnapshot: snapshot,
		VariableBindings: bindings, AttemptCount: row.AttemptCount,
		TemplateID: *row.TemplateID, TemplateVersionID: *row.TemplateVersionID,
	}, true, nil
}

func (r *Repository) SetRecipientQueuedTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, emailMessageID uuid.UUID) error {
	_, err := r.queries.WithTx(tx).SetBroadcastRecipientQueued(ctx, dbsqlc.SetBroadcastRecipientQueuedParams{EmailMessageID: &emailMessageID, ID: recipient.ID, TeamID: recipient.TeamID, BroadcastID: recipient.BroadcastID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("mark broadcast recipient queued: %w", err)
	}
	return nil
}

func (r *Repository) RecheckRecipientEligibilityTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient) (string, bool, error) {
	if tx == nil {
		return "", false, errors.New("broadcast recipient eligibility transaction is required")
	}
	reason, err := r.queries.WithTx(tx).RecheckBroadcastRecipientEligibility(ctx, dbsqlc.RecheckBroadcastRecipientEligibilityParams{ID: recipient.ID, TeamID: recipient.TeamID, BroadcastID: recipient.BroadcastID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("recheck broadcast recipient eligibility: %w", err)
	}
	if reason == nil {
		return "", false, nil
	}
	return *reason, true, nil
}

func (r *Repository) RetryRecipientFanoutTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, nextAttemptAt time.Time, code, message string) error {
	_, err := r.queries.WithTx(tx).RetryBroadcastRecipientFanout(ctx, dbsqlc.RetryBroadcastRecipientFanoutParams{
		NextAttemptAt: pgconv.TimestamptzFromTime(nextAttemptAt), ErrorCode: stringPointer(code), ErrorMessage: stringPointer(message),
		ID: recipient.ID, TeamID: recipient.TeamID, BroadcastID: recipient.BroadcastID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("schedule broadcast recipient retry: %w", err)
	}
	return nil
}

func (r *Repository) FailRecipientFanoutTx(ctx context.Context, tx pgx.Tx, recipient FanoutRecipient, code, message string) error {
	_, err := r.queries.WithTx(tx).FailBroadcastRecipientFanout(ctx, dbsqlc.FailBroadcastRecipientFanoutParams{
		ErrorCode: stringPointer(code), ErrorMessage: stringPointer(message), ID: recipient.ID, TeamID: recipient.TeamID, BroadcastID: recipient.BroadcastID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("fail broadcast recipient fanout: %w", err)
	}
	return nil
}

func (r *Repository) FinalizeBroadcastFanoutTx(ctx context.Context, tx pgx.Tx, teamID, broadcastID uuid.UUID) (Broadcast, error) {
	row, err := r.queries.WithTx(tx).FinalizeBroadcastFanout(ctx, dbsqlc.FinalizeBroadcastFanoutParams{BroadcastID: broadcastID, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Broadcast{}, ErrConflict
	}
	if err != nil {
		return Broadcast{}, fmt.Errorf("finalize broadcast fanout: %w", err)
	}
	return broadcastFromSQLC(row)
}

type MaterializationResult struct {
	BroadcastID   uuid.UUID
	TeamID        uuid.UUID
	AudienceCount int64
	EligibleCount int64
	ExcludedCount int64
}

func (r *Repository) MaterializeNextQueuedRecipients(ctx context.Context) (MaterializationResult, bool, error) {
	if r == nil || r.db == nil || r.queries == nil {
		return MaterializationResult{}, false, errors.New("broadcast repository is not configured")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterializationResult{}, false, fmt.Errorf("begin recipient materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)

	candidate, err := queries.ClaimNextQueuedBroadcastForMaterialization(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return MaterializationResult{}, false, nil
	}
	if err != nil {
		return MaterializationResult{}, false, fmt.Errorf("claim queued broadcast: %w", err)
	}
	if err := queries.MaterializeBroadcastRecipients(ctx, dbsqlc.MaterializeBroadcastRecipientsParams{TopicID: candidate.TopicID, TeamID: candidate.TeamID, SegmentID: candidate.SegmentID, BroadcastID: candidate.ID}); err != nil {
		return MaterializationResult{}, false, fmt.Errorf("insert broadcast recipients: %w", err)
	}
	completed, err := queries.CompleteBroadcastRecipientMaterialization(ctx, dbsqlc.CompleteBroadcastRecipientMaterializationParams{BroadcastID: candidate.ID, TeamID: candidate.TeamID})
	if err != nil {
		return MaterializationResult{}, false, fmt.Errorf("complete recipient materialization: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterializationResult{}, false, fmt.Errorf("commit recipient materialization: %w", err)
	}
	return MaterializationResult{BroadcastID: completed.BroadcastID, TeamID: completed.TeamID, AudienceCount: completed.AudienceCount, EligibleCount: completed.EligibleCount, ExcludedCount: completed.ExcludedCount}, true, nil
}

type broadcastTransition struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason,omitempty"`
}

type broadcastSummary struct {
	AudienceCount   int64 `json:"audience_count"`
	EligibleCount   int64 `json:"eligible_count"`
	SuppressedCount int64 `json:"suppressed_count"`
	QueuedCount     int64 `json:"queued_count"`
	FailedCount     int64 `json:"failed_count"`
}

func emitBroadcastEvent(ctx context.Context, tx pgx.Tx, emitter eventEmitter, eventType platformevent.Type, value Broadcast, from, reason string, failure map[string]any) error {
	if emitter == nil {
		emitter = platformevent.DefaultEmitter()
	}
	if emitter == nil {
		return nil
	}
	teamID, err := uuid.Parse(value.TeamID)
	if err != nil {
		return fmt.Errorf("parse broadcast team id: %w", err)
	}
	objectID, err := uuid.Parse(value.ID)
	if err != nil {
		return fmt.Errorf("parse broadcast id: %w", err)
	}
	payload := map[string]any{
		"broadcast": value,
		"transition": broadcastTransition{From: from, To: value.Status, Reason: reason},
		"summary": broadcastSummary{AudienceCount: value.AudienceCount, EligibleCount: value.EligibleCount, SuppressedCount: value.SuppressedCount, QueuedCount: value.QueuedCount, FailedCount: value.FailedCount},
	}
	if failure != nil {
		payload["failure"] = failure
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode broadcast event: %w", err)
	}
	_, err = emitter.EmitTx(ctx, tx, platformevent.Envelope{Type: eventType, TeamID: teamID, ObjectType: "broadcast", ObjectID: &objectID, Data: data})
	return err
}

func broadcastFromSQLC(row dbsqlc.Broadcast) (Broadcast, error) {
	bindings, err := decodeObject(row.VariableBindings, "broadcast variable bindings")
	if err != nil {
		return Broadcast{}, err
	}
	return Broadcast{
		ID: row.ID.String(), TeamID: row.TeamID.String(), Name: row.Name, Status: row.Status,
		SegmentID: row.SegmentID.String(), TopicID: pgconv.UUIDStringPtr(row.TopicID),
		FromEmail: pointerString(row.FromEmail), FromName: row.FromName, ReplyToEmail: row.ReplyToEmail,
		Subject: row.Subject, PreviewText: row.PreviewText, HTML: row.HtmlBody, Text: row.TextBody,
		VariableBindings: bindings, ScheduledAt: pgconv.TimestamptzToTimePtr(row.ScheduledAt),
		QueuedAt: pgconv.TimestamptzToTimePtr(row.QueuedAt), SentAt: pgconv.TimestamptzToTimePtr(row.SentAt),
		CanceledAt: pgconv.TimestamptzToTimePtr(row.CanceledAt), AudienceCount: row.AudienceCount,
		EligibleCount: row.EligibleCount, SuppressedCount: row.SuppressedCount, QueuedCount: row.QueuedCount,
		FailedCount: row.FailedCount, Revision: row.Revision, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func recipientFromSQLC(row dbsqlc.BroadcastRecipient) (Recipient, error) {
	snapshot, err := decodeObject(row.ContactSnapshot, "broadcast recipient snapshot")
	if err != nil {
		return Recipient{}, err
	}
	return Recipient{
		ID: row.ID.String(), BroadcastID: row.BroadcastID.String(), ContactID: pgconv.UUIDStringPtr(row.ContactID),
		Email: row.Email, FirstName: row.FirstName, LastName: row.LastName, ContactSnapshot: snapshot,
		Status: row.Status, ExclusionReason: row.ExclusionReason, EmailMessageID: pgconv.UUIDStringPtr(row.EmailMessageID),
		CreatedAt: row.CreatedAt.Time, QueuedAt: pgconv.TimestamptzToTimePtr(row.QueuedAt),
	}, nil
}

func encodeBindings(bindings map[string]any) ([]byte, error) {
	if bindings == nil {
		bindings = map[string]any{}
	}
	encoded, err := json.Marshal(bindings)
	if err != nil {
		return nil, fmt.Errorf("encode broadcast variable bindings: %w", err)
	}
	return encoded, nil
}

func decodeObject(data []byte, label string) (map[string]any, error) {
	value := map[string]any{}
	if len(data) == 0 {
		return value, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	return value, nil
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
