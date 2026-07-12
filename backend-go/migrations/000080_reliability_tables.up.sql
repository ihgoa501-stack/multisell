CREATE TABLE IF NOT EXISTS agent_status (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL UNIQUE,
    agent_name VARCHAR(255) NOT NULL DEFAULT '',
    squad VARCHAR(100) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    failed_count INTEGER NOT NULL DEFAULT 0,
    error_reason TEXT NOT NULL DEFAULT '',
    is_paused BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS llm_cost_records (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL DEFAULT '',
    model_name VARCHAR(100) NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_cost_agent_id ON llm_cost_records(agent_id);
CREATE INDEX IF NOT EXISTS idx_llm_cost_created_at ON llm_cost_records(created_at);

CREATE TABLE IF NOT EXISTS failure_records (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(255) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_failure_agent_id ON failure_records(agent_id);
CREATE INDEX IF NOT EXISTS idx_failure_status ON failure_records(status);
