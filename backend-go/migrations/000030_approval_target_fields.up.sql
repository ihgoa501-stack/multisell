ALTER TABLE approval_request
  ADD COLUMN IF NOT EXISTS target_type VARCHAR(64),
  ADD COLUMN IF NOT EXISTS target_id BIGINT,
  ADD COLUMN IF NOT EXISTS risk_level VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_approval_request_target
  ON approval_request(target_type, target_id, request_type, status);
