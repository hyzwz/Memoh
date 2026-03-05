-- name: InsertAuditLog :one
INSERT INTO audit_logs (user_id, bot_id, action, resource_type, resource_id, detail, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: BatchInsertAuditLogs :copyfrom
INSERT INTO audit_logs (user_id, bot_id, action, resource_type, resource_id, detail, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id))
  AND (sqlc.narg(bot_id)::uuid IS NULL OR bot_id = sqlc.narg(bot_id))
  AND (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action))
  AND (sqlc.narg(resource_type)::text IS NULL OR resource_type = sqlc.narg(resource_type))
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountAuditLogs :one
SELECT COUNT(*)::bigint AS count FROM audit_logs
WHERE (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id))
  AND (sqlc.narg(bot_id)::uuid IS NULL OR bot_id = sqlc.narg(bot_id))
  AND (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action))
  AND (sqlc.narg(resource_type)::text IS NULL OR resource_type = sqlc.narg(resource_type));
