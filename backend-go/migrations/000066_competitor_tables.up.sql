CREATE TABLE IF NOT EXISTS competitor_product (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    product_url TEXT,
    sku_code VARCHAR(200),
    category VARCHAR(200),
    brand VARCHAR(200),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS price_snapshot (
    id BIGSERIAL PRIMARY KEY,
    competitor_id BIGINT NOT NULL REFERENCES competitor_product(id) ON DELETE CASCADE,
    price NUMERIC(10,2) NOT NULL,
    original_price NUMERIC(10,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    sales_last_30d INT NOT NULL DEFAULT 0,
    rating NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count INT NOT NULL DEFAULT 0,
    is_in_stock BOOLEAN NOT NULL DEFAULT true,
    snapshot_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_snapshot_competitor_id ON price_snapshot(competitor_id);
CREATE INDEX IF NOT EXISTS idx_price_snapshot_date ON price_snapshot(snapshot_date);
CREATE INDEX IF NOT EXISTS idx_competitor_product_platform ON competitor_product(platform);
