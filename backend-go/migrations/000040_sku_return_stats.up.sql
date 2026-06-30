CREATE TABLE IF NOT EXISTS sku_return_stats (
    sku_id BIGINT NOT NULL PRIMARY KEY,
    total_orders BIGINT NOT NULL DEFAULT 0,
    total_returns BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
