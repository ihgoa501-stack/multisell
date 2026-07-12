#!/bin/sh
set -eu

DATABASE_URL=${DATABASE_URL:-}
[ -n "$DATABASE_URL" ] || { echo "[ERROR] DATABASE_URL is required." >&2; exit 2; }
command -v psql >/dev/null 2>&1 || { echo "[ERROR] psql is required." >&2; exit 2; }

# Everything runs in one rolled-back transaction. The verifier may therefore
# use an already-migrated CI database without leaving test records behind.
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 <<'SQL'
BEGIN;

DO $$
DECLARE
    approval bigint;
BEGIN
    INSERT INTO approval_request (
        product_id, request_type, requester, reviewer, status,
        target_type, target_id, risk_level, created_at, updated_at
    ) VALUES (
        0, 'user_role_assign', 'state-machine-verifier', 'state-machine-verifier',
        'approved', 'user', 1, 'high', NOW(), NOW()
    ) RETURNING id INTO approval;

    INSERT INTO approval_execution (
        approval_id, idempotency_key, action_type, target_type, target_id, state
    ) VALUES (
        approval, 'approval-state-machine-verifier', 'user_role_assign', 'user', '1', 'processing'
    );

    UPDATE approval_execution
       SET state = 'failed', error_message = 'expected failure', completed_at = NOW(), updated_at = NOW()
     WHERE approval_id = approval;
    UPDATE approval_execution
       SET state = 'processing', error_message = '', completed_at = NULL, updated_at = NOW()
     WHERE approval_id = approval;
    UPDATE approval_execution
       SET state = 'succeeded', error_message = '', completed_at = NOW(), updated_at = NOW()
     WHERE approval_id = approval;

    BEGIN
        UPDATE approval_execution SET target_id = '2' WHERE approval_id = approval;
        RAISE EXCEPTION 'binding mutation was accepted';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM = 'binding mutation was accepted' THEN RAISE; END IF;
    END;

    BEGIN
        UPDATE approval_execution
           SET state = 'failed', error_message = 'tamper', completed_at = NOW()
         WHERE approval_id = approval;
        RAISE EXCEPTION 'terminal rollback was accepted';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM = 'terminal rollback was accepted' THEN RAISE; END IF;
    END;

    BEGIN
        UPDATE approval_execution SET updated_at = NOW() + interval '1 second' WHERE approval_id = approval;
        RAISE EXCEPTION 'terminal metadata mutation was accepted';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM = 'terminal metadata mutation was accepted' THEN RAISE; END IF;
    END;

    BEGIN
        DELETE FROM approval_execution WHERE approval_id = approval;
        RAISE EXCEPTION 'record deletion was accepted';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM = 'record deletion was accepted' THEN RAISE; END IF;
    END;
END;
$$;

ROLLBACK;
SQL

echo "approval execution PostgreSQL state machine verified"
