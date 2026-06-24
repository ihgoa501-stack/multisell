-- Approval policy engine tables
CREATE TABLE IF NOT EXISTS approval_policy (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    squad_id VARCHAR(50),
    agent_id VARCHAR(20),
    action_type VARCHAR(100),
    risk_level VARCHAR(20),
    condition_expr TEXT,
    effect VARCHAR(20) NOT NULL DEFAULT 'escalate',
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_approval_policy_agent ON approval_policy(agent_id);
CREATE INDEX idx_approval_policy_squad ON approval_policy(squad_id);
CREATE INDEX idx_approval_policy_enabled ON approval_policy(enabled);

-- Seed default policies
INSERT INTO approval_policy (name, squad_id, agent_id, action_type, risk_level, effect, priority, enabled, description) VALUES
('High risk auto-escalate', NULL, NULL, NULL, 'critical', 'escalate', 100, true, '所有critical风险动作自动升级到管理员'),
('Medium risk supervised', NULL, NULL, NULL, 'medium', 'auto_approve', 50, true, 'medium风险自动审批（可配置）'),
('Growth squad low risk', 'growth', NULL, NULL, 'low', 'auto_approve', 40, true, '增长小队低风险动作自动执行'),
('Inventory replenish auto', NULL, 'A5', 'replenish', 'medium', 'auto_approve', 60, true, 'A5补货建议medium风险自动审批')
ON CONFLICT DO NOTHING;
