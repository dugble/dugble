-- name: GetTeamWallet :one
SELECT *
FROM team_wallets
WHERE team_id = sqlc.arg(team_id);

-- name: GetActiveProductRate :one
SELECT *
FROM product_rates
WHERE product = sqlc.arg(product)
  AND meter = sqlc.arg(meter)
  AND billing_market = sqlc.arg(billing_market)
  AND tier = sqlc.arg(tier)
  AND effective_from <= sqlc.arg(priced_at)
  AND (effective_until IS NULL OR effective_until > sqlc.arg(priced_at))
ORDER BY effective_from DESC
LIMIT 1;

-- name: ListWalletLedger :many
SELECT *
FROM wallet_ledger
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CreditTeamWallet :one
WITH locked_wallet AS MATERIALIZED (
    SELECT wallet.*
    FROM team_wallets AS wallet
    WHERE wallet.team_id = sqlc.arg(team_id)
    FOR UPDATE
),
inserted_ledger AS (
    INSERT INTO wallet_ledger (
        team_id,
        amount_units,
        transaction_type,
        reference_id
    )
    SELECT
        wallet.team_id,
        sqlc.arg(amount_units),
        sqlc.arg(transaction_type),
        sqlc.arg(reference_id)
    FROM locked_wallet AS wallet
    WHERE sqlc.arg(amount_units)::bigint > 0
      AND sqlc.arg(transaction_type)::text <> 'usage'
    ON CONFLICT (team_id, transaction_type, reference_id) DO NOTHING
    RETURNING team_id, amount_units
),
updated_wallet AS (
    UPDATE team_wallets AS wallet
    SET balance_units = wallet.balance_units + ledger.amount_units,
        updated_at = now()
    FROM inserted_ledger AS ledger
    WHERE wallet.team_id = ledger.team_id
    RETURNING wallet.*
)
SELECT *
FROM updated_wallet;

-- name: DebitTeamWallet :one
WITH locked_wallet AS MATERIALIZED (
    SELECT wallet.*
    FROM team_wallets AS wallet
    WHERE wallet.team_id = sqlc.arg(team_id)
    FOR UPDATE
),
inserted_ledger AS (
    INSERT INTO wallet_ledger (
        team_id,
        amount_units,
        transaction_type,
        reference_id
    )
    SELECT
        wallet.team_id,
        -sqlc.arg(amount_units)::bigint,
        sqlc.arg(transaction_type),
        sqlc.arg(reference_id)
    FROM locked_wallet AS wallet
    WHERE sqlc.arg(amount_units)::bigint > 0
      AND sqlc.arg(transaction_type)::text <> 'usage'
      AND wallet.balance_units >= sqlc.arg(amount_units)
    ON CONFLICT (team_id, transaction_type, reference_id) DO NOTHING
    RETURNING team_id, amount_units
),
updated_wallet AS (
    UPDATE team_wallets AS wallet
    SET balance_units = wallet.balance_units + ledger.amount_units,
        updated_at = now()
    FROM inserted_ledger AS ledger
    WHERE wallet.team_id = ledger.team_id
    RETURNING wallet.*
)
SELECT *
FROM updated_wallet;

