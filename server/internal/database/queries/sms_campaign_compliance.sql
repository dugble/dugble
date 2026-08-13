-- name: RecordSMSOptOut :one
WITH contact_record AS (
    SELECT id FROM contacts
    WHERE team_id = sqlc.arg(team_id) AND normalized_phone = sqlc.arg(phone)
    FOR UPDATE
), updated_contact AS (
    UPDATE contacts
    SET sms_consent_status = 'opted_out',
        sms_consent_updated_at = now(),
        sms_consent_source = sqlc.arg(source),
        updated_at = now()
    WHERE id = (SELECT id FROM contact_record) AND team_id = sqlc.arg(team_id)
    RETURNING id
), suppression AS (
    INSERT INTO channel_suppressions (
        team_id, channel, address, normalized_address, reason, origin, source_id
    ) VALUES (
        sqlc.arg(team_id), 'sms', sqlc.arg(phone), sqlc.arg(phone),
        'user_opt_out', sqlc.arg(source), sqlc.narg(source_id)
    )
    ON CONFLICT (team_id, channel, normalized_address)
    DO UPDATE SET reason = 'user_opt_out', origin = EXCLUDED.origin, source_id = EXCLUDED.source_id
    RETURNING id
)
INSERT INTO sms_consent_events (team_id, contact_id, phone, action, source, source_id)
VALUES (
    sqlc.arg(team_id), (SELECT id FROM contact_record), sqlc.arg(phone),
    'opted_out', sqlc.arg(source), sqlc.narg(source_id)
)
ON CONFLICT (team_id, source, source_id) WHERE source_id IS NOT NULL
DO UPDATE SET source_id = EXCLUDED.source_id
RETURNING *;

-- name: GetSMSCampaignExclusionSummary :many
SELECT exclusion_reason, count(*)::bigint AS total
FROM sms_campaign_recipients
WHERE campaign_id = sqlc.arg(campaign_id) AND team_id = sqlc.arg(team_id)
  AND status = 'excluded'
GROUP BY exclusion_reason
ORDER BY exclusion_reason;

-- name: GetSMSCampaignAnalytics :one
SELECT
    campaign.id AS campaign_id,
    campaign.audience_count,
    campaign.eligible_count,
    campaign.excluded_count,
    count(recipient.id) FILTER (WHERE recipient.status = 'queued')::bigint AS queued_count,
    count(recipient.id) FILTER (WHERE recipient.status = 'failed')::bigint AS failed_count,
    count(message.id) FILTER (WHERE message.status IN ('submitted', 'sent', 'delivered'))::bigint AS sent_count,
    count(message.id) FILTER (WHERE message.status = 'delivered')::bigint AS delivered_count,
    count(message.id) FILTER (WHERE message.status IN ('undelivered', 'rejected', 'failed', 'expired'))::bigint AS delivery_failed_count,
    campaign.estimated_segments,
    campaign.estimated_cost_units,
    campaign.estimated_billable_cost_units,
    campaign.actual_segments,
    campaign.actual_charge_units,
    campaign.currency
FROM sms_campaigns AS campaign
LEFT JOIN sms_campaign_recipients AS recipient
  ON recipient.campaign_id = campaign.id AND recipient.team_id = campaign.team_id
LEFT JOIN sms_messages AS message
  ON message.id = recipient.sms_message_id AND message.team_id = recipient.team_id
WHERE campaign.id = sqlc.arg(campaign_id) AND campaign.team_id = sqlc.arg(team_id)
GROUP BY campaign.id;
