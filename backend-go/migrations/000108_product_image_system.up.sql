CREATE TABLE IF NOT EXISTS product_image_assets (
    id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, blob_id VARCHAR(64) NOT NULL,
    filename VARCHAR(255) NOT NULL, content_type VARCHAR(100) NOT NULL, size_bytes BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL, truth VARCHAR(20) NOT NULL DEFAULT 'unknown', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_product_image_assets_owner_created ON product_image_assets(owner_id, created_at DESC);
CREATE TABLE IF NOT EXISTS product_image_tasks (
    id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, asset_id BIGINT NOT NULL REFERENCES product_image_assets(id),
    image_service_job_id VARCHAR(100), output_blob_id VARCHAR(64), idempotency_key VARCHAR(100) NOT NULL,
    manifest_hash VARCHAR(64) NOT NULL, operation VARCHAR(64) NOT NULL, width INTEGER NOT NULL, height INTEGER NOT NULL,
    format VARCHAR(20) NOT NULL, status VARCHAR(32) NOT NULL, error_code VARCHAR(100), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_product_image_tasks_owner_created ON product_image_tasks(owner_id, created_at DESC);
CREATE TABLE IF NOT EXISTS product_image_reviews (
    id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, task_id BIGINT NOT NULL REFERENCES product_image_tasks(id),
    decision VARCHAR(32) NOT NULL, truth VARCHAR(20) NOT NULL DEFAULT 'unknown', notes TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS product_image_sets (
    id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, listing_id BIGINT NOT NULL REFERENCES product_listing(id), channel VARCHAR(64) NOT NULL,
    locale VARCHAR(32) NOT NULL, version INTEGER NOT NULL, based_on_set_id BIGINT REFERENCES product_image_sets(id),
    status VARCHAR(16) NOT NULL, manifest_sha VARCHAR(64), selected_by BIGINT, selected_at TIMESTAMPTZ, frozen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, listing_id, channel, locale, version)
);
CREATE TABLE IF NOT EXISTS product_image_set_items (
    id BIGSERIAL PRIMARY KEY, image_set_id BIGINT NOT NULL REFERENCES product_image_sets(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL, ordinal INTEGER NOT NULL, locale VARCHAR(32) NOT NULL, channel VARCHAR(64) NOT NULL,
    asset_sha VARCHAR(64) NOT NULL, task_id BIGINT NOT NULL REFERENCES product_image_tasks(id), output_blob_id VARCHAR(64) NOT NULL,
    task_manifest_hash VARCHAR(64) NOT NULL, operation VARCHAR(64) NOT NULL, processor VARCHAR(64) NOT NULL,
    image_service_job_id VARCHAR(100) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(image_set_id, ordinal)
);
