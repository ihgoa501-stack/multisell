ALTER TABLE supplier
    ADD COLUMN IF NOT EXISTS owner_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source_system VARCHAR(32) NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS external_business_id VARCHAR(300) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    ADD COLUMN IF NOT EXISTS identity_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS truth_status VARCHAR(16) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS observed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verified_by BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_supplier_owner_source_business
    ON supplier(owner_id, source_system, external_business_id)
    WHERE owner_id > 0 AND external_business_id <> '';

ALTER TABLE supplier
    ADD CONSTRAINT ck_supplier_truth_status
    CHECK (truth_status IN ('actual', 'quoted', 'estimated', 'unknown')) NOT VALID;

ALTER TABLE product_supplier
    ADD COLUMN IF NOT EXISTS owner_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sourcing_product_id BIGINT REFERENCES sourcing_1688_product(id),
    ADD COLUMN IF NOT EXISTS source_snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    ADD COLUMN IF NOT EXISTS product_opportunity_id BIGINT REFERENCES product_opportunity(id),
    ADD COLUMN IF NOT EXISTS truth_status VARCHAR(16) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS source_uri TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS observed_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS ux_product_supplier_sourcing_authority
    ON product_supplier(owner_id, sourcing_product_id, product_id, supplier_id)
    WHERE owner_id > 0 AND sourcing_product_id IS NOT NULL;

ALTER TABLE product_supplier
    ADD CONSTRAINT ck_product_supplier_truth_status
    CHECK (truth_status IN ('actual', 'quoted', 'estimated', 'unknown')) NOT VALID;

CREATE OR REPLACE FUNCTION reject_authoritative_supplier_identity_mutation()
RETURNS trigger AS $$
BEGIN
    IF OLD.source_system <> 'legacy' THEN
        RAISE EXCEPTION 'authoritative supplier identity is immutable';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_authoritative_supplier_identity_immutable
BEFORE UPDATE OR DELETE ON supplier
FOR EACH ROW EXECUTE FUNCTION reject_authoritative_supplier_identity_mutation();
