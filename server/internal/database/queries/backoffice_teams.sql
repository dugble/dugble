-- name: BackofficeListTeams :many
SELECT id, name, status, created_at
FROM teams
WHERE (sqlc.arg(search)::text = '' OR name ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)::int
OFFSET sqlc.arg(page_offset)::int;
-- name: BackofficeGetTeam :one
SELECT id,name,status,created_at FROM teams WHERE id=sqlc.arg(id);
-- name: BackofficeUpdateTeamStatus :execrows
UPDATE teams SET status=sqlc.arg(status),updated_at=now() WHERE id=sqlc.arg(id);
-- name: BackofficeListTeamMembers :many
SELECT u.id AS user_id,u.email,u.name,tm.role,tm.status,tm.created_at FROM team_members tm JOIN users u ON u.id=tm.user_id WHERE tm.team_id=sqlc.arg(team_id) ORDER BY tm.created_at DESC;
-- name: BackofficeListTeamSMS :many
SELECT s.id,t.name AS team_name,s.to_number,s.from_name,s.status,coalesce(s.provider_id,''),coalesce(s.error_message,''),s.created_at FROM sms_messages s JOIN teams t ON t.id=s.team_id WHERE s.team_id=sqlc.arg(team_id) ORDER BY s.created_at DESC LIMIT 25;
