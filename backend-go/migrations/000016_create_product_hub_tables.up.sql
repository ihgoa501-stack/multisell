-- Migration: 000016_create_product_hub_tables
-- Up
CREATE TABLE IF NOT EXISTS product_master (
    id BIGSERIAL PRIMARY KEY,
    product_code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(256) NOT NULL,
    brand_id BIGINT,
    category_id BIGINT,
    business_model VARCHAR(32) NOT NULL DEFAULT 'catalog',
    lifecycle_status VARCHAR(32) NOT NULL DEFAULT 'idea',
    owner_id BIGINT NOT NULL,
    team_id BIGINT,
    description TEXT,
    target_market VARCHAR(128),
    target_price NUMERIC(12,2),
    target_margin NUMERIC(12,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_master_lifecycle ON product_master(lifecycle_status);
CREATE INDEX IF NOT EXISTS idx_product_master_owner ON product_master(owner_id);
