DROP INDEX IF EXISTS idx_event_processed_state;
ALTER TABLE event_processed DROP COLUMN IF EXISTS created_at;
ALTER TABLE event_processed DROP COLUMN IF EXISTS state;
