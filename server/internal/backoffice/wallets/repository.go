package wallets

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbsqlc "github.com/coffeyvidzro/dugble/server/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db, queries: dbsqlc.New(db)} }

func (r *Repository) List(ctx context.Context, limit, offset int32) ([]Wallet, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice wallets repository is not configured")
	}
	rows, err := r.queries.BackofficeListWallets(ctx, dbsqlc.BackofficeListWalletsParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	items := make([]Wallet, 0, len(rows))
	for _, row := range rows {
		items = append(items, walletFromListRow(row))
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, teamID uuid.UUID) (Wallet, error) {
	if r == nil || r.queries == nil {
		return Wallet{}, errors.New("backoffice wallets repository is not configured")
	}
	row, err := r.queries.BackofficeGetWallet(ctx, dbsqlc.BackofficeGetWalletParams{TeamID: teamID})
	if err != nil {
		return Wallet{}, fmt.Errorf("get wallet: %w", err)
	}
	return Wallet{TeamID: row.TeamID.String(), TeamName: row.TeamName, BillingMarket: row.BillingMarket, Currency: row.Currency, BalanceUnits: row.BalanceUnits, Tier: row.Tier, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}, nil
}

func (r *Repository) ListTransactions(ctx context.Context, teamID *uuid.UUID, limit, offset int32) ([]Transaction, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("backoffice wallets repository is not configured")
	}
	rows, err := r.queries.BackofficeListWalletTransactions(ctx, dbsqlc.BackofficeListWalletTransactionsParams{TeamID: teamID, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions: %w", err)
	}
	items := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		items = append(items, transactionFromValues(row.ID, row.TeamID, row.TeamName, row.UsageAuthorizationID, row.SubscriptionChargeID, row.PaymentTransactionID, row.AmountUnits, row.Currency, row.TransactionType, row.ReferenceID, row.CreatedAt.Time))
	}
	return items, nil
}
func (r *Repository) GetTransaction(ctx context.Context, teamID, transactionID uuid.UUID) (Transaction, error) {
	row, err := r.queries.BackofficeGetWalletTransaction(ctx, dbsqlc.BackofficeGetWalletTransactionParams{TeamID: teamID, TransactionID: transactionID})
	if err != nil {
		return Transaction{}, fmt.Errorf("get wallet transaction: %w", err)
	}
	return transactionFromValues(row.ID, row.TeamID, row.TeamName, row.UsageAuthorizationID, row.SubscriptionChargeID, row.PaymentTransactionID, row.AmountUnits, row.Currency, row.TransactionType, row.ReferenceID, row.CreatedAt.Time), nil
}

func (r *Repository) Adjust(ctx context.Context, teamID uuid.UUID, amount int64, reference, reason string, actorID uuid.UUID, sessionID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin wallet adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := dbsqlc.New(tx).BackofficeAdjustWallet(ctx, dbsqlc.BackofficeAdjustWalletParams{TeamID: teamID, AmountUnits: amount, ReferenceID: reference})
	if err != nil {
		return fmt.Errorf("adjust wallet: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (team_id,actor_type,actor_user_id,actor_session_id,action,resource_type,resource_id,metadata) VALUES ($1,'user',$2,NULLIF($3,''),'wallet.adjust','wallet',$1::text,jsonb_build_object('amount_units',$4,'reference_id',$5,'reason',$6,'balance_units',$7))`, teamID, actorID, sessionID, amount, reference, reason, row.BalanceUnits)
	if err != nil {
		return fmt.Errorf("audit wallet adjustment: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit wallet adjustment: %w", err)
	}
	return nil
}
func stringID(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	text := value.String()
	return &text
}
func transactionFromValues(id, teamID uuid.UUID, teamName string, authorizationID, chargeID, paymentID *uuid.UUID, amount int64, currency, kind, reference string, created time.Time) Transaction {
	return Transaction{ID: id.String(), TeamID: teamID.String(), TeamName: teamName, UsageAuthorizationID: stringID(authorizationID), SubscriptionChargeID: stringID(chargeID), PaymentTransactionID: stringID(paymentID), AmountUnits: amount, Currency: currency, TransactionType: kind, ReferenceID: reference, CreatedAt: created}
}

func walletFromListRow(row dbsqlc.BackofficeListWalletsRow) Wallet {
	return Wallet{TeamID: row.TeamID.String(), TeamName: row.TeamName, BillingMarket: row.BillingMarket, Currency: row.Currency, BalanceUnits: row.BalanceUnits, Tier: row.Tier, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
