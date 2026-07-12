ALTER TABLE sourcing_1688_product
    ADD COLUMN IF NOT EXISTS source_product_fingerprint CHAR(64),
    ADD COLUMN IF NOT EXISTS supplier_business_id VARCHAR(120);

ALTER TABLE sourcing_1688_snapshot
    ADD COLUMN IF NOT EXISTS observed_supplier_business_id VARCHAR(120),
    ADD COLUMN IF NOT EXISTS product_fingerprint CHAR(64);

CREATE INDEX IF NOT EXISTS idx_sourcing_1688_fingerprint
    ON sourcing_1688_product(source_product_fingerprint);
CREATE INDEX IF NOT EXISTS idx_sourcing_1688_supplier_business
    ON sourcing_1688_product(supplier_business_id);

CREATE TABLE IF NOT EXISTS sourcing_1688_change_event (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    previous_snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    current_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    change_type VARCHAR(40) NOT NULL,
    before_value JSONB,
    after_value JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (current_snapshot_id, change_type)
);

CREATE TABLE IF NOT EXISTS sourcing_1688_duplicate_candidate (
    id BIGSERIAL PRIMARY KEY,
    source_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    matched_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    match_type VARCHAR(32) NOT NULL,
    fingerprint CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review',
    reviewed_by BIGINT REFERENCES "user"(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_product_id, matched_product_id, match_type),
    CONSTRAINT ck_sourcing_duplicate_distinct CHECK (source_product_id <> matched_product_id),
    CONSTRAINT ck_sourcing_duplicate_status CHECK (status IN ('pending_review','same_product','different_product'))
);
