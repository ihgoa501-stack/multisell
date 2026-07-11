CREATE TABLE IF NOT EXISTS demand_research_batch (
    id BIGSERIAL PRIMARY KEY, batch_key VARCHAR(100) NOT NULL,
    owner_id BIGINT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(owner_id,batch_key)
);
CREATE INDEX IF NOT EXISTS idx_demand_research_batch_owner ON demand_research_batch(owner_id);
CREATE TABLE IF NOT EXISTS demand_research_snapshot (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES demand_research_batch(id) ON DELETE CASCADE,
    owner_id BIGINT NOT NULL,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE CASCADE,
    run_id VARCHAR(80) NOT NULL, run_type VARCHAR(24) NOT NULL CHECK (run_type IN ('scout_result','falsifier_result','data_reality_result')),
    source_uri TEXT NOT NULL, collected_at TIMESTAMPTZ NOT NULL,
    raw_payload TEXT NOT NULL, raw_sha256 VARCHAR(64) NOT NULL CHECK (length(raw_sha256)=64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(owner_id,run_id, run_type)
);
CREATE INDEX IF NOT EXISTS idx_demand_research_snapshot_case ON demand_research_snapshot(demand_case_id);
ALTER TABLE demand_evidence ADD COLUMN IF NOT EXISTS snapshot_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_demand_evidence_snapshot ON demand_evidence(snapshot_id);
