-- Compliance Risk Engine Phase 1: check results table

CREATE TABLE IF NOT EXISTS compliance_check_result (
    id              BIGSERIAL PRIMARY KEY,
    product_id      BIGINT NOT NULL,
    platform_id     BIGINT,
    check_type      VARCHAR(50) NOT NULL DEFAULT 'compliance',
    status          VARCHAR(20) NOT NULL DEFAULT 'pass',
    risk_level      VARCHAR(10),              -- low / medium / high / unknown
    rule_version    INT NOT NULL DEFAULT 1,
    evidence        JSONB,                     -- array of {rule, field, value, source, timestamp}
    scanned_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_scan_at    TIMESTAMPTZ,
    is_suppressed   BOOLEAN NOT NULL DEFAULT false,
    suppressed_reason VARCHAR(500),
    suppressed_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Idempotency: one result per (product_id, platform_id, check_type) per scan window
CREATE UNIQUE INDEX IF NOT EXISTS idx_compliance_result_product_platform_type
    ON compliance_check_result(product_id, COALESCE(platform_id, 0), check_type, CAST(scanned_at AS date));

CREATE INDEX IF NOT EXISTS idx_compliance_result_product
    ON compliance_check_result(product_id);

CREATE INDEX IF NOT EXISTS idx_compliance_result_status
    ON compliance_check_result(status);

CREATE INDEX IF NOT EXISTS idx_compliance_result_scanned_at
    ON compliance_check_result(scanned_at DESC);
