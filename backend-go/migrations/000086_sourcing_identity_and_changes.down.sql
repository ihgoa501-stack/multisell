DROP TABLE IF EXISTS sourcing_1688_duplicate_candidate;
DROP TABLE IF EXISTS sourcing_1688_change_event;
DROP INDEX IF EXISTS idx_sourcing_1688_supplier_business;
DROP INDEX IF EXISTS idx_sourcing_1688_fingerprint;
ALTER TABLE sourcing_1688_product
    DROP COLUMN IF EXISTS supplier_business_id,
    DROP COLUMN IF EXISTS source_product_fingerprint;
ALTER TABLE sourcing_1688_snapshot
    DROP COLUMN IF EXISTS product_fingerprint,
    DROP COLUMN IF EXISTS observed_supplier_business_id;
