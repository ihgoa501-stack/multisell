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
