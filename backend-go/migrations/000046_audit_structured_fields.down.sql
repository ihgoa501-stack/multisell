-- +migrate Down
-- +migrate StatementBegin

DROP INDEX IF EXISTS idx_operation_log_approval;
DROP INDEX IF EXISTS idx_operation_log_entity;
DROP INDEX IF EXISTS idx_operation_log_trigger;
ALTER TABLE operation_log DROP COLUMN IF EXISTS entity_id;
ALTER TABLE operation_log DROP COLUMN IF EXISTS entity_type;
ALTER TABLE operation_log DROP COLUMN IF EXISTS approval_id;
ALTER TABLE operation_log DROP COLUMN IF EXISTS agent_suggestion_id;
ALTER TABLE operation_log DROP COLUMN IF EXISTS trigger_type;

-- +migrate StatementEnd
