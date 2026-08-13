-- name: CreateSMSMessage :one
INSERT INTO sms_messages (
    team_id,
    sender_id,
    to_number,
    from_name,
    body,
    status,
    segments,
    metadata,
    scheduled_at,
    destination_country
) VALUES (
    sqlc.arg(team_id),
    sqlc.narg(sender_id),
    sqlc.arg(to_number),
    sqlc.arg(from_name),
    sqlc.arg(body),
    sqlc.arg(status),
    sqlc.arg(segments),
    sqlc.arg(metadata),
    sqlc.narg(scheduled_at),
    sqlc.arg(destination_country)
)
RETURNING *;

-- name: ListSMSMessages :many
SELECT *
FROM sms_messages
WHERE team_id = sqlc.arg(team_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: GetSMSMessage :one
SELECT *
FROM sms_messages
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id);

-- name: ListSMSMessageEvents :many
SELECT event.id::text AS id,
       event.type,
       event.occurred_at,
       event.provider,
       event.code,
       event.message
FROM (
    SELECT message.id::text || ':accepted' AS id,
           'accepted'::text AS type,
           message.created_at AS occurred_at,
           NULL::text AS provider,
           NULL::text AS code,
           NULL::text AS message
    FROM sms_messages AS message
    WHERE message.id = sqlc.arg(message_id) AND message.team_id = sqlc.arg(team_id)
    UNION ALL
    SELECT attempt.id::text,
           attempt.status,
           COALESCE(attempt.terminal_at, attempt.submitted_at, attempt.request_completed_at,
                    attempt.request_started_at, attempt.claimed_at),
           attempt.provider,
           CASE attempt.status
               WHEN 'rejected' THEN 'SMS_REJECTED'
               WHEN 'expired' THEN 'SMS_EXPIRED'
               WHEN 'permanent_failure' THEN 'SMS_FAILED'
               WHEN 'retryable_failure' THEN 'SMS_RETRYABLE_FAILURE'
               WHEN 'submission_unknown' THEN 'SMS_SUBMISSION_UNKNOWN'
               ELSE NULL
           END,
           CASE attempt.status
               WHEN 'rejected' THEN 'SMS was rejected'
               WHEN 'expired' THEN 'SMS delivery expired'
               WHEN 'permanent_failure' THEN 'SMS delivery failed'
               WHEN 'retryable_failure' THEN 'SMS delivery will be retried'
               WHEN 'submission_unknown' THEN 'SMS submission status is unknown'
               ELSE NULL
           END
    FROM message_delivery_attempts AS attempt
    WHERE attempt.sms_message_id = sqlc.arg(message_id) AND attempt.team_id = sqlc.arg(team_id)
) AS event
ORDER BY event.occurred_at ASC, event.id ASC
LIMIT sqlc.arg(limit_count);

-- name: MarkSMSMessageSubmitted :one
UPDATE sms_messages
SET provider_id = sqlc.arg(provider_id),
    provider_message_id = sqlc.arg(provider_message_id),
    status = sqlc.arg(status),
    error_message = NULL,
    submitted_at = COALESCE(submitted_at, now()),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
RETURNING *;

-- name: MarkSMSMessageFailed :one
UPDATE sms_messages
SET status = 'failed',
    error_message = sqlc.arg(error_message),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
RETURNING *;

-- name: UpdateSMSMessageStatus :one
UPDATE sms_messages
SET status = sqlc.arg(status),
    delivered_at = CASE
        WHEN sqlc.arg(status) = 'delivered'
            THEN COALESCE(delivered_at, now())
        ELSE delivered_at
    END,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND status <> sqlc.arg(status)
  AND status NOT IN ('delivered', 'undelivered', 'rejected', 'failed', 'expired', 'unknown', 'canceled')
  AND (
      sqlc.arg(status) IN ('delivered', 'undelivered', 'rejected', 'failed', 'expired')
      OR CASE status
          WHEN 'queued' THEN 0
          WHEN 'processing' THEN 1
          WHEN 'submitted' THEN 2
          WHEN 'sent' THEN 3
          ELSE 4
      END < CASE sqlc.arg(status)
          WHEN 'queued' THEN 0
          WHEN 'processing' THEN 1
          WHEN 'submitted' THEN 2
          WHEN 'sent' THEN 3
          ELSE -1
      END
  )
RETURNING *;

-- name: FindApprovedSMSSender :one
SELECT sender_id.id
FROM sender_ids AS sender_id
WHERE sender_id.team_id = sqlc.arg(team_id)
  AND sender_id.normalized_name = lower(trim(sqlc.arg(name)))
  AND sender_id.status = 'approved'
  AND sender_id.provider_whitelisted
ORDER BY sender_id.created_at DESC
LIMIT 1;

-- name: MarkSMSMessageProcessing :one
UPDATE sms_messages
SET status = 'processing',
    error_message = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND status = 'queued'
  AND provider_message_id IS NULL
RETURNING *;

-- name: MarkSMSMessageDeliveryUnknown :one
UPDATE sms_messages
SET status = 'unknown',
    error_message = sqlc.arg(error_message),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND team_id = sqlc.arg(team_id)
  AND status = 'processing'
  AND provider_message_id IS NULL
RETURNING *;
