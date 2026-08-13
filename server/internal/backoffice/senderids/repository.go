package senderids

import (
	"context"
	"fmt"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter) ([]Row, error) {
	rows, e := r.queries.BackofficeListSenderIDs(ctx, dbsqlc.BackofficeListSenderIDsParams{Search: f.Query, Status: f.Status})
	if e != nil {
		return nil, fmt.Errorf("list sender ids: %w", e)
	}
	out := make([]Row, 0, len(rows))
	for _, v := range rows {
		out = append(out, Row{v.ID.String(), v.TeamName, v.Name, v.CountryCode, v.Status, v.CreatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Detail(ctx context.Context, id string) (Detail, error) {
	uid, e := uuid.Parse(id)
	if e != nil {
		return Detail{}, fmt.Errorf("parse sender id: %w", e)
	}
	v, e := r.queries.BackofficeGetSenderID(ctx, dbsqlc.BackofficeGetSenderIDParams{ID: uid})
	if e != nil {
		return Detail{}, fmt.Errorf("get sender id detail: %w", e)
	}
	purpose := ""
	if v.Purpose != nil {
		purpose = *v.Purpose
	}
	return Detail{ID: v.ID.String(), TeamID: v.TeamID.String(), TeamName: v.TeamName, Name: v.Name, CountryCode: v.CountryCode, Purpose: purpose, Status: v.Status, Provider: v.Provider, RejectionReason: v.RejectionReason, ApprovedAt: v.ApprovedAt, RejectedAt: v.RejectedAt, SuspendedAt: v.SuspendedAt, CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time}, nil
}
func (r *Repository) Approve(ctx context.Context, id string) error {
	uid, e := uuid.Parse(id)
	if e != nil {
		return fmt.Errorf("parse sender id: %w", e)
	}
	n, e := r.queries.BackofficeApproveSenderID(ctx, dbsqlc.BackofficeApproveSenderIDParams{ID: uid})
	if e != nil {
		return fmt.Errorf("approve sender id: %w", e)
	}
	if n == 0 {
		return fmt.Errorf("approve sender id: %w", pgx.ErrNoRows)
	}
	return nil
}
func (r *Repository) Reject(ctx context.Context, id, reason string) error {
	uid, e := uuid.Parse(id)
	if e != nil {
		return fmt.Errorf("parse sender id: %w", e)
	}
	n, e := r.queries.BackofficeRejectSenderID(ctx, dbsqlc.BackofficeRejectSenderIDParams{ID: uid, Reason: &reason})
	if e != nil {
		return fmt.Errorf("reject sender id: %w", e)
	}
	if n == 0 {
		return fmt.Errorf("reject sender id: %w", pgx.ErrNoRows)
	}
	return nil
}
