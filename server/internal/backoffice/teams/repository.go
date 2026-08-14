package teams

import (
	"context"
	"fmt"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter) ([]Row, error) {
	rows, e := r.queries.BackofficeListTeams(ctx, dbsqlc.BackofficeListTeamsParams{Search: f.Query, Status: f.Status, PageLimit: f.Limit, PageOffset: f.Offset})
	if e != nil {
		return nil, fmt.Errorf("list teams: %w", e)
	}
	out := make([]Row, 0, len(rows))
	for _, v := range rows {
		out = append(out, Row{v.ID.String(), v.Name, v.Status, v.CreatedAt.Time})
	}
	return out, nil
}
func (r *Repository) Detail(ctx context.Context, id string) (Detail, error) {
	uid, e := uuid.Parse(id)
	if e != nil {
		return Detail{}, fmt.Errorf("parse team id: %w", e)
	}
	v, e := r.queries.BackofficeGetTeam(ctx, dbsqlc.BackofficeGetTeamParams{ID: uid})
	if e != nil {
		return Detail{}, fmt.Errorf("get team detail: %w", e)
	}
	d := Detail{Team: Row{v.ID.String(), v.Name, v.Status, v.CreatedAt.Time}}
	if e = r.loadMembers(ctx, uid, &d); e != nil {
		return Detail{}, e
	}
	if e = r.loadSMS(ctx, uid, &d); e != nil {
		return Detail{}, e
	}
	return d, nil
}
func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	uid, e := uuid.Parse(id)
	if e != nil {
		return fmt.Errorf("parse team id: %w", e)
	}
	n, e := r.queries.BackofficeUpdateTeamStatus(ctx, dbsqlc.BackofficeUpdateTeamStatusParams{ID: uid, Status: status})
	if e != nil {
		return fmt.Errorf("update team status: %w", e)
	}
	if n == 0 {
		return fmt.Errorf("update team status: %w", pgx.ErrNoRows)
	}
	return nil
}
func (r *Repository) loadMembers(ctx context.Context, id uuid.UUID, d *Detail) error {
	rows, e := r.queries.BackofficeListTeamMembers(ctx, dbsqlc.BackofficeListTeamMembersParams{TeamID: id})
	if e != nil {
		return fmt.Errorf("list team members: %w", e)
	}
	d.Members = make([]MemberRow, 0, len(rows))
	for _, v := range rows {
		d.Members = append(d.Members, MemberRow{v.UserID.String(), v.Email, v.Name, v.Role, v.Status, v.CreatedAt.Time})
	}
	return nil
}
func (r *Repository) loadSMS(ctx context.Context, id uuid.UUID, d *Detail) error {
	rows, e := r.queries.BackofficeListTeamSMS(ctx, dbsqlc.BackofficeListTeamSMSParams{TeamID: id})
	if e != nil {
		return fmt.Errorf("list team sms messages: %w", e)
	}
	d.SMS = make([]SMSRow, 0, len(rows))
	for _, v := range rows {
		d.SMS = append(d.SMS, SMSRow{v.ID.String(), v.TeamName, v.ToNumber, v.FromName, v.Status, v.ProviderID, v.ErrorMessage, v.CreatedAt.Time})
	}
	return nil
}
