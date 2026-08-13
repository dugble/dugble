package smscampaign

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const applyCommunicationCreditPreflightSQL = `
WITH entitlement AS (
    SELECT
        subscription.id,
        subscription.current_period_start,
        subscription.current_period_end
    FROM team_subscriptions AS subscription
    WHERE subscription.team_id = $1
      AND subscription.status = 'active'
      AND subscription.current_period_start <= now()
      AND subscription.current_period_end > now()
),
communication_credit AS (
    SELECT GREATEST(credit.granted_units - credit.consumed_units, 0)::bigint AS remaining_units
    FROM subscription_credits AS credit
    JOIN entitlement AS subscription
      ON subscription.id = credit.subscription_id
     AND subscription.current_period_start = credit.period_start
     AND subscription.current_period_end = credit.period_end
    JOIN team_wallets AS wallet
      ON wallet.team_id = credit.team_id
     AND wallet.currency = credit.currency
    WHERE credit.team_id = $1
    ORDER BY credit.period_start DESC, credit.created_at DESC
    LIMIT 1
),
preflight AS (
    SELECT
        campaign.id,
        campaign.failed_count,
        campaign.preflight_balance_units,
        GREATEST(
            campaign.estimated_cost_units - COALESCE(communication_credit.remaining_units, 0),
            0
        )::bigint AS billable_cost_units
    FROM SMS_campaigns AS campaign
    LEFT JOIN communication_credit ON true
    WHERE campaign.id = $2
      AND campaign.team_id = $1
      AND campaign.preflight_at IS NOT NULL
)
UPDATE sms_campaigns AS campaign
SET estimated_billable_cost_units = preflight.billable_cost_units,
    preflight_allowance_segments = 0,
    status = CASE
        WHEN preflight.failed_count > 0 OR preflight.preflight_balance_units IS NULL THEN 'failed'
        WHEN preflight.preflight_balance_units < preflight.billable_cost_units THEN 'failed'
        ELSE 'sending'
    END,
    updated_at = now()
FROM preflight
WHERE campaign.id = preflight.id
`

// ApplyCommunicationCreditPreflight replaces the legacy quantity-allowance
// estimate with the current period's monetary communication-credit snapshot.
// It does not consume credit; runtime usage authorization remains authoritative.
func ApplyCommunicationCreditPreflight(
	ctx context.Context,
	tx pgx.Tx,
	teamID uuid.UUID,
	campaignID uuid.UUID,
) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, teamID); err != nil {
		return fmt.Errorf("lock SMS campaign billing preflight: %w", err)
	}
	if _, err := tx.Exec(ctx, applyCommunicationCreditPreflightSQL, teamID, campaignID); err != nil {
		return fmt.Errorf("apply SMS campaign communication credit preflight: %w", err)
	}
	return nil
}
