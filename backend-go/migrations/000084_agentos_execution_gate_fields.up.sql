-- 000064: Add execution gate fields to unified_action and approval_request
-- Part of P0-P1 AgentOS Execution Gate (PR #276)

-- unified_action
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS execution_mode text NOT NULL DEFAULT 'production';
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS idempotency_key text DEFAULT '';
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS approved_by_user_id bigint;
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS executed_by_user_id bigint;
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS rejected_by_user_id bigint;

-- approval_request
ALTER TABLE approval_request ADD COLUMN IF NOT EXISTS requester_user_id bigint;
ALTER TABLE approval_request ADD COLUMN IF NOT EXISTS reviewer_user_id bigint;

-- indices for execution gate queries
CREATE INDEX IF NOT EXISTS idx_unified_action_idempotency_key ON unified_action (idempotency_key) WHERE idempotency_key != '';
CREATE INDEX IF NOT EXISTS idx_unified_action_execution_mode ON unified_action (execution_mode);
CREATE INDEX IF NOT EXISTS idx_unified_action_approved_by_user_id ON unified_action (approved_by_user_id);
CREATE INDEX IF NOT EXISTS idx_unified_action_executed_by_user_id ON unified_action (executed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_approval_request_requester_user_id ON approval_request (requester_user_id);
CREATE INDEX IF NOT EXISTS idx_approval_request_reviewer_user_id ON approval_request (reviewer_user_id);
