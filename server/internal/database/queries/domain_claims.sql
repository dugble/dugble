-- name: GetActiveDomainForClaimByName :one
SELECT domain_record.*
FROM domains AS domain_record
WHERE domain_record.normalized_name = lower(trim(sqlc.arg(name)))
  AND domain_record.disabled_at IS NULL
ORDER BY domain_record.created_at DESC
LIMIT 1
FOR SHARE;

-- name: GetDomainForClaimByID :one
SELECT domain_record.*
FROM domains AS domain_record
WHERE domain_record.id = sqlc.arg(id);

-- name: CreateDomainClaim :one
INSERT INTO domain_claims (
    target_domain_id,
    source_domain_id,
    normalized_name,
    source_team_id,
    target_team_id,
    provider_region,
    custom_return_path,
    tls_mode,
    record_name,
    record_value,
    record_ttl,
    expires_at,
    created_by
) VALUES (
    sqlc.arg(target_domain_id),
    sqlc.arg(source_domain_id),
    lower(trim(sqlc.arg(normalized_name))),
    sqlc.arg(source_team_id),
    sqlc.arg(target_team_id),
    lower(trim(sqlc.arg(provider_region))),
    lower(trim(sqlc.arg(custom_return_path))),
    sqlc.arg(tls_mode),
    sqlc.arg(record_name),
    sqlc.arg(record_value),
    sqlc.arg(record_ttl),
    sqlc.arg(expires_at),
    sqlc.arg(created_by)
)
RETURNING *;

-- name: GetDomainClaim :one
SELECT claim.*
FROM domain_claims AS claim
WHERE claim.target_domain_id = sqlc.arg(target_domain_id)
  AND claim.target_team_id = sqlc.arg(target_team_id);

-- name: GetDomainClaimByID :one
SELECT claim.*
FROM domain_claims AS claim
WHERE claim.id = sqlc.arg(id);

-- name: RequestDomainClaimVerification :one
UPDATE domain_claims
SET verification_requested_at = now(),
    blocked_reason = NULL,
    failure_reason = NULL,
    updated_at = now()
WHERE target_domain_id = sqlc.arg(target_domain_id)
  AND target_team_id = sqlc.arg(target_team_id)
  AND status IN ('pending', 'blocked')
  AND expires_at > now()
RETURNING *;

-- name: ClaimPendingDomainClaims :many
WITH candidates AS (
    SELECT claim.id
    FROM domain_claims AS claim
    WHERE claim.status IN ('pending', 'verified', 'blocked')
      AND claim.verification_requested_at IS NOT NULL
      AND claim.expires_at > now()
      AND (
          claim.reconcile_locked_at IS NULL
          OR claim.reconcile_locked_at < sqlc.arg(stale_before)
      )
    ORDER BY claim.verification_requested_at, claim.created_at
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(claim_limit)
), updated AS (
    UPDATE domain_claims AS claim
    SET reconcile_locked_at = now(),
        reconcile_locked_by = trim(sqlc.arg(worker_id)),
        updated_at = now()
    FROM candidates
    WHERE claim.id = candidates.id
    RETURNING claim.*
)
SELECT updated.*
FROM updated
ORDER BY updated.verification_requested_at, updated.created_at;

-- name: MarkDomainClaimVerified :one
UPDATE domain_claims
SET status = 'verified',
    verified_at = COALESCE(verified_at, now()),
    blocked_reason = NULL,
    failure_reason = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id))
RETURNING *;

-- name: MarkDomainClaimBlocked :one
UPDATE domain_claims
SET status = 'blocked',
    blocked_reason = sqlc.arg(blocked_reason),
    failure_reason = NULL,
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id))
RETURNING *;

-- name: MarkDomainClaimFailed :one
UPDATE domain_claims
SET status = 'failed',
    failure_reason = sqlc.arg(failure_reason),
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id))
RETURNING *;

-- name: MarkDomainClaimCompleted :one
UPDATE domain_claims
SET status = 'completed',
    completed_at = COALESCE(completed_at, now()),
    blocked_reason = NULL,
    failure_reason = NULL,
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'verified'
  AND reconcile_locked_by = trim(sqlc.arg(worker_id))
