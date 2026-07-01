-- +migrate Up
-- +migrate StatementBegin

CREATE TABLE IF NOT EXISTS profit_calculation (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    sku_id BIGINT NOT NULL,
    revenue NUMERIC(15,2) NOT NULL DEFAULT 0,
    platform_fee NUMERIC(15,2) NOT NULL DEFAULT 0,
    logistics_fee NUMERIC(15,2) NOT NULL DEFAULT 0,
    advertising_cost NUMERIC(15,2) NOT NULL DEFAULT 0,
    purchase_cost NUMERIC(15,2) NOT NULL DEFAULT 0,
    other_cost NUMERIC(15,2) NOT NULL DEFAULT 0,
    net_profit NUMERIC(15,2) NOT NULL DEFAULT 0,
    profit_margin NUMERIC(15,2) NOT NULL DEFAULT 0,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    period_start DATE NOT NULL DEFAULT '1970-01-01',
    period_end DATE NOT NULL DEFAULT '1970-01-01'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_profit_order_sku ON profit_calculation(order_id, sku_id);

-- +migrate StatementEnd
