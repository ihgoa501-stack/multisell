DROP TRIGGER IF EXISTS trg_operation_log_chain_repair_snapshot_immutable
    ON operation_log_chain_repair_snapshot;
DROP FUNCTION IF EXISTS protect_operation_log_chain_repair_snapshot();
DROP FUNCTION IF EXISTS audit_operation_chain_status();

ALTER TABLE operation_log DISABLE TRIGGER trg_operation_log_append_only;

UPDATE operation_log current
SET previous_hash = snapshot.old_previous_hash,
    record_hash = snapshot.old_record_hash
FROM operation_log_chain_repair_snapshot snapshot
WHERE snapshot.repair_version = 152
  AND snapshot.record_id = current.id;

ALTER TABLE operation_log ENABLE TRIGGER trg_operation_log_append_only;

CREATE OR REPLACE FUNCTION protect_operation_log_chain() RETURNS TRIGGER AS $$
DECLARE
    chain_head TEXT;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'operation_log is append-only';
    END IF;
    PERFORM pg_advisory_xact_lock(61420260712);
    NEW.created_at := COALESCE(NEW.created_at, NOW());
    SELECT record_hash INTO chain_head FROM operation_log ORDER BY id DESC LIMIT 1;
    NEW.previous_hash := COALESCE(chain_head, '');
    NEW.record_hash := audit_operation_hash(NEW.previous_hash, NEW.module, NEW.action,
        NEW.resource_id, NEW.content, NEW.operator, NEW.user_id, NEW.result, NEW.ip,
        NEW.duration, NEW.trigger_type, NEW.agent_suggestion_id, NEW.approval_id,
        NEW.entity_type, NEW.entity_id, NEW.created_at, NEW.correlation_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP INDEX IF EXISTS idx_operation_log_previous_hash;
DROP TABLE IF EXISTS operation_log_chain_repair_snapshot;
