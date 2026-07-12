CREATE TABLE IF NOT EXISTS problem_case (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, problem_key VARCHAR(160) NOT NULL, region VARCHAR(80) NOT NULL,
 observable_population VARCHAR(300) NOT NULL, problem_scenario TEXT NOT NULL, current_workaround TEXT NOT NULL,
 responsibility VARCHAR(32) NOT NULL CHECK(responsibility IN ('consumer_controlled','shared','landlord','employer','manufacturer','medical','public_service','unknown')),
 product_solvability VARCHAR(24) NOT NULL CHECK(product_solvability IN ('plausible','partial','structural','unknown')),
 harm_risk VARCHAR(16) NOT NULL CHECK(harm_risk IN ('low','medium','high','unknown')),
 next_minimum_evidence TEXT NOT NULL, status VARCHAR(32) NOT NULL DEFAULT 'lead' CHECK(status IN ('lead','evidence_missing','survives_falsification','rejected')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_problem_case_owner ON problem_case(owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_problem_case_owner_key ON problem_case(owner_id, problem_key);
CREATE TABLE IF NOT EXISTS problem_evidence (
 id BIGSERIAL PRIMARY KEY, problem_case_id BIGINT NOT NULL REFERENCES problem_case(id) ON DELETE CASCADE,
 kind VARCHAR(16) NOT NULL CHECK(kind IN ('support','counter')), title TEXT NOT NULL, source_uri TEXT NOT NULL,
 observed_at TIMESTAMPTZ NOT NULL, collector VARCHAR(120) NOT NULL, raw_sha256 VARCHAR(64) NOT NULL CHECK(length(raw_sha256)=64), raw_payload TEXT NOT NULL, trusted_run BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_problem_evidence_case ON problem_evidence(problem_case_id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_problem_evidence_identity ON problem_evidence(problem_case_id, kind, collector, raw_sha256);
