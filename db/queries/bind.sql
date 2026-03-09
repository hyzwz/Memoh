-- name: CreateBindCode :one
INSERT INTO channel_identity_bind_codes (token, issued_by_user_id, channel_type, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at;

-- name: CreatePendingBindCode :one
INSERT INTO channel_identity_bind_codes (token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at;

-- name: GetBindCode :one
SELECT id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at
FROM channel_identity_bind_codes
WHERE token = $1;

-- name: GetLatestPendingBindCodeForRequester :one
SELECT id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at
FROM channel_identity_bind_codes
WHERE issued_by_user_id = $1
  AND channel_type IS NOT DISTINCT FROM $2
  AND requested_by_channel_identity_id = $3
  AND used_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY created_at DESC
LIMIT 1;

-- name: GetBindCodeForUpdate :one
SELECT id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at
FROM channel_identity_bind_codes
WHERE token = $1
FOR UPDATE;

-- name: MarkBindCodeUsed :one
UPDATE channel_identity_bind_codes
SET used_at = now(), used_by_channel_identity_id = $2
WHERE id = $1
  AND used_at IS NULL
RETURNING id, token, issued_by_user_id, channel_type, expires_at, requested_by_channel_identity_id, used_at, used_by_channel_identity_id, created_at;
