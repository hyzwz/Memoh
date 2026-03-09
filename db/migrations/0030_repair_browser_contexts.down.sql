-- 0030_repair_browser_contexts (rollback)
-- Remove the repaired browser_contexts objects.

ALTER TABLE bots
  DROP COLUMN IF EXISTS browser_context_id;

DROP TABLE IF EXISTS browser_contexts;
