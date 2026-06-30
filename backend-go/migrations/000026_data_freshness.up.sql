CREATE TABLE IF NOT EXISTS data_freshness (
    id              BIGSERIAL PRIMARY KEY,
    product_id      BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    dimension       VARCHAR(50) NOT NULL,
    last_verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_check_at   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    freshness_days  INTEGER NOT NULL DEFAULT 30,
    status          VARCHAR(20) NOT NULL DEFAULT 'fresh',
    drift_detected  BOOLEAN NOT NULL DEFAULT FALSE,
    last_value      TEXT,
    current_value   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_data_freshness_product_id ON data_freshness(product_id);
CREATE UNIQUE INDEX idx_data_freshness_product_dimension ON data_freshness(product_id, dimension);
CREATE INDEX idx_data_freshness_next_check_at ON data_freshness(next_check_at);
CREATE INDEX idx_data_freshness_status ON data_freshness(status);
