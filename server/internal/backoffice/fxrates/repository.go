package fxrates

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

func (r *Repository) List(ctx context.Context, limit, offset int32) ([]FXRate, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice FX rates repository is not configured")
	}
	rows, err := r.queries.BackofficeListFXRates(ctx, dbsqlc.BackofficeListFXRatesParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list FX rates: %w", err)
	}
	out := make([]FXRate, 0, len(rows))
	for _, row := range rows {
		out = append(out, fxRateFromSQLC(row))
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (FXRate, error) {
	row, err := r.queries.BackofficeGetFXRate(ctx, dbsqlc.BackofficeGetFXRateParams{ID: id})
	if err != nil {
		return FXRate{}, err
	}
	return fxRateFromSQLC(row), nil
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (FXRate, error) {
	rate, err := numericRate(in.Rate)
	if err != nil {
		return FXRate{}, err
	}
	row, err := r.queries.BackofficeCreateFXRate(ctx, dbsqlc.BackofficeCreateFXRateParams{
		BaseCurrency:   in.BaseCurrency,
		QuoteCurrency:  in.QuoteCurrency,
		Rate:           rate,
		EffectiveFrom:  pgtype.Timestamptz{Time: in.EffectiveFrom, Valid: true},
		EffectiveUntil: nullableTime(in.EffectiveUntil),
	})
	if err != nil {
		return FXRate{}, err
	}
	return fxRateFromSQLC(row), nil
}

func (r *Repository) Close(ctx context.Context, id uuid.UUID, effectiveUntil time.Time) (FXRate, error) {
	row, err := r.queries.BackofficeCloseFXRate(ctx, dbsqlc.BackofficeCloseFXRateParams{
		ID: id, EffectiveUntil: pgtype.Timestamptz{Time: effectiveUntil, Valid: true},
	})
	if err != nil {
		return FXRate{}, err
	}
	return fxRateFromSQLC(row), nil
}

func (r *Repository) Replace(ctx context.Context, id uuid.UUID, in ReplaceInput) (FXRate, error) {
	rate, err := numericRate(in.Rate)
	if err != nil {
		return FXRate{}, err
	}
	row, err := r.queries.BackofficeReplaceFXRate(ctx, dbsqlc.BackofficeReplaceFXRateParams{
		TargetID:      id,
		Rate:          rate,
		EffectiveFrom: pgtype.Timestamptz{Time: in.EffectiveFrom, Valid: true},
	})
	if err != nil {
		return FXRate{}, err
	}
	return fxRateFromSQLC(row), nil
}

func numericRate(value string) (pgtype.Numeric, error) {
	var rate pgtype.Numeric
	if err := rate.Scan(value); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("parse FX rate: %w", err)
	}
	return rate, nil
}

func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func fxRateFromSQLC(row dbsqlc.FxRate) FXRate {
	var until *time.Time
	if row.EffectiveUntil.Valid {
		v := row.EffectiveUntil.Time
		until = &v
	}
	value, _ := row.Rate.Value()
	return FXRate{ID: row.ID.String(), BaseCurrency: row.BaseCurrency, QuoteCurrency: row.QuoteCurrency, Rate: fmt.Sprint(value), EffectiveFrom: row.EffectiveFrom.Time, EffectiveUntil: until, CreatedAt: row.CreatedAt.Time}
}
