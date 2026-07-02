-- Consolidation Module
-- Consolidation groups and items for bulk shipping rate negotiation.

CREATE TABLE IF NOT EXISTS consolidation_group (
    id BIGSERIAL PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    total_weight_kg DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_volume_m3 DECIMAL(12,2) NOT NULL DEFAULT 0,
    destination VARCHAR(255) NOT NULL,
    carrier_id BIGINT,
    negotiated_rate DECIMAL(12,2),
    discount_rate DECIMAL(5,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consolidation_item (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    sku_id BIGINT NOT NULL,
    weight_kg DECIMAL(12,2) NOT NULL,
    volume_m3 DECIMAL(12,2) NOT NULL DEFAULT 0,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consolidation_item_group_id ON consolidation_item(group_id);
