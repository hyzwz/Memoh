-- name: UpsertCockpitDailyReport :one
INSERT INTO cockpit_daily_reports (bot_id, user_id, hand_execution_id, report_date, summary, tasks, pending_tasks, total_estimated_saved_hours, total_ai_time_minutes, efficiency_multiplier, innovation_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (bot_id, user_id, report_date) DO UPDATE SET
    hand_execution_id = $3,
    summary = $5,
    tasks = $6,
    pending_tasks = $7,
    total_estimated_saved_hours = $8,
    total_ai_time_minutes = $9,
    efficiency_multiplier = $10,
    innovation_count = $11
RETURNING *;

-- name: GetCockpitDailyReport :one
SELECT * FROM cockpit_daily_reports
WHERE bot_id = $1 AND user_id = $2 AND report_date = $3;

-- name: ListCockpitDailyReports :many
SELECT * FROM cockpit_daily_reports
WHERE bot_id = $1
  AND report_date >= sqlc.arg(start_date)
  AND report_date <= sqlc.arg(end_date)
ORDER BY report_date DESC;

-- name: ListCockpitDailyReportsByUser :many
SELECT * FROM cockpit_daily_reports
WHERE user_id = $1
  AND report_date >= sqlc.arg(start_date)
  AND report_date <= sqlc.arg(end_date)
ORDER BY report_date DESC;

-- name: GetCockpitConfig :one
SELECT * FROM cockpit_config WHERE key = $1;

-- name: UpsertCockpitConfig :one
INSERT INTO cockpit_config (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now()
RETURNING *;

-- name: ListCockpitConfigs :many
SELECT * FROM cockpit_config ORDER BY key;

-- name: CockpitDailyMessageStats :one
-- Counts messages and unique conversations for a bot on a given date.
SELECT
    COUNT(*)::bigint AS message_count,
    COUNT(DISTINCT m.conversation_id)::bigint AS conversation_count
FROM bot_history_messages m
WHERE m.bot_id = sqlc.arg(bot_id)
  AND m.created_at >= sqlc.arg(day_start)
  AND m.created_at < sqlc.arg(day_end);

-- name: CockpitDailyHandExecutionStats :one
-- Counts hand executions for a bot on a given date.
SELECT
    COUNT(*)::bigint AS total_executions,
    COUNT(*) FILTER (WHERE status = 'completed')::bigint AS completed_executions,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_executions
FROM hand_execution_logs
WHERE bot_id = sqlc.arg(bot_id)
  AND created_at >= sqlc.arg(day_start)
  AND created_at < sqlc.arg(day_end);

-- name: CockpitAggregateByDateRange :one
SELECT
    COUNT(*)::bigint AS total_reports,
    COALESCE(SUM(total_estimated_saved_hours), 0)::numeric AS total_saved_hours,
    COALESCE(SUM(total_ai_time_minutes), 0)::bigint AS total_ai_minutes,
    COALESCE(SUM(innovation_count), 0)::bigint AS total_innovations,
    COALESCE(AVG(efficiency_multiplier), 0)::numeric AS avg_efficiency
FROM cockpit_daily_reports
WHERE bot_id = $1
  AND report_date >= sqlc.arg(start_date)
  AND report_date <= sqlc.arg(end_date);
