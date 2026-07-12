CREATE TABLE IF NOT EXISTS sourcing_1688_image_processing (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    source_url TEXT NOT NULL,
    source_sha256 CHAR(64) NOT NULL,
    processed_sha256 CHAR(64) NOT NULL,
    output_format VARCHAR(8) NOT NULL,
    output_width INTEGER NOT NULL,
    output_height INTEGER NOT NULL,
    quality INTEGER NOT NULL,
    processor_version VARCHAR(40) NOT NULL,
    operations JSONB NOT NULL,
    rights_evidence_uri TEXT NOT NULL,
    rights_truth_status VARCHAR(16) NOT NULL,
    rights_observed_at TIMESTAMPTZ NOT NULL,
    channel_rule_uri TEXT NOT NULL,
    evidence_fingerprint CHAR(64) NOT NULL,
    processed_bytes BYTEA NOT NULL,
    processed_by BIGINT NOT NULL REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (sourcing_product_id, snapshot_id, source_sha256, output_width, output_height, output_format, quality, processor_version, evidence_fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_sourcing_image_processing_source ON sourcing_1688_image_processing(sourcing_product_id, snapshot_id);
