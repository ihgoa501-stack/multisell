CREATE TABLE IF NOT EXISTS forbidden_action (
    id BIGSERIAL PRIMARY KEY,
    action_type VARCHAR(100) NOT NULL,
    agent_id VARCHAR(100) DEFAULT '',
    risk_level VARCHAR(20) DEFAULT '',
    reason TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_forbidden_action ON forbidden_action (action_type, agent_id);

-- Seed critical forbidden actions for safety
INSERT INTO forbidden_action (action_type, agent_id, risk_level, reason) VALUES
('price_update', '*', 'high', 'AI auto price change is forbidden without Owner approval policy'),
('inventory_change', '*', 'high', 'AI auto inventory change is forbidden without Owner approval policy'),
('listing_publish', '*', 'high', 'AI auto platform publish is forbidden without explicit approval'),
('credential_change', '*', 'high', 'AI cannot change credentials'),
('permission_change', '*', 'high', 'AI cannot change permissions or RBAC rules'),
('destructive_data_change', '*', 'high', 'AI cannot delete business data')
ON CONFLICT DO NOTHING;
