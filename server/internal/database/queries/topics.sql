-- name: CreateTopic :one
INSERT INTO topics (
    team_id,
    name,
    description,
    default_subscription,
    visibility
) VALUES (
    sqlc.arg(team_id),
    sqlc.arg(name),
    sqlc.narg(description),
    sqlc.arg(default_subscription),
    sqlc.arg(visibility)
)
RETURNING id,
          team_id,
          name,
          description,
          default_subscription,
          visibility,
          created_at,
          updated_at;

-- name: ListTopics :many
SELECT t.id,
       t.team_id,
       t.name,
       t.description,
       t.default_subscription,
       t.visibility,
       t.created_at,
       t.updated_at
FROM topics AS t
WHERE t.team_id = sqlc.arg(team_id)
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: GetTopic :one
SELECT t.id,
       t.team_id,
       t.name,
       t.description,
       t.default_subscription,
       t.visibility,
       t.created_at,
       t.updated_at
FROM topics AS t
WHERE t.id = sqlc.arg(id)
  AND t.team_id = sqlc.arg(team_id);

-- name: UpdateTopic :one
UPDATE topics
SET name = sqlc.arg(name),
    description = sqlc.narg(description),
    visibility = sqlc.arg(visibility),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
RETURNING id,
          team_id,
          name,
          description,
          default_subscription,
          visibility,
          created_at,
          updated_at;

-- name: DeleteTopic :one
DELETE FROM topics
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
RETURNING id,
          team_id,
          name,
          description,
          default_subscription,
          visibility,
          created_at,
          updated_at;
