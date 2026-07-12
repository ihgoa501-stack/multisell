DROP TRIGGER IF EXISTS trg_apply_sourcing_1688_draft_approval ON approval_request;
DROP FUNCTION IF EXISTS apply_sourcing_1688_draft_approval();
DROP TRIGGER IF EXISTS trg_sourcing_1688_lifecycle_sync ON sourcing_1688_product;
DROP FUNCTION IF EXISTS sync_sourcing_1688_lifecycle_from_legacy_status();
DROP INDEX IF EXISTS idx_sourcing_1688_lifecycle;
DROP INDEX IF EXISTS ux_sourcing_listing_draft_approval;
ALTER TABLE sourcing_listing_draft
    DROP COLUMN IF EXISTS approval_rejection_reason,
    DROP COLUMN IF EXISTS approval_status,
    DROP COLUMN IF EXISTS approval_id;
ALTER TABLE sourcing_1688_product DROP CONSTRAINT IF EXISTS ck_sourcing_1688_lifecycle_status;
ALTER TABLE sourcing_1688_product
    DROP COLUMN IF EXISTS lifecycle_updated_at,
    DROP COLUMN IF EXISTS lifecycle_reason,
    DROP COLUMN IF EXISTS lifecycle_actor_id,
    DROP COLUMN IF EXISTS lifecycle_status;
