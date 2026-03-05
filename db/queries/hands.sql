-- name: CreateHand :one
INSERT INTO hands (bot_id, name, description, content, type, schedule_pattern, schedule_timezone, schedule_max_calls, triggers, allowed_tools, model_config, output_config, permissions, is_enabled, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetHandByID :one
SELECT * FROM hands WHERE id = $1;

-- name: GetHandByName :one
SELECT * FROM hands WHERE bot_id = $1 AND name = $2;

-- name: ListHands :many
SELECT * FROM hands
WHERE bot_id = $1
ORDER BY name;

-- name: ListEnabledHands :many
SELECT * FROM hands
WHERE bot_id = $1 AND is_enabled = true
ORDER BY name;

-- name: UpdateHand :one
UPDATE hands
SET name = $2, description = $3, content = $4, type = $5,
    schedule_pattern = $6, schedule_timezone = $7, schedule_max_calls = $8,
    triggers = $9, allowed_tools = $10, model_config = $11, output_config = $12,
    permissions = $13, is_enabled = $14, metadata = $15, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteHand :exec
DELETE FROM hands WHERE id = $1;

-- name: CreateHandExecutionLog :one
INSERT INTO hand_execution_logs (hand_id, bot_id, trigger_type, trigger_detail, status, model_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateHandExecutionLog :one
UPDATE hand_execution_logs
SET status = $2, result_text = $3, error_message = $4, usage = $5, completed_at = $6
WHERE id = $1
RETURNING *;

-- name: GetHandExecutionLog :one
SELECT * FROM hand_execution_logs WHERE id = $1;

-- name: ListHandExecutionLogs :many
SELECT * FROM hand_execution_logs
WHERE hand_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
