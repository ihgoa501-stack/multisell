CREATE TABLE business_decision_case (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, question TEXT NOT NULL, target TEXT NOT NULL,
 object_type VARCHAR(64) NOT NULL CHECK (object_type='platform_order_ingest'), object_id BIGINT NOT NULL REFERENCES platform_order_ingest(id) ON DELETE RESTRICT,
 truth_status VARCHAR(24) NOT NULL, unknowns_json TEXT NOT NULL, manifest_sha256 CHAR(64) NOT NULL CHECK(manifest_sha256 ~ '^[0-9a-f]{64}$'),
 idempotency_key VARCHAR(128) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(owner_id,idempotency_key));
CREATE INDEX idx_business_decision_case_owner ON business_decision_case(owner_id,id DESC);
CREATE TABLE business_decision_fact_snapshot (
 id BIGSERIAL PRIMARY KEY, decision_case_id BIGINT NOT NULL REFERENCES business_decision_case(id) ON DELETE RESTRICT,
 owner_id BIGINT NOT NULL, object_type VARCHAR(64) NOT NULL, object_id BIGINT NOT NULL, truth_status VARCHAR(24) NOT NULL,
 source_table VARCHAR(96) NOT NULL, source_observed_at TIMESTAMPTZ NOT NULL, payload_json TEXT NOT NULL,
 payload_sha256 CHAR(64) NOT NULL CHECK(payload_sha256 ~ '^[0-9a-f]{64}$'), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(decision_case_id));
CREATE TABLE business_ai_recommendation (
 id BIGSERIAL PRIMARY KEY, decision_case_id BIGINT NOT NULL REFERENCES business_decision_case(id) ON DELETE RESTRICT,
 owner_id BIGINT NOT NULL, recommendation TEXT NOT NULL, rationale TEXT NOT NULL,
 truth_status VARCHAR(24) NOT NULL CHECK(truth_status IN ('quoted','estimated','unknown','mock','inferred')),
 unknowns_json TEXT NOT NULL, manifest_sha256 CHAR(64) NOT NULL, idempotency_key VARCHAR(128) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(owner_id,idempotency_key));
CREATE TABLE business_owner_decision (
 id BIGSERIAL PRIMARY KEY, decision_case_id BIGINT NOT NULL REFERENCES business_decision_case(id) ON DELETE RESTRICT,
 owner_id BIGINT NOT NULL, recommendation_id BIGINT REFERENCES business_ai_recommendation(id) ON DELETE RESTRICT,
 decision VARCHAR(32) NOT NULL CHECK(decision IN ('selected','rejected','paused','request_more_evidence')),
 capability_id VARCHAR(120) NOT NULL DEFAULT '', command_type VARCHAR(120) NOT NULL DEFAULT '',
 target_type VARCHAR(80) NOT NULL DEFAULT '', target_id VARCHAR(160) NOT NULL DEFAULT '', input_sha256 CHAR(64) NOT NULL DEFAULT '',
 reason TEXT NOT NULL, manifest_sha256 CHAR(64) NOT NULL, idempotency_key VARCHAR(128) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(owner_id,idempotency_key),
 CHECK ((decision='selected' AND capability_id<>'' AND command_type<>'' AND target_type<>'' AND target_id<>'' AND input_sha256 ~ '^[0-9a-f]{64}$') OR
        (decision<>'selected' AND capability_id='' AND command_type='' AND target_type='' AND target_id='' AND input_sha256='')));
CREATE OR REPLACE FUNCTION guard_business_decision_immutable() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'business decision records are immutable'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_business_case_immutable BEFORE UPDATE OR DELETE ON business_decision_case FOR EACH ROW EXECUTE FUNCTION guard_business_decision_immutable();
CREATE TRIGGER trg_business_snapshot_immutable BEFORE UPDATE OR DELETE ON business_decision_fact_snapshot FOR EACH ROW EXECUTE FUNCTION guard_business_decision_immutable();
CREATE TRIGGER trg_business_ai_immutable BEFORE UPDATE OR DELETE ON business_ai_recommendation FOR EACH ROW EXECUTE FUNCTION guard_business_decision_immutable();
CREATE TRIGGER trg_business_owner_immutable BEFORE UPDATE OR DELETE ON business_owner_decision FOR EACH ROW EXECUTE FUNCTION guard_business_decision_immutable();
