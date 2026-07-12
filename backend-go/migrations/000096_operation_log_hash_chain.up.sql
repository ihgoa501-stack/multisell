CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE operation_log
    ADD COLUMN IF NOT EXISTS previous_hash VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS record_hash VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(128) NOT NULL DEFAULT '';

CREATE OR REPLACE FUNCTION audit_operation_hash(
    p_previous_hash TEXT, p_module TEXT, p_action TEXT, p_resource_id TEXT,
    p_content TEXT, p_operator TEXT, p_user_id BIGINT, p_result TEXT, p_ip TEXT,
    p_duration INTEGER, p_trigger_type TEXT, p_agent_suggestion_id BIGINT,
    p_approval_id BIGINT, p_entity_type TEXT, p_entity_id BIGINT, p_created_at TIMESTAMPTZ,
    p_correlation_id TEXT
) RETURNS TEXT AS $$
    SELECT encode(digest(concat_ws('|',
        COALESCE(p_previous_hash, ''), COALESCE(p_module, ''), COALESCE(p_action, ''),
        COALESCE(p_resource_id, ''), COALESCE(p_content, ''), COALESCE(p_operator, ''),
        COALESCE(p_user_id::TEXT, ''), COALESCE(p_result, ''), COALESCE(p_ip, ''),
        COALESCE(p_duration::TEXT, ''), COALESCE(p_trigger_type, ''),
        COALESCE(p_agent_suggestion_id::TEXT, ''), COALESCE(p_approval_id::TEXT, ''),
        COALESCE(p_entity_type, ''), COALESCE(p_entity_id::TEXT, ''),
        COALESCE(p_created_at::TEXT, ''), COALESCE(p_correlation_id, '')
    ), 'sha256'), 'hex');
$$ LANGUAGE SQL IMMUTABLE;

DO $$
DECLARE
    row_record operation_log%ROWTYPE;
    chain_head TEXT := '';
    calculated TEXT;
BEGIN
    FOR row_record IN SELECT * FROM operation_log ORDER BY id LOOP
        calculated := audit_operation_hash(chain_head, row_record.module, row_record.action,
            row_record.resource_id, row_record.content, row_record.operator, row_record.user_id,
            row_record.result, row_record.ip, row_record.duration, row_record.trigger_type,
            row_record.agent_suggestion_id, row_record.approval_id, row_record.entity_type,
            row_record.entity_id, row_record.created_at, row_record.correlation_id);
        UPDATE operation_log SET previous_hash = chain_head, record_hash = calculated WHERE id = row_record.id;
        chain_head := calculated;
    END LOOP;
END $$;

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

DROP TRIGGER IF EXISTS trg_operation_log_append_only ON operation_log;
CREATE TRIGGER trg_operation_log_append_only
BEFORE INSERT OR UPDATE OR DELETE ON operation_log
FOR EACH ROW EXECUTE FUNCTION protect_operation_log_chain();

CREATE UNIQUE INDEX IF NOT EXISTS idx_operation_log_record_hash ON operation_log(record_hash);