-- name: AuthorizeSMSCharge :one
WITH clock AS MATERIALIZED (
    SELECT now() AS priced_at
),
team_record AS MATERIALIZED (
    SELECT team.id, team.status, team.market_code FROM teams AS team
    WHERE team.id = sqlc.arg(team_id)
),
pricing_context AS MATERIALIZED (
    SELECT team.id AS team_id,
      CASE WHEN team.market_code = sqlc.arg(destination_country)::char(2) THEN 'local' ELSE 'intl' END::text AS route_type
    FROM team_record AS team
),
market_record AS MATERIALIZED (
    SELECT market.code, market.currency FROM billing_markets AS market
    JOIN team_record AS team ON team.market_code = market.code
    WHERE market.is_enabled = true
),
entitlement_record AS MATERIALIZED (
    SELECT subscription.id, subscription.team_id, subscription.plan_code AS tier,
      subscription.current_period_start, subscription.current_period_end
    FROM team_subscriptions AS subscription
    WHERE subscription.team_id = sqlc.arg(team_id)
      AND subscription.status = 'active'
      AND subscription.current_period_start <= now()
      AND subscription.current_period_end > now()
),
wallet_record AS MATERIALIZED (
    SELECT wallet.*, entitlement.tier FROM team_wallets AS wallet
    JOIN entitlement_record AS entitlement ON entitlement.team_id = wallet.team_id
    WHERE wallet.team_id = sqlc.arg(team_id)
    FOR UPDATE OF wallet
),
existing_authorization AS MATERIALIZED (
    SELECT * FROM usage_authorizations AS usage_auth
    WHERE usage_auth.team_id = sqlc.arg(team_id)
      AND usage_auth.product = 'sms' AND usage_auth.meter = 'sms_segment'
      AND usage_auth.reference_id = sqlc.arg(reference_id)
),
credit_record AS MATERIALIZED (
    SELECT credit.* FROM subscription_credits AS credit
    JOIN entitlement_record AS entitlement
      ON entitlement.id = credit.subscription_id AND entitlement.team_id = credit.team_id
     AND entitlement.current_period_start = credit.period_start
     AND entitlement.current_period_end = credit.period_end
    JOIN wallet_record AS wallet ON wallet.currency = credit.currency
    WHERE credit.consumed_units < credit.granted_units
    LIMIT 1 FOR UPDATE OF credit
),
rate_record AS MATERIALIZED (
    SELECT rate.* FROM sms_rates AS rate
    CROSS JOIN clock CROSS JOIN pricing_context AS pricing
    JOIN wallet_record AS wallet ON wallet.tier = rate.tier
    WHERE rate.destination_country = sqlc.arg(destination_country)
      AND rate.route_type = pricing.route_type
      AND rate.effective_from <= clock.priced_at
      AND (rate.effective_until IS NULL OR rate.effective_until > clock.priced_at)
    ORDER BY rate.effective_from DESC LIMIT 1
),
fx_record AS MATERIALIZED (
    SELECT fx.* FROM fx_rates AS fx CROSS JOIN clock
    JOIN rate_record AS rate ON fx.base_currency = rate.currency
    JOIN wallet_record AS wallet ON fx.quote_currency = wallet.currency
    WHERE rate.currency <> wallet.currency
      AND fx.effective_from <= clock.priced_at
      AND (fx.effective_until IS NULL OR fx.effective_until > clock.priced_at)
    ORDER BY fx.effective_from DESC LIMIT 1
),
priced_plan AS MATERIALIZED (
    SELECT wallet.team_id, wallet.billing_market, wallet.currency, wallet.tier,
      wallet.balance_units, pricing.route_type, rate.id AS sms_rate_id,
      CASE WHEN rate.currency <> wallet.currency THEN fx.id END AS fx_rate_id,
      (rate.id IS NOT NULL AND rate.currency <> wallet.currency AND fx.id IS NULL) AS fx_missing,
      CASE WHEN rate.id IS NULL THEN 0::bigint
           WHEN rate.currency = wallet.currency THEN rate.cost_units::bigint
           WHEN fx.id IS NOT NULL THEN round(rate.cost_units::numeric * fx.rate)::bigint
           ELSE 0::bigint END AS unit_cost_units,
      credit.id AS subscription_credit_id,
      GREATEST(COALESCE(credit.granted_units-credit.consumed_units,0),0)::bigint AS available_credit_units,
      clock.priced_at
    FROM wallet_record AS wallet CROSS JOIN clock CROSS JOIN pricing_context AS pricing
    LEFT JOIN rate_record AS rate ON true LEFT JOIN fx_record AS fx ON true LEFT JOIN credit_record AS credit ON true
),
full_cost_plan AS MATERIALIZED (
    SELECT plan.*, CASE
      WHEN plan.unit_cost_units > 9223372036854775807 / NULLIF(sqlc.arg(quantity)::bigint,0) THEN NULL::bigint
      ELSE plan.unit_cost_units * sqlc.arg(quantity)::bigint END AS full_cost_units
    FROM priced_plan AS plan
),
charge_plan AS MATERIALIZED (
    SELECT plan.*,
      LEAST(COALESCE(plan.full_cost_units,0),plan.available_credit_units)::bigint AS credit_consumed_units,
      GREATEST(COALESCE(plan.full_cost_units,0)-plan.available_credit_units,0)::bigint AS wallet_debit_units
    FROM full_cost_plan AS plan
),
inserted_authorization AS (
    INSERT INTO usage_authorizations (
      team_id,product,meter,reference_id,sms_rate_id,fx_rate_id,billing_market,destination_country,route_type,
      total_quantity,allowance_quantity,billable_quantity,unit_cost_units,amount_units,subscription_credit_id,
      full_cost_units,credit_consumed_units,wallet_debit_units,currency,tier,priced_at
    )
    SELECT plan.team_id,'sms','sms_segment',sqlc.arg(reference_id),plan.sms_rate_id,plan.fx_rate_id,
      plan.billing_market,sqlc.arg(destination_country),plan.route_type,sqlc.arg(quantity),0,sqlc.arg(quantity),
      plan.unit_cost_units,plan.wallet_debit_units,
      CASE WHEN plan.credit_consumed_units>0 THEN plan.subscription_credit_id END,
      plan.full_cost_units,plan.credit_consumed_units,plan.wallet_debit_units,plan.currency,plan.tier,plan.priced_at
    FROM charge_plan AS plan
    JOIN team_record AS team ON team.id=plan.team_id AND team.status='active'
    JOIN market_record AS market ON market.code=plan.billing_market AND market.currency=plan.currency
    WHERE sqlc.arg(quantity)::bigint>0 AND NOT EXISTS(SELECT 1 FROM existing_authorization)
      AND plan.full_cost_units IS NOT NULL AND plan.sms_rate_id IS NOT NULL AND NOT plan.fx_missing
      AND plan.unit_cost_units>0 AND plan.wallet_debit_units<=plan.balance_units
    ON CONFLICT(team_id,product,meter,reference_id) DO NOTHING RETURNING *
),
updated_credit AS (
    UPDATE subscription_credits AS credit
    SET consumed_units=credit.consumed_units+usage_auth.credit_consumed_units,updated_at=now()
    FROM inserted_authorization AS usage_auth
    WHERE credit.id=usage_auth.subscription_credit_id AND usage_auth.credit_consumed_units>0
      AND credit.consumed_units+usage_auth.credit_consumed_units<=credit.granted_units
    RETURNING credit.granted_units,credit.consumed_units
),
inserted_ledger AS (
    INSERT INTO wallet_ledger(team_id,usage_authorization_id,amount_units,transaction_type,reference_id)
    SELECT usage_auth.team_id,usage_auth.id,-usage_auth.wallet_debit_units,'usage',usage_auth.reference_id
    FROM inserted_authorization AS usage_auth WHERE usage_auth.wallet_debit_units>0
    RETURNING team_id,amount_units
),
updated_wallet AS (
    UPDATE team_wallets AS wallet SET balance_units=wallet.balance_units+ledger.amount_units,updated_at=now()
    FROM inserted_ledger AS ledger WHERE wallet.team_id=ledger.team_id RETURNING wallet.balance_units
),
resolved_authorization AS MATERIALIZED (
    SELECT * FROM existing_authorization UNION ALL SELECT * FROM inserted_authorization LIMIT 1
)
SELECT CASE
    WHEN NOT EXISTS(SELECT 1 FROM team_record) THEN 'team_not_found'
    WHEN EXISTS(SELECT 1 FROM team_record WHERE status<>'active') THEN 'team_inactive'
    WHEN NOT EXISTS(SELECT 1 FROM market_record) THEN 'unsupported_market'
    WHEN NOT EXISTS(SELECT 1 FROM entitlement_record) THEN 'subscription_unavailable'
    WHEN NOT EXISTS(SELECT 1 FROM wallet_record) THEN 'wallet_not_found'
    WHEN EXISTS(SELECT 1 FROM existing_authorization) THEN 'already_applied'
    WHEN EXISTS(SELECT 1 FROM priced_plan WHERE sms_rate_id IS NULL) THEN 'rate_not_found'
    WHEN EXISTS(SELECT 1 FROM priced_plan WHERE fx_missing) THEN 'fx_rate_not_found'
    WHEN EXISTS(SELECT 1 FROM full_cost_plan WHERE full_cost_units IS NULL) THEN 'amount_overflow'
    WHEN EXISTS(SELECT 1 FROM charge_plan WHERE wallet_debit_units>balance_units) THEN 'insufficient_balance'
    WHEN EXISTS(SELECT 1 FROM inserted_authorization WHERE credit_consumed_units=full_cost_units) THEN 'credit_applied'
    WHEN EXISTS(SELECT 1 FROM inserted_authorization) THEN 'applied' ELSE 'already_applied' END AS outcome,
  COALESCE((SELECT billing_market FROM resolved_authorization),(SELECT billing_market FROM wallet_record),'')::text AS market_code,
  COALESCE((SELECT currency FROM resolved_authorization),(SELECT currency FROM wallet_record),'')::text AS currency,
  COALESCE((SELECT tier FROM resolved_authorization),(SELECT tier FROM wallet_record),'')::text AS tier,
  'sms'::text AS product,
  COALESCE((SELECT unit_cost_units FROM resolved_authorization),0)::bigint AS unit_cost_units,
  sqlc.arg(quantity)::bigint AS quantity,
  COALESCE((SELECT amount_units FROM resolved_authorization),0)::bigint AS amount_units,
  COALESCE((SELECT balance_units FROM updated_wallet),(SELECT balance_units FROM wallet_record),0)::bigint AS balance_units,
  (SELECT subscription_credit_id FROM resolved_authorization) AS subscription_credit_id,
  COALESCE((SELECT full_cost_units FROM resolved_authorization),0)::bigint AS full_cost_units,
  COALESCE((SELECT credit_consumed_units FROM resolved_authorization),0)::bigint AS credit_consumed_units,
  COALESCE((SELECT wallet_debit_units FROM resolved_authorization),0)::bigint AS wallet_debit_units,
  COALESCE((SELECT granted_units-consumed_units FROM updated_credit),(SELECT granted_units-consumed_units FROM credit_record),0)::bigint AS remaining_credit_units;
