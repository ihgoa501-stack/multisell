CREATE TABLE IF NOT EXISTS product_image_manual_imports (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL REFERENCES product_image_assets(id),
    asset_sha VARCHAR(64) NOT NULL CHECK (asset_sha ~ '^[0-9a-f]{64}$'),
    parent_asset_id BIGINT NOT NULL REFERENCES product_image_assets(id),
    parent_asset_sha VARCHAR(64) NOT NULL CHECK (parent_asset_sha ~ '^[0-9a-f]{64}$'),
    import_kind VARCHAR(32) NOT NULL CHECK (import_kind IN ('manual_import','channel_native_import')),
    tool VARCHAR(100) NOT NULL,
    operation VARCHAR(255) NOT NULL,
    fee_amount VARCHAR(32) NOT NULL CHECK (fee_amount ~ '^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$'),
    fee_currency VARCHAR(3) NOT NULL CHECK (fee_currency IN ('USD','EUR','CNY','GBP','JPY')),
    model VARCHAR(100) NOT NULL,
    model_version VARCHAR(100) NOT NULL,
    original_channel VARCHAR(64),
    channel_restriction VARCHAR(64) NOT NULL,
    source_observed_at TIMESTAMPTZ NOT NULL,
    truth VARCHAR(20) NOT NULL DEFAULT 'unknown' CHECK (truth = 'unknown'),
    idempotency_key VARCHAR(100) NOT NULL,
    request_hash VARCHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key),
    CHECK (import_kind <> 'channel_native_import' OR (original_channel IS NOT NULL AND original_channel = channel_restriction))
);
CREATE INDEX IF NOT EXISTS idx_product_image_manual_import_owner_created ON product_image_manual_imports(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_image_manual_import_parent ON product_image_manual_imports(owner_id, parent_asset_id);
