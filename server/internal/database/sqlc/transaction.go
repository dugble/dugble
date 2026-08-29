package sqlc

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrTransactionsUnsupported = errors.New("sqlc database connection does not support transactions")

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// Begin starts a transaction using the connection supplied to New. Keeping
// transaction access on Queries lets module repositories retain sqlc as their
// only database dependency and bind generated queries with WithTx.
func (q *Queries) Begin(ctx context.Context) (pgx.Tx, error) {
	if q == nil {
		return nil, ErrTransactionsUnsupported
	}
	beginner, ok := q.db.(transactionBeginner)
	if !ok {
		return nil, ErrTransactionsUnsupported
	}
	return beginner.Begin(ctx)
}
