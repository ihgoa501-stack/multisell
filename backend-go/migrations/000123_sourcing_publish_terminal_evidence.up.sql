CREATE TABLE sourcing_publish_terminal_evidence (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id) ON DELETE RESTRICT,
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT,
    publish_attempt_id BIGINT NOT NULL UNIQUE REFERENCES sourcing_publish_attempt(id) ON DELETE RESTRICT,
    platform_id BIGINT NOT NULL REFERENCES platform(id) ON DELETE RESTRICT,
    platform_account_id BIGINT NOT NULL REFERENCES platform_integration_account(id) ON DELETE RESTRICT,
    outcome VARCHAR(16) NOT NULL CHECK (outcome IN ('succeeded','failed')),
    source_type VARCHAR(40) NOT NULL CHECK (source_type IN ('platform_receipt','controlled_reconciliation')),
    evidence_id VARCHAR(200) NOT NULL,
    external_receipt_id VARCHAR(200) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    platform_product_id VARCHAR(255) NOT NULL DEFAULT '',
    failure_code VARCHAR(120) NOT NULL DEFAULT '',
    failure_message TEXT NOT NULL DEFAULT '',
    receipt_payload JSONB NOT NULL,
    receipt_sha256 CHAR(64) NOT NULL CHECK (receipt_sha256 ~ '^[0-9a-f]{64}$'),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, platform_id, evidence_id),
    CHECK ((outcome = 'succeeded' AND platform_product_id <> '' AND failure_code = '' AND failure_message = '')
        OR (outcome = 'failed' AND failure_code <> '' AND failure_message <> ''))
);

CREATE OR REPLACE FUNCTION reject_sourcing_publish_terminal_evidence_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing publish terminal evidence is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_publish_terminal_evidence_no_update
BEFORE UPDATE ON sourcing_publish_terminal_evidence FOR EACH ROW
EXECUTE FUNCTION reject_sourcing_publish_terminal_evidence_mutation();
CREATE TRIGGER trg_sourcing_publish_terminal_evidence_no_delete
BEFORE DELETE ON sourcing_publish_terminal_evidence FOR EACH ROW
EXECUTE FUNCTION reject_sourcing_publish_terminal_evidence_mutation();

COMMENT ON TABLE sourcing_publish_terminal_evidence IS
  'Immutable externally observed or controlled-reconciliation evidence that alone closes an exact-task publish attempt.';
