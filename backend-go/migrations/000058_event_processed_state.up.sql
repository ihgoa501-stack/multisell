-- Add state column for atomic claim model:
--   processing → handler is running (added before dispatch)
--   succeeded  → handler completed successfully
--   failed     → handler failed, event in DLQ (reclaimable for replay)
ALTER TABLE event_processed ADD COLUMN IF NOT EXISTS state VARCHAR(20) NOT NULL DEFAULT 'processing';
ALTER TABLE event_processed ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_event_processed_state ON event_processed(state);
