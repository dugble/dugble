package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const chargePeriodSQL = `
WITH wallet AS MATERIALIZED (
    SELECT wallet.*
    FROM team_wallets AS wallet
    WHERE wallet.team_id = $2
    FOR UPDATE
),
price AS MATERIALIZED (
    SELECT price.*
    FROM plan_prices AS price
    JOIN wallet
      ON wallet.billing_market = price.billing_market
     AND wallet.currency = price.currency
    WHERE price.plan_code = $3
      AND price.billing_interval = 'monthly'
      AND price.effective_from <= $4
      AND (price.effective_until IS NULL OR price.effective_until > $4)
    ORDER BY price.effective_from DESC
    LIMIT 1
),
existing AS MATERIALIZED (
    SELECT charge.*
    FROM subscription_charges AS charge
    WHERE charge.subscription_id = $1
      AND charge.period_start = $4
),
charge AS (
    INSERT INTO subscription_charges (
        subscription_id, team_id, plan_price_id, plan_code,
        billing_market, currency, period_start, period_end,
        amount_units, status, failure_code, applied_at, reference_id
    )
    SELECT
        $1, $2, price.id, price.plan_code,
        price.billing_market, price.currency, $4, $5,
        price.amount_units,
        CASE WHEN wallet.balance_units >= price.amount_units THEN 'applied' ELSE 'failed' END,
        CASE WHEN wallet.balance_units >= price.amount_units THEN NULL ELSE 'insufficient_balance' END,
        CASE WHEN wallet.balance_units >= price.amount_units THEN now() ELSE NULL END,
        'subscription:' || $1::text || ':' || extract(epoch FROM $4::timestamptz)::bigint::text
    FROM price
    JOIN wallet ON true
    ON CONFLICT (subscription_id, period_start) DO UPDATE
    SET status = CASE
            WHEN subscription_charges.status = 'applied' THEN 'applied'
            WHEN subscription_charges.amount_units <= (SELECT balance_units FROM wallet) THEN 'applied'
            ELSE 'failed'
        END,
        failure_code = CASE
            WHEN subscription_charges.status = 'applied'
              OR subscription_charges.amount_units <= (SELECT balance_units FROM wallet) THEN NULL
            ELSE 'insufficient_balance'
        END,
        attempt_count = CASE WHEN subscription_charges.status = 'applied'
            THEN subscription_charges.attempt_count ELSE subscription_charges.attempt_count + 1 END,
        last_attempted_at = CASE WHEN subscription_charges.status = 'applied'
            THEN subscription_charges.last_attempted_at ELSE now() END,
        applied_at = CASE
            WHEN subscription_charges.status = 'applied' THEN subscription_charges.applied_at
            WHEN subscription_charges.amount_units <= (SELECT balance_units FROM wallet) THEN now()
            ELSE NULL
        END
    RETURNING *
),
ledger AS (
    INSERT INTO wallet_ledger (
        team_id, subscription_charge_id, amount_units, transaction_type, reference_id
    )
    SELECT charge.team_id, charge.id, -charge.amount_units, 'subscription', charge.reference_id
    FROM charge
    WHERE charge.status = 'applied' AND charge.amount_units > 0
    ON CONFLICT (subscription_charge_id) WHERE subscription_charge_id IS NOT NULL DO NOTHING
    RETURNING team_id, amount_units
),
updated_wallet AS (
    UPDATE team_wallets AS wallet
    SET balance_units = wallet.balance_units + ledger.amount_units, updated_at = now()
    FROM ledger
    WHERE wallet.team_id = ledger.team_id
    RETURNING wallet.balance_units
),
credit AS (
    INSERT INTO subscription_credits (
        subscription_id, subscription_charge_id, team_id, plan_code,
        billing_market, currency, period_start, period_end,
        granted_units
    )
    SELECT
        charge.subscription_id, charge.id, charge.team_id, charge.plan_code,
        charge.billing_market, charge.currency, charge.period_start, charge.period_end,
        charge.amount_units
    FROM charge
    WHERE charge.status = 'applied'
      AND charge.amount_units > 0
    ON CONFLICT (subscription_charge_id) DO UPDATE
    SET subscription_charge_id = EXCLUDED.subscription_charge_id
    RETURNING id, granted_units
)
SELECT
    charge.id,
    CASE
        WHEN existing.status = 'applied' THEN 'already_applied'
        WHEN charge.status = 'applied' THEN 'applied'
        ELSE 'insufficient_balance'
    END::text,
    charge.status,
    charge.failure_code,
    charge.attempt_count,
    charge.last_attempted_at,
    charge.applied_at,
    charge.plan_code,
    charge.currency::text,
    charge.amount_units,
    COALESCE((SELECT balance_units FROM updated_wallet), (SELECT balance_units FROM wallet))::bigint,
    (SELECT id FROM credit),
    COALESCE((SELECT granted_units FROM credit), 0)::bigint
FROM charge
LEFT JOIN existing ON true
`

type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

func (*Repository) ChargePeriod(ctx context.Context, tx pgx.Tx, input Input) (Result, error) {
	if tx == nil {
		return Result{}, errors.New("subscription charge transaction is required")
	}
	var result Result
	var outcome string
	var lastAttemptedAt, appliedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, chargePeriodSQL,
		input.SubscriptionID, input.TeamID, input.PlanCode, input.PeriodStart, input.PeriodEnd,
	).Scan(
		&result.ChargeID, &outcome, &result.Status, &result.FailureCode,
		&result.AttemptCount, &lastAttemptedAt, &appliedAt, &result.PlanCode,
		&result.Currency, &result.AmountUnits, &result.RemainingBalance,
		&result.CreditID, &result.CreditGranted,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Outcome: OutcomePriceUnavailable, PlanCode: input.PlanCode}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("charge subscription period: %w", err)
	}
	result.Outcome = Outcome(outcome)
	if lastAttemptedAt.Valid {
		result.LastAttemptedAt = &lastAttemptedAt.Time
	}
	if appliedAt.Valid {
		result.AppliedAt = &appliedAt.Time
	}
	return result, nil
}
