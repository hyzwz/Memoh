-- name: CreateModelRoute :one
INSERT INTO bot_model_routes (bot_id, name, model_id, priority, complexity_tier, is_enabled, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetModelRouteByID :one
SELECT * FROM bot_model_routes WHERE id = $1;

-- name: ListModelRoutes :many
SELECT * FROM bot_model_routes
WHERE bot_id = $1
ORDER BY priority DESC, name;

-- name: ListEnabledModelRoutes :many
SELECT * FROM bot_model_routes
WHERE bot_id = $1 AND is_enabled = true
ORDER BY priority DESC;

-- name: GetModelRouteByTier :one
SELECT * FROM bot_model_routes
WHERE bot_id = $1 AND complexity_tier = $2 AND is_enabled = true
ORDER BY priority DESC
LIMIT 1;

-- name: UpdateModelRoute :one
UPDATE bot_model_routes
SET name = $2, model_id = $3, priority = $4, complexity_tier = $5, is_enabled = $6, metadata = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteModelRoute :exec
DELETE FROM bot_model_routes WHERE id = $1;
