DROP INDEX IF EXISTS idx_approval_request_target;

ALTER TABLE approval_request
  DROP COLUMN IF EXISTS risk_level,
  DROP COLUMN IF EXISTS target_id,
  DROP COLUMN IF EXISTS target_type;
