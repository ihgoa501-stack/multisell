CREATE TABLE IF NOT EXISTS sourcing_publish_attempt (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    draft_id BIGINT NOT NULL REFERENCES sourcing_listing_draft(id),
    product_id BIGINT NOT NULL REFERENCES product(id),
    listing_id BIGINT NOT NULL REFERENCES product_listing(id),
    experiment_id VARCHAR(40) NOT NULL REFERENCES experiment_case(experiment_id),
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    platform_account_id BIGINT NOT NULL REFERENCES platform_integration_account(id),
    approval_id BIGINT UNIQUE REFERENCES approval_request(id),
    idempotency_key VARCHAR(160) NOT NULL UNIQUE,
    request_sha256 CHAR(64) NOT NULL,
    request_payload JSONB NOT NULL,
    adapter_request_payload JSONB NOT NULL,
    response_payload JSONB,
    response_sha256 CHAR(64),
    status VARCHAR(24) NOT NULL DEFAULT 'pending_approval',
    error_message TEXT NOT NULL DEFAULT '',
    requested_by BIGINT NOT NULL REFERENCES "user"(id),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by BIGINT REFERENCES "user"(id),
    approved_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    CONSTRAINT ck_sourcing_publish_status CHECK (status IN (
        'pending_approval', 'approved', 'rejected', 'executing', 'submitted', 'reconcile_required', 'succeeded', 'failed'
    ))
);

CREATE INDEX IF NOT EXISTS idx_sourcing_publish_source
    ON sourcing_publish_attempt(sourcing_product_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_sourcing_publish_listing
    ON sourcing_publish_attempt(listing_id);

-- Once an external call starts, its frozen request and account binding are an
-- immutable audit record. Completion may only add the response/error/status.
CREATE OR REPLACE FUNCTION protect_sourcing_publish_attempt()
RETURNS trigger AS $$
BEGIN
    IF (
        NEW.sourcing_product_id IS DISTINCT FROM OLD.sourcing_product_id OR
        NEW.draft_id IS DISTINCT FROM OLD.draft_id OR
        NEW.product_id IS DISTINCT FROM OLD.product_id OR
        NEW.listing_id IS DISTINCT FROM OLD.listing_id OR
        NEW.experiment_id IS DISTINCT FROM OLD.experiment_id OR
        NEW.platform_id IS DISTINCT FROM OLD.platform_id OR
        NEW.platform_account_id IS DISTINCT FROM OLD.platform_account_id OR
        (OLD.approval_id IS NOT NULL AND NEW.approval_id IS DISTINCT FROM OLD.approval_id) OR
        NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key OR
        NEW.request_sha256 IS DISTINCT FROM OLD.request_sha256 OR
        NEW.request_payload IS DISTINCT FROM OLD.request_payload OR
        NEW.adapter_request_payload IS DISTINCT FROM OLD.adapter_request_payload
    ) THEN
        RAISE EXCEPTION 'executed sourcing publish request is immutable';
    END IF;
    IF OLD.status NOT IN ('executing', 'reconcile_required') AND (
        NEW.response_payload IS DISTINCT FROM OLD.response_payload OR
        NEW.response_sha256 IS DISTINCT FROM OLD.response_sha256 OR
        NEW.error_message IS DISTINCT FROM OLD.error_message OR
        NEW.completed_at IS DISTINCT FROM OLD.completed_at
    ) THEN
        RAISE EXCEPTION 'publish result may only be recorded while executing';
    END IF;
    IF OLD.status IN ('submitted', 'succeeded', 'failed') AND (
        NEW.status IS DISTINCT FROM OLD.status OR
        NEW.response_payload IS DISTINCT FROM OLD.response_payload OR
        NEW.response_sha256 IS DISTINCT FROM OLD.response_sha256 OR
        NEW.error_message IS DISTINCT FROM OLD.error_message OR
        NEW.completed_at IS DISTINCT FROM OLD.completed_at
    ) THEN
        RAISE EXCEPTION 'completed sourcing publish result is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_protect_sourcing_publish_attempt ON sourcing_publish_attempt;
CREATE TRIGGER trg_protect_sourcing_publish_attempt
BEFORE UPDATE ON sourcing_publish_attempt
FOR EACH ROW EXECUTE FUNCTION protect_sourcing_publish_attempt();
