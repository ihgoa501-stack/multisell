CREATE TABLE IF NOT EXISTS scheduler_retry (
    id VARCHAR(36) PRIMARY KEY,
    task_id VARCHAR(100) NOT NULL,
    agent_id VARCHAR(100) NOT NULL,
    decision_point VARCHAR(100) NOT NULL,
    failed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_error TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scheduler_retry_attempts CHECK (attempts >= 1)
);
CREATE INDEX IF NOT EXISTS idx_scheduler_retry_task ON scheduler_retry(task_id);
CREATE INDEX IF NOT EXISTS idx_scheduler_retry_failed_at ON scheduler_retry(failed_at);
