package markets

import (
	"context"
	"errors"
	"fmt"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func fromRow(code, currency string, minor int16, enabled bool) Market {
	return Market{Code: code, Currency: currency, MinorUnit: minor, IsEnabled: enabled}
}
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]Market, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice billing markets repository is not configured")
	}
	rows, e := r.queries.BackofficeListBillingMarkets(ctx, dbsqlc.BackofficeListBillingMarketsParams{PageLimit: limit, PageOffset: offset})
	if e != nil {
		return nil, fmt.Errorf("list billing markets: %w", e)
	}
	out := make([]Market, 0, len(rows))
	for _, v := range rows {
		out = append(out, fromRow(v.Code, v.Currency, v.MinorUnit, v.IsEnabled))
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, code string) (Market, error) {
	v, e := r.queries.BackofficeGetBillingMarket(ctx, dbsqlc.BackofficeGetBillingMarketParams{Code: code})
	if e != nil {
		return Market{}, e
	}
	return fromRow(v.Code, v.Currency, v.MinorUnit, v.IsEnabled), nil
}
func (r *Repository) GetCurrency(ctx context.Context, code string) (bool, error) {
	v, e := r.queries.BackofficeGetCurrency(ctx, dbsqlc.BackofficeGetCurrencyParams{Code: code})
	if e != nil {
		return false, fmt.Errorf("get market currency: %w", e)
	}
	return v.IsEnabled, nil
}
func (r *Repository) Create(ctx context.Context, in CreateInput) (Market, error) {
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	e := r.queries.BackofficeCreateBillingMarket(ctx, dbsqlc.BackofficeCreateBillingMarketParams{Code: in.Code, Currency: in.Currency, IsEnabled: enabled})
	if e != nil {
		return Market{}, fmt.Errorf("create billing market: %w", e)
	}
	return r.Get(ctx, in.Code)
}
func (r *Repository) SetEnabled(ctx context.Context, code string, enabled bool) (Market, error) {
	n, e := r.queries.BackofficeSetBillingMarketEnabled(ctx, dbsqlc.BackofficeSetBillingMarketEnabledParams{Code: code, IsEnabled: enabled})
	if e != nil {
		return Market{}, fmt.Errorf("update billing market: %w", e)
	}
	if n == 0 {
		return Market{}, pgx.ErrNoRows
	}
	return r.Get(ctx, code)
}
