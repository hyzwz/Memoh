-- 1010_remotefs: Add remote device management and file access audit
-- Supports desktop client file API via Tailscale

-- Add external identity mapping (WeChat Work, WeChat, etc.)
ALTER TABLE users ADD COLUMN IF NOT EXISTS external_ids JSONB NOT NULL DEFAULT '{}';

-- Partial indexes for external identity lookup
CREATE INDEX IF NOT EXISTS idx_users_external_wecom
  ON users ((external_ids->>'wecom'))
  WHERE external_ids->>'wecom' IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_external_wechat
  ON users ((external_ids->>'wechat'))
  WHERE external_ids->>'wechat' IS NOT NULL;

-- remote_devices: tracks desktop clients connected via Tailscale
CREATE TABLE IF NOT EXISTS remote_devices (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  hostname        TEXT NOT NULL,
  ts_ip           INET,
  ts_node_key     TEXT,
  platform        TEXT NOT NULL DEFAULT 'unknown',
  client_version  TEXT,
  status          TEXT NOT NULL DEFAULT 'offline',
  last_seen_at    TIMESTAMPTZ,
  share_policy    JSONB NOT NULL DEFAULT '{}',

  -- File access credentials
  api_token_hash  TEXT,
  token_issued_at TIMESTAMPTZ,
  token_expires_at TIMESTAMPTZ,

  -- Device lifecycle
  paired_at       TIMESTAMPTZ,
  revoked_at      TIMESTAMPTZ,
  revoke_reason   TEXT,

  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT remote_devices_user_hostname_unique UNIQUE(user_id, hostname)
);

CREATE INDEX IF NOT EXISTS idx_remote_devices_user_id ON remote_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_remote_devices_status ON remote_devices(status) WHERE status = 'online';

-- remotefs_audit_log: file operation audit trail (no FK for durability)
CREATE TABLE IF NOT EXISTS remotefs_audit_log (
  id          BIGSERIAL PRIMARY KEY,
  device_id   UUID NOT NULL,
  bot_id      UUID NOT NULL,
  user_id     UUID NOT NULL,
  action      TEXT NOT NULL,
  file_path   TEXT NOT NULL,
  file_size   BIGINT,
  result      TEXT NOT NULL,
  error_msg   TEXT,
  metadata    JSONB DEFAULT '{}',
  duration_ms INTEGER,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_remotefs_audit_device ON remotefs_audit_log(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_remotefs_audit_bot    ON remotefs_audit_log(bot_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_remotefs_audit_user   ON remotefs_audit_log(user_id, created_at DESC);
