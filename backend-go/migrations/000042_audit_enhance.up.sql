-- 000029: Add user_id and result columns to operation_log for audit enhancement

ALTER TABLE operation_log
  ADD COLUMN IF NOT EXISTS user_id BIGINT;

ALTER TABLE operation_log
  ADD COLUMN IF NOT EXISTS result VARCHAR(20) DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_operation_log_user_id ON operation_log(user_id);
