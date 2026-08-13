-- name: BackofficeListSenderIDs :many
SELECT s.id,t.name AS team_name,s.name,s.country_code,s.status,s.created_at FROM sender_ids s JOIN teams t ON t.id=s.team_id WHERE (sqlc.arg(search)::text='' OR t.name ILIKE '%'||sqlc.arg(search)::text||'%' OR s.name ILIKE '%'||sqlc.arg(search)::text||'%' OR s.country_code ILIKE '%'||sqlc.arg(search)::text||'%') AND (sqlc.arg(status)::text='' OR s.status=sqlc.arg(status)) ORDER BY s.created_at DESC LIMIT 100;
-- name: BackofficeGetSenderID :one
SELECT s.id,s.team_id,t.name AS team_name,s.name,s.country_code,s.purpose,s.status,coalesce(s.provider,'') AS provider,coalesce(s.rejection_reason,'') AS rejection_reason,coalesce(to_char(s.approved_at,'YYYY-MM-DD HH24:MI'),'')::text AS approved_at,coalesce(to_char(s.rejected_at,'YYYY-MM-DD HH24:MI'),'')::text AS rejected_at,coalesce(to_char(s.suspended_at,'YYYY-MM-DD HH24:MI'),'')::text AS suspended_at,coalesce(s.created_by::text,'')::text AS created_by,s.created_at,s.updated_at FROM sender_ids s JOIN teams t ON t.id=s.team_id WHERE s.id=sqlc.arg(id);
-- name: BackofficeApproveSenderID :execrows
UPDATE sender_ids SET status='approved',approved_at=now(),rejected_at=NULL,rejection_reason=NULL,updated_at=now() WHERE id=sqlc.arg(id);
-- name: BackofficeRejectSenderID :execrows
UPDATE sender_ids SET status='rejected',rejected_at=now(),approved_at=NULL,rejection_reason=sqlc.arg(reason),updated_at=now() WHERE id=sqlc.arg(id);
