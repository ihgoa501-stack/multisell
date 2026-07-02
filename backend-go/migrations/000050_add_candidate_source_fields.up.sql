ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS source_url varchar(2048) DEFAULT '';
ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS source_platform varchar(64) DEFAULT '';
ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS raw_payload jsonb;
ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS completeness_status varchar(32) DEFAULT 'incomplete';
ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS collected_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_candidate_product_source_url ON candidate_product(source_url);
CREATE INDEX IF NOT EXISTS idx_candidate_product_completeness_status ON candidate_product(completeness_status);
