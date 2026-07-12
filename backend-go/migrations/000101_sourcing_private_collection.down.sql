DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sourcing_1688_task_link)
       OR EXISTS (SELECT 1 FROM sourcing_1688_snapshot WHERE capture_mode = 'extension_click')
       OR EXISTS (SELECT 1 FROM sourcing_1688_product WHERE lifecycle_status = 'unverified_lead') THEN
        RAISE EXCEPTION '000101 rollback is unsafe after private collection data exists; export or migrate those records first';
    END IF;
END $$;

ALTER TABLE sourcing_1688_product
    DROP CONSTRAINT IF EXISTS ck_sourcing_1688_lifecycle_status;
ALTER TABLE sourcing_1688_product
    ADD CONSTRAINT ck_sourcing_1688_lifecycle_status CHECK (lifecycle_status IN (
        'capture_failed', 'pending_review', 'rejected', 'ready_for_product',
        'editing', 'pending_approval', 'approved_draft'
    ));

ALTER TABLE sourcing_1688_snapshot
    DROP CONSTRAINT IF EXISTS ck_sourcing_snapshot_capture_mode;
ALTER TABLE sourcing_1688_snapshot
    ADD CONSTRAINT ck_sourcing_snapshot_capture_mode
    CHECK (capture_mode IN ('controlled_fetch', 'manual_import', 'legacy_unknown'));
ALTER TABLE sourcing_1688_snapshot DROP COLUMN IF EXISTS extension_version;

DROP TABLE IF EXISTS sourcing_1688_task_link;
DROP INDEX IF EXISTS idx_sourcing_1688_owner_status;
DROP INDEX IF EXISTS ux_sourcing_1688_owner_offer;
CREATE UNIQUE INDEX IF NOT EXISTS ux_sourcing_1688_offer_id
    ON sourcing_1688_product(source_offer_id) WHERE source_offer_id IS NOT NULL;

ALTER TABLE sourcing_1688_product
    ADD CONSTRAINT sourcing_1688_product_source_url_key UNIQUE (source_url);
ALTER TABLE sourcing_1688_product DROP COLUMN IF EXISTS owner_id;
