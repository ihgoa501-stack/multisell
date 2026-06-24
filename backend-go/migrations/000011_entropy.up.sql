CREATE TABLE IF NOT EXISTS rule_conflict (
    id BIGSERIAL PRIMARY KEY,
    rule_a_id BIGINT NOT NULL,
    rule_b_id BIGINT NOT NULL,
    agent_id VARCHAR(20),
    conflict_type VARCHAR(20) NOT NULL,
    similarity NUMERIC(5,4),
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    resolution TEXT,
    payload JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS spc_control_limit (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(80) NOT NULL,
    metric VARCHAR(50) NOT NULL,
    mean NUMERIC(6,4) NOT NULL DEFAULT 0,
    std_dev NUMERIC(6,4) NOT NULL DEFAULT 0,
    upper_control NUMERIC(6,4),
    lower_control NUMERIC(6,4),
    upper_warning NUMERIC(6,4),
    lower_warning NUMERIC(6,4),
    sample_size INT DEFAULT 0,
    out_of_control BOOLEAN DEFAULT false,
    last_value NUMERIC(6,4),
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (agent_id, decision_point, metric)
);

CREATE INDEX IF NOT EXISTS idx_spc_agent ON spc_control_limit(agent_id);
CREATE INDEX IF NOT EXISTS idx_rule_conflict_status ON rule_conflict(status);
