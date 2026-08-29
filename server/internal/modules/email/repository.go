package email

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

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var ErrNotFound = errors.New("email message not found")
var ErrNotCancelable = errors.New("email message is not a pending scheduled email")
var ErrSenderDomainNotFound = errors.New("sender domain not found")
var ErrActiveEmailTenantNotFound = errors.New("active email tenant not found")
var ErrSandboxRecipientNotFound = errors.New("sandbox recipient not found")

type SenderDomainRoute struct {
	ID           uuid.UUID
	Provider     string
	Region       string
	Status       string
	HealthStatus string
	Disabled     bool
}

type Repository struct {
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: dbsqlc.New(db)}
}

func (r *Repository) ResolveSandboxRecipientForToken(ctx context.Context, teamID, tokenID uuid.UUID) (string, bool, error) {
	row, err := r.queries.ResolveEmailSandboxRecipientForToken(ctx, dbsqlc.ResolveEmailSandboxRecipientForTokenParams{
		TokenID: tokenID,
		TeamID:  teamID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrSandboxRecipientNotFound
	}
	if err != nil {
		return "", false, fmt.Errorf("resolve sandbox recipient for team token: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(row.Email)), row.EmailVerified, nil
}

func (r *Repository) ResolveActiveCustomerRouteTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID, provider, region, stream string) (platformemail.DeliveryRoute, error) {
	if tx == nil {
		return platformemail.DeliveryRoute{}, errors.New("email route transaction is required")
	}
	tenantName, err := r.queries.WithTx(tx).GetActiveEmailTenantExternalName(ctx, dbsqlc.GetActiveEmailTenantExternalNameParams{
		TeamID: teamID, Provider: strings.ToLower(strings.TrimSpace(provider)), Region: strings.ToLower(strings.TrimSpace(region)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return platformemail.DeliveryRoute{}, ErrActiveEmailTenantNotFound
	}
	if err != nil {
		return platformemail.DeliveryRoute{}, fmt.Errorf("resolve active customer email tenant: %w", err)
	}
	route, err := platformemail.CustomerDeliveryRoute(stream, tenantName)
	if err != nil {
		return platformemail.DeliveryRoute{}, fmt.Errorf("build customer email route: %w", err)
	}
	return route, nil
}

func (r *Repository) CreateTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID, req validatedSend) (Message, error) {
	recipients, err := json.Marshal(map[string][]EmailAddress{"to": req.To, "cc": req.CC, "bcc": req.BCC, "reply_to": req.ReplyTo})
	if err != nil {
		return Message{}, fmt.Errorf("encode email recipients: %w", err)
	}
	headers, err := json.Marshal(platformemail.PersistDeliveryRoute(req.Headers, req.DeliveryRoute))
	if err != nil {
		return Message{}, fmt.Errorf("encode email headers: %w", err)
	}
	attachments, err := json.Marshal(req.Attachments)
	if err != nil {
		return Message{}, fmt.Errorf("encode email attachments: %w", err)
	}
	tags, err := json.Marshal(req.Tags)
	if err != nil {
		return Message{}, fmt.Errorf("encode email tags: %w", err)
	}
	row, err := r.queries.WithTx(tx).CreateEmailMessage(ctx, dbsqlc.CreateEmailMessageParams{
		TeamID:           teamID,
		SenderDomainID:   req.SenderDomainID,
		DeliveryProvider: req.Provider,
		ProviderRegion:   req.ProviderRegion,
		MessageType:      req.MessageType,
		FromEmail:        req.FromEmail,
		FromName:         req.FromName,
		ReplyToEmail:     req.ReplyToEmail,
		ToEmail:          req.ToEmail,
		ToName:           req.ToName,
		Subject:          req.Subject,
		HtmlBody:         req.HTMLBody,
		TextBody:         req.TextBody,
		Metadata:         req.Metadata,
		Recipients:       recipients,
		Headers:          headers,
		Attachments:      attachments,
		Tags:             tags,
		ScheduledAt:      pgconv.NullableTimestamptz(req.ScheduledAt),
	})
	if err != nil {
		return Message{}, fmt.Errorf("create email message: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) ListEvents(ctx context.Context, teamID, messageID uuid.UUID, limit, offset int32) ([]Event, error) {
	rows, err := r.queries.ListEmailMessageEvents(ctx, dbsqlc.ListEmailMessageEventsParams{
		MessageID: messageID, TeamID: teamID, LimitCount: limit, OffsetCount: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list email events: %w", err)
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

func (r *Repository) ResolveSenderDomain(ctx context.Context, teamID uuid.UUID, domainName string) (SenderDomainRoute, error) {
	row, err := r.queries.ResolveEmailSenderDomain(ctx, dbsqlc.ResolveEmailSenderDomainParams{
		TeamID:     teamID,
		DomainName: domainName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SenderDomainRoute{}, ErrSenderDomainNotFound
	}
	if err != nil {
		return SenderDomainRoute{}, fmt.Errorf("resolve sender domain: %w", err)
	}
	route := SenderDomainRoute{
		ID:           row.ID,
		Provider:     row.Provider,
		Region:       row.ProviderRegion,
		Status:       row.Status,
		HealthStatus: row.HealthStatus,
	}
	if route.HealthStatus == "degraded" {
		route.Status = "degraded"
	}
	return route, nil
}

func (r *Repository) CancelTx(ctx context.Context, tx pgx.Tx, id, teamID uuid.UUID) error {
	row, err := r.queries.WithTx(tx).GetEmailMessageScheduleForUpdate(ctx, dbsqlc.GetEmailMessageScheduleForUpdateParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock email message for cancellation: %w", err)
	}
	if row.Status != StatusQueued || !row.ScheduledAt.Valid || !row.ScheduledAt.Time.After(time.Now().UTC()) {
		return ErrNotCancelable
	}
	if err := r.queries.WithTx(tx).CancelEmailMessage(ctx, dbsqlc.CancelEmailMessageParams{ID: id, TeamID: teamID}); err != nil {
		return fmt.Errorf("cancel email message: %w", err)
	}
	return nil
}

func (r *Repository) RescheduleTx(ctx context.Context, tx pgx.Tx, id, teamID uuid.UUID, scheduledAt time.Time) error {
	row, err := r.queries.WithTx(tx).GetEmailMessageScheduleForUpdate(ctx, dbsqlc.GetEmailMessageScheduleForUpdateParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock email message for rescheduling: %w", err)
	}
	if row.Status != StatusQueued || !row.ScheduledAt.Valid || !row.ScheduledAt.Time.After(time.Now().UTC()) {
		return ErrNotCancelable
	}
	if err := r.queries.WithTx(tx).RescheduleEmailMessage(ctx, dbsqlc.RescheduleEmailMessageParams{
		ID: id, TeamID: teamID, ScheduledAt: pgconv.TimestamptzFromTime(scheduledAt),
	}); err != nil {
		return fmt.Errorf("reschedule email message: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id, teamID uuid.UUID) (Message, error) {
	row, err := r.queries.GetEmailMessage(ctx, dbsqlc.GetEmailMessageParams{ID: id, TeamID: teamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, fmt.Errorf("get email message: %w", err)
	}
	return messageFromSQLC(row), nil
}

func (r *Repository) List(ctx context.Context, teamID uuid.UUID, limit, offset int32) ([]MessageSummary, error) {
	rows, err := r.queries.ListEmailMessages(ctx, dbsqlc.ListEmailMessagesParams{
		TeamID: teamID, LimitCount: limit, OffsetCount: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list email messages: %w", err)
	}
	messages := make([]MessageSummary, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, MessageSummary{
			ID: row.ID.String(), ToEmail: row.ToEmail, ToName: row.ToName, Subject: row.Subject,
			Status: row.Status, Provider: row.Provider, QueuedAt: row.QueuedAt.Time,
			SubmittedAt: pgconv.TimestamptzToTimePtr(row.SubmittedAt), DeliveredAt: pgconv.TimestamptzToTimePtr(row.DeliveredAt),
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return messages, nil
}

func messageFromSQLC(row dbsqlc.EmailMessage) Message {
	message := Message{
		ID:                row.ID.String(),
		TeamID:            row.TeamID.String(),
		MessageType:       row.MessageType,
		FromEmail:         row.FromEmail,
		FromName:          row.FromName,
		ReplyToEmail:      row.ReplyToEmail,
		ToEmail:           row.ToEmail,
		ToName:            row.ToName,
		Subject:           row.Subject,
		HTMLBody:          row.HtmlBody,
		TextBody:          row.TextBody,
		Status:            row.Status,
		Provider:          row.Provider,
		ProviderMessageID: row.ProviderMessageID,
		ErrorCode:         row.ErrorCode,
		ErrorMessage:      row.ErrorMessage,
		Metadata:          json.RawMessage(row.Metadata),
		ScheduledAt:       pgconv.TimestamptzToTimePtr(row.ScheduledAt),
		QueuedAt:          row.QueuedAt.Time,
		ProcessingAt:      pgconv.TimestamptzToTimePtr(row.ProcessingAt),
		SubmittedAt:       pgconv.TimestamptzToTimePtr(row.SubmittedAt),
		DeliveredAt:       pgconv.TimestamptzToTimePtr(row.DeliveredAt),
		FailedAt:          pgconv.TimestamptzToTimePtr(row.FailedAt),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
	var recipients struct {
		To      []EmailAddress `json:"to"`
		CC      []EmailAddress `json:"cc"`
		BCC     []EmailAddress `json:"bcc"`
		ReplyTo []EmailAddress `json:"reply_to"`
	}
	_ = json.Unmarshal(row.Recipients, &recipients)
	message.To, message.CC, message.BCC, message.ReplyTo = recipients.To, recipients.CC, recipients.BCC, recipients.ReplyTo
	var persistedHeaders map[string]string
	_ = json.Unmarshal(row.Headers, &persistedHeaders)
	_, message.Headers = platformemail.ExtractDeliveryRoute(persistedHeaders)
	_ = json.Unmarshal(row.Attachments, &message.Attachments)
	_ = json.Unmarshal(row.Tags, &message.Tags)
	return message
}

func (r *Repository) GetAnalytics(ctx context.Context, teamID uuid.UUID) (AnalyticsResponse, error) {
	windows := make([]AnalyticsWindow, 0, 3)
	for _, days := range []int32{7, 30, 90} {
		points, err := r.emailAnalyticsSeries(ctx, teamID, days)
		if err != nil {
			return AnalyticsResponse{}, err
		}
		windows = append(windows, AnalyticsWindow{Days: days, Rates: emailRates(points), Series: points})
	}
	return AnalyticsResponse{Object: "email.analytics", Windows: windows}, nil
}

func (r *Repository) emailAnalyticsSeries(ctx context.Context, teamID uuid.UUID, days int32) ([]AnalyticsPoint, error) {
	rows, err := r.queries.GetEmailAnalyticsSeries(ctx, dbsqlc.GetEmailAnalyticsSeriesParams{
		WindowDays: days,
		TeamID:     teamID,
	})
	if err != nil {
		return nil, fmt.Errorf("get email analytics series: %w", err)
	}
	points := make([]AnalyticsPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, AnalyticsPoint{
			Date: row.Date, Total: row.Total, Delivered: row.Delivered,
			Opened: row.Opened, Clicked: row.Clicked, Bounced: row.Bounced,
		})
	}
	return points, nil
}

func emailRates(points []AnalyticsPoint) []AnalyticsRate {
	var total, delivered, opened, clicked, bounced int64
	for _, point := range points {
		total += point.Total
		delivered += point.Delivered
		opened += point.Opened
		clicked += point.Clicked
		bounced += point.Bounced
	}
	return []AnalyticsRate{{Name: "delivery_rate", Value: percentage(delivered, total)}, {Name: "open_rate", Value: percentage(opened, delivered)}, {Name: "click_rate", Value: percentage(clicked, delivered)}, {Name: "bounce_rate", Value: percentage(bounced, total)}}
}

func percentage(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total)
}
