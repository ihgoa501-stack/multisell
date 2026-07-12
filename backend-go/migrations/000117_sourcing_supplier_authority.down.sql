DROP TRIGGER IF EXISTS trg_authoritative_supplier_identity_immutable ON supplier;
DROP FUNCTION IF EXISTS reject_authoritative_supplier_identity_mutation();

DROP INDEX IF EXISTS ux_product_supplier_sourcing_authority;
ALTER TABLE product_supplier
    DROP CONSTRAINT IF EXISTS ck_product_supplier_truth_status,
    DROP COLUMN IF EXISTS observed_at,
    DROP COLUMN IF EXISTS source_uri,
    DROP COLUMN IF EXISTS truth_status,
    DROP COLUMN IF EXISTS product_opportunity_id,
    DROP COLUMN IF EXISTS source_snapshot_id,
    DROP COLUMN IF EXISTS sourcing_product_id,
    DROP COLUMN IF EXISTS owner_id;

DROP INDEX IF EXISTS ux_supplier_owner_source_business;
ALTER TABLE supplier
    DROP CONSTRAINT IF EXISTS ck_supplier_truth_status,
    DROP COLUMN IF EXISTS verified_by,
    DROP COLUMN IF EXISTS observed_at,
    DROP COLUMN IF EXISTS truth_status,
    DROP COLUMN IF EXISTS identity_sha256,
    DROP COLUMN IF EXISTS source_snapshot_id,
    DROP COLUMN IF EXISTS external_business_id,
    DROP COLUMN IF EXISTS source_system,
    DROP COLUMN IF EXISTS owner_id;
