CREATE TABLE IF NOT EXISTS metabolism_log_archive (
    id BIGINT PRIMARY KEY,
    event_id BIGINT NOT NULL,
    source VARCHAR(100) NOT NULL,
    total_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    impact_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    ref_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    semantic_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    sem_skipped BOOLEAN NOT NULL DEFAULT false,
    dimensions JSONB,
    excretable BOOLEAN NOT NULL DEFAULT false,
    reason TEXT,
    excreted_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_metabolism_log_archive_excreted_at ON metabolism_log_archive(excreted_at);
CREATE INDEX idx_metabolism_log_archive_source ON metabolism_log_archive(source);
