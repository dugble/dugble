-- name: CreateMessageTemplate :one
INSERT INTO message_templates (team_id, name, alias, category)
VALUES (sqlc.arg(team_id), sqlc.arg(name), sqlc.narg(alias), sqlc.arg(category)::message_template_category)
RETURNING *;

-- name: ListMessageTemplates :many
SELECT mt.*
FROM message_templates AS mt
WHERE mt.team_id = sqlc.arg(team_id)
  AND mt.deleted_at IS NULL
  AND (mt.alias IS NULL OR left(mt.alias, length('__broadcast_')) <> '__broadcast_')
ORDER BY mt.created_at DESC, mt.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: GetMessageTemplateByID :one
SELECT mt.*
FROM message_templates AS mt
WHERE mt.id = sqlc.arg(id)
  AND mt.team_id = sqlc.arg(team_id)
  AND mt.deleted_at IS NULL;

-- name: GetMessageTemplateByAlias :one
SELECT mt.*
FROM message_templates AS mt
WHERE mt.team_id = sqlc.arg(team_id)
  AND lower(mt.alias) = lower(sqlc.arg(alias))
  AND mt.deleted_at IS NULL;

-- name: LockMessageTemplate :one
SELECT mt.*
FROM message_templates AS mt
WHERE mt.id = sqlc.arg(id)
  AND mt.team_id = sqlc.arg(team_id)
  AND mt.deleted_at IS NULL
FOR UPDATE;

-- name: UpdateMessageTemplateMetadata :one
UPDATE message_templates
SET name = sqlc.arg(name),
    alias = sqlc.narg(alias),
    category = sqlc.arg(category)::message_template_category,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SetMessageTemplateCurrentVersion :one
UPDATE message_templates
SET current_version_id = sqlc.arg(version_id),
    next_version_number = next_version_number + 1,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: PublishMessageTemplateVersion :one
UPDATE message_templates
SET published_version_id = sqlc.arg(version_id),
    published_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteMessageTemplate :one
UPDATE message_templates
SET deleted_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: CreateMessageTemplatePublication :one
INSERT INTO message_template_publications (team_id, template_id, version_id)
VALUES (sqlc.arg(team_id), sqlc.arg(template_id), sqlc.arg(version_id))
RETURNING id, team_id, template_id, version_id, published_at;

-- Deprecated compatibility query. Broadcasts no longer reference message
-- templates, so generated cleanup calls intentionally have no effect until the
-- old helper methods are removed from the template package.
-- name: DeleteUnreferencedBroadcastTemplate :exec
DELETE FROM message_templates
WHERE id = sqlc.arg(template_id)
  AND team_id = sqlc.arg(team_id)
  AND false;
