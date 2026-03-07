-- Expression index for user-level conversation isolation queries.
-- Speeds up queries filtering by metadata->>'context_user_id'.
CREATE INDEX IF NOT EXISTS idx_bot_history_messages_context_user_id
  ON bot_history_messages ((metadata->>'context_user_id'))
  WHERE metadata->>'context_user_id' IS NOT NULL;
