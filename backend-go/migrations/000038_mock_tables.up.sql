CREATE TABLE IF NOT EXISTS mock_order (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    order_no VARCHAR(50) NOT NULL,
    product_name VARCHAR(200) DEFAULT '',
    quantity INT NOT NULL DEFAULT 1,
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    order_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_seed_data BOOLEAN NOT NULL DEFAULT false,
    extra_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mock_order_platform ON mock_order(platform_id);
CREATE INDEX IF NOT EXISTS idx_mock_order_status ON mock_order(status);

CREATE TABLE IF NOT EXISTS mock_settlement (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    period VARCHAR(10) NOT NULL,
    total_revenue NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    net_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    order_count INT NOT NULL DEFAULT 0,
    is_seed_data BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mock_settlement_platform ON mock_settlement(platform_id);

CREATE TABLE IF NOT EXISTS mock_sync_status (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL,
    platform_name VARCHAR(100) NOT NULL,
    sync_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    records_synced INT NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    last_sync_at TIMESTAMPTZ,
    is_mock_data BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mock_sync_platform ON mock_sync_status(platform_id);
CREATE INDEX IF NOT EXISTS idx_mock_sync_type ON mock_sync_status(sync_type);
