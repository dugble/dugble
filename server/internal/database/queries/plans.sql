-- name: ListEnabledPlans :many
SELECT *
FROM plans
WHERE is_enabled = true
ORDER BY CASE code
    WHEN 'growth' THEN 1
    WHEN 'scale' THEN 2
    WHEN 'enterprise' THEN 3
    ELSE 4
END, code;

-- name: GetPlan :one
SELECT *
FROM plans
WHERE code = sqlc.arg(code);

-- name: GetActivePlanPrice :one
SELECT *
FROM plan_prices
WHERE plan_code = sqlc.arg(plan_code)
  AND billing_market = sqlc.arg(billing_market)
  AND billing_interval = 'monthly'
  AND effective_from <= sqlc.arg(priced_at)
  AND (effective_until IS NULL OR effective_until > sqlc.arg(priced_at))
ORDER BY effective_from DESC
LIMIT 1;

-- name: ListPlansForTeam :many
SELECT
    plan.code,
    plan.name,
    subscription.plan_code AS current_plan_code,
    subscription.pending_plan_code,
    subscription.current_period_end,
    COALESCE(price.id, '00000000-0000-0000-0000-000000000000'::uuid) AS plan_price_id,
    COALESCE(price.currency, '')::text AS currency,
    COALESCE(price.amount_units, 0)::bigint AS amount_units,
    (price.id IS NOT NULL)::boolean AS is_available
FROM team_subscriptions AS subscription
JOIN team_wallets AS wallet ON wallet.team_id = subscription.team_id
CROSS JOIN plans AS plan
LEFT JOIN LATERAL (
    SELECT plan_price.*
    FROM plan_prices AS plan_price
    WHERE plan_price.plan_code = plan.code
      AND plan_price.billing_market = wallet.billing_market
      AND plan_price.billing_interval = 'monthly'
      AND plan_price.effective_from <= subscription.current_period_end
      AND (
          plan_price.effective_until IS NULL
          OR plan_price.effective_until > subscription.current_period_end
      )
    ORDER BY plan_price.effective_from DESC
    LIMIT 1
) AS price ON true
WHERE subscription.team_id = sqlc.arg(team_id)
  AND plan.is_enabled = true
ORDER BY CASE plan.code
    WHEN 'growth' THEN 1
    WHEN 'scale' THEN 2
    WHEN 'enterprise' THEN 3
    ELSE 4
END, plan.code;
