CREATE TABLE IF NOT EXISTS command_execution (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    action_type VARCHAR(100) NOT NULL,
    agent_id VARCHAR(100) NOT NULL,
    state VARCHAR(20) NOT NULL,
    result_json TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    lease_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_command_execution_state CHECK (state IN ('processing', 'succeeded', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_command_execution_state_lease
    ON command_execution(state, lease_expires_at);

CREATE INDEX IF NOT EXISTS idx_command_execution_created_at
    ON command_execution(created_at);
