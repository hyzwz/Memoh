-- 0026_enterprise_features.down.sql
-- Reverse enterprise features migration

DROP TABLE IF EXISTS cockpit_config CASCADE;
DROP TABLE IF EXISTS cockpit_daily_reports CASCADE;
DROP TRIGGER IF EXISTS trg_hand_execution_bot_consistency ON hand_execution_logs;
DROP FUNCTION IF EXISTS check_hand_bot_consistency();
DROP TABLE IF EXISTS hand_execution_logs CASCADE;
DROP TABLE IF EXISTS hands CASCADE;
DROP TABLE IF EXISTS budgets CASCADE;
DROP TABLE IF EXISTS model_pricing CASCADE;
DROP TABLE IF EXISTS bot_model_routes CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS bot_permissions CASCADE;
DROP TABLE IF EXISTS bot_department_access CASCADE;
DROP TABLE IF EXISTS department_members CASCADE;
DROP TABLE IF EXISTS departments CASCADE;

-- Note: Cannot remove enum values from user_role in PostgreSQL.
-- The added values (dept_admin, org_admin, super_admin) will remain.
