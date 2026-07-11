CREATE TABLE IF NOT EXISTS demand_data_access (
 id BIGSERIAL PRIMARY KEY,
 demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE CASCADE,
 field_name VARCHAR(120) NOT NULL,
 status VARCHAR(32) NOT NULL CHECK(status IN ('available','requires_owner_access','requires_listing','requires_transaction','unavailable','unknown')),
 required_scope VARCHAR(160) NOT NULL DEFAULT '',
 decision_purpose TEXT NOT NULL,
 refusal_impact TEXT NOT NULL,
 source_uri TEXT NOT NULL,
 run_id VARCHAR(80) NOT NULL,
 snapshot_id BIGINT NOT NULL REFERENCES demand_research_snapshot(id) ON DELETE RESTRICT,
 access_mode VARCHAR(16) NOT NULL DEFAULT 'read_only' CHECK(access_mode='read_only'),
 preflight_required BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(demand_case_id,field_name,run_id)
);
ALTER TABLE demand_research_snapshot ADD COLUMN IF NOT EXISTS collector VARCHAR(120) NOT NULL DEFAULT 'legacy_unknown';
CREATE INDEX IF NOT EXISTS idx_demand_data_access_case ON demand_data_access(demand_case_id);
