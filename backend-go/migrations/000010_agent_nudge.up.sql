CREATE TABLE IF NOT EXISTS agent_nudge (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    agent_id VARCHAR(20) NOT NULL,
    current_level VARCHAR(20) NOT NULL,
    target_level VARCHAR(20) NOT NULL,
    trust_score NUMERIC(5,4),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    message TEXT,
    metrics JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_nudge_user ON agent_nudge(user_id);
CREATE INDEX IF NOT EXISTS idx_nudge_agent ON agent_nudge(agent_id);
CREATE INDEX IF NOT EXISTS idx_nudge_status ON agent_nudge(status);
