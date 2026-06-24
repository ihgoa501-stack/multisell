CREATE TABLE IF NOT EXISTS agent_trust_score (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(20) NOT NULL UNIQUE,
    agent_name VARCHAR(100) NOT NULL,
    squad_id VARCHAR(50),
    total_actions INTEGER NOT NULL DEFAULT 0,
    adopted_actions INTEGER NOT NULL DEFAULT 0,
    rejected_actions INTEGER NOT NULL DEFAULT 0,
    failed_actions INTEGER NOT NULL DEFAULT 0,
    auto_approved INTEGER NOT NULL DEFAULT 0,
    adoption_rate NUMERIC(5,4) NOT NULL DEFAULT 0,
    execution_success NUMERIC(5,4) NOT NULL DEFAULT 0,
    avg_confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    trust_score NUMERIC(5,4) NOT NULL DEFAULT 0,
    autonomy_level VARCHAR(20) NOT NULL DEFAULT 'advisory',
    target_level VARCHAR(20),
    estimated_savings NUMERIC(12,2) NOT NULL DEFAULT 0,
    last_action_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_trust_score ON agent_trust_score(trust_score DESC);

INSERT INTO agent_trust_score (agent_id, agent_name, squad_id, autonomy_level, trust_score, target_level) VALUES
('A1', 'Product Scout', 'autonomous', 'advisory', 0.00, 'guided'),
('A2', 'Listing Optimizer', 'autonomous', 'guided', 0.35, 'supervised'),
('A3', 'Ad Advice', 'autonomous', 'advisory', 0.00, 'guided'),
('A4', 'Customer Service', 'autonomous', 'supervised', 0.60, 'autonomous'),
('A5', 'Inventory Alert', 'autonomous', 'supervised', 0.60, 'autonomous'),
('A6', 'Profit Watch', 'autonomous', 'supervised', 0.60, 'autonomous'),
('A7', 'Compliance Guard', 'autonomous', 'supervised', 0.55, 'autonomous'),
('G1', 'Dashboard', 'governance', 'advisory', 0.00, 'guided'),
('G2', 'Warehouse Customs', 'governance', 'supervised', 0.50, 'autonomous'),
('G3', 'Discount Risk', 'governance', 'supervised', 0.65, 'autonomous')
ON CONFLICT (agent_id) DO NOTHING;
