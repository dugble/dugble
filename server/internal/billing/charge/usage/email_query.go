package usage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const ensureEmailAllowanceSQL = `
WITH clock AS MATERIALIZED (
    SELECT
        now() AS priced_at,
        date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS period_start,
        (date_trunc('month', now() AT TIME ZONE 'UTC') + interval '1 month') AT TIME ZONE 'UTC' AS period_end
),
eligible_team AS MATERIALIZED (
    SELECT wallet.team_id, wallet.billing_market, wallet.currency, subscription.plan_code AS tier
    FROM teams AS team
    JOIN billing_markets AS market
      ON market.code = team.market_code
     AND market.is_enabled = true
    JOIN team_wallets AS wallet
      ON wallet.team_id = team.id
     AND wallet.billing_market = market.code
     AND wallet.currency = market.currency
    JOIN team_subscriptions AS subscription
      ON subscription.team_id = team.id
     AND subscription.status = 'active'
     AND subscription.current_period_start <= now()
     AND subscription.current_period_end > now()
    WHERE team.id = $1
      AND team.status = 'active'
),
allowance_policy_record AS MATERIALIZED (
    SELECT policy.*
    FROM allowance_policies AS policy
    CROSS JOIN clock
    JOIN eligible_team AS team
      ON team.billing_market = policy.billing_market
     AND team.tier = policy.tier
    WHERE policy.product = 'email'
      AND policy.meter = 'email_recipient'
      AND policy.cadence = 'monthly'
      AND policy.effective_from <= clock.priced_at
      AND (policy.effective_until IS NULL OR policy.effective_until > clock.priced_at)
    ORDER BY policy.effective_from DESC
    LIMIT 1
)
INSERT INTO usage_allowances (
    team_id, allowance_policy_id, product, meter, billing_market, tier,
    period_start, period_end, included_quantity
)
SELECT
    team.team_id, policy.id, 'email', 'email_recipient',
    policy.billing_market, policy.tier,
    clock.period_start, clock.period_end, policy.included_quantity
FROM eligible_team AS team
CROSS JOIN clock
JOIN allowance_policy_record AS policy ON true
ON CONFLICT (team_id, product, meter, period_start, period_end) DO NOTHING
`

