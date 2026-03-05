-- name: UpsertModelPricing :one
INSERT INTO model_pricing (model_id, input_price_per_mtok, output_price_per_mtok, cache_read_price_per_mtok, cache_write_price_per_mtok, reasoning_price_per_mtok, effective_from)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (model_id, effective_from) DO UPDATE SET
    input_price_per_mtok = $2,
    output_price_per_mtok = $3,
    cache_read_price_per_mtok = $4,
    cache_write_price_per_mtok = $5,
    reasoning_price_per_mtok = $6
RETURNING *;

-- name: GetCurrentModelPricing :one
SELECT * FROM model_pricing
WHERE model_id = $1 AND effective_from <= now()
ORDER BY effective_from DESC
LIMIT 1;

-- name: ListModelPricing :many
SELECT * FROM model_pricing
WHERE model_id = $1
ORDER BY effective_from DESC;

-- name: CreateBudget :one
INSERT INTO budgets (scope_type, scope_id, period, limit_amount, alert_threshold, action_on_exceed, is_enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBudget :one
SELECT * FROM budgets
WHERE scope_type = $1 AND scope_id = $2 AND period = $3;

-- name: ListBudgets :many
SELECT * FROM budgets
WHERE (sqlc.narg(scope_type)::text IS NULL OR scope_type = sqlc.narg(scope_type))
ORDER BY scope_type, scope_id;

-- name: UpdateBudget :one
UPDATE budgets
SET limit_amount = $2, alert_threshold = $3, action_on_exceed = $4, is_enabled = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteBudget :exec
DELETE FROM budgets WHERE id = $1;
