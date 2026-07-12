ALTER TABLE sourcing_1688_task_link DROP CONSTRAINT IF EXISTS ck_sourcing_task_link_authority;
DROP INDEX IF EXISTS ux_sourcing_task_link_product_opportunity;
DROP INDEX IF EXISTS idx_sourcing_task_link_opportunity;
ALTER TABLE sourcing_1688_task_link
    DROP COLUMN IF EXISTS authority_kind,
    DROP COLUMN IF EXISTS opportunity_decision_id,
    DROP COLUMN IF EXISTS product_opportunity_id;
