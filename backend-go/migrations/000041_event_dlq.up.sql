-- Event DLQ (dead-letter queue) table for failed events that exceeded
-- max delivery retries.
CREATE TABLE IF NOT EXISTS event_dlq (
    id BIGSERIAL PRIMARY KEY,
    original_event_id VARCHAR(36) NOT NULL,
    topic VARCHAR(100) NOT NULL,
    source VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    priority INT NOT NULL DEFAULT 0,
    correlation_id VARCHAR(36) DEFAULT '',
    error_message TEXT DEFAULT '',
    delivery_attempts INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    replayed_at TIMESTAMPTZ,
    replayed_by VARCHAR(100) DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_event_dlq_created ON event_dlq(created_at);
CREATE INDEX IF NOT EXISTS idx_event_dlq_topic ON event_dlq(topic);
