-- 0029_context_user_id_index (rollback)
-- Remove the expression index used for user-level conversation isolation queries.
DROP INDEX IF EXISTS idx_bot_history_messages_context_user_id;
