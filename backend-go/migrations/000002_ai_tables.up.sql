-- AI Trace tables
CREATE TABLE IF NOT EXISTS ai_trace (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT,
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(80) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'running',
    model_provider VARCHAR(80),
    model_name VARCHAR(120),
    prompt_version VARCHAR(80),
    input_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    final_output JSONB,
    confidence NUMERIC(5,4),
    risk_level VARCHAR(20),
    token_count INTEGER,
    latency_ms INTEGER,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_trace_event (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL REFERENCES ai_trace(trace_id),
    event_type VARCHAR(64) NOT NULL,
    seq INTEGER NOT NULL,
    content TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (trace_id, seq)
);

CREATE TABLE IF NOT EXISTS ai_evidence_ref (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL REFERENCES ai_trace(trace_id),
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS unified_action (
    id BIGSERIAL PRIMARY KEY,
    source_table VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    agent_id VARCHAR(20),
    squad_id VARCHAR(50),
    user_id BIGINT,
    action_type VARCHAR(100) NOT NULL,
    business_object_type VARCHAR(64),
    business_object_id VARCHAR(128),
    title TEXT NOT NULL,
    description TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    before_snapshot JSONB,
    after_snapshot JSONB,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',
    requires_approval BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(32) NOT NULL DEFAULT 'suggested',
    confidence NUMERIC(5,4),
    proposed_by VARCHAR(100),
    approved_by VARCHAR(100),
    rejected_by VARCHAR(100),
    executed_by VARCHAR(100),
    rejection_reason TEXT,
    proposed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    executing_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_table, source_id)
);

-- Indexes
CREATE INDEX idx_ai_trace_agent_id ON ai_trace(agent_id);
CREATE INDEX idx_ai_trace_status ON ai_trace(status);
CREATE INDEX idx_ai_trace_started_at ON ai_trace(started_at);
CREATE INDEX idx_ai_trace_event_trace_id ON ai_trace_event(trace_id);
CREATE INDEX idx_ai_evidence_ref_trace_id ON ai_evidence_ref(trace_id);
CREATE INDEX idx_unified_action_status ON unified_action(status);
CREATE INDEX idx_unified_action_agent_id ON unified_action(agent_id);
CREATE INDEX idx_unified_action_user_id ON unified_action(user_id);
CREATE INDEX idx_unified_action_proposed_at ON unified_action(proposed_at);
