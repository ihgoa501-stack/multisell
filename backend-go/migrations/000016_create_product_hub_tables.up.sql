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

CREATE TABLE IF NOT EXISTS product_variant (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    sku_product_id BIGINT,
    sku_code VARCHAR(64),
    variant_label VARCHAR(128),
    barcode VARCHAR(64),
    weight DOUBLE PRECISION,
    dimensions VARCHAR(64),
    country_of_origin VARCHAR(8),
    hs_code VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_variant_master ON product_variant(product_master_id);
CREATE INDEX IF NOT EXISTS idx_product_variant_sku ON product_variant(sku_product_id);

CREATE TABLE IF NOT EXISTS product_concept (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    brief TEXT,
    target_customer TEXT,
    pain_point TEXT,
    market_research TEXT,
    competitor_info TEXT,
    design_source VARCHAR(32),
    attachment_urls JSONB DEFAULT '[]',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_concept_master ON product_concept(product_master_id);

CREATE TABLE IF NOT EXISTS supplier_offer (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    offer_type VARCHAR(32),
    unit_cost DOUBLE PRECISION,
    currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    moq INT,
    lead_time_days INT,
    incoterm VARCHAR(32),
    is_preferred BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_offer_master ON supplier_offer(product_master_id);
CREATE INDEX IF NOT EXISTS idx_supplier_offer_supplier ON supplier_offer(supplier_id);

CREATE TABLE IF NOT EXISTS sample_request (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    supplier_offer_id BIGINT,
    supplier_id BIGINT NOT NULL,
    quantity INT,
    requirements TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    iteration_no INT NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ,
    evaluation TEXT,
    quality_score DOUBLE PRECISION,
    decision VARCHAR(32),
    image_urls JSONB DEFAULT '[]',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sample_request_master ON sample_request(product_master_id);
