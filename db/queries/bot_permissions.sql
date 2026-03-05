-- name: UpsertBotPermission :one
INSERT INTO bot_permissions (bot_id, role, resource, action, allowed)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (bot_id, role, resource, action) DO UPDATE SET allowed = $5
RETURNING *;

-- name: DeleteBotPermission :exec
DELETE FROM bot_permissions WHERE id = $1;

-- name: ListBotPermissions :many
SELECT * FROM bot_permissions
WHERE bot_id = $1
ORDER BY role, resource, action;

-- name: GetBotPermission :one
SELECT * FROM bot_permissions
WHERE bot_id = $1 AND role = $2 AND resource = $3 AND action = $4;

-- name: DeleteBotPermissionsByBot :exec
DELETE FROM bot_permissions WHERE bot_id = $1;
