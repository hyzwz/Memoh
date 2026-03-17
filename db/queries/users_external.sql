-- name: GetUserByWeComID :one
SELECT * FROM users
WHERE external_ids->>'wecom' = $1 AND is_active = true;

-- name: GetUserByWeChatID :one
SELECT * FROM users
WHERE external_ids->>'wechat' = $1 AND is_active = true;

-- name: UpdateUserExternalIDs :one
UPDATE users
SET external_ids = sqlc.arg(external_ids),
    updated_at = now()
WHERE id = $1
RETURNING *;
