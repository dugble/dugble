package payments

import (
	"context"
	"fmt"
	"time"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db, queries: dbsqlc.New(db)} }
func (r *Repository) List(ctx context.Context, f Filter, teamID *uuid.UUID) ([]Payment, error) {
	rows, e := r.queries.BackofficeListPayments(ctx, dbsqlc.BackofficeListPaymentsParams{TeamID: teamID, Status: f.Status, Provider: f.Provider, PageLimit: f.Limit, PageOffset: f.Offset})
	if e != nil {
		return nil, fmt.Errorf("list payments: %w", e)
	}
	out := make([]Payment, 0, len(rows))
	for _, v := range rows {
		out = append(out, payment(v.ID, v.TeamID, v.TeamName, v.Provider, v.ClientReference, v.Currency, v.AmountUnits, v.Status, v.ProviderTransactionID, v.PaidAt.Time, v.PaidAt.Valid, v.CreatedAt.Time, v.UpdatedAt.Time))
	}
	return out, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Payment, error) {
	v, e := r.queries.BackofficeGetPayment(ctx, dbsqlc.BackofficeGetPaymentParams{ID: id})
	if e != nil {
		return Payment{}, e
	}
	return payment(v.ID, v.TeamID, v.TeamName, v.Provider, v.ClientReference, v.Currency, v.AmountUnits, v.Status, v.ProviderTransactionID, v.PaidAt.Time, v.PaidAt.Valid, v.CreatedAt.Time, v.UpdatedAt.Time), nil
}

// Reconcile marks one verified pending payment paid, credits its wallet, and
// records the administrator's reason in the same transaction.
func (r *Repository) Reconcile(ctx context.Context, payment Payment, providerTransactionID, reason string, actorID uuid.UUID, sessionID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin payment reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	paymentID, _ := uuid.Parse(payment.ID)
	teamID, _ := uuid.Parse(payment.TeamID)
	result, err := tx.Exec(ctx, `UPDATE payment_transactions SET status='paid',provider_transaction_id=$2,paid_at=now(),updated_at=now() WHERE id=$1 AND status='pending'`, paymentID, providerTransactionID)
	if err != nil {
		return fmt.Errorf("mark payment reconciled: %w", err)
	}
	if result.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	result, err = tx.Exec(ctx, `INSERT INTO wallet_ledger(team_id,payment_transaction_id,amount_units,transaction_type,reference_id) VALUES($1,$2,$3,'deposit',$4) ON CONFLICT(payment_transaction_id) DO NOTHING`, teamID, paymentID, payment.AmountUnits, payment.ClientReference)
	if err != nil {
		return fmt.Errorf("record reconciled deposit: %w", err)
	}
	if result.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	result, err = tx.Exec(ctx, `UPDATE team_wallets SET balance_units=balance_units+$1,updated_at=now() WHERE team_id=$2`, payment.AmountUnits, teamID)
	if err != nil {
		return fmt.Errorf("credit reconciled payment: %w", err)
	}
	if result.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(team_id,actor_type,actor_user_id,actor_session_id,action,resource_type,resource_id,metadata) VALUES($1,'user',$2,NULLIF($3,''),'payment.reconcile','payment',$4,jsonb_build_object('provider_transaction_id',$5,'amount_units',$6,'currency',$7,'reason',$8))`, teamID, actorID, sessionID, payment.ID, providerTransactionID, payment.AmountUnits, payment.Currency, reason)
	if err != nil {
		return fmt.Errorf("audit payment reconciliation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit payment reconciliation: %w", err)
	}
	return nil
}
func payment(id, team uuid.UUID, name, provider, reference, currency string, amount int64, status string, providerID *string, paid time.Time, paidValid bool, created, updated time.Time) Payment {
	var paidAt *time.Time
	if paidValid {
		paidAt = &paid
	}
	return Payment{ID: id.String(), TeamID: team.String(), TeamName: name, Provider: provider, ClientReference: reference, Currency: currency, AmountUnits: amount, Status: status, ProviderTransactionID: providerID, PaidAt: paidAt, CreatedAt: created, UpdatedAt: updated}
}
