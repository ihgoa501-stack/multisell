CREATE TABLE IF NOT EXISTS landed_cost (
    id              BIGSERIAL       PRIMARY KEY,
    product_id      BIGINT          NOT NULL,
    platform_id     BIGINT          NOT NULL,

    -- Cost components
    unit_cost_cny       DECIMAL(12,2) NOT NULL DEFAULT 0,
    freight_cny         DECIMAL(12,2) NOT NULL DEFAULT 0,
    insurance_cny       DECIMAL(12,2) NOT NULL DEFAULT 0,
    duty_rate           DECIMAL(5,2)  NOT NULL DEFAULT 0,
    duty_cny            DECIMAL(12,2) NOT NULL DEFAULT 0,
    vat_rate            DECIMAL(5,2)  NOT NULL DEFAULT 0,
    vat_cny             DECIMAL(12,2) NOT NULL DEFAULT 0,
    platform_fee_pct    DECIMAL(5,2)  NOT NULL DEFAULT 0,
    platform_fee_cny    DECIMAL(12,2) NOT NULL DEFAULT 0,
    clearing_fee_cny    DECIMAL(12,2) NOT NULL DEFAULT 0,

    -- Results
    total_cost_cny      DECIMAL(12,2) NOT NULL DEFAULT 0,
    exchange_rate       DECIMAL(10,4) NOT NULL DEFAULT 0,
    total_cost_local    DECIMAL(12,2) NOT NULL DEFAULT 0,
    target_price        DECIMAL(12,2) NOT NULL DEFAULT 0,

    calculated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_landed_cost_product_id ON landed_cost (product_id);
CREATE INDEX idx_landed_cost_product_platform ON landed_cost (product_id, platform_id);
