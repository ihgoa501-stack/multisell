DROP INDEX IF EXISTS idx_operation_log_record_hash;
DROP TRIGGER IF EXISTS trg_operation_log_append_only ON operation_log;
DROP FUNCTION IF EXISTS protect_operation_log_chain();
DROP FUNCTION IF EXISTS audit_operation_hash(TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, BIGINT, TEXT, TEXT, INTEGER, TEXT, BIGINT, BIGINT, TEXT, BIGINT, TIMESTAMPTZ, TEXT);
ALTER TABLE operation_log DROP COLUMN IF EXISTS correlation_id, DROP COLUMN IF EXISTS record_hash, DROP COLUMN IF EXISTS previous_hash;
