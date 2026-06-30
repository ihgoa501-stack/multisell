DROP INDEX IF EXISTS idx_event_outbox_event_id;

ALTER TABLE event_outbox
  DROP COLUMN IF EXISTS event_id,
  DROP COLUMN IF EXISTS delivery_attempts,
  DROP COLUMN IF EXISTS last_error;
