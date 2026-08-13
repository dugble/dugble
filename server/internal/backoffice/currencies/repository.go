package currencies

import (
	"context"
	"errors"
	"fmt"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ queries *dbsqlc.Queries }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, limit, offset int32) ([]Currency, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice currencies repository is not configured")
	}
	rows, e := r.queries.BackofficeListCurrencies(ctx, dbsqlc.BackofficeListCurrenciesParams{PageLimit: limit, PageOffset: offset})
	if e != nil {
		return nil, fmt.Errorf("list currencies: %w", e)
	}
	out := make([]Currency, 0, len(rows))
	for _, v := range rows {
		out = append(out, Currency{v.Code, v.MinorUnit, v.IsEnabled})
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, code string) (Currency, error) {
	v, e := r.queries.BackofficeGetCurrency(ctx, dbsqlc.BackofficeGetCurrencyParams{Code: code})
	if e != nil {
		return Currency{}, fmt.Errorf("get currency: %w", e)
	}
	return Currency{v.Code, v.MinorUnit, v.IsEnabled}, nil
}
func (r *Repository) Create(ctx context.Context, in CreateInput) (Currency, error) {
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	v, e := r.queries.BackofficeCreateCurrency(ctx, dbsqlc.BackofficeCreateCurrencyParams{Code: in.Code, MinorUnit: in.MinorUnit, IsEnabled: enabled})
	if e != nil {
		return Currency{}, fmt.Errorf("create currency: %w", e)
	}
	return Currency{v.Code, v.MinorUnit, v.IsEnabled}, nil
}
func (r *Repository) SetEnabled(ctx context.Context, code string, enabled bool) (Currency, error) {
	v, e := r.queries.BackofficeSetCurrencyEnabled(ctx, dbsqlc.BackofficeSetCurrencyEnabledParams{Code: code, IsEnabled: enabled})
	if e != nil {
		return Currency{}, fmt.Errorf("update currency: %w", e)
	}
	return Currency{v.Code, v.MinorUnit, v.IsEnabled}, nil
}
