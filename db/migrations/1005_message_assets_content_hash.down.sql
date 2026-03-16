-- Revert bot_history_message_assets back to asset_id schema.

ALTER TABLE bot_history_message_assets
  ADD COLUMN IF NOT EXISTS asset_id UUID;

ALTER TABLE bot_history_message_assets
  DROP CONSTRAINT IF EXISTS message_asset_content_unique;

ALTER TABLE bot_history_message_assets
  ADD CONSTRAINT message_asset_unique UNIQUE (message_id, asset_id);

CREATE INDEX IF NOT EXISTS idx_message_assets_asset_id ON bot_history_message_assets(asset_id);

ALTER TABLE bot_history_message_assets
  DROP COLUMN IF EXISTS content_hash;
