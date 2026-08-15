-- name: ResolveEmailSandboxRecipientForToken :one
SELECT users.email, users.email_verified
FROM team_tokens AS token
JOIN users ON users.id = token.created_by
WHERE token.id = sqlc.arg(token_id)
  AND token.team_id = sqlc.arg(team_id)
  AND token.revoked_at IS NULL
  AND (token.expires_at IS NULL OR token.expires_at > now());

-- name: ResolveEmailSenderDomain :one
SELECT domain_record.id,
       domain_record.provider,
       domain_record.provider_region,
       domain_record.status,
       domain_record.health_status
FROM domains AS domain_record
WHERE domain_record.team_id = sqlc.arg(team_id)
  AND domain_record.normalized_name = lower(trim(sqlc.arg(domain_name)))
  AND domain_record.disabled_at IS NULL
ORDER BY domain_record.created_at DESC
LIMIT 1;

-- name: CreateEmailMessage :one
INSERT INTO email_messages (
    team_id,
    sender_domain_id,
    delivery_provider,
    provider_region,
    message_type,
    from_email,
    from_name,
    reply_to_email,
    to_email,
    to_name,
    subject,
    html_body,
    text_body,
    status,
    metadata,
    recipients,
    headers,
    attachments,
    tags,
    scheduled_at
)
SELECT
    team.id,
    sqlc.narg(sender_domain_id),
    sqlc.arg(delivery_provider),
    sqlc.arg(provider_region),
    sqlc.arg(message_type),
    sqlc.arg(from_email),
    sqlc.narg(from_name),
    sqlc.narg(reply_to_email),
    sqlc.arg(to_email),
    sqlc.narg(to_name),
    sqlc.arg(subject),
    sqlc.narg(html_body),
    sqlc.narg(text_body),
    'queued',
    sqlc.arg(metadata),
    sqlc.arg(recipients),
    sqlc.arg(headers),
    sqlc.arg(attachments),
    sqlc.arg(tags),
    sqlc.narg(scheduled_at)
FROM teams AS team
WHERE team.id = sqlc.arg(team_id)
  AND team.status = 'active'
RETURNING *;

-- name: GetEmailMessage :one
SELECT message.*
FROM email_messages AS message
JOIN teams AS team ON team.id = message.team_id
WHERE message.id = sqlc.arg(id)
  AND message.team_id = sqlc.arg(team_id)
  AND team.status = 'active';

-- name: ListEmailMessages :many
SELECT
    message.id,
    message.team_id,
    message.to_email,
    message.to_name,
    message.subject,
    message.status,
    message.provider,
    message.queued_at,
    message.submitted_at,
    message.delivered_at,
    message.created_at
FROM email_messages AS message
JOIN teams AS team ON team.id = message.team_id
WHERE message.team_id = sqlc.arg(team_id)
  AND team.status = 'active'
ORDER BY message.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: ListEmailMessageEvents :many
SELECT event.id::text AS id, event.type, event.occurred_at, event.provider, event.code, event.message
FROM (
    SELECT message.id::text || ':accepted' AS id,
           'accepted'::text AS type,
           message.created_at AS occurred_at,
           NULL::text AS provider,
           NULL::text AS code,
           NULL::text AS message
    FROM email_messages AS message
    WHERE message.id = sqlc.arg(message_id) AND message.team_id = sqlc.arg(team_id)
    UNION ALL
    SELECT attempt.id::text,
           attempt.status,
           COALESCE(attempt.terminal_at, attempt.submitted_at, attempt.request_completed_at,
                    attempt.request_started_at, attempt.claimed_at),
           attempt.provider,
           CASE attempt.status
               WHEN 'rejected' THEN 'EMAIL_REJECTED'
               WHEN 'expired' THEN 'EMAIL_EXPIRED'
               WHEN 'permanent_failure' THEN 'EMAIL_FAILED'
               WHEN 'retryable_failure' THEN 'EMAIL_RETRYABLE_FAILURE'
               WHEN 'submission_unknown' THEN 'EMAIL_SUBMISSION_UNKNOWN'
               ELSE NULL
           END,
           CASE attempt.status
               WHEN 'rejected' THEN 'Email was rejected'
               WHEN 'expired' THEN 'Email delivery expired'
               WHEN 'permanent_failure' THEN 'Email delivery failed'
               WHEN 'retryable_failure' THEN 'Email delivery will be retried'
               WHEN 'submission_unknown' THEN 'Email submission status is unknown'
               ELSE NULL
           END
    FROM message_delivery_attempts AS attempt
    WHERE attempt.email_message_id = sqlc.arg(message_id) AND attempt.team_id = sqlc.arg(team_id)
    UNION ALL
    SELECT provider_event.id::text,
           CASE provider_event.event_type
               WHEN 'delivery' THEN 'delivered'
               WHEN 'delivery_delay' THEN 'delayed'
               WHEN 'bounce' THEN 'bounced'
               WHEN 'complaint' THEN 'complained'
               WHEN 'reject' THEN 'rejected'
               ELSE provider_event.event_type
           END,
           provider_event.occurred_at,
           provider_event.provider,
           CASE provider_event.event_type
               WHEN 'bounce' THEN 'EMAIL_BOUNCED'
               WHEN 'complaint' THEN 'EMAIL_COMPLAINED'
               WHEN 'reject' THEN 'EMAIL_REJECTED'
               ELSE NULL
           END,
           NULL::text
    FROM email_provider_events AS provider_event
    JOIN email_messages AS message ON message.id = provider_event.email_message_id
    WHERE provider_event.email_message_id = sqlc.arg(message_id) AND message.team_id = sqlc.arg(team_id)
) AS event
ORDER BY event.occurred_at ASC, event.id ASC
LIMIT sqlc.arg(limit_count);

-- name: GetEmailMessageScheduleForUpdate :one
SELECT status, scheduled_at
FROM email_messages
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id)
FOR UPDATE;

-- name: CancelEmailMessage :exec
UPDATE email_messages
SET status = 'canceled', updated_at = now()
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id);

-- name: RescheduleEmailMessage :exec
UPDATE email_messages
SET scheduled_at = sqlc.arg(scheduled_at), updated_at = now()
WHERE id = sqlc.arg(id) AND team_id = sqlc.arg(team_id);
