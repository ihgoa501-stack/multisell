CREATE OR REPLACE FUNCTION enforce_approval_execution_state_machine()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'approval_execution is append-preserving: deletion is forbidden';
    END IF;

    IF NEW.approval_id IS DISTINCT FROM OLD.approval_id
        OR NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key
        OR NEW.action_type IS DISTINCT FROM OLD.action_type
        OR NEW.target_type IS DISTINCT FROM OLD.target_type
        OR NEW.target_id IS DISTINCT FROM OLD.target_id
        OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'approval_execution binding is immutable';
    END IF;

    IF OLD.state = 'succeeded' THEN
        RAISE EXCEPTION 'succeeded approval_execution is terminal';
    END IF;

    IF NOT (
        (OLD.state = 'processing' AND NEW.state IN ('succeeded', 'failed'))
        OR (OLD.state = 'failed' AND NEW.state = 'processing')
    ) THEN
        RAISE EXCEPTION 'invalid approval_execution transition: % -> %', OLD.state, NEW.state;
    END IF;

    IF NEW.state = 'processing' AND (NEW.completed_at IS NOT NULL OR NEW.error_message <> '') THEN
        RAISE EXCEPTION 'processing approval_execution cannot contain completion data';
    END IF;
    IF NEW.state = 'succeeded' AND (NEW.completed_at IS NULL OR NEW.error_message <> '') THEN
        RAISE EXCEPTION 'succeeded approval_execution requires completed_at and no error';
    END IF;
    IF NEW.state = 'failed' AND (NEW.completed_at IS NULL OR NEW.error_message = '') THEN
        RAISE EXCEPTION 'failed approval_execution requires completed_at and an error';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS approval_execution_state_machine ON approval_execution;
CREATE TRIGGER approval_execution_state_machine
BEFORE UPDATE OR DELETE ON approval_execution
FOR EACH ROW EXECUTE FUNCTION enforce_approval_execution_state_machine();
