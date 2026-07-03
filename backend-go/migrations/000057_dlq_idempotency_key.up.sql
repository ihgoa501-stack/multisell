ALTER TABLE event_dlq ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_event_dlq_idempotency_key ON event_dlq(idempotency_key);