RETURNING *;

-- name: CancelDomainClaim :one
UPDATE domain_claims
SET status = 'canceled',
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE target_domain_id = sqlc.arg(target_domain_id)
  AND target_team_id = sqlc.arg(target_team_id)
  AND status IN ('pending', 'verified', 'blocked')
RETURNING *;

-- name: ExpireDomainClaims :many
UPDATE domain_claims
SET status = 'expired',
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE status IN ('pending', 'verified', 'blocked')
  AND expires_at <= now()
RETURNING *;

-- name: DomainHasPendingScheduledEmails :one
SELECT EXISTS (
    SELECT 1
    FROM email_messages AS message
    WHERE message.sender_domain_id = sqlc.arg(domain_id)
      AND message.status = 'queued'
      AND message.scheduled_at IS NOT NULL
      AND message.scheduled_at > now()
);

-- name: DomainHasRecentOwnerActivity :one
SELECT EXISTS (
    SELECT 1
    FROM email_messages AS message
    WHERE message.sender_domain_id = sqlc.arg(domain_id)
      AND message.created_at >= sqlc.arg(since)
);

-- name: ArchiveClaimSourceDomain :one
UPDATE domains
SET status = 'disabled',
    disabled_at = COALESCE(disabled_at, now()),
    next_check_at = now(),
    reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
RETURNING *;

-- name: CreateClaimTargetDomain :one
INSERT INTO domains (
    id,
    team_id,
    name,
    normalized_name,
    provider,
    provider_account,
    provider_region,
    status,
    provider_status,
    tls_mode,
    custom_return_path,
    health_status,
    submitted_at,
    next_check_at,
    created_by
) VALUES (
    sqlc.arg(id),
    sqlc.arg(team_id),
    sqlc.arg(name),
    lower(trim(sqlc.arg(name))),
    lower(trim(sqlc.arg(provider))),
    lower(trim(sqlc.arg(provider_account))),
    lower(trim(sqlc.arg(provider_region))),
    'pending',
    'pending',
    sqlc.arg(tls_mode),
    lower(trim(sqlc.arg(custom_return_path))),
    'unknown',
    now(),
    now() + interval '1 minute',
    sqlc.narg(created_by)
)
RETURNING *;

-- name: DeleteCurrentClaimTargetDNSRecords :exec
DELETE FROM domain_dns_records
WHERE domain_id = sqlc.arg(domain_id)
  AND is_current
  AND purpose <> 'tracking';

-- name: CreateClaimTargetDNSRecord :one
INSERT INTO domain_dns_records (
    domain_id,
    purpose,
    record,
    name,
    type,
    value,
    ttl,
    priority,
    status,
    is_current,
    verified_at
) VALUES (
    sqlc.arg(domain_id),
    lower(trim(sqlc.arg(purpose))),
    sqlc.arg(record),
    sqlc.arg(name),
    sqlc.arg(type),
    sqlc.arg(value),
    sqlc.arg(ttl),
    sqlc.narg(priority),
    sqlc.arg(status),
    true,
    CASE WHEN sqlc.arg(status) = 'verified' THEN now() ELSE NULL END
)
ON CONFLICT (domain_id, purpose, name, type, value)
DO UPDATE SET record = EXCLUDED.record,
              ttl = EXCLUDED.ttl,
              priority = EXCLUDED.priority,
              status = EXCLUDED.status,
              is_current = true,
              verified_at = CASE
                  WHEN EXCLUDED.status = 'verified'
                      THEN COALESCE(domain_dns_records.verified_at, now())
                  ELSE domain_dns_records.verified_at
              END,
              superseded_at = NULL,
              updated_at = now()
RETURNING *;

-- name: ReleaseDomainClaim :one
UPDATE domain_claims
SET reconcile_locked_at = NULL,
    reconcile_locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND reconcile_locked_by = trim(sqlc.arg(worker_id))
RETURNING *;
