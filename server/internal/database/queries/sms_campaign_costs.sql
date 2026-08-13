-- name: ClaimNextSMSCampaignRecipientForEstimate :one
SELECT recipient.*, campaign.body AS campaign_body,
       campaign.sender_id, sender.name AS sender_name
FROM sms_campaign_recipients AS recipient
JOIN sms_campaigns AS campaign
  ON campaign.id = recipient.campaign_id AND campaign.team_id = recipient.team_id
JOIN sender_ids AS sender
  ON sender.id = campaign.sender_id AND sender.team_id = campaign.team_id
WHERE recipient.status = 'pending' AND recipient.estimated_segments IS NULL
  AND campaign.status = 'estimating'
ORDER BY recipient.created_at, recipient.id
FOR UPDATE OF recipient SKIP LOCKED
LIMIT 1;

-- name: EstimateSMSCampaignRecipientCost :one
SELECT
    CASE
        WHEN rate.currency = wallet.currency THEN rate.cost_units::bigint
        ELSE round(rate.cost_units::numeric * fx.rate)::bigint
    END AS unit_cost_units,
    (
        CASE
            WHEN rate.currency = wallet.currency THEN rate.cost_units::bigint
            ELSE round(rate.cost_units::numeric * fx.rate)::bigint
        END * sqlc.arg(segments)::bigint
    )::bigint AS cost_units,
    wallet.currency::text AS currency
FROM team_wallets AS wallet
JOIN teams AS team ON team.id = wallet.team_id AND team.status = 'active'
JOIN team_subscriptions AS subscription
  ON subscription.team_id = wallet.team_id
 AND subscription.status = 'active'
 AND subscription.current_period_start <= now()
 AND subscription.current_period_end > now()
JOIN sms_rates AS rate
  ON rate.tier = subscription.plan_code
 AND rate.destination_country = sqlc.arg(destination_country)::char(2)
 AND rate.route_type = CASE WHEN team.market_code = sqlc.arg(destination_country)::char(2) THEN 'local' ELSE 'intl' END
 AND rate.effective_from <= now()
 AND (rate.effective_until IS NULL OR rate.effective_until > now())
LEFT JOIN fx_rates AS fx
  ON rate.currency <> wallet.currency
 AND fx.base_currency = rate.currency
 AND fx.quote_currency = wallet.currency
 AND fx.effective_from <= now()
 AND (fx.effective_until IS NULL OR fx.effective_until > now())
WHERE wallet.team_id = sqlc.arg(team_id)
  AND (rate.currency = wallet.currency OR fx.id IS NOT NULL)
ORDER BY rate.effective_from DESC, fx.effective_from DESC NULLS LAST
LIMIT 1;

-- name: SetSMSCampaignRecipientEstimate :exec
UPDATE sms_campaign_recipients
SET rendered_body = sqlc.arg(rendered_body),
    encoding = sqlc.arg(encoding),
    estimated_segments = sqlc.arg(estimated_segments),
    estimated_unit_cost_units = sqlc.arg(estimated_unit_cost_units),
    estimated_cost_units = sqlc.arg(estimated_cost_units)
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
  AND status = 'pending' AND estimated_segments IS NULL;

-- name: FailSMSCampaignRecipientEstimate :exec
UPDATE sms_campaign_recipients
SET status = 'failed', failure_code = sqlc.arg(failure_code),
    failure_message = sqlc.arg(failure_message)
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
  AND status = 'pending' AND estimated_segments IS NULL;

-- name: FinalizeSMSCampaignCostPreflight :one
WITH estimate AS (
    SELECT
        COALESCE(sum(estimated_segments), 0)::bigint AS segments,
        COALESCE(sum(estimated_cost_units), 0)::bigint AS cost_units,
        COALESCE(min(estimated_unit_cost_units), 0)::bigint AS minimum_unit_cost_units,
        count(*) FILTER (WHERE status = 'failed')::bigint AS failures
    FROM sms_campaign_recipients
    WHERE campaign_id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
), wallet AS (
    SELECT team_wallet.balance_units, team_wallet.currency::text AS currency
    FROM (VALUES (1)) AS singleton(value)
    LEFT JOIN team_wallets AS team_wallet ON team_wallet.team_id = sqlc.arg(team_id)
), allowance AS (
    SELECT COALESCE(max(included_quantity - consumed_quantity), 0)::bigint AS remaining
    FROM usage_allowances
    WHERE team_id = sqlc.arg(team_id) AND product = 'sms' AND meter = 'sms_segment'
      AND period_start <= now() AND period_end > now()
), preflight AS (
    SELECT estimate.*,
        LEAST(estimate.segments, allowance.remaining)::bigint AS allowance_segments,
        GREATEST(
            estimate.cost_units
              - LEAST(estimate.segments, allowance.remaining) * estimate.minimum_unit_cost_units,
            0
        )::bigint AS billable_cost_units
    FROM estimate, allowance
)
UPDATE sms_campaigns AS campaign
SET estimated_segments = preflight.segments,
    estimated_cost_units = preflight.cost_units,
    estimated_billable_cost_units = preflight.billable_cost_units,
    preflight_allowance_segments = preflight.allowance_segments,
    currency = wallet.currency,
    preflight_balance_units = wallet.balance_units,
    preflight_at = now(),
    failed_count = preflight.failures,
    status = CASE
        WHEN preflight.failures > 0 OR wallet.balance_units IS NULL THEN 'failed'
        WHEN wallet.balance_units < preflight.billable_cost_units THEN 'failed'
        ELSE 'sending'
    END,
    updated_at = now()
FROM preflight, wallet
WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id)
  AND campaign.status = 'estimating'
  AND NOT EXISTS (
      SELECT 1 FROM sms_campaign_recipients
      WHERE campaign_id = campaign.id AND status = 'pending'
        AND estimated_segments IS NULL
  )
RETURNING campaign.*;

-- name: GetSMSCampaignCostEstimate :one
SELECT
    campaign.id AS campaign_id,
    campaign.currency,
    campaign.estimated_segments,
    campaign.estimated_cost_units,
    campaign.estimated_billable_cost_units,
    campaign.preflight_allowance_segments,
    campaign.actual_segments,
    campaign.actual_charge_units,
    campaign.preflight_balance_units,
    campaign.preflight_at,
    count(recipient.id) FILTER (WHERE recipient.status <> 'excluded')::bigint AS recipients,
    COALESCE(min(recipient.estimated_cost_units), 0)::bigint AS minimum_recipient_cost_units,
    COALESCE(max(recipient.estimated_cost_units), 0)::bigint AS maximum_recipient_cost_units
FROM sms_campaigns AS campaign
LEFT JOIN sms_campaign_recipients AS recipient
  ON recipient.campaign_id = campaign.id AND recipient.team_id = campaign.team_id
WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id)
GROUP BY campaign.id;
