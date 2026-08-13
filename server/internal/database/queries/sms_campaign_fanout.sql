-- name: QueueNextDueSMSCampaign :one
WITH due AS (
    SELECT id FROM sms_campaigns
    WHERE status = 'scheduled' AND scheduled_at <= now()
    ORDER BY scheduled_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE sms_campaigns AS campaign
SET status = 'queued', queued_at = now(), revision = revision + 1, updated_at = now()
FROM due
WHERE campaign.id = due.id
RETURNING campaign.*;

-- name: ClaimNextSMSCampaignForMaterialization :one
WITH candidate AS (
    SELECT id FROM sms_campaigns
    WHERE status = 'queued'
    ORDER BY queued_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE sms_campaigns AS campaign
SET status = 'materializing', updated_at = now()
FROM candidate
WHERE campaign.id = candidate.id
RETURNING campaign.*;

-- name: MaterializeClaimedSMSCampaignRecipients :execrows
WITH audience AS (
    SELECT
        contact.id AS contact_id,
        contact.team_id,
        contact.normalized_phone,
        contact.phone_country,
        contact.sms_consent_status,
        jsonb_build_object(
            'email', contact.email,
            'phone', contact.phone,
            'first_name', contact.first_name,
            'last_name', contact.last_name,
            'properties', COALESCE(properties.value, '{}'::jsonb)
        ) AS snapshot,
        suppression.id IS NOT NULL AS suppressed,
        row_number() OVER (
            PARTITION BY contact.normalized_phone
            ORDER BY membership.created_at, contact.id
        ) AS phone_rank
    FROM sms_campaigns AS campaign
    JOIN contact_segments AS membership
      ON membership.segment_id = campaign.segment_id AND membership.team_id = campaign.team_id
    JOIN contacts AS contact
      ON contact.id = membership.contact_id AND contact.team_id = membership.team_id
    LEFT JOIN channel_suppressions AS suppression
      ON suppression.team_id = contact.team_id
     AND suppression.channel = 'sms'
     AND suppression.normalized_address = contact.normalized_phone
    LEFT JOIN LATERAL (
        SELECT jsonb_object_agg(property.key,
            CASE property.value_type WHEN 'number' THEN to_jsonb(value.number_value) ELSE to_jsonb(value.string_value) END
        ) AS value
        FROM contact_property_values AS value
        JOIN contact_properties AS property ON property.id = value.contact_property_id
        WHERE value.contact_id = contact.id AND value.team_id = contact.team_id
    ) AS properties ON true
    WHERE campaign.id = sqlc.arg(campaign_id)
      AND campaign.team_id = sqlc.arg(team_id)
      AND campaign.status = 'materializing'
)
INSERT INTO sms_campaign_recipients (
    team_id, campaign_id, contact_id, phone, phone_country,
    contact_snapshot, status, exclusion_reason
)
SELECT
    team_id, sqlc.arg(campaign_id), contact_id, normalized_phone, phone_country, snapshot,
    CASE
        WHEN normalized_phone IS NULL THEN 'excluded'
        WHEN phone_rank > 1 THEN 'excluded'
        WHEN sms_consent_status <> 'opted_in' THEN 'excluded'
        WHEN suppressed THEN 'excluded'
        ELSE 'pending'
    END,
    CASE
        WHEN normalized_phone IS NULL THEN 'missing_phone'
        WHEN phone_rank > 1 THEN 'duplicate_phone'
        WHEN sms_consent_status = 'opted_out' THEN 'opted_out'
        WHEN sms_consent_status <> 'opted_in' THEN 'consent_unknown'
        WHEN suppressed THEN 'suppressed'
        ELSE NULL
    END
FROM audience
ON CONFLICT (campaign_id, contact_id) DO NOTHING;

-- name: FinishSMSCampaignMaterialization :one
WITH counts AS (
    SELECT count(*)::bigint AS audience,
           count(*) FILTER (WHERE status = 'pending')::bigint AS eligible,
           count(*) FILTER (WHERE status = 'excluded')::bigint AS excluded
    FROM sms_campaign_recipients
    WHERE campaign_id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
)
UPDATE sms_campaigns AS campaign
SET status = CASE WHEN counts.eligible = 0 THEN 'sent' ELSE 'estimating' END,
    audience_count = counts.audience,
    eligible_count = counts.eligible,
    excluded_count = counts.excluded,
    materialized_at = now(),
    sent_at = CASE WHEN counts.eligible = 0 THEN now() ELSE NULL END,
    updated_at = now()
FROM counts
WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id)
  AND campaign.status = 'materializing'
RETURNING campaign.*;

-- name: ClaimNextSMSCampaignRecipient :one
WITH candidate AS (
    SELECT recipient.id
    FROM sms_campaign_recipients AS recipient
    JOIN sms_campaigns AS campaign
      ON campaign.id = recipient.campaign_id AND campaign.team_id = recipient.team_id
    WHERE recipient.status = 'pending' AND recipient.next_attempt_at <= now()
      AND campaign.status = 'sending' AND recipient.estimated_segments IS NOT NULL
      AND (
          SELECT count(*) FROM sms_campaign_recipients AS sent_this_second
          WHERE sent_this_second.campaign_id = campaign.id
            AND sent_this_second.status = 'queued'
            AND sent_this_second.queued_at >= date_trunc('second', now())
      ) < campaign.rate_limit_per_second
      AND (
          campaign.daily_send_limit IS NULL
          OR (
              SELECT count(*) FROM sms_campaign_recipients AS sent_today
              WHERE sent_today.campaign_id = campaign.id
                AND sent_today.status = 'queued'
                AND sent_today.queued_at >= date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'
          ) < campaign.daily_send_limit
      )
    ORDER BY recipient.next_attempt_at, recipient.created_at, recipient.id
    FOR UPDATE OF recipient, campaign SKIP LOCKED
    LIMIT 1
)
UPDATE sms_campaign_recipients AS recipient
SET status = 'processing', attempt_count = attempt_count + 1
FROM candidate, sms_campaigns AS campaign, sender_ids AS sender
WHERE recipient.id = candidate.id
  AND campaign.id = recipient.campaign_id AND campaign.team_id = recipient.team_id
  AND sender.id = campaign.sender_id AND sender.team_id = campaign.team_id
RETURNING recipient.*, campaign.body AS campaign_body,
          campaign.sender_id, sender.name AS sender_name;

-- name: RecheckSMSCampaignRecipientEligibility :one
WITH eligibility AS (
    SELECT CASE
        WHEN contact.id IS NULL OR contact.normalized_phone IS DISTINCT FROM recipient.phone THEN 'invalid_phone'
        WHEN contact.sms_consent_status = 'opted_out' THEN 'opted_out'
        WHEN contact.sms_consent_status <> 'opted_in' THEN 'consent_unknown'
        WHEN EXISTS (
            SELECT 1 FROM channel_suppressions AS suppression
            WHERE suppression.team_id = recipient.team_id
              AND suppression.channel = 'sms'
              AND suppression.normalized_address = recipient.phone
        ) THEN 'suppressed'
        ELSE NULL
    END AS reason
    FROM sms_campaign_recipients AS recipient
    LEFT JOIN contacts AS contact
      ON contact.id = recipient.contact_id AND contact.team_id = recipient.team_id
    WHERE recipient.id = sqlc.arg(recipient_id) AND recipient.team_id = sqlc.arg(team_id)
      AND recipient.status = 'processing'
), excluded AS (
    UPDATE sms_campaign_recipients AS recipient
    SET status = 'excluded', exclusion_reason = eligibility.reason
    FROM eligibility
    WHERE id = sqlc.arg(recipient_id) AND team_id = sqlc.arg(team_id)
      AND eligibility.reason IS NOT NULL
    RETURNING campaign_id
), adjusted AS (
    UPDATE sms_campaigns
    SET eligible_count = eligible_count - 1,
        excluded_count = excluded_count + 1,
        updated_at = now()
    WHERE id = (SELECT campaign_id FROM excluded) AND team_id = sqlc.arg(team_id)
    RETURNING id
)
SELECT COALESCE(reason, '')::text FROM eligibility;

-- name: SetSMSCampaignRecipientQueued :exec
WITH queued AS (
    UPDATE sms_campaign_recipients AS recipient
    SET status = 'queued', sms_message_id = sqlc.arg(sms_message_id),
        rendered_body = sqlc.arg(rendered_body), queued_at = now(),
        actual_segments = sqlc.arg(actual_segments),
        actual_charge_units = sqlc.arg(actual_charge_units),
        failure_code = NULL, failure_message = NULL
    WHERE recipient.id = sqlc.arg(id) AND recipient.team_id = sqlc.arg(team_id) AND recipient.status = 'processing'
    RETURNING recipient.campaign_id, recipient.actual_segments, recipient.actual_charge_units
)
UPDATE sms_campaigns AS campaign
SET actual_segments = campaign.actual_segments + queued.actual_segments,
    actual_charge_units = campaign.actual_charge_units + queued.actual_charge_units,
    updated_at = now()
FROM queued
WHERE campaign.id = queued.campaign_id AND campaign.team_id = sqlc.arg(team_id);

-- name: FailSMSCampaignRecipient :exec
UPDATE sms_campaign_recipients
SET status = 'failed', failure_code = sqlc.arg(failure_code),
    failure_message = sqlc.arg(failure_message)
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id) AND status = 'processing';

-- name: FinalizeSMSCampaignFanout :one
UPDATE sms_campaigns AS campaign
SET status = CASE
        WHEN EXISTS (
            SELECT 1 FROM sms_campaign_recipients
            WHERE campaign_id = campaign.id AND status = 'failed'
        ) THEN 'failed'
        ELSE 'sent'
    END,
    failed_count = (
        SELECT count(*) FROM sms_campaign_recipients
        WHERE campaign_id = campaign.id AND status = 'failed'
    ),
    actual_segments = COALESCE((
        SELECT sum(actual_segments) FROM sms_campaign_recipients
        WHERE campaign_id = campaign.id AND status = 'queued'
    ), 0),
    actual_charge_units = COALESCE((
        SELECT sum(actual_charge_units) FROM sms_campaign_recipients
        WHERE campaign_id = campaign.id AND status = 'queued'
    ), 0),
    sent_at = now(), updated_at = now()
WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id)
  AND campaign.status = 'sending'
  AND NOT EXISTS (
      SELECT 1 FROM sms_campaign_recipients
      WHERE campaign_id = campaign.id AND status IN ('pending', 'processing')
  )
RETURNING campaign.*;
