-- name: GetTeamSubscription :one
SELECT *
FROM team_subscriptions
WHERE team_id = sqlc.arg(team_id);

-- name: ListTeamSubscriptionCharges :many
SELECT *
FROM subscription_charges
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: ScheduleTeamPlanChange :one
WITH locked_subscription AS MATERIALIZED (
    SELECT subscription.*, wallet.billing_market
    FROM team_subscriptions AS subscription
    JOIN team_wallets AS wallet ON wallet.team_id = subscription.team_id
    WHERE subscription.team_id = sqlc.arg(team_id)
    FOR UPDATE OF subscription, wallet
),
target_plan AS MATERIALIZED (
    SELECT plan.code
    FROM plans AS plan
    WHERE plan.code = sqlc.arg(plan_code)
      AND plan.is_enabled = true
),
target_price AS MATERIALIZED (
    SELECT price.id
    FROM plan_prices AS price
    JOIN locked_subscription AS subscription
      ON price.billing_market = subscription.billing_market
    JOIN target_plan AS plan ON price.plan_code = plan.code
    WHERE price.billing_interval = 'monthly'
      AND price.effective_from <= subscription.current_period_end
      AND (
          price.effective_until IS NULL
          OR price.effective_until > subscription.current_period_end
      )
    ORDER BY price.effective_from DESC
    LIMIT 1
),
updated_subscription AS (
    UPDATE team_subscriptions AS subscription
    SET pending_plan_code = CASE
            WHEN subscription.plan_code = sqlc.arg(plan_code)::text THEN NULL
            ELSE sqlc.arg(plan_code)::text
        END,
        pending_plan_effective_at = CASE
            WHEN subscription.plan_code = sqlc.arg(plan_code)::text THEN NULL
            ELSE subscription.current_period_end
        END,
        updated_at = now()
    FROM locked_subscription, target_plan, target_price
    WHERE subscription.id = locked_subscription.id
      AND subscription.status IN ('active', 'past_due')
      AND subscription.cancel_at_period_end = false
    RETURNING subscription.*
)
SELECT *
FROM updated_subscription;

-- name: CancelTeamPlanChange :one
WITH updated_subscription AS (
    UPDATE team_subscriptions AS subscription
    SET pending_plan_code = NULL,
        pending_plan_effective_at = NULL,
        updated_at = now()
    WHERE subscription.team_id = sqlc.arg(team_id)
      AND subscription.status IN ('active', 'past_due')
    RETURNING subscription.*
)
SELECT *
FROM updated_subscription;

-- name: CancelTeamSubscription :one
WITH updated_subscription AS (
    UPDATE team_subscriptions AS subscription
    SET cancel_at_period_end = true,
        pending_plan_code = NULL,
        pending_plan_effective_at = NULL,
        updated_at = now()
    WHERE subscription.team_id = sqlc.arg(team_id)
      AND subscription.status IN ('active', 'past_due')
      AND subscription.cancel_at_period_end = false
    RETURNING subscription.*
)
SELECT *
FROM updated_subscription;

-- name: ReactivateTeamSubscription :one
UPDATE team_subscriptions AS subscription
SET cancel_at_period_end = false,
    updated_at = now()
WHERE subscription.team_id = sqlc.arg(team_id)
  AND subscription.status IN ('active', 'past_due')
  AND subscription.cancel_at_period_end = true
RETURNING subscription.*;
