-- 0031_bind_code_pairing_requests
-- Remove pending pairing request tracking from bind codes.

DROP INDEX IF EXISTS idx_channel_identity_bind_codes_requested_by;

ALTER TABLE channel_identity_bind_codes
  DROP COLUMN IF EXISTS requested_by_channel_identity_id;
