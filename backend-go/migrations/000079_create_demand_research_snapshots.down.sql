DROP INDEX IF EXISTS idx_demand_evidence_snapshot;
ALTER TABLE demand_evidence DROP COLUMN IF EXISTS snapshot_id;
DROP TABLE IF EXISTS demand_research_snapshot;
DROP TABLE IF EXISTS demand_research_batch;
