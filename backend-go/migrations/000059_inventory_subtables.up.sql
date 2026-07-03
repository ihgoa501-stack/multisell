-- ============================================================
-- Migration 000059: Inventory subtables
-- Creates inventory_alert, inventory_alert_rule,
-- inventory_transfer, and bin_location tables.
-- ============================================================

-- inventory_alert: triggered stock alerts (low stock, overstock, aging)
CREATE TABLE IF NOT EXISTS inventory_alert (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL,
    alert_type VARCHAR(50),
    message TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_alert_sku_id ON inventory_alert(sku_id);

-- inventory_alert_rule: stock alert threshold rules per SKU
CREATE TABLE IF NOT EXISTS inventory_alert_rule (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL,
    min_level INTEGER NOT NULL DEFAULT 0,
    max_level INTEGER NOT NULL DEFAULT 0,
    lead_time_days INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_alert_rule_sku_id ON inventory_alert_rule(sku_id);

-- inventory_transfer: warehouse-to-warehouse inventory transfers
CREATE TABLE IF NOT EXISTS inventory_transfer (
    id BIGSERIAL PRIMARY KEY,
    from_warehouse VARCHAR(255) NOT NULL,
    to_warehouse VARCHAR(255) NOT NULL,
    sku_id BIGINT NOT NULL,
    quantity INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    carrier VARCHAR(255),
    tracking_no VARCHAR(255),
    estimated_arrival TIMESTAMPTZ,
    note TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- bin_location: warehouse bin/location management for physical inventory
CREATE TABLE IF NOT EXISTS bin_location (
    id BIGSERIAL PRIMARY KEY,
    warehouse VARCHAR(255) NOT NULL,
    location_code VARCHAR(255) NOT NULL,
    sku_id BIGINT,
    capacity INTEGER NOT NULL DEFAULT 0,
    used INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bin_location_location_code ON bin_location(location_code);
