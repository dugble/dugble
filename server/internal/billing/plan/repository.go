package plan

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }

func (r *Repository) List(ctx context.Context, teamID uuid.UUID) ([]Plan, error) {
	rows, err := r.queries.ListPlansForTeam(ctx, dbsqlc.ListPlansForTeamParams{TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("list plans for team: %w", err)
	}
	plans := make([]Plan, 0, len(rows))
	for _, row := range rows {
		item := Plan{
			Code: row.Code, Name: row.Name, Available: row.IsAvailable,
			Current:     row.Code == row.CurrentPlanCode,
			Pending:     row.PendingPlanCode != nil && row.Code == *row.PendingPlanCode,
			EffectiveAt: row.CurrentPeriodEnd.Time,
		}
		if row.IsAvailable {
			item.Price = &Price{ID: row.PlanPriceID.String(), Currency: row.Currency, AmountUnits: row.AmountUnits}
		}
		plans = append(plans, item)
	}
	return plans, nil
}
