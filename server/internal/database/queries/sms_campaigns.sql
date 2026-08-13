-- name: CreateSMSCampaign :one
INSERT INTO sms_campaigns (team_id, name, segment_id, sender_id, body, rate_limit_per_second, daily_send_limit)
VALUES (sqlc.arg(team_id), sqlc.arg(name), sqlc.arg(segment_id), sqlc.arg(sender_id), sqlc.arg(body), sqlc.arg(rate_limit_per_second), sqlc.narg(daily_send_limit))
RETURNING *;

-- name: ListSMSCampaigns :many
SELECT * FROM sms_campaigns
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetSMSCampaign :one
SELECT * FROM sms_campaigns WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id);

-- name: UpdateSMSCampaignDraft :one
UPDATE sms_campaigns
SET name = sqlc.arg(name), segment_id = sqlc.arg(segment_id), sender_id = sqlc.arg(sender_id),
    body = sqlc.arg(body), rate_limit_per_second = sqlc.arg(rate_limit_per_second),
    daily_send_limit = sqlc.narg(daily_send_limit), revision = revision + 1, updated_at = now()
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
  AND status = 'draft' AND revision = sqlc.arg(revision)
RETURNING *;

-- name: DeleteSMSCampaign :one
DELETE FROM sms_campaigns
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id) AND status IN ('draft', 'canceled')
RETURNING *;

-- name: DuplicateSMSCampaign :one
INSERT INTO sms_campaigns (team_id, name, segment_id, sender_id, body, rate_limit_per_second, daily_send_limit)
SELECT team_id, sqlc.arg(name), segment_id, sender_id, body, rate_limit_per_second, daily_send_limit
FROM sms_campaigns AS source
WHERE source.id = sqlc.arg(source_id) AND source.team_id = sqlc.arg(team_id)
RETURNING *;

-- name: IsApprovedSMSCampaignSender :one
SELECT EXISTS (
    SELECT 1 FROM sms_campaigns AS campaign
    JOIN sender_ids AS sender ON sender.id = campaign.sender_id AND sender.team_id = campaign.team_id
    WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id)
      AND sender.status = 'approved' AND sender.provider_whitelisted
);

-- name: ActivateSMSCampaign :one
UPDATE sms_campaigns AS campaign
SET status = CASE WHEN sqlc.narg(scheduled_at)::timestamptz IS NULL THEN 'queued' ELSE 'scheduled' END,
    scheduled_at = sqlc.narg(scheduled_at),
    queued_at = CASE WHEN sqlc.narg(scheduled_at)::timestamptz IS NULL THEN now() ELSE NULL END,
    revision = revision + 1,
    updated_at = now()
WHERE campaign.id = sqlc.arg(id) AND campaign.team_id = sqlc.arg(team_id) AND campaign.status = 'draft'
  AND EXISTS (
      SELECT 1 FROM sender_ids AS sender
      WHERE sender.id = campaign.sender_id AND sender.team_id = campaign.team_id
        AND sender.status = 'approved' AND sender.provider_whitelisted
  )
RETURNING campaign.*;

-- name: CancelSMSCampaign :one
UPDATE sms_campaigns
SET status = 'canceled', canceled_at = now(), revision = revision + 1, updated_at = now()
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id) AND status IN ('scheduled', 'queued')
RETURNING *;

-- name: ListSMSCampaignRecipients :many
SELECT * FROM sms_campaign_recipients
WHERE campaign_id = sqlc.arg(campaign_id) AND team_id = sqlc.arg(team_id)
ORDER BY created_at, id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
