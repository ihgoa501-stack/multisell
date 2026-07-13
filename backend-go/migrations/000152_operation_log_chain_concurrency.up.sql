CREATE TABLE IF NOT EXISTS operation_log_chain_repair_snapshot (
    repair_version INTEGER NOT NULL,
    record_id BIGINT NOT NULL,
    old_previous_hash VARCHAR(64) NOT NULL,
    old_record_hash VARCHAR(64) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (repair_version, record_id)
);

INSERT INTO operation_log_chain_repair_snapshot (
    repair_version, record_id, old_previous_hash, old_record_hash
)
SELECT 152, id, previous_hash, record_hash
FROM operation_log
ON CONFLICT DO NOTHING;

ALTER TABLE operation_log DISABLE TRIGGER trg_operation_log_append_only;

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
        UPDATE operation_log
        SET previous_hash = chain_head, record_hash = calculated
        WHERE id = row_record.id;
        chain_head := calculated;
    END LOOP;
END $$;

ALTER TABLE operation_log ENABLE TRIGGER trg_operation_log_append_only;

CREATE INDEX IF NOT EXISTS idx_operation_log_previous_hash
    ON operation_log(previous_hash);

CREATE OR REPLACE FUNCTION protect_operation_log_chain() RETURNS TRIGGER AS $$
DECLARE
    chain_head TEXT;
    tip_count BIGINT;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'operation_log is append-only';
    END IF;

    PERFORM pg_advisory_xact_lock(61420260712);

    -- ponytail: scan the indexed chain for its unique tip; use a dedicated
    -- head row only if measured audit volume makes this lookup material.
    SELECT COUNT(*), MIN(candidate.record_hash)
    INTO tip_count, chain_head
    FROM operation_log candidate
    WHERE NOT EXISTS (
        SELECT 1 FROM operation_log child
        WHERE child.previous_hash = candidate.record_hash
    );

    IF EXISTS (SELECT 1 FROM operation_log) AND tip_count <> 1 THEN
        RAISE EXCEPTION 'operation_log chain has % tips', tip_count;
    END IF;

    NEW.created_at := COALESCE(NEW.created_at, NOW());
    NEW.previous_hash := COALESCE(chain_head, '');
    NEW.record_hash := audit_operation_hash(NEW.previous_hash, NEW.module, NEW.action,
        NEW.resource_id, NEW.content, NEW.operator, NEW.user_id, NEW.result, NEW.ip,
        NEW.duration, NEW.trigger_type, NEW.agent_suggestion_id, NEW.approval_id,
        NEW.entity_type, NEW.entity_id, NEW.created_at, NEW.correlation_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION audit_operation_chain_status()
RETURNS TABLE (
    total BIGINT,
    self_hash_bad BIGINT,
    roots BIGINT,
    missing_predecessors BIGINT,
    fork_points BIGINT,
    tips BIGINT,
    reachable BIGINT
) AS $$
WITH RECURSIVE
checks AS (
    SELECT record_hash, previous_hash,
        record_hash IS DISTINCT FROM audit_operation_hash(
            previous_hash, module, action, resource_id, content, operator, user_id,
            result, ip, duration, trigger_type, agent_suggestion_id, approval_id,
            entity_type, entity_id, created_at, correlation_id
        ) AS self_hash_bad
    FROM operation_log
),
refs AS (
    SELECT previous_hash, COUNT(*) AS children
    FROM operation_log
    WHERE previous_hash <> ''
    GROUP BY previous_hash
),
walk(record_hash) AS (
    SELECT record_hash FROM operation_log WHERE previous_hash = ''
    UNION
    SELECT child.record_hash
    FROM walk parent
    JOIN operation_log child ON child.previous_hash = parent.record_hash
)
SELECT
    (SELECT COUNT(*) FROM checks),
    (SELECT COUNT(*) FROM checks WHERE self_hash_bad),
    (SELECT COUNT(*) FROM checks WHERE previous_hash = ''),
    (SELECT COUNT(*) FROM checks c
        WHERE c.previous_hash <> ''
          AND NOT EXISTS (
              SELECT 1 FROM operation_log predecessor
              WHERE predecessor.record_hash = c.previous_hash
          )),
    (SELECT COUNT(*) FROM refs WHERE children > 1),
    (SELECT COUNT(*) FROM operation_log candidate
        WHERE NOT EXISTS (
            SELECT 1 FROM operation_log child
            WHERE child.previous_hash = candidate.record_hash
        )),
    (SELECT COUNT(*) FROM walk);
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION protect_operation_log_chain_repair_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'operation_log_chain_repair_snapshot is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_operation_log_chain_repair_snapshot_immutable
BEFORE INSERT OR UPDATE OR DELETE ON operation_log_chain_repair_snapshot
FOR EACH ROW EXECUTE FUNCTION protect_operation_log_chain_repair_snapshot();
