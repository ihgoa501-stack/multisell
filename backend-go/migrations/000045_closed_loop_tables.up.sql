-- +migrate Up
-- +migrate StatementBegin

ALTER TABLE approval_request ADD COLUMN IF NOT EXISTS entity_type VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE approval_request ADD COLUMN IF NOT EXISTS entity_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS approval_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_approval_request_entity ON approval_request(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_listing_task_approval ON listing_task(approval_id);

-- +migrate StatementEnd
