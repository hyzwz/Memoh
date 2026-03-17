-- 1010_remotefs down: remove remote device management

DROP INDEX IF EXISTS idx_remotefs_audit_user;
DROP INDEX IF EXISTS idx_remotefs_audit_bot;
DROP INDEX IF EXISTS idx_remotefs_audit_device;
DROP TABLE IF EXISTS remotefs_audit_log;

DROP INDEX IF EXISTS idx_remote_devices_status;
DROP INDEX IF EXISTS idx_remote_devices_user_id;
DROP TABLE IF EXISTS remote_devices;

DROP INDEX IF EXISTS idx_users_external_wechat;
DROP INDEX IF EXISTS idx_users_external_wecom;
ALTER TABLE users DROP COLUMN IF EXISTS external_ids;
