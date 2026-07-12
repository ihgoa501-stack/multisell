CREATE TABLE IF NOT EXISTS approval_execution (
    approval_id BIGINT PRIMARY KEY REFERENCES approval_request(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    action_type VARCHAR(100) NOT NULL,
    target_type VARCHAR(100) NOT NULL DEFAULT '',
    target_id VARCHAR(255) NOT NULL DEFAULT '',
    state VARCHAR(20) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT chk_approval_execution_state CHECK (state IN ('processing', 'succeeded', 'failed'))
);
CREATE INDEX IF NOT EXISTS idx_approval_execution_state ON approval_execution(state);
