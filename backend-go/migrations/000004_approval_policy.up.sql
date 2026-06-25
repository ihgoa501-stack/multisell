CREATE TABLE IF NOT EXISTS approval_policy_rule (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    risk_level VARCHAR(20),
    action_type VARCHAR(100),
    squad_id VARCHAR(50),
    agent_id VARCHAR(20),
    business_object_type VARCHAR(64),
    max_amount NUMERIC(12,2),
    max_quantity INTEGER,
    min_confidence NUMERIC(5,4),
    auto_approve BOOLEAN NOT NULL DEFAULT false,
    outcome VARCHAR(20) NOT NULL DEFAULT 'escalate',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_approval_policy_enabled ON approval_policy_rule(enabled);

INSERT INTO approval_policy_rule (name, description, risk_level, action_type, squad_id, agent_id, business_object_type, max_amount, max_quantity, min_confidence, auto_approve, outcome, priority, enabled) VALUES
('L3 Auto-Approve: Low Risk', 'Low risk actions auto-approved', 'low', '*', '*', '*', '*', NULL, NULL, 0.70, true, 'auto_approve', 100, true),
('A5 Stock Alert Auto', 'Stock alerts with confidence >= 0.8 auto-approved', 'medium', 'stock_alert', '*', 'A5', 'inventory', NULL, NULL, 0.80, true, 'auto_approve', 90, true),
('G3 Small Discount Auto', 'Discounts under ¥200 auto-approved', 'medium', 'discount_check', '*', 'G3', 'price_rule', 200.00, 5, 0.75, true, 'auto_approve', 80, true),
('A2 Listing Draft Auto', 'Listing drafts auto-approved', 'medium', 'listing_optimize', '*', 'A2', 'listing', NULL, NULL, 0.70, true, 'auto_approve', 85, true),
('Escalate: Medium Risk Over ¥1000', 'Medium risk over ¥1000 requires human', 'medium', '*', '*', '*', '*', 1000.00, NULL, NULL, false, 'escalate', 50, true),
('Always Escalate: High Risk', 'All high risk actions require human', 'high', '*', '*', '*', '*', NULL, NULL, NULL, false, 'escalate', 40, true),
('Block: Critical Risk', 'Critical risk actions blocked', 'critical', '*', '*', '*', '*', NULL, NULL, NULL, false, 'block', 30, true),
('A7 Compliance Manual', 'Compliance always requires human', 'high', 'compliance_check', '*', 'A7', 'product', NULL, NULL, NULL, false, 'escalate', 60, true),
('A6 Low Confidence Escalate', 'Profit watch with confidence < 0.7 requires human', 'medium', 'profit_check', '*', 'A6', 'sku', NULL, NULL, 0.70, false, 'escalate', 70, true),
('Batch Ops Over 10 SKUs', 'Batch ops over 10 SKUs require human', 'medium', '*', '*', '*', '*', NULL, 10, NULL, false, 'escalate', 55, true);
