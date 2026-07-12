-- Restore the index contract defined by 000029. A historical draft attempted
-- to include scanned_at::date, but timestamptz-to-date depends on timezone and
-- is not immutable, so PostgreSQL cannot use it in an index expression.
DROP INDEX IF EXISTS idx_compliance_result_product_platform_type;
CREATE UNIQUE INDEX idx_compliance_result_product_platform_type
    ON compliance_check_result(product_id, COALESCE(platform_id, 0), check_type);
