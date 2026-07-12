DROP TRIGGER IF EXISTS trg_sourcing_1688_snapshot_immutable ON sourcing_1688_snapshot;
DROP FUNCTION IF EXISTS reject_sourcing_1688_snapshot_mutation();
DROP TABLE IF EXISTS sourcing_listing_draft;
DROP TABLE IF EXISTS product_cost_input;
DROP TABLE IF EXISTS product_media_asset;
ALTER TABLE sourcing_1688_product DROP CONSTRAINT IF EXISTS fk_sourcing_1688_snapshot;
DROP TABLE IF EXISTS sourcing_1688_snapshot;
ALTER TABLE sourcing_1688_product
    DROP COLUMN IF EXISTS review_notes,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS snapshot_id,
    DROP COLUMN IF EXISTS experiment_id,
    DROP COLUMN IF EXISTS demand_case_id,
    DROP COLUMN IF EXISTS source_offer_id;
