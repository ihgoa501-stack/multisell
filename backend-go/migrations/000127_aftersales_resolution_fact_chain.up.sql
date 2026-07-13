CREATE TABLE aftersales_resolution_case (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
 after_sales_id BIGINT REFERENCES after_sales_order(id) ON DELETE RESTRICT, order_id BIGINT NOT NULL REFERENCES sales_order(id) ON DELETE RESTRICT,
 platform_account_id BIGINT NOT NULL REFERENCES platform_integration_account(id) ON DELETE RESTRICT,
 kind VARCHAR(16) NOT NULL CHECK(kind IN('refund','return','dispute')), requested_minor BIGINT NOT NULL CHECK(requested_minor>0), currency CHAR(3) NOT NULL,
 reason TEXT NOT NULL, request_source VARCHAR(24) NOT NULL CHECK(request_source IN('platform_request','buyer_request')), request_evidence_id VARCHAR(200) NOT NULL,
 request_observed_at TIMESTAMPTZ NOT NULL, request_key VARCHAR(200) NOT NULL, status VARCHAR(24) NOT NULL CHECK(status IN('requested','approved','rejected','execution_submitted','succeeded','failed')),
 decision_reason TEXT NOT NULL DEFAULT '', decision_key VARCHAR(200) NOT NULL DEFAULT '', decided_by BIGINT REFERENCES "user"(id), decided_at TIMESTAMPTZ,
 execution_key VARCHAR(200) NOT NULL DEFAULT '', external_request_id VARCHAR(200) NOT NULL DEFAULT '', submitted_at TIMESTAMPTZ,
 consequence_status VARCHAR(16) NOT NULL DEFAULT 'deferred' CHECK(consequence_status='deferred'), created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(owner_id,request_key), UNIQUE(owner_id,request_source,request_evidence_id),
 CHECK((status='requested' AND decided_by IS NULL AND decided_at IS NULL) OR status<>'requested'),
 CHECK((status IN('execution_submitted','succeeded','failed') AND external_request_id<>'' AND submitted_at IS NOT NULL) OR status NOT IN('execution_submitted','succeeded','failed'))
);
CREATE UNIQUE INDEX uq_aftersales_resolution_external_request ON aftersales_resolution_case(owner_id,platform_account_id,external_request_id) WHERE external_request_id<>'';
CREATE FUNCTION enforce_aftersales_resolution_owner_authority() RETURNS trigger AS $$
BEGIN
 IF (SELECT count(*) FROM platform_order_ingest WHERE owner_id=NEW.owner_id AND account_id=NEW.platform_account_id AND normalized_order_id=NEW.order_id AND truth_status='external_observed' AND processing_status='applied') <> 1 THEN
  RAISE EXCEPTION 'aftersales resolution lacks one applied external Owner order fact';
 END IF;
 RETURN NEW;
END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_aftersales_resolution_owner_authority BEFORE INSERT ON aftersales_resolution_case FOR EACH ROW EXECUTE FUNCTION enforce_aftersales_resolution_owner_authority();
CREATE FUNCTION protect_aftersales_resolution_identity() RETURNS trigger AS $$
BEGIN
 IF NEW.owner_id<>OLD.owner_id OR NEW.after_sales_id IS DISTINCT FROM OLD.after_sales_id OR NEW.order_id<>OLD.order_id OR NEW.platform_account_id<>OLD.platform_account_id OR NEW.kind<>OLD.kind OR NEW.requested_minor<>OLD.requested_minor OR NEW.currency<>OLD.currency OR NEW.request_source<>OLD.request_source OR NEW.request_evidence_id<>OLD.request_evidence_id OR NEW.request_observed_at<>OLD.request_observed_at OR NEW.request_key<>OLD.request_key THEN
  RAISE EXCEPTION 'aftersales resolution request identity is immutable';
 END IF;
 RETURN NEW;
END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_aftersales_resolution_identity_immutable BEFORE UPDATE ON aftersales_resolution_case FOR EACH ROW EXECUTE FUNCTION protect_aftersales_resolution_identity();
CREATE TABLE aftersales_resolution_receipt (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
 resolution_id BIGINT NOT NULL UNIQUE REFERENCES aftersales_resolution_case(id) ON DELETE RESTRICT,
 outcome VARCHAR(16) NOT NULL CHECK(outcome IN('succeeded','failed')), source_type VARCHAR(32) NOT NULL CHECK(source_type IN('platform_receipt','controlled_reconciliation')),
 evidence_id VARCHAR(200) NOT NULL, external_receipt_id VARCHAR(200) NOT NULL, observed_at TIMESTAMPTZ NOT NULL,
 actual_minor BIGINT NOT NULL CHECK(actual_minor>=0), currency CHAR(3) NOT NULL, failure_code VARCHAR(120) NOT NULL DEFAULT '',
 receipt_payload JSONB NOT NULL, receipt_sha256 CHAR(64) NOT NULL CHECK(receipt_sha256 ~ '^[0-9a-f]{64}$'), recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(owner_id,evidence_id), CHECK((outcome='succeeded' AND actual_minor>0 AND failure_code='') OR (outcome='failed' AND failure_code<>''))
);
CREATE FUNCTION reject_aftersales_resolution_receipt_mutation() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'aftersales resolution receipt is immutable'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_aftersales_resolution_receipt_no_update BEFORE UPDATE ON aftersales_resolution_receipt FOR EACH ROW EXECUTE FUNCTION reject_aftersales_resolution_receipt_mutation();
CREATE TRIGGER trg_aftersales_resolution_receipt_no_delete BEFORE DELETE ON aftersales_resolution_receipt FOR EACH ROW EXECUTE FUNCTION reject_aftersales_resolution_receipt_mutation();
