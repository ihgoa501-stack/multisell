ALTER TABLE collect_leads
    ADD COLUMN IF NOT EXISTS collection_driver VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS evidence_id BIGINT,
    ADD COLUMN IF NOT EXISTS confidence_state VARCHAR(32) NOT NULL DEFAULT 'unverified',
    ADD COLUMN IF NOT EXISTS canonical_key VARCHAR(64);

-- Agent runs already persist parent_trace_id in the active GORM model; older
-- local databases need the column before real collection can start a trace.
ALTER TABLE ai_trace
    ADD COLUMN IF NOT EXISTS parent_trace_id VARCHAR(64);

CREATE TABLE IF NOT EXISTS collection_evidence (
    id BIGSERIAL PRIMARY KEY,
    source_url VARCHAR(2048) NOT NULL,
    driver VARCHAR(64) NOT NULL,
    raw_payload JSONB NOT NULL,
    parser_version VARCHAR(64) NOT NULL,
    evidence_sha256 VARCHAR(64) NOT NULL,
    correlation_id VARCHAR(128) NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_collection_evidence_sha256 ON collection_evidence(evidence_sha256);
CREATE INDEX IF NOT EXISTS idx_collection_evidence_correlation ON collection_evidence(correlation_id);
CREATE INDEX IF NOT EXISTS idx_collect_leads_evidence ON collect_leads(evidence_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_collect_leads_canonical_key
    ON collect_leads(canonical_key);
CREATE INDEX IF NOT EXISTS idx_ai_trace_parent_trace_id ON ai_trace(parent_trace_id);

ALTER TABLE collect_leads
    ADD CONSTRAINT fk_collect_leads_evidence
    FOREIGN KEY (evidence_id) REFERENCES collection_evidence(id);
