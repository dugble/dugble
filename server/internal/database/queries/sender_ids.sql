-- name: CreateSenderID :one
INSERT INTO sender_ids (
    team_id,
    name,
    normalized_name,
    country_code,
    purpose,
    provider,
    created_by
)
SELECT
    team.id,
    sqlc.arg(name),
    lower(trim(sqlc.arg(name))),
    upper(trim(sqlc.arg(country_code))),
    NULLIF(trim(sqlc.arg(purpose)), ''),
    sqlc.narg(provider),
    sqlc.narg(created_by)
FROM teams AS team
WHERE team.id = sqlc.arg(team_id)
  AND team.status = 'active'
RETURNING *;

-- name: ListSenderIDs :many
SELECT sender_id.*
FROM sender_ids AS sender_id
JOIN teams AS team ON team.id = sender_id.team_id
WHERE sender_id.team_id = sqlc.arg(team_id)
  AND team.status = 'active'
ORDER BY sender_id.created_at DESC;

-- name: GetSenderID :one
SELECT sender_id.*
FROM sender_ids AS sender_id
JOIN teams AS team ON team.id = sender_id.team_id
WHERE sender_id.id = sqlc.arg(id)
  AND sender_id.team_id = sqlc.arg(team_id)
  AND team.status = 'active';

-- name: DeactivateSenderID :one
UPDATE sender_ids AS sender_id
SET status = 'inactive',
    provider_whitelisted = false,
    disabled_at = COALESCE(sender_id.disabled_at, now()),
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
FROM teams AS team
WHERE sender_id.id = sqlc.arg(id)
  AND sender_id.team_id = sqlc.arg(team_id)
  AND team.id = sender_id.team_id
  AND team.status = 'active'
RETURNING sender_id.*;
