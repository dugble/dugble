package productrates

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]ProductRate, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice product rates repository is not configured")
	}
	rows, err := r.queries.BackofficeListProductRates(ctx, dbsqlc.BackofficeListProductRatesParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list product rates: %w", err)
	}
	items := make([]ProductRate, 0, len(rows))
	for _, row := range rows {
		items = append(items, productRateFromSQLC(row))
	}
	return items, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (ProductRate, error) {
	row, err := r.queries.BackofficeGetProductRate(ctx, dbsqlc.BackofficeGetProductRateParams{ID: id})
	if err != nil {
		return ProductRate{}, err
	}
	return productRateFromSQLC(row), nil
}
func (r *Repository) Create(ctx context.Context, input CreateInput) (ProductRate, error) {
	row, err := r.queries.BackofficeCreateProductRate(ctx, dbsqlc.BackofficeCreateProductRateParams{
		Product:        input.Product,
		Meter:          input.Meter,
		BillingMarket:  input.BillingMarket,
		Tier:           input.Tier,
		Currency:       input.Currency,
		CostUnits:      input.CostUnits,
		EffectiveFrom:  pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil: nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return ProductRate{}, err
	}
	return productRateFromSQLC(row), nil
}
func (r *Repository) Close(ctx context.Context, id uuid.UUID, effectiveUntil time.Time) (ProductRate, error) {
	row, err := r.queries.BackofficeCloseProductRate(ctx, dbsqlc.BackofficeCloseProductRateParams{ID: id, EffectiveUntil: pgtype.Timestamptz{Time: effectiveUntil, Valid: true}})
	if err != nil {
		return ProductRate{}, err
	}
	return productRateFromSQLC(row), nil
}
func (r *Repository) Replace(ctx context.Context, id uuid.UUID, input CreateInput) (ProductRate, error) {
	row, err := r.queries.BackofficeReplaceProductRate(ctx, dbsqlc.BackofficeReplaceProductRateParams{
		TargetID: id, CostUnits: input.CostUnits,
		EffectiveFrom:  pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil: nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return ProductRate{}, err
	}
	return productRateFromSQLC(row), nil
}
func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
func productRateFromSQLC(row dbsqlc.ProductRate) ProductRate {
	var until *time.Time
	if row.EffectiveUntil.Valid {
		value := row.EffectiveUntil.Time
		until = &value
	}
	return ProductRate{ID: row.ID.String(),
		Product:       row.Product,
		Meter:         row.Meter,
		BillingMarket: row.BillingMarket,
		Tier:          row.Tier,
		Currency:      row.Currency,
		CostUnits:     row.CostUnits,
		EffectiveFrom: row.EffectiveFrom.Time, EffectiveUntil: until, CreatedAt: row.CreatedAt.Time}
}
