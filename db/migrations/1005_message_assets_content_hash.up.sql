-- Migrate bot_history_message_assets from asset_id schema to content_hash schema.
-- The canonical 0001_init.up.sql already uses content_hash; this brings existing
-- databases in line.

-- Step 1: add content_hash column if it doesn't exist.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'bot_history_message_assets' AND column_name = 'content_hash'
  ) THEN
    ALTER TABLE bot_history_message_assets ADD COLUMN content_hash TEXT;
  END IF;
END $$;

-- Step 2: backfill content_hash from the media_assets table via asset_id.
UPDATE bot_history_message_assets bma
SET content_hash = ma.content_hash
FROM media_assets ma
WHERE bma.asset_id = ma.id
  AND bma.content_hash IS NULL;

-- Step 3: default any remaining NULLs (orphaned rows) to empty string.
UPDATE bot_history_message_assets
SET content_hash = ''
WHERE content_hash IS NULL;

-- Step 4: make the column NOT NULL now that all rows are backfilled.
ALTER TABLE bot_history_message_assets ALTER COLUMN content_hash SET NOT NULL;

-- Step 5: drop the old unique constraint and create the new one.
ALTER TABLE bot_history_message_assets
  DROP CONSTRAINT IF EXISTS message_asset_unique;

ALTER TABLE bot_history_message_assets
  ADD CONSTRAINT message_asset_content_unique UNIQUE (message_id, content_hash);

-- Step 6: drop the asset_id column and its index (no longer needed).
DROP INDEX IF EXISTS idx_message_assets_asset_id;

ALTER TABLE bot_history_message_assets
  DROP COLUMN IF EXISTS asset_id;
