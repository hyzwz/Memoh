-- name: GetDeviceByID :one
SELECT * FROM remote_devices WHERE id = $1;

-- name: GetOnlineDevicesByUser :many
SELECT * FROM remote_devices
WHERE user_id = $1 AND status = 'online'
ORDER BY last_seen_at DESC;

-- name: GetDevicesByUser :many
SELECT * FROM remote_devices
WHERE user_id = $1 AND status != 'revoked'
ORDER BY last_seen_at DESC NULLS LAST;

-- name: UpsertDevice :one
INSERT INTO remote_devices (
  user_id, hostname, ts_ip, ts_node_key, platform, client_version,
  status, api_token_hash, token_issued_at, token_expires_at, paired_at,
  last_seen_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(hostname), sqlc.arg(ts_ip), sqlc.arg(ts_node_key),
  sqlc.arg(platform), sqlc.arg(client_version),
  'online', sqlc.arg(api_token_hash), now(), sqlc.arg(token_expires_at), now(),
  now()
)
ON CONFLICT (user_id, hostname) DO UPDATE SET
  ts_ip = EXCLUDED.ts_ip,
  ts_node_key = EXCLUDED.ts_node_key,
  platform = EXCLUDED.platform,
  client_version = EXCLUDED.client_version,
  status = 'online',
  api_token_hash = EXCLUDED.api_token_hash,
  token_issued_at = now(),
  token_expires_at = EXCLUDED.token_expires_at,
  last_seen_at = now(),
  revoked_at = NULL,
  revoke_reason = NULL,
  updated_at = now()
RETURNING *;

-- name: UpdateDeviceStatus :exec
UPDATE remote_devices
SET status = sqlc.arg(status),
    last_seen_at = CASE WHEN sqlc.arg(status)::text = 'online' THEN now() ELSE last_seen_at END,
    updated_at = now()
WHERE id = $1;

-- name: UpdateDeviceHeartbeat :exec
UPDATE remote_devices
SET last_seen_at = now(),
    ts_ip = sqlc.arg(ts_ip),
    updated_at = now()
WHERE id = $1 AND status = 'online';

-- name: RevokeDevice :exec
UPDATE remote_devices
SET status = 'revoked',
    revoked_at = now(),
    revoke_reason = sqlc.arg(revoke_reason),
    updated_at = now()
WHERE id = $1;

-- name: RefreshDeviceToken :exec
UPDATE remote_devices
SET api_token_hash = sqlc.arg(api_token_hash),
    token_issued_at = now(),
    token_expires_at = sqlc.arg(token_expires_at),
    updated_at = now()
WHERE id = $1;

-- name: CreateAuditLog :exec
INSERT INTO remotefs_audit_log (
  device_id, bot_id, user_id, action, file_path,
  file_size, result, error_msg, metadata, duration_ms
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: ListAuditLogsByDevice :many
SELECT * FROM remotefs_audit_log
WHERE device_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)
OFFSET sqlc.arg(offset_val);

-- name: ListAuditLogsByBot :many
SELECT * FROM remotefs_audit_log
WHERE bot_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)
OFFSET sqlc.arg(offset_val);
