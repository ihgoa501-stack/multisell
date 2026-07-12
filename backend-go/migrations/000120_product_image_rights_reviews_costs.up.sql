CREATE TABLE IF NOT EXISTS product_image_rights_grants (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    asset_id BIGINT REFERENCES product_image_assets(id),
    asset_sha VARCHAR(64) NOT NULL CHECK (asset_sha ~ '^[0-9a-f]{64}$'),
    can_copy BOOLEAN NOT NULL DEFAULT FALSE,
    can_modify BOOLEAN NOT NULL DEFAULT FALSE,
    can_third_party_ai BOOLEAN NOT NULL DEFAULT FALSE,
    can_cross_border BOOLEAN NOT NULL DEFAULT FALSE,
    can_commercial_publish BOOLEAN NOT NULL DEFAULT FALSE,
    can_platform_sublicense BOOLEAN NOT NULL DEFAULT FALSE,
    trademark_cleared BOOLEAN NOT NULL DEFAULT FALSE,
    likeness_cleared BOOLEAN NOT NULL DEFAULT FALSE,
    purpose VARCHAR(64) NOT NULL,
    jurisdiction VARCHAR(64) NOT NULL,
    channel VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    region VARCHAR(64) NOT NULL,
    grantor VARCHAR(255) NOT NULL,
    rights_chain TEXT NOT NULL,
    evidence_sha VARCHAR(64) NOT NULL CHECK (evidence_sha ~ '^[0-9a-f]{64}$'),
    owner_verified BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT,
    revocation_idempotency_key VARCHAR(100),
    revocation_request_hash VARCHAR(64),
    idempotency_key VARCHAR(100) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key),
    CHECK (valid_until IS NULL OR valid_until > valid_from),
    CHECK ((revoked_at IS NULL AND revocation_reason IS NULL) OR (revoked_at IS NOT NULL AND length(trim(revocation_reason)) > 0))
);
CREATE INDEX IF NOT EXISTS idx_product_image_rights_scope ON product_image_rights_grants(owner_id, asset_sha, purpose, channel);

ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS asset_sha VARCHAR(64);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS purpose VARCHAR(64);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS channel VARCHAR(64);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS product_authenticity VARCHAR(16);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS rights_status VARCHAR(16);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS channel_rules VARCHAR(16);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS claims_scene VARCHAR(16);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS technical_visual VARCHAR(16);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS evidence_sha VARCHAR(64);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS evidence_truth VARCHAR(20);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(100);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS request_hash VARCHAR(64);
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS expected_task_version BIGINT;
ALTER TABLE product_image_reviews ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_product_image_review_owner_idem ON product_image_reviews(owner_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_product_image_reviews_gate ON product_image_reviews(owner_id, task_id, asset_sha, purpose, channel);

CREATE TABLE IF NOT EXISTS product_image_cost_entries (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL REFERENCES product_image_tasks(id),
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('estimated','actual')),
    category VARCHAR(32) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    amount VARCHAR(32) NOT NULL CHECK (amount ~ '^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$'),
    currency VARCHAR(3) NOT NULL CHECK (currency IN ('USD','EUR','CNY','GBP','JPY')),
    exchange_rate VARCHAR(32) NOT NULL CHECK (exchange_rate ~ '^([1-9][0-9]{0,9})(\.[0-9]{1,4})?$|^0\.[0-9]{0,3}[1-9]$'),
    exchange_rate_source VARCHAR(255) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    billing_status VARCHAR(24) NOT NULL CHECK (billing_status IN ('estimated','pending','invoiced','paid','reconciled','unknown')),
    evidence_sha VARCHAR(64),
    idempotency_key VARCHAR(100) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    expected_task_version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_product_image_cost_task ON product_image_cost_entries(owner_id, task_id, created_at DESC);
