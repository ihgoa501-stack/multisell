-- Migration 000006: add pre_listing_decision table

CREATE TABLE IF NOT EXISTS pre_listing_decision (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL,
    platform_id BIGINT,
    country_code VARCHAR(10),
    decision_point VARCHAR(50) DEFAULT 'pre_listing',
    estimated_revenue NUMERIC(12,2) DEFAULT 0,
    estimated_product_cost NUMERIC(12,2) DEFAULT 0,
    estimated_shipping_cost NUMERIC(12,2) DEFAULT 0,
    estimated_platform_fee NUMERIC(12,2) DEFAULT 0,
    estimated_payment_fee NUMERIC(12,2) DEFAULT 0,
    estimated_other_fee NUMERIC(12,2) DEFAULT 0,
    estimated_profit NUMERIC(12,2) DEFAULT 0,
    profit_margin NUMERIC(10,4) DEFAULT 0,
    confidence_score NUMERIC(5,4) DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'medium',
    recommendation TEXT,
    reasoning TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    decided_by VARCHAR(100),
    decided_at TIMESTAMPTZ,
    trace_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pre_listing_decision_sku_id ON pre_listing_decision(sku_id);
CREATE INDEX idx_pre_listing_decision_platform_id ON pre_listing_decision(platform_id);
CREATE INDEX idx_pre_listing_decision_status ON pre_listing_decision(status);
