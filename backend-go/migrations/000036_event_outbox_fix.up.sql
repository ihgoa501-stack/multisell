-- Add event_id, delivery_attempts, and last_error columns to event_outbox
-- for real outbox status tracking (pending → delivered/failed).

ALTER TABLE event_outbox
  ADD COLUMN IF NOT EXISTS event_id VARCHAR(36) DEFAULT '',
  ADD COLUMN IF NOT EXISTS delivery_attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_error TEXT DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_event_outbox_event_id ON event_outbox(event_id);
