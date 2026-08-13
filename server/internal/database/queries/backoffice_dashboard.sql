-- name: BackofficeDashboardStats :one
SELECT
 (SELECT count(*) FROM users) AS users,
 (SELECT count(*) FROM teams) AS teams,
 (SELECT count(*) FROM sms_messages WHERE created_at >= date_trunc('day', now())) AS sms_today,
 (SELECT count(*) FROM sms_messages WHERE created_at >= now() - interval '24 hours' AND status IN ('failed','undelivered','rejected','expired')) AS failed_sms_24_hours,
 (SELECT count(*) FROM sender_ids WHERE status = 'pending') AS pending_sender_ids,
 (SELECT count(*) FROM domains WHERE status = 'pending') AS pending_domains;

-- name: BackofficeDashboardFailedSMS :many
SELECT s.id, t.name AS team_name, s.to_number, s.status,
       coalesce(s.error_message, '') AS error_message, s.created_at
FROM sms_messages s
JOIN teams t ON t.id = s.team_id
WHERE s.created_at >= now() - interval '24 hours'
  AND s.status IN ('failed', 'undelivered', 'rejected', 'expired')
ORDER BY s.created_at DESC
LIMIT 8;

-- name: BackofficeDashboardPendingSenderIDs :many
SELECT s.id, t.name AS team_name, s.name, s.country_code, s.created_at
FROM sender_ids s
JOIN teams t ON t.id = s.team_id
WHERE s.status = 'pending'
ORDER BY s.created_at ASC
LIMIT 6;

-- name: BackofficeDashboardPendingDomains :many
SELECT d.id, coalesce(t.name, '') AS team_name, d.normalized_name AS name, d.created_at
FROM domains d
LEFT JOIN teams t ON t.id = d.team_id
WHERE d.status = 'pending'
ORDER BY d.created_at ASC
LIMIT 6;

-- name: BackofficeDashboardRecentActivity :many
SELECT action, resource_type, resource_id, outcome, actor_type, created_at
FROM audit_events
ORDER BY created_at DESC, id DESC
LIMIT 8;
