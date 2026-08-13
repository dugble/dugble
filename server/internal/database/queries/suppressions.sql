-- name: CreateSuppression :one
INSERT INTO channel_suppressions (
    team_id, channel, address, normalized_address, reason, origin, source_id
) VALUES (
    sqlc.arg(team_id),
    'email',
    sqlc.arg(email),
    lower(btrim(sqlc.arg(email))),
    sqlc.arg(origin),
    CASE WHEN sqlc.arg(origin)::text = 'manual' THEN 'manual' ELSE 'provider' END,
    sqlc.narg(source_id)
)
RETURNING *;

-- name: CreateSuppressions :many
INSERT INTO channel_suppressions (
    team_id, channel, address, normalized_address, reason, origin
)
SELECT sqlc.arg(team_id),
       'email',
       batch_email.email,
       lower(btrim(batch_email.email)),
       'manual',
       'manual'
FROM unnest(sqlc.arg(emails)::text[]) WITH ORDINALITY AS batch_email(email, position)
ORDER BY batch_email.position
RETURNING *;

-- name: ListSuppressions :many
SELECT s.*
FROM channel_suppressions AS s
WHERE s.team_id = sqlc.arg(team_id)
  AND s.channel = 'email'
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: ListSuppressionsFiltered :many
SELECT s.*
FROM channel_suppressions AS s
WHERE s.team_id = sqlc.arg(team_id)
  AND s.channel = 'email'
  AND (sqlc.narg(filter_origin)::text IS NULL OR s.reason = sqlc.narg(filter_origin))
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListSuppressionsAfter :many
SELECT s.*
FROM channel_suppressions AS s
WHERE s.team_id = sqlc.arg(scope_team_id)
  AND s.channel = 'email'
  AND (sqlc.narg(filter_origin)::text IS NULL OR s.reason = sqlc.narg(filter_origin))
  AND (s.created_at, s.id) < (
      SELECT cursor_suppression.created_at, cursor_suppression.id
      FROM channel_suppressions AS cursor_suppression
      WHERE cursor_suppression.id = sqlc.arg(cursor_id)
        AND cursor_suppression.team_id = sqlc.arg(scope_team_id)
        AND cursor_suppression.channel = 'email'
  )
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListSuppressionsBefore :many
SELECT s.*
FROM channel_suppressions AS s
WHERE s.team_id = sqlc.arg(scope_team_id)
  AND s.channel = 'email'
  AND (sqlc.narg(filter_origin)::text IS NULL OR s.reason = sqlc.narg(filter_origin))
  AND (s.created_at, s.id) > (
      SELECT cursor_suppression.created_at, cursor_suppression.id
      FROM channel_suppressions AS cursor_suppression
      WHERE cursor_suppression.id = sqlc.arg(cursor_id)
        AND cursor_suppression.team_id = sqlc.arg(scope_team_id)
        AND cursor_suppression.channel = 'email'
  )
ORDER BY s.created_at ASC, s.id ASC
LIMIT sqlc.arg(page_limit);

-- name: SuppressionCursorExists :one
SELECT EXISTS (
    SELECT 1
    FROM channel_suppressions AS s
    WHERE s.id = sqlc.arg(cursor_id)
      AND s.team_id = sqlc.arg(team_id)
      AND s.channel = 'email'
);

-- name: GetSuppressionByID :one
SELECT s.*
FROM channel_suppressions AS s
WHERE s.id = sqlc.arg(id)
  AND s.team_id = sqlc.arg(team_id)
  AND s.channel = 'email';

-- name: GetSuppressionByEmail :one
SELECT s.*
FROM channel_suppressions AS s
WHERE s.team_id = sqlc.arg(team_id)
  AND s.channel = 'email'
  AND s.normalized_address = lower(btrim(sqlc.arg(email)));

-- name: DeleteSuppressionByID :one
DELETE FROM channel_suppressions
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND channel = 'email'
RETURNING *;

-- name: DeleteSuppressionByEmail :one
DELETE FROM channel_suppressions
WHERE team_id = sqlc.arg(team_id)
  AND channel = 'email'
  AND normalized_address = lower(btrim(sqlc.arg(email)))
RETURNING *;

-- name: DeleteSuppressionsByIDs :many
DELETE FROM channel_suppressions
WHERE team_id = sqlc.arg(team_id)
  AND channel = 'email'
  AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING *;

-- name: DeleteSuppressionsByEmails :many
DELETE FROM channel_suppressions
WHERE team_id = sqlc.arg(team_id)
  AND channel = 'email'
  AND normalized_address = ANY(sqlc.arg(emails)::text[])
RETURNING *;

-- name: CreateChannelSuppression :one
INSERT INTO channel_suppressions (
    team_id, channel, address, normalized_address, reason, origin, source_id
) VALUES (
    sqlc.arg(team_id), sqlc.arg(channel), sqlc.arg(address),
    sqlc.arg(normalized_address), sqlc.arg(reason), sqlc.arg(origin), sqlc.narg(source_id)
)
RETURNING *;

-- name: IsChannelAddressSuppressed :one
SELECT EXISTS (
    SELECT 1 FROM channel_suppressions
    WHERE team_id = sqlc.arg(team_id)
      AND channel = sqlc.arg(channel)
      AND normalized_address = sqlc.arg(normalized_address)
);

-- name: DeleteChannelSuppression :one
DELETE FROM channel_suppressions
WHERE team_id = sqlc.arg(team_id)
  AND channel = sqlc.arg(channel)
  AND normalized_address = sqlc.arg(normalized_address)
RETURNING *;
