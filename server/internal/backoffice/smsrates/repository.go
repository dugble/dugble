package smsrates

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
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]SMSRate, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice SMS rates repository is not configured")
	}
	rows, err := r.queries.BackofficeListSMSRates(ctx, dbsqlc.BackofficeListSMSRatesParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list SMS rates: %w", err)
	}
	items := make([]SMSRate, 0, len(rows))
	for _, row := range rows {
		items = append(items, smsRateFromSQLC(row))
	}
	return items, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (SMSRate, error) {
	row, err := r.queries.BackofficeGetSMSRate(ctx, dbsqlc.BackofficeGetSMSRateParams{ID: id})
	if err != nil {
		return SMSRate{}, err
	}
	return smsRateFromSQLC(row), nil
}
func (r *Repository) Create(ctx context.Context, input CreateInput) (SMSRate, error) {
	row, err := r.queries.BackofficeCreateSMSRate(ctx, dbsqlc.BackofficeCreateSMSRateParams{
		DestinationCountry: input.DestinationCountry,
		RouteType:          input.RouteType,
		Tier:               input.Tier,
		Currency:           input.Currency,
		CostUnits:          input.CostUnits,
		EffectiveFrom:      pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil:     nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return SMSRate{}, err
	}
	return smsRateFromSQLC(row), nil
}
func (r *Repository) Close(ctx context.Context, id uuid.UUID, effectiveUntil time.Time) (SMSRate, error) {
	row, err := r.queries.BackofficeCloseSMSRate(ctx, dbsqlc.BackofficeCloseSMSRateParams{ID: id, EffectiveUntil: pgtype.Timestamptz{Time: effectiveUntil, Valid: true}})
	if err != nil {
		return SMSRate{}, err
	}
	return smsRateFromSQLC(row), nil
}
func (r *Repository) Replace(ctx context.Context, id uuid.UUID, input CreateInput) (SMSRate, error) {
	row, err := r.queries.BackofficeReplaceSMSRate(ctx, dbsqlc.BackofficeReplaceSMSRateParams{
		TargetID: id, CostUnits: input.CostUnits,
		EffectiveFrom:  pgtype.Timestamptz{Time: input.EffectiveFrom, Valid: true},
		EffectiveUntil: nullableTime(input.EffectiveUntil),
	})
	if err != nil {
		return SMSRate{}, err
	}
	return smsRateFromSQLC(row), nil
}
func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
func smsRateFromSQLC(row dbsqlc.SmsRate) SMSRate {
	var until *time.Time
	if row.EffectiveUntil.Valid {
		value := row.EffectiveUntil.Time
		until = &value
	}
	return SMSRate{ID: row.ID.String(),
		DestinationCountry: row.DestinationCountry,
		RouteType:          row.RouteType,
		Tier:               row.Tier,
		Currency:           row.Currency,
		CostUnits:          row.CostUnits,
		EffectiveFrom:      row.EffectiveFrom.Time, EffectiveUntil: until, CreatedAt: row.CreatedAt.Time}
}
