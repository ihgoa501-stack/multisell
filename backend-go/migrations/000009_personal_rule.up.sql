CREATE TABLE IF NOT EXISTS personal_rule (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(80),
    rule_type VARCHAR(20) NOT NULL,
    name VARCHAR(200) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    effect JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_personal_rule_user ON personal_rule(user_id, agent_id);
