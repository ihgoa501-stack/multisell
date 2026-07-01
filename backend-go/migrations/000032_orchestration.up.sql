-- Product Lifecycle Orchestration Pipeline
-- Tracks each step of a product's lifecycle from sourcing through delisting.

CREATE TABLE IF NOT EXISTS lifecycle_step (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    step VARCHAR(50) NOT NULL,
    agent_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_step_product_id ON lifecycle_step(product_id);
CREATE INDEX IF NOT EXISTS idx_lifecycle_step_status ON lifecycle_step(status);

CREATE TABLE IF NOT EXISTS orchestration_config (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    steps JSONB NOT NULL DEFAULT '[]',
    failure_action VARCHAR(20) NOT NULL DEFAULT 'stop',
    auto_approve_pct NUMERIC(5,2) NOT NULL DEFAULT 80,
    auto_retry_count INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
