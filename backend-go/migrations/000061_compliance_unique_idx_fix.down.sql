-- Restore the old unique index with scanned_at::date included.
DROP INDEX IF EXISTS idx_compliance_result_product_platform_type;
CREATE UNIQUE INDEX idx_compliance_result_product_platform_type
    ON compliance_check_result(product_id, COALESCE(platform_id, 0), check_type, scanned_at::date);
