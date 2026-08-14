package usage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dugble/dugble/server/internal/platform/systemmail"
)

const (
	BalanceLevelLow       = "low"
	BalanceLevelExhausted = "exhausted"
)

type balanceAlertSender interface {
	SendWalletBalanceAlert(context.Context, pgx.Tx, systemmail.SendWalletBalanceAlertInput) error
}

type BalanceNotifier struct {
	db     *pgxpool.Pool
	sender balanceAlertSender
}

func NewBalanceNotifier(db *pgxpool.Pool, sender balanceAlertSender) *BalanceNotifier {
	return &BalanceNotifier{db: db, sender: sender}
}

func (notifier *BalanceNotifier) Notify(ctx context.Context, charge CommittedCharge) error {
	if notifier == nil || notifier.db == nil || notifier.sender == nil {
		return errors.New("wallet balance notifier is not configured")
	}
	level := balanceLevel(charge)
	if level == "" {
		return nil
	}
	tx, err := notifier.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin wallet balance notification: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var claimed bool
	err = tx.QueryRow(ctx, `
INSERT INTO wallet_balance_notifications (team_id, level, balance_units)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, level) DO UPDATE
SET balance_units = EXCLUDED.balance_units, notified_at = now()
WHERE wallet_balance_notifications.notified_at <= now() - interval '7 days'
RETURNING true`, charge.TeamID, level, charge.RemainingBalance).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("claim wallet balance notification: %w", err)
	}
	if !claimed {
		return nil
	}

	rows, err := tx.Query(ctx, `
SELECT users.name, users.email, teams.name
FROM team_members
JOIN users ON users.id = team_members.user_id
JOIN teams ON teams.id = team_members.team_id
WHERE team_members.team_id = $1
  AND team_members.role = 'owner'
  AND team_members.status = 'active'
ORDER BY users.id`, charge.TeamID)
	if err != nil {
		return fmt.Errorf("list wallet balance recipients: %w", err)
	}
	inputs := make([]systemmail.SendWalletBalanceAlertInput, 0, 1)
	for rows.Next() {
		var input systemmail.SendWalletBalanceAlertInput
		if err := rows.Scan(&input.Name, &input.ToEmail, &input.TeamName); err != nil {
			return fmt.Errorf("scan wallet balance recipient: %w", err)
		}
		input.Currency, input.BalanceUnits, input.Level = charge.Currency, charge.RemainingBalance, level
		inputs = append(inputs, input)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("list wallet balance recipients: %w", err)
	}
	rows.Close()
	for _, input := range inputs {
		if err := notifier.sender.SendWalletBalanceAlert(ctx, tx, input); err != nil {
			return fmt.Errorf("queue wallet balance notification: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit wallet balance notification: %w", err)
	}
	return nil
}

func balanceLevel(charge CommittedCharge) string {
	if charge.WalletDebitUnits > 0 && charge.RemainingBalance <= 0 {
		return BalanceLevelExhausted
	}
	// A balance that cannot fund another charge of the same wallet-debit size
	// is actionable without relying on currency-specific absolute thresholds.
	if charge.WalletDebitUnits > 0 && charge.RemainingBalance < charge.WalletDebitUnits {
		return BalanceLevelLow
	}
	return ""
}
