-- Migration 000022: add agent learning tables
-- Decision evaluation tracks predicted vs actual outcomes for agent decisions.
-- Agent accuracy stores aggregate accuracy metrics per agent and period.

CREATE TABLE IF NOT EXISTS decision_evaluation (
    id BIGSERIAL PRIMARY KEY,
    decision_trace_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    agent_id VARCHAR(20) NOT NULL,
    predicted_outcome TEXT,
    actual_outcome TEXT,
    score NUMERIC(6,4) NOT NULL DEFAULT 0,
    evaluated_at TIMESTAMPTZ,
    evaluation_type VARCHAR(10) NOT NULL DEFAULT 'T+30',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_decision_evaluation_trace ON decision_evaluation(decision_trace_id);
CREATE INDEX idx_decision_evaluation_product ON decision_evaluation(product_id);
CREATE INDEX idx_decision_evaluation_agent ON decision_evaluation(agent_id);
CREATE INDEX idx_decision_evaluation_created ON decision_evaluation(created_at);

CREATE TABLE IF NOT EXISTS agent_accuracy (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(20) NOT NULL,
    period VARCHAR(5) NOT NULL,
    total_decisions INT NOT NULL DEFAULT 0,
    correct_decisions INT NOT NULL DEFAULT 0,
    accuracy_pct NUMERIC(6,2) NOT NULL DEFAULT 0,
    trend VARCHAR(20) NOT NULL DEFAULT 'stable',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_agent_period ON agent_accuracy(agent_id, period);
CREATE INDEX idx_agent_accuracy_agent ON agent_accuracy(agent_id);
