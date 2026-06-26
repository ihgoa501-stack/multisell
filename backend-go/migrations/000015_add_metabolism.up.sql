-- Metabolism M1 Phase 1: metabolism_log table and event_outbox columns
-- Adds self-cleansing (excretion) metadata to the event outbox for
-- tracking which events were metabolized and why.

CREATE TABLE IF NOT EXISTS metabolism_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL,
    source VARCHAR(100) NOT NULL,
    total_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    impact_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    ref_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    semantic_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    sem_skipped BOOLEAN NOT NULL DEFAULT false,
    raw_score_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS excreted_at TIMESTAMPTZ;
ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS excretion_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_metabolism_log_created_at ON metabolism_log(created_at);
CREATE INDEX IF NOT EXISTS idx_metabolism_log_source ON metabolism_log(source);
CREATE INDEX IF NOT EXISTS idx_metabolism_log_high_score ON metabolism_log(total_score) WHERE total_score >= 0.70;
