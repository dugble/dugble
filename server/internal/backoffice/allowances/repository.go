package allowances

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]AllowancePolicy, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice allowance policies repository is not configured")
	}
	rows, err := r.queries.BackofficeListAllowancePolicies(ctx, dbsqlc.BackofficeListAllowancePoliciesParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list allowance policies: %w", err)
	}
	items := make([]AllowancePolicy, 0, len(rows))
	for _, row := range rows {
		items = append(items, allowancePolicyFromSQLC(row))
	}
	return items, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (AllowancePolicy, error) {
	row, err := r.queries.BackofficeGetAllowancePolicy(ctx, dbsqlc.BackofficeGetAllowancePolicyParams{ID: id})
	if err != nil {
		return AllowancePolicy{}, err
	}
	return allowancePolicyFromSQLC(row), nil
}
func (r *Repository) Create(ctx context.Context, input CreateInput) (AllowancePolicy, error) {
	row, err := r.queries.BackofficeCreateAllowancePolicy(ctx, dbsqlc.BackofficeCreateAllowancePolicyParams{
		Product:          input.Product,
		Meter:            input.Meter,
		BillingMarket:    input.BillingMarket,
		Tier:             input.Tier,
		IncludedQuantity: input.IncludedQuantity,
		Cadence:          input.Cadence,
		EffectiveFrom:    pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil:   nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return AllowancePolicy{}, err
	}
	return allowancePolicyFromSQLC(row), nil
}
func (r *Repository) Close(ctx context.Context, id uuid.UUID, effectiveUntil time.Time) (AllowancePolicy, error) {
	row, err := r.queries.BackofficeCloseAllowancePolicy(ctx, dbsqlc.BackofficeCloseAllowancePolicyParams{ID: id, EffectiveUntil: pgtype.Timestamptz{Time: effectiveUntil, Valid: true}})
	if err != nil {
		return AllowancePolicy{}, err
	}
	return allowancePolicyFromSQLC(row), nil
}
func (r *Repository) Replace(ctx context.Context, id uuid.UUID, input CreateInput) (AllowancePolicy, error) {
	row, err := r.queries.BackofficeReplaceAllowancePolicy(ctx, dbsqlc.BackofficeReplaceAllowancePolicyParams{
		TargetID: id, IncludedQuantity: input.IncludedQuantity,
		EffectiveFrom:  pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil: nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return AllowancePolicy{}, err
	}
	return allowancePolicyFromSQLC(row), nil
}
func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
func allowancePolicyFromSQLC(row dbsqlc.AllowancePolicy) AllowancePolicy {
	var until *time.Time
	if row.EffectiveUntil.Valid {
		value := row.EffectiveUntil.Time
		until = &value
	}
	return AllowancePolicy{ID: row.ID.String(),
		Product:          row.Product,
		Meter:            row.Meter,
		BillingMarket:    row.BillingMarket,
		Tier:             row.Tier,
		IncludedQuantity: row.IncludedQuantity,
		Cadence:          row.Cadence,
		EffectiveFrom:    row.EffectiveFrom.Time, EffectiveUntil: until, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
