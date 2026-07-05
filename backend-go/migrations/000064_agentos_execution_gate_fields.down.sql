-- 000064 down: Remove execution gate fields

ALTER TABLE unified_action DROP COLUMN IF EXISTS execution_mode;
ALTER TABLE unified_action DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE unified_action DROP COLUMN IF EXISTS approved_by_user_id;
ALTER TABLE unified_action DROP COLUMN IF EXISTS executed_by_user_id;
ALTER TABLE unified_action DROP COLUMN IF EXISTS rejected_by_user_id;

ALTER TABLE approval_request DROP COLUMN IF EXISTS requester_user_id;
ALTER TABLE approval_request DROP COLUMN IF EXISTS reviewer_user_id;

DROP INDEX IF EXISTS idx_unified_action_idempotency_key;
DROP INDEX IF EXISTS idx_unified_action_execution_mode;
DROP INDEX IF EXISTS idx_unified_action_approved_by_user_id;
DROP INDEX IF EXISTS idx_unified_action_executed_by_user_id;
DROP INDEX IF EXISTS idx_approval_request_requester_user_id;
DROP INDEX IF EXISTS idx_approval_request_reviewer_user_id;
