package email

import (
	"context"
	"fmt"
	"time"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter) ([]Row, error) {
	rows, err := r.queries.BackofficeListEmailMessages(ctx, dbsqlc.BackofficeListEmailMessagesParams{Search: f.Query, Status: f.Status})
	if err != nil {
		return nil, fmt.Errorf("list backoffice email messages: %w", err)
	}
	out := make([]Row, 0, len(rows))
	for _, v := range rows {
		out = append(out, Row{v.ID.String(), v.TeamName, v.FromEmail, v.ToEmail, v.Subject, v.Status, v.Provider, v.ErrorMessage, v.CreatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Detail(ctx context.Context, id uuid.UUID) (Detail, error) {
	v, err := r.queries.BackofficeGetEmailMessage(ctx, dbsqlc.BackofficeGetEmailMessageParams{ID: id})
	if err != nil {
		return Detail{}, fmt.Errorf("get backoffice email message: %w", err)
	}
	rows, err := r.queries.BackofficeListEmailRecipients(ctx, dbsqlc.BackofficeListEmailRecipientsParams{MessageID: id})
	if err != nil {
		return Detail{}, fmt.Errorf("list backoffice email recipients: %w", err)
	}
	d := Detail{ID: v.ID.String(), TeamID: v.TeamID.String(), TeamName: v.TeamName, MessageType: v.MessageType, FromEmail: v.FromEmail, ToEmail: v.ToEmail, Subject: v.Subject, Status: v.Status, DeliveryProvider: v.DeliveryProvider, ProviderRegion: v.ProviderRegion, Metadata: string(v.Metadata), RecipientsJSON: string(v.Recipients), Headers: string(v.Headers), Attachments: string(v.Attachments), Tags: string(v.Tags), QueuedAt: v.QueuedAt.Time, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}
	if v.SenderDomainID != nil {
		d.SenderDomainID = v.SenderDomainID.String()
	}
	if v.FromName != nil {
		d.FromName = *v.FromName
	}
	if v.ReplyToEmail != nil {
		d.ReplyToEmail = *v.ReplyToEmail
	}
	if v.ToName != nil {
		d.ToName = *v.ToName
	}
	if v.HtmlBody != nil {
		d.HTMLBody = *v.HtmlBody
	}
	if v.TextBody != nil {
		d.TextBody = *v.TextBody
	}
	if v.Provider != nil {
		d.Provider = *v.Provider
	}
	if v.ProviderMessageID != nil {
		d.ProviderMessageID = *v.ProviderMessageID
	}
	if v.ErrorCode != nil {
		d.ErrorCode = *v.ErrorCode
	}
	if v.ErrorMessage != nil {
		d.ErrorMessage = *v.ErrorMessage
	}
	d.ScheduledAt = timePtr(v.ScheduledAt)
	d.ProcessingAt = timePtr(v.ProcessingAt)
	d.SubmittedAt = timePtr(v.SubmittedAt)
	d.DeliveredAt = timePtr(v.DeliveredAt)
	d.FailedAt = timePtr(v.FailedAt)
	for _, x := range rows {
		recipient := Recipient{Email: x.RecipientEmail, Type: x.RecipientType, Status: x.Status, LastEventAt: timePtr(x.LastEventAt), DeliveredAt: timePtr(x.DeliveredAt), FailedAt: timePtr(x.FailedAt)}
		if x.LastEventType != nil {
			recipient.LastEvent = *x.LastEventType
		}
		if x.ErrorCode != nil {
			recipient.ErrorCode = *x.ErrorCode
		}
		if x.ErrorMessage != nil {
			recipient.ErrorMessage = *x.ErrorMessage
		}
		d.Recipients = append(d.Recipients, recipient)
	}
	return d, nil
}
func timePtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
