DROP INDEX IF EXISTS idx_candidate_product_source_url;
DROP INDEX IF EXISTS idx_candidate_product_completeness_status;
ALTER TABLE candidate_product DROP COLUMN IF EXISTS collected_at;
ALTER TABLE candidate_product DROP COLUMN IF EXISTS completeness_status;
ALTER TABLE candidate_product DROP COLUMN IF EXISTS raw_payload;
ALTER TABLE candidate_product DROP COLUMN IF EXISTS source_platform;
ALTER TABLE candidate_product DROP COLUMN IF EXISTS source_url;
