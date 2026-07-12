DROP INDEX IF EXISTS idx_event_outbox_pending_created;
ALTER TABLE event_outbox
    DROP COLUMN IF EXISTS idempotency_key,
    DROP COLUMN IF EXISTS correlation_id,
    DROP COLUMN IF EXISTS entity_type,
    DROP COLUMN IF EXISTS entity_id,
    DROP COLUMN IF EXISTS actor,
    DROP COLUMN IF EXISTS version;
