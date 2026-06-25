-- 000011_entropy
-- rule_conflict already created in 000001; add columns idempotently.
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'open';
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS resolution TEXT;
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS payload JSONB DEFAULT '{}'::jsonb;
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE rule_conflict ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;

-- spc_control_limit already created in 000001; modify existing columns.
-- The original 000001 schema uses: user_id, metric_name, upper_limit, lower_limit.
-- The 000011 version uses: agent_id, decision_point, metric, mean, std_dev, etc.
-- Use ALTER TABLE to add new columns from the 000011 schema.
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS decision_point VARCHAR(80);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS mean NUMERIC(6,4) NOT NULL DEFAULT 0;
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS std_dev NUMERIC(6,4) NOT NULL DEFAULT 0;
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS upper_control NUMERIC(6,4);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS lower_control NUMERIC(6,4);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS upper_warning NUMERIC(6,4);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS lower_warning NUMERIC(6,4);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS sample_size INT DEFAULT 0;
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS out_of_control BOOLEAN DEFAULT false;
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS last_value NUMERIC(6,4);
ALTER TABLE spc_control_limit ADD COLUMN IF NOT EXISTS calculated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_spc_agent ON spc_control_limit(agent_id, metric);
CREATE INDEX IF NOT EXISTS idx_rule_conflict_status ON rule_conflict(status);
