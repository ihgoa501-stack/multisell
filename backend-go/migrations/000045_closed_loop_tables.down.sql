-- +migrate Down
-- +migrate StatementBegin

DROP INDEX IF EXISTS idx_approval_request_entity;
DROP INDEX IF EXISTS idx_listing_task_approval;
ALTER TABLE listing_task DROP COLUMN IF EXISTS approval_id;
ALTER TABLE approval_request DROP COLUMN IF EXISTS entity_id;
ALTER TABLE approval_request DROP COLUMN IF EXISTS entity_type;

-- +migrate StatementEnd
