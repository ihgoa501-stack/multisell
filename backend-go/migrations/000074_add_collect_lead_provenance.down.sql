ALTER TABLE collect_leads DROP CONSTRAINT IF EXISTS fk_collect_leads_evidence;
DROP INDEX IF EXISTS idx_collect_leads_evidence;
DROP INDEX IF EXISTS idx_collect_leads_canonical_key;
DROP INDEX IF EXISTS idx_ai_trace_parent_trace_id;
DROP TABLE IF EXISTS collection_evidence;

ALTER TABLE collect_leads
    DROP COLUMN IF EXISTS canonical_key,
    DROP COLUMN IF EXISTS confidence_state,
    DROP COLUMN IF EXISTS evidence_id,
    DROP COLUMN IF EXISTS collection_driver;

ALTER TABLE ai_trace DROP COLUMN IF EXISTS parent_trace_id;
