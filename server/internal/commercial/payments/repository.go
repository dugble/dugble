package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db, queries: dbsqlc.New(db)} }

func (r *Repository) Create(ctx context.Context, teamID uuid.UUID, input CreateInput) (Transaction, error) {
	row, err := r.queries.CreatePaymentTransaction(ctx, dbsqlc.CreatePaymentTransactionParams{
		TeamID:          teamID,
		Provider:        input.Provider,
		ClientReference: input.ClientReference,
		Currency:        input.Currency,
		AmountUnits:     input.AmountUnits,
	})
	if err != nil {
		return Transaction{}, fmt.Errorf("create payment transaction: %w", err)
	}
	return transactionFromSQLC(row), nil
}

func (r *Repository) GetByClientReference(ctx context.Context, provider, reference string) (Transaction, error) {
	row, err := r.queries.GetPaymentTransactionByClientReference(ctx, dbsqlc.GetPaymentTransactionByClientReferenceParams{Provider: provider, ClientReference: reference})
	if err != nil {
		return Transaction{}, fmt.Errorf("get payment transaction: %w", err)
	}
	return transactionFromSQLC(row), nil
}

func (r *Repository) MarkFailed(ctx context.Context, transaction Transaction) (Transaction, error) {
	id, err := uuid.Parse(transaction.ID)
	if err != nil {
		return Transaction{}, fmt.Errorf("parse payment transaction id: %w", err)
	}
	teamID, err := uuid.Parse(transaction.TeamID)
	if err != nil {
		return Transaction{}, fmt.Errorf("parse payment team id: %w", err)
	}
	row, err := r.queries.MarkPaymentTransactionFailed(ctx, dbsqlc.MarkPaymentTransactionFailedParams{ID: id, TeamID: teamID})
	if err != nil {
		return Transaction{}, fmt.Errorf("mark payment failed: %w", err)
	}
	return transactionFromSQLC(row), nil
}

func (r *Repository) ListRecipients(ctx context.Context, teamID uuid.UUID) ([]Recipient, error) {
	rows, err := r.queries.ListActiveTeamOwnerRecipients(ctx, dbsqlc.ListActiveTeamOwnerRecipientsParams{TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("list payment recipients: %w", err)
	}
	recipients := make([]Recipient, 0, len(rows))
	for _, row := range rows {
		recipients = append(recipients, Recipient{Name: row.Name, Email: row.Email, TeamName: row.TeamName})
	}
	return recipients, nil
}

// MarkPaidAndCredit records the provider result and credits the wallet in one
// database transaction. The ledger uniqueness constraints make retries safe.
func (r *Repository) MarkPaidAndCredit(ctx context.Context, transaction Transaction, providerTransactionID string) (Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin payment completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, _ := uuid.Parse(transaction.ID)
	teamID, _ := uuid.Parse(transaction.TeamID)
	row, err := dbsqlc.New(tx).MarkPaymentTransactionPaid(ctx, dbsqlc.MarkPaymentTransactionPaidParams{ProviderTransactionID: &providerTransactionID, ID: id, TeamID: teamID, AmountUnits: transaction.AmountUnits})
	if errors.Is(err, pgx.ErrNoRows) {
		current, getErr := dbsqlc.New(tx).GetPaymentTransactionByID(ctx, dbsqlc.GetPaymentTransactionByIDParams{ID: id, TeamID: teamID})
		if getErr == nil && current.Status == StatusPaid && current.ProviderTransactionID != nil && *current.ProviderTransactionID == providerTransactionID {
			return transactionFromSQLC(current), nil
		}
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("mark payment paid: %w", err)
	}
	insert, err := tx.Exec(ctx, `INSERT INTO wallet_ledger (team_id, payment_transaction_id, amount_units, transaction_type, reference_id) VALUES ($1,$2,$3,'deposit',$4) ON CONFLICT (payment_transaction_id) DO NOTHING`, teamID, id, transaction.AmountUnits, transaction.ClientReference)
	if err != nil {
		return Transaction{}, fmt.Errorf("record payment deposit: %w", err)
	}
	if insert.RowsAffected() != 1 {
		return Transaction{}, fmt.Errorf("record payment deposit: %w", pgx.ErrNoRows)
	}
	result, err := tx.Exec(ctx, `UPDATE team_wallets SET balance_units=balance_units+$1, updated_at=now() WHERE team_id=$2`, transaction.AmountUnits, teamID)
	if err != nil || result.RowsAffected() != 1 {
		if err == nil {
			err = pgx.ErrNoRows
		}
		return Transaction{}, fmt.Errorf("credit payment deposit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit payment completion: %w", err)
	}
	return transactionFromSQLC(row), nil
}

func transactionFromSQLC(row dbsqlc.PaymentTransaction) Transaction {
	result := Transaction{ID: row.ID.String(), TeamID: row.TeamID.String(), Provider: row.Provider, ClientReference: row.ClientReference, Currency: row.Currency, AmountUnits: row.AmountUnits, Status: row.Status, ProviderTransactionID: row.ProviderTransactionID, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
	if row.PaidAt.Valid {
		value := row.PaidAt.Time
		result.PaidAt = &value
	}
	return result
}
