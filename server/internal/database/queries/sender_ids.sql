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

-- name: ClaimPendingSenderIDRegistrations :many
WITH candidates AS (
    SELECT sender_id.id
    FROM sender_ids AS sender_id
    WHERE sender_id.country_code = 'GH'
      AND lower(sender_id.provider) = lower(trim(sqlc.arg(provider_id)))
      AND sender_id.status = 'pending'
      AND sender_id.next_check_at <= now()
      AND (
          sender_id.reconcile_locked_at IS NULL
          OR sender_id.reconcile_locked_at < sqlc.arg(stale_before)
      )
    ORDER BY sender_id.next_check_at, sender_id.created_at, sender_id.id
    FOR UPDATE OF sender_id SKIP LOCKED
    LIMIT sqlc.arg(claim_limit)
), updated AS (
    UPDATE sender_ids AS sender_id
    SET reconcile_locked_at = now(),
        reconcile_locked_by = trim(sqlc.arg(worker_id)),
        reconciliation_attempts = sender_id.reconciliation_attempts + 1,
        updated_at = now()
    FROM candidates
    WHERE sender_id.id = candidates.id
    RETURNING sender_id.*
)
SELECT sender_id.id,
       sender_id.team_id,
       sender_id.name,
       sender_id.country_code::text AS country_code,
       COALESCE(sender_id.purpose, '') AS purpose,
       COALESCE(sender_id.provider, '') AS provider,
       COALESCE(sender_id.provider_status, '') AS provider_status,
       sender_id.submitted_at,
       sender_id.reconciliation_attempts
FROM updated AS sender_id
ORDER BY sender_id.next_check_at, sender_id.created_at, sender_id.id;

-- name: CompleteSenderIDSubmission :execrows
UPDATE sender_ids
SET provider_status = trim(sqlc.arg(provider_status)),
    submitted_at = COALESCE(submitted_at, now()),
    last_checked_at = now(),
    next_check_at = sqlc.arg(next_check_at),
    reconciliation_attempts = 0,
    last_error = NULL,
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id));

-- name: CompleteSenderIDStatus :execrows
UPDATE sender_ids
SET status = lower(trim(sqlc.arg(status))),
    provider_status = trim(sqlc.arg(provider_status)),
    provider_whitelisted = sqlc.arg(whitelisted),
    health_status = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'approved' AND sqlc.arg(whitelisted) THEN 'healthy'
        WHEN lower(trim(sqlc.arg(status))) IN ('rejected', 'suspended') THEN 'degraded'
        ELSE health_status
    END,
    submitted_at = COALESCE(submitted_at, now()),
    last_checked_at = now(),
    next_check_at = sqlc.arg(next_check_at),
    reconciliation_attempts = 0,
    last_error = NULL,
    rejection_reason = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'rejected' THEN sqlc.narg(rejection_reason)
        ELSE NULL
    END,
    approved_at = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'approved' THEN COALESCE(approved_at, now())
        ELSE approved_at
    END,
    rejected_at = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'rejected' THEN COALESCE(rejected_at, now())
        ELSE rejected_at
    END,
    suspended_at = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'suspended' THEN COALESCE(suspended_at, now())
        ELSE suspended_at
    END,
    disabled_at = CASE
        WHEN lower(trim(sqlc.arg(status))) = 'inactive' THEN COALESCE(disabled_at, now())
        ELSE disabled_at
    END,
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id));

-- name: RecordSenderIDProviderFailure :execrows
UPDATE sender_ids
SET provider_status = COALESCE(NULLIF(trim(sqlc.arg(provider_status)), ''), provider_status),
    last_checked_at = now(),
    next_check_at = sqlc.arg(next_check_at),
    last_error = sqlc.arg(last_error),
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id));
