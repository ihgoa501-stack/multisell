-- 000029 rollback: Remove audit enhancement columns and index

DROP INDEX IF EXISTS idx_operation_log_user_id;

ALTER TABLE operation_log
  DROP COLUMN IF EXISTS user_id;

ALTER TABLE operation_log
  DROP COLUMN IF EXISTS result;
