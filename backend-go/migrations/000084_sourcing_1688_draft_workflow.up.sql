ALTER TABLE sourcing_1688_product
    ADD COLUMN IF NOT EXISTS source_offer_id VARCHAR(80),
    ADD COLUMN IF NOT EXISTS demand_case_id BIGINT REFERENCES demand_case(id),
    ADD COLUMN IF NOT EXISTS experiment_id VARCHAR(40) REFERENCES experiment_case(experiment_id),
    ADD COLUMN IF NOT EXISTS snapshot_id BIGINT,
    ADD COLUMN IF NOT EXISTS reviewed_by BIGINT REFERENCES "user"(id),
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS review_notes TEXT;

CREATE TABLE IF NOT EXISTS sourcing_1688_snapshot (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    source_url TEXT NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL,
    collected_by BIGINT NOT NULL REFERENCES "user"(id),
    driver VARCHAR(40) NOT NULL,
    parser_version VARCHAR(40) NOT NULL,
    raw_payload JSONB NOT NULL,
    raw_sha256 CHAR(64) NOT NULL,
    observed_title TEXT,
    observed_price NUMERIC(12,2),
    observed_moq INTEGER NOT NULL DEFAULT 1,
    observed_supplier TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (sourcing_product_id, raw_sha256)
);

ALTER TABLE sourcing_1688_product
    ADD CONSTRAINT fk_sourcing_1688_snapshot
    FOREIGN KEY (snapshot_id) REFERENCES sourcing_1688_snapshot(id);

CREATE TABLE IF NOT EXISTS product_media_asset (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    source_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    source_url TEXT NOT NULL,
    processed_url TEXT NOT NULL,
    media_role VARCHAR(20) NOT NULL,
    rights_status VARCHAR(20) NOT NULL,
    rights_evidence_uri TEXT NOT NULL,
    operations JSONB NOT NULL DEFAULT '[]'::jsonb,
    content_sha256 CHAR(64),
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    has_watermark BOOLEAN NOT NULL DEFAULT false,
    has_chinese_text BOOLEAN NOT NULL DEFAULT false,
    has_brand_mark BOOLEAN NOT NULL DEFAULT false,
    channel_rule_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_product_media_rights CHECK (rights_status IN ('verified', 'blocked'))
);

CREATE TABLE IF NOT EXISTS product_cost_input (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    experiment_id VARCHAR(40) NOT NULL REFERENCES experiment_case(experiment_id),
    cost_type VARCHAR(40) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(8) NOT NULL,
    truth_status VARCHAR(16) NOT NULL,
    source_uri TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id, cost_type),
    CONSTRAINT ck_product_cost_truth CHECK (truth_status IN ('actual', 'quoted', 'estimated'))
);

CREATE TABLE IF NOT EXISTS sourcing_listing_draft (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL UNIQUE REFERENCES sourcing_1688_product(id),
    snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    product_id BIGINT NOT NULL REFERENCES product(id),
    listing_id BIGINT NOT NULL UNIQUE REFERENCES product_listing(id),
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id),
    experiment_id VARCHAR(40) NOT NULL REFERENCES experiment_case(experiment_id),
    created_by BIGINT NOT NULL REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sourcing_1688_demand_case ON sourcing_1688_product(demand_case_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_sourcing_1688_offer_id ON sourcing_1688_product(source_offer_id) WHERE source_offer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sourcing_1688_experiment ON sourcing_1688_product(experiment_id);
CREATE INDEX IF NOT EXISTS idx_product_media_asset_product ON product_media_asset(product_id);
CREATE INDEX IF NOT EXISTS idx_product_cost_input_product ON product_cost_input(product_id);

CREATE OR REPLACE FUNCTION reject_sourcing_1688_snapshot_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing_1688_snapshot is immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sourcing_1688_snapshot_immutable ON sourcing_1688_snapshot;
CREATE TRIGGER trg_sourcing_1688_snapshot_immutable
BEFORE UPDATE OR DELETE ON sourcing_1688_snapshot
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_1688_snapshot_mutation();
