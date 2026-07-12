CREATE TABLE IF NOT EXISTS tool_execution (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    call_hash VARCHAR(64) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    target_type VARCHAR(100) NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    state VARCHAR(20) NOT NULL,
    result_json TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    lease_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_tool_execution_state CHECK (state IN ('processing', 'succeeded', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_tool_execution_state_lease ON tool_execution(state, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_tool_execution_created_at ON tool_execution(created_at);
