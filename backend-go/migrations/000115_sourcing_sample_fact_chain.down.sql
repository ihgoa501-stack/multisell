DROP TRIGGER IF EXISTS trg_sourcing_sample_event_append_only ON sourcing_sample_event;
DROP FUNCTION IF EXISTS reject_sourcing_sample_event_mutation();
DROP TABLE IF EXISTS sourcing_sample_event;
DROP TABLE IF EXISTS sourcing_sample;
