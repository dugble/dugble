-- name: BackofficeListEmailMessages :many
SELECT m.id, t.name AS team_name, m.from_email, m.to_email, m.subject,
       m.status, coalesce(m.provider, m.delivery_provider) AS provider,
       coalesce(m.error_message, '') AS error_message, m.created_at
FROM email_messages m
JOIN teams t ON t.id = m.team_id
WHERE (sqlc.arg(search)::text = '' OR t.name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.from_email ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.to_email ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.subject ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.id::text = sqlc.arg(search)::text)
  AND (sqlc.arg(status)::text = '' OR m.status = sqlc.arg(status))
ORDER BY m.created_at DESC
LIMIT 100;

-- name: BackofficeGetEmailMessage :one
SELECT m.id, m.team_id, t.name AS team_name, m.sender_domain_id,
       m.message_type, m.from_email, m.from_name, m.reply_to_email,
       m.to_email, m.to_name, m.subject, m.html_body, m.text_body,
       m.status, m.delivery_provider, m.provider_region, m.provider,
       m.provider_message_id, m.error_code, m.error_message,
       m.metadata, m.recipients, m.headers, m.attachments, m.tags,
       m.scheduled_at, m.queued_at, m.processing_at, m.submitted_at,
       m.delivered_at, m.failed_at, m.created_at, m.updated_at
FROM email_messages m JOIN teams t ON t.id = m.team_id
WHERE m.id = sqlc.arg(id);

-- name: BackofficeListEmailRecipients :many
SELECT recipient_email, recipient_type, status, last_event_type, last_event_at,
       error_code, error_message, delivered_at, failed_at
FROM email_recipients WHERE email_message_id = sqlc.arg(message_id)
ORDER BY recipient_type, recipient_email;
