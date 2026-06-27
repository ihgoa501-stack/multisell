-- Product version history table for snapshot/rollback support.
CREATE TABLE IF NOT EXISTS product_version (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    version_data JSONB,
    snapshot JSONB,
    agent_id VARCHAR(255),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_version_product_id ON product_version(product_id);
CREATE INDEX IF NOT EXISTS idx_product_version_created_at ON product_version(created_at);
