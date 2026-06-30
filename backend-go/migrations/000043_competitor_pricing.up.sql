-- 000043: Competitor price monitoring and dynamic pricing engine

CREATE TABLE IF NOT EXISTS competitor_prices (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL,
    platform VARCHAR(50) NOT NULL DEFAULT '',
    competitor_name VARCHAR(200) NOT NULL,
    price NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_url VARCHAR(500) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_competitor_prices_sku ON competitor_prices(sku_id);
CREATE INDEX IF NOT EXISTS idx_competitor_prices_sku_captured ON competitor_prices(sku_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS pricing_strategies (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT,
    strategy_type VARCHAR(30) NOT NULL,
    parameters JSONB DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pricing_strategies_sku ON pricing_strategies(sku_id);

CREATE TABLE IF NOT EXISTS pricing_recommendations (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL,
    current_price NUMERIC(10,2) NOT NULL,
    recommended_price NUMERIC(10,2) NOT NULL,
    strategy_used VARCHAR(30) DEFAULT '',
    reason VARCHAR(500) DEFAULT '',
    risk_level VARCHAR(10) NOT NULL DEFAULT 'medium',
    competitor_count INT NOT NULL DEFAULT 0,
    applied BOOLEAN NOT NULL DEFAULT false,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pricing_recommendations_sku ON pricing_recommendations(sku_id);
CREATE INDEX IF NOT EXISTS idx_pricing_recommendations_applied ON pricing_recommendations(applied);
