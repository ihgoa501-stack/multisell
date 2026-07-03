DROP INDEX IF EXISTS idx_event_dlq_idempotency_key;
ALTER TABLE event_dlq DROP COLUMN IF EXISTS idempotency_key;
