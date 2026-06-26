-- Rollback Metabolism M1 Phase 1

DROP TABLE IF EXISTS metabolism_log;

ALTER TABLE event_outbox DROP COLUMN IF EXISTS excreted_at;
ALTER TABLE event_outbox DROP COLUMN IF EXISTS excretion_reason;
