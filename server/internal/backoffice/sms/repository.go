package sms

import (
	"context"
	"fmt"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter) ([]Row, error) {
	rows, e := r.queries.BackofficeListSMS(ctx, dbsqlc.BackofficeListSMSParams{Search: f.Query, Status: f.Status})
	if e != nil {
		return nil, fmt.Errorf("list sms messages: %w", e)
	}
	out := make([]Row, 0, len(rows))
	for _, v := range rows {
		out = append(out, Row{v.ID.String(), v.TeamName, v.ToNumber, v.FromName, v.Status, v.ProviderID, v.ErrorMessage, v.CreatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Detail(ctx context.Context, id string) (Detail, error) {
	uid, e := uuid.Parse(id)
	if e != nil {
		return Detail{}, fmt.Errorf("parse sms id: %w", e)
	}
	v, e := r.queries.BackofficeGetSMS(ctx, dbsqlc.BackofficeGetSMSParams{ID: uid})
	if e != nil {
		return Detail{}, fmt.Errorf("get sms detail: %w", e)
	}
	return Detail{ID: v.ID.String(), TeamID: v.TeamID.String(), TeamName: v.TeamName, SenderID: v.SenderID, ToNumber: v.ToNumber, FromName: v.FromName, Body: v.Body, Status: v.Status, ProviderID: v.ProviderID, ProviderMessageID: v.ProviderMessageID, Segments: v.Segments, ErrorMessage: v.ErrorMessage, Metadata: v.Metadata, SubmittedAt: v.SubmittedAt, DeliveredAt: v.DeliveredAt, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}, nil
}
