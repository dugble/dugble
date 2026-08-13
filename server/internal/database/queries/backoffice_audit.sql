-- name: BackofficeListAuditEvents :many
SELECT a.id, a.action, a.resource_type, a.resource_id, a.outcome,
       a.actor_type, coalesce(u.email, '') AS actor_email,
       coalesce(t.name, '') AS team_name, a.request_id, a.ip_address,
       a.metadata, a.created_at
FROM audit_events a
LEFT JOIN users u ON u.id = a.actor_user_id
LEFT JOIN teams t ON t.id = a.team_id
WHERE (sqlc.arg(search)::text = ''
       OR a.action ILIKE '%' || sqlc.arg(search)::text || '%'
       OR a.resource_type ILIKE '%' || sqlc.arg(search)::text || '%'
       OR a.resource_id ILIKE '%' || sqlc.arg(search)::text || '%'
       OR u.email ILIKE '%' || sqlc.arg(search)::text || '%'
       OR t.name ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(outcome)::text = '' OR a.outcome = sqlc.arg(outcome))
  AND (sqlc.arg(actor_type)::text = '' OR a.actor_type = sqlc.arg(actor_type))
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;