const chargeEmailUsageSQL = `
WITH clock AS MATERIALIZED (
    SELECT
        now() AS priced_at,
        date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS period_start,
        (date_trunc('month', now() AT TIME ZONE 'UTC') + interval '1 month') AT TIME ZONE 'UTC' AS period_end
),
team_record AS MATERIALIZED (
    SELECT team.id, team.status, team.market_code
    FROM teams AS team
    WHERE team.id = $2
),
market_record AS MATERIALIZED (
    SELECT market.code, market.currency
    FROM billing_markets AS market
    JOIN team_record AS team ON team.market_code = market.code
    WHERE market.is_enabled = true
),
entitlement_record AS MATERIALIZED (
    SELECT subscription.id, subscription.team_id, subscription.plan_code AS tier,
           subscription.current_period_start, subscription.current_period_end
    FROM team_subscriptions AS subscription
    WHERE subscription.team_id = $2
      AND subscription.status = 'active'
      AND subscription.current_period_start <= now()
      AND subscription.current_period_end > now()
),
wallet_record AS MATERIALIZED (
    SELECT wallet.*, entitlement.tier
    FROM team_wallets AS wallet
    JOIN entitlement_record AS entitlement ON entitlement.team_id = wallet.team_id
    WHERE wallet.team_id = $2
    FOR UPDATE OF wallet
),
existing_charge AS MATERIALIZED (
    SELECT *
    FROM usage_authorizations AS usage_charge
    WHERE usage_charge.team_id = $2
      AND usage_charge.product = 'email'
      AND usage_charge.meter = 'email_recipient'
      AND usage_charge.reference_id = $3
),
allowance_record AS MATERIALIZED (
    SELECT allowance.*
    FROM usage_allowances AS allowance
    CROSS JOIN clock
    JOIN wallet_record AS wallet
      ON wallet.team_id = allowance.team_id
     AND wallet.billing_market = allowance.billing_market
     AND wallet.tier = allowance.tier
    WHERE allowance.product = 'email'
      AND allowance.meter = 'email_recipient'
      AND allowance.period_start = clock.period_start
      AND allowance.period_end = clock.period_end
    LIMIT 1
    FOR UPDATE OF allowance
),
quantity_plan AS MATERIALIZED (
    SELECT
        wallet.team_id,
        wallet.billing_market,
        wallet.currency,
        wallet.tier,
        wallet.balance_units,
        allowance.id AS usage_allowance_id,
        LEAST(
            $1::bigint,
            GREATEST(
                COALESCE(allowance.included_quantity - allowance.consumed_quantity, 0),
                0
            )
        )::bigint AS allowance_quantity,
        GREATEST(
            $1::bigint - LEAST(
                $1::bigint,
                GREATEST(
                    COALESCE(allowance.included_quantity - allowance.consumed_quantity, 0),
                    0
                )
            ),
            0
        )::bigint AS billable_quantity
    FROM wallet_record AS wallet
    LEFT JOIN allowance_record AS allowance ON allowance.team_id = wallet.team_id
),
credit_record AS MATERIALIZED (
    SELECT credit.*
    FROM subscription_credits AS credit
    JOIN entitlement_record AS entitlement
      ON entitlement.id = credit.subscription_id
     AND entitlement.team_id = credit.team_id
     AND entitlement.current_period_start = credit.period_start
     AND entitlement.current_period_end = credit.period_end
    JOIN wallet_record AS wallet ON wallet.currency = credit.currency
    WHERE credit.consumed_units < credit.granted_units
    LIMIT 1
    FOR UPDATE OF credit
),
rate_record AS MATERIALIZED (
    SELECT rate.*
    FROM product_rates AS rate
    CROSS JOIN clock
    JOIN quantity_plan AS quantity
      ON quantity.billing_market = rate.billing_market
     AND quantity.currency = rate.currency
     AND quantity.tier = rate.tier
    WHERE quantity.billable_quantity > 0
      AND rate.product = 'email'
      AND rate.meter = 'email_recipient'
      AND rate.effective_from <= clock.priced_at
      AND (rate.effective_until IS NULL OR rate.effective_until > clock.priced_at)
    ORDER BY rate.effective_from DESC
    LIMIT 1
),
priced_plan AS MATERIALIZED (
    SELECT quantity.team_id, quantity.billing_market, quantity.currency, quantity.tier,
           quantity.balance_units, quantity.usage_allowance_id,
           quantity.allowance_quantity, quantity.billable_quantity,
           CASE WHEN quantity.billable_quantity > 0 THEN rate.id END AS product_rate_id,
           CASE WHEN quantity.billable_quantity > 0 THEN COALESCE(rate.cost_units, 0) ELSE 0 END::bigint AS unit_cost_units,
           CASE
             WHEN quantity.billable_quantity = 0 THEN 0::bigint
             WHEN rate.cost_units IS NULL THEN 0::bigint
             WHEN rate.cost_units > 9223372036854775807 / NULLIF(quantity.billable_quantity, 0) THEN NULL::bigint
             ELSE rate.cost_units * quantity.billable_quantity
           END AS full_cost_units,
           credit.id AS subscription_credit_id,
           GREATEST(COALESCE(credit.granted_units - credit.consumed_units, 0), 0)::bigint AS available_credit_units,
           clock.priced_at
    FROM quantity_plan AS quantity
    CROSS JOIN clock
    LEFT JOIN rate_record AS rate ON true
    LEFT JOIN credit_record AS credit ON true
),
charge_plan AS MATERIALIZED (
    SELECT plan.*,
           LEAST(COALESCE(plan.full_cost_units, 0), plan.available_credit_units)::bigint AS credit_consumed_units,
           GREATEST(COALESCE(plan.full_cost_units, 0) - plan.available_credit_units, 0)::bigint AS wallet_debit_units
    FROM priced_plan AS plan
),
inserted_charge AS (
    INSERT INTO usage_authorizations (
        team_id, product, meter, reference_id, usage_allowance_id,
        product_rate_id, billing_market,
        total_quantity, allowance_quantity, billable_quantity, unit_cost_units,
        amount_units, subscription_credit_id, full_cost_units,
        credit_consumed_units, wallet_debit_units, currency, tier, priced_at
    )
    SELECT plan.team_id, 'email', 'email_recipient', $3,
           CASE WHEN plan.allowance_quantity > 0 THEN plan.usage_allowance_id END,
           plan.product_rate_id, plan.billing_market,
           $1, plan.allowance_quantity, plan.billable_quantity, plan.unit_cost_units,
           plan.wallet_debit_units,
           CASE WHEN plan.credit_consumed_units > 0 THEN plan.subscription_credit_id END,
           plan.full_cost_units, plan.credit_consumed_units, plan.wallet_debit_units,
           plan.currency, plan.tier, plan.priced_at
    FROM charge_plan AS plan
    JOIN team_record AS team ON team.id = plan.team_id AND team.status = 'active'
    JOIN market_record AS market
      ON market.code = plan.billing_market AND market.currency = plan.currency
    WHERE $1::bigint > 0
      AND NOT EXISTS (SELECT 1 FROM existing_charge)
      AND plan.full_cost_units IS NOT NULL
      AND (plan.billable_quantity = 0 OR plan.product_rate_id IS NOT NULL)
      AND plan.wallet_debit_units <= plan.balance_units
    ON CONFLICT (team_id, product, meter, reference_id) DO NOTHING
    RETURNING *
),
updated_allowance AS (
    UPDATE usage_allowances AS allowance
    SET consumed_quantity = allowance.consumed_quantity + usage_charge.allowance_quantity,
        updated_at = now()
    FROM inserted_charge AS usage_charge
    WHERE allowance.id = usage_charge.usage_allowance_id
      AND usage_charge.allowance_quantity > 0
      AND allowance.consumed_quantity + usage_charge.allowance_quantity <= allowance.included_quantity
    RETURNING allowance.included_quantity, allowance.consumed_quantity
),
updated_credit AS (
    UPDATE subscription_credits AS credit
    SET consumed_units = credit.consumed_units + usage_charge.credit_consumed_units,
        updated_at = now()
    FROM inserted_charge AS usage_charge
    WHERE credit.id = usage_charge.subscription_credit_id
      AND usage_charge.credit_consumed_units > 0
      AND credit.consumed_units + usage_charge.credit_consumed_units <= credit.granted_units
    RETURNING credit.granted_units, credit.consumed_units
),
inserted_ledger AS (
    INSERT INTO wallet_ledger (
        team_id, usage_authorization_id, amount_units, transaction_type, reference_id
    )
    SELECT usage_charge.team_id, usage_charge.id, -usage_charge.wallet_debit_units,
           'usage', usage_charge.reference_id
    FROM inserted_charge AS usage_charge
    WHERE usage_charge.wallet_debit_units > 0
    RETURNING team_id, amount_units
),
updated_wallet AS (
    UPDATE team_wallets AS wallet
    SET balance_units = wallet.balance_units + ledger.amount_units, updated_at = now()
    FROM inserted_ledger AS ledger
    WHERE wallet.team_id = ledger.team_id
    RETURNING wallet.balance_units
),
resolved_charge AS MATERIALIZED (
    SELECT * FROM existing_charge
    UNION ALL
    SELECT * FROM inserted_charge
    LIMIT 1
)
SELECT
    CASE
      WHEN NOT EXISTS (SELECT 1 FROM team_record) THEN 'team_not_found'
      WHEN EXISTS (SELECT 1 FROM team_record WHERE status <> 'active') THEN 'team_inactive'
      WHEN NOT EXISTS (SELECT 1 FROM market_record) THEN 'unsupported_market'
      WHEN NOT EXISTS (SELECT 1 FROM entitlement_record) THEN 'subscription_unavailable'
      WHEN NOT EXISTS (SELECT 1 FROM wallet_record) THEN 'wallet_not_found'
      WHEN EXISTS (SELECT 1 FROM existing_charge) THEN 'already_applied'
      WHEN EXISTS (SELECT 1 FROM priced_plan WHERE billable_quantity > 0 AND product_rate_id IS NULL) THEN 'rate_not_found'
      WHEN EXISTS (SELECT 1 FROM priced_plan WHERE full_cost_units IS NULL) THEN 'amount_overflow'
      WHEN EXISTS (SELECT 1 FROM charge_plan WHERE wallet_debit_units > balance_units) THEN 'insufficient_balance'
      WHEN EXISTS (
          SELECT 1 FROM inserted_charge
          WHERE full_cost_units > 0 AND credit_consumed_units = full_cost_units
      ) THEN 'credit_applied'
      WHEN EXISTS (SELECT 1 FROM inserted_charge) THEN 'applied'
      ELSE 'already_applied'
    END::text AS outcome,
    COALESCE((SELECT billing_market FROM resolved_charge), (SELECT billing_market FROM wallet_record), '')::text,
    COALESCE((SELECT currency FROM resolved_charge), (SELECT currency FROM wallet_record), '')::text,
    COALESCE((SELECT tier FROM resolved_charge), (SELECT tier FROM wallet_record), '')::text,
    'email'::text,
    COALESCE((SELECT unit_cost_units FROM resolved_charge), 0)::bigint,
    COALESCE((SELECT total_quantity FROM resolved_charge), $1::bigint)::bigint,
    COALESCE((SELECT amount_units FROM resolved_charge), 0)::bigint,
    COALESCE((SELECT balance_units FROM updated_wallet), (SELECT balance_units FROM wallet_record), 0)::bigint,
    (SELECT subscription_credit_id FROM resolved_charge),
    COALESCE((SELECT full_cost_units FROM resolved_charge), 0)::bigint,
    COALESCE((SELECT credit_consumed_units FROM resolved_charge), 0)::bigint,
    COALESCE((SELECT wallet_debit_units FROM resolved_charge), 0)::bigint,
    COALESCE(
      (SELECT granted_units - consumed_units FROM updated_credit),
      (SELECT granted_units - consumed_units FROM credit_record),
      0
    )::bigint
`

func ensureEmailAllowance(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	_, err := tx.Exec(ctx, ensureEmailAllowanceSQL, teamID)
	return err
}

func chargeEmailUsage(
	ctx context.Context,
	tx pgx.Tx,
	input EmailChargeInput,
) (Charge, error) {
	var outcome string
	var product string
	var charge Charge
	err := tx.QueryRow(
		ctx,
		chargeEmailUsageSQL,
		input.RecipientCount,
		input.TeamID,
		input.MessageID.String(),
	).Scan(
		&outcome,
		&charge.MarketCode,
		&charge.Currency,
		&charge.Tier,
		&product,
		&charge.UnitCostUnits,
		&charge.Quantity,
		&charge.AmountUnits,
		&charge.RemainingBalance,
		&charge.SubscriptionCreditID,
		&charge.FullCostUnits,
		&charge.CreditConsumedUnits,
		&charge.WalletDebitUnits,
		&charge.RemainingCreditUnits,
	)
	charge.Outcome = Outcome(outcome)
	charge.Product = Product(product)
	return charge, err
}
