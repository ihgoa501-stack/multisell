DROP INDEX IF EXISTS ux_sourcing_publish_task_one_active;
DROP INDEX IF EXISTS ux_sourcing_publish_legacy_one_active;
ALTER TABLE sourcing_publish_attempt DROP COLUMN IF EXISTS task_link_id;
ALTER TABLE sourcing_listing_draft DROP COLUMN IF EXISTS editable_version;
CREATE UNIQUE INDEX ux_sourcing_publish_one_active
    ON sourcing_publish_attempt(sourcing_product_id)
    WHERE status IN ('pending_approval','approved','executing','submitted','reconcile_required');

ALTER TABLE sourcing_1688_task_link DROP CONSTRAINT IF EXISTS ck_sourcing_task_link_workflow_status;
ALTER TABLE sourcing_1688_task_link ADD CONSTRAINT ck_sourcing_task_link_workflow_status CHECK (workflow_status IN (
    'needs_review','ready_for_draft','converting','converted_to_draft',
    'blocked','conversion_failed','reconcile_required','archived'
));

-- A down migration is allowed only when no task has advanced into a new state;
-- otherwise restoring the legacy product-global approval function would erase
-- meaning. The constraint above fails safely before this point in that case.
