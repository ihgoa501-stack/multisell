-- Forward repair: tables silently skipped due to duplicate migration version
-- numbers in the 000026 and 000028 ranges.
--
-- 000026_data_freshness was applied; 000026_tariff_rule was silently skipped.
-- 000028_landed_cost was applied; 000028_sku_return_stats was silently skipped.
--
-- Using IF NOT EXISTS for idempotent replay safety.

-- === 000026_tariff_rule (skipped) ===
CREATE TABLE IF NOT EXISTS tariff_rule (
    id BIGSERIAL PRIMARY KEY,
    country_code VARCHAR(10) NOT NULL,
    hs_code VARCHAR(20),
    hs_code_prefix VARCHAR(10),
    duty_rate_pct NUMERIC(10,4) DEFAULT 0,
    vat_rate_pct NUMERIC(10,4) DEFAULT 0,
    other_tax_rate_pct NUMERIC(10,4) DEFAULT 0,
    min_threshold_usd NUMERIC(14,2) DEFAULT 0,
    max_threshold_usd NUMERIC(14,2) DEFAULT 0,
    incoterm VARCHAR(10) NOT NULL DEFAULT 'DDU',
    priority INTEGER DEFAULT 0,
    effective_from TIMESTAMPTZ,
    effective_to TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'active',
    remark TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tariff_rule_country ON tariff_rule(country_code);
CREATE INDEX IF NOT EXISTS idx_tariff_rule_status ON tariff_rule(status);
CREATE INDEX IF NOT EXISTS idx_tariff_rule_country_status ON tariff_rule(country_code, status);

-- === 000028_sku_return_stats (skipped) ===
CREATE TABLE IF NOT EXISTS sku_return_stats (
    sku_id BIGINT NOT NULL PRIMARY KEY,
    total_orders BIGINT NOT NULL DEFAULT 0,
    total_returns BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
