-- +migrate Up
-- +migrate StatementBegin

ALTER TABLE operation_log ADD COLUMN IF NOT EXISTS trigger_type VARCHAR(50);
ALTER TABLE operation_log ADD COLUMN IF NOT EXISTS agent_suggestion_id BIGINT;
ALTER TABLE operation_log ADD COLUMN IF NOT EXISTS approval_id BIGINT;
ALTER TABLE operation_log ADD COLUMN IF NOT EXISTS entity_type VARCHAR(50);
ALTER TABLE operation_log ADD COLUMN IF NOT EXISTS entity_id BIGINT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_operation_log_trigger ON operation_log(trigger_type);
CREATE INDEX IF NOT EXISTS idx_operation_log_entity ON operation_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_operation_log_approval ON operation_log(approval_id);

-- +migrate StatementEnd
