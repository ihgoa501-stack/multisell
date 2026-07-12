DO $$
BEGIN
    IF EXISTS (
        SELECT sourcing_product_id
        FROM sourcing_listing_draft
        GROUP BY sourcing_product_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot roll back task draft workflows while a source has multiple drafts';
    END IF;
END $$;

ALTER TABLE sourcing_1688_task_link DROP CONSTRAINT IF EXISTS fk_sourcing_task_link_draft;
DROP INDEX IF EXISTS ux_sourcing_task_link_draft;
DROP INDEX IF EXISTS ux_sourcing_draft_task_link;
DROP INDEX IF EXISTS ux_sourcing_draft_legacy_source;
ALTER TABLE sourcing_listing_draft DROP COLUMN IF EXISTS task_link_id;
ALTER TABLE sourcing_listing_draft
    ADD CONSTRAINT sourcing_listing_draft_sourcing_product_id_key UNIQUE (sourcing_product_id);

ALTER TABLE sourcing_1688_task_link
    DROP CONSTRAINT IF EXISTS ck_sourcing_task_link_workflow_status,
    DROP COLUMN IF EXISTS workflow_updated_at,
    DROP COLUMN IF EXISTS draft_id,
    DROP COLUMN IF EXISTS workflow_status;
