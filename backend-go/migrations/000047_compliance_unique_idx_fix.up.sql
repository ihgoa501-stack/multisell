-- Replace compliance unique index to not include scanned_at::date.
-- The per-day uniqueness was causing duplicate results for the same
-- product/platform/check_type combination accumulated over time.
-- Previously the unique constraint was part of 000029_compliance.up.sql
-- but was moved to a standalone migration to avoid in-place edits.

DROP INDEX IF EXISTS idx_compliance_result_product_platform_type;
CREATE UNIQUE INDEX idx_compliance_result_product_platform_type
    ON compliance_check_result(product_id, COALESCE(platform_id, 0), check_type);
