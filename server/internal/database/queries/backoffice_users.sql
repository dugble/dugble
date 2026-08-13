-- name: BackofficeListUsers :many
SELECT
    id,
    email,
    name,
    email_verified,
    created_at
FROM users
WHERE
    sqlc.arg(search)::text = ''
    OR email ILIKE '%' || sqlc.arg(search)::text || '%'
    OR name ILIKE '%' || sqlc.arg(search)::text || '%'
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)::int
OFFSET sqlc.arg(page_offset)::int;

-- name: BackofficeGetUser :one
SELECT
    id,
    email,
    name,
    email_verified,
    created_at
FROM users
WHERE id = sqlc.arg(id);

-- name: BackofficeListUserTeams :many
SELECT
    team.id,
    team.name,
    membership.role,
    membership.status
FROM team_members AS membership
JOIN teams AS team ON team.id = membership.team_id
WHERE membership.user_id = sqlc.arg(user_id)
ORDER BY team.created_at DESC;
