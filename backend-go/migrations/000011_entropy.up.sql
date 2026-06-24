-- rule_conflict is already created in 000001_init_schema;
-- add columns that 000011 depends on (idempotent)
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'open';
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS resolution TEXT;
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS payload JSONB DEFAULT '{}'::jsonb;
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;

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
