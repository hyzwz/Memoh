-- 0031_bind_code_pairing_requests
-- Track pending pairing requests on bind codes so owners can approve them later.

ALTER TABLE channel_identity_bind_codes
  ADD COLUMN IF NOT EXISTS requested_by_channel_identity_id UUID REFERENCES channel_identities(id);

CREATE INDEX IF NOT EXISTS idx_channel_identity_bind_codes_requested_by
  ON channel_identity_bind_codes(requested_by_channel_identity_id);
