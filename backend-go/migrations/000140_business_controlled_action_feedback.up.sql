CREATE TABLE business_controlled_action (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    owner_decision_id BIGINT NOT NULL REFERENCES business_owner_decision(id),
    capability_id VARCHAR(120) NOT NULL,
    command_type VARCHAR(120) NOT NULL,
    target_type VARCHAR(80) NOT NULL,
    target_id VARCHAR(160) NOT NULL,
    approval_id BIGINT NOT NULL REFERENCES approval_request(id),
    idempotency_key VARCHAR(200) NOT NULL,
    input_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_sha256 CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'approved_pending_execution',
    command_business_id VARCHAR(200) NOT NULL DEFAULT '',
    failure_message TEXT NOT NULL DEFAULT '',
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_business_controlled_action_key UNIQUE(owner_id, idempotency_key),
    CONSTRAINT ck_business_controlled_action_status CHECK (status IN ('approved_pending_execution','executing','succeeded','failed','reconcile_required'))
);

CREATE TABLE business_action_observation (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    controlled_action_id BIGINT NOT NULL REFERENCES business_controlled_action(id),
    evidence_kind VARCHAR(16) NOT NULL,
    truth_status VARCHAR(24) NOT NULL,
    source_object_type VARCHAR(40) NOT NULL,
    source_object_id BIGINT NOT NULL,
    source_manifest_sha256 CHAR(64) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    target_metric VARCHAR(120) NOT NULL,
    target_value VARCHAR(160) NOT NULL,
    actual_value VARCHAR(160) NOT NULL,
    comparison_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_business_action_observation_source UNIQUE(owner_id, controlled_action_id, source_object_type, source_object_id),
    CONSTRAINT ck_business_action_observation_kind CHECK (evidence_kind IN ('support','counter','conflict')),
    CONSTRAINT ck_business_action_observation_truth CHECK (truth_status IN ('actual','external_observed')),
    CONSTRAINT ck_business_action_observation_source CHECK (source_object_type IN ('platform_order_ingest','order_final_profit_version','cash_reconciliation'))
);

CREATE TABLE business_next_action_recommendation (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    controlled_action_id BIGINT NOT NULL REFERENCES business_controlled_action(id),
    recommendation_text TEXT NOT NULL,
    rationale TEXT NOT NULL,
    truth_status VARCHAR(16) NOT NULL DEFAULT 'inferred',
    status VARCHAR(20) NOT NULL DEFAULT 'proposed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_business_next_action_truth CHECK (truth_status = 'inferred'),
    CONSTRAINT ck_business_next_action_status CHECK (status IN ('proposed','superseded'))
);

CREATE OR REPLACE FUNCTION prevent_business_feedback_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'business decision feedback evidence is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION guard_business_controlled_action_authority() RETURNS trigger AS $$
BEGIN
    IF NEW.owner_id <> OLD.owner_id OR NEW.owner_decision_id <> OLD.owner_decision_id OR
       NEW.capability_id <> OLD.capability_id OR NEW.command_type <> OLD.command_type OR
       NEW.target_type <> OLD.target_type OR NEW.target_id <> OLD.target_id OR
       NEW.approval_id <> OLD.approval_id OR NEW.idempotency_key <> OLD.idempotency_key OR
       NEW.input_payload <> OLD.input_payload OR NEW.input_sha256 <> OLD.input_sha256 THEN
        RAISE EXCEPTION 'controlled action authority is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER business_action_observation_immutable BEFORE UPDATE OR DELETE ON business_action_observation FOR EACH ROW EXECUTE FUNCTION prevent_business_feedback_mutation();
CREATE TRIGGER business_next_action_recommendation_immutable BEFORE UPDATE OR DELETE ON business_next_action_recommendation FOR EACH ROW EXECUTE FUNCTION prevent_business_feedback_mutation();
CREATE TRIGGER business_controlled_action_authority_immutable BEFORE UPDATE ON business_controlled_action FOR EACH ROW EXECUTE FUNCTION guard_business_controlled_action_authority();
