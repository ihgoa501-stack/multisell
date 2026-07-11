CREATE TABLE IF NOT EXISTS demand_case (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    region VARCHAR(80) NOT NULL,
    consumer VARCHAR(240) NOT NULL,
    need_scenario VARCHAR(400) NOT NULL,
    sales_channel VARCHAR(160) NOT NULL,
    stop_condition TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'lead' CHECK (status IN ('lead', 'evidence_missing', 'rejected', 'experiment_ready')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_demand_case_owner_id ON demand_case(owner_id);

CREATE TABLE IF NOT EXISTS demand_evidence (
    id BIGSERIAL PRIMARY KEY,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE CASCADE,
    dimension VARCHAR(40) NOT NULL CHECK (dimension IN ('demand', 'competition', 'acquisition', 'fulfillment', 'compliance', 'payment', 'aftersales', 'profit_verifiability')),
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('support', 'counter', 'conflict')),
    truth_status VARCHAR(16) NOT NULL CHECK (truth_status IN ('actual', 'quoted', 'estimated', 'unknown', 'mock', 'inferred')),
    title TEXT NOT NULL,
    source_uri TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ,
    run_id VARCHAR(80) NOT NULL,
    fatal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_demand_evidence_case_id ON demand_evidence(demand_case_id);

CREATE TABLE IF NOT EXISTS demand_verdict (
    id BIGSERIAL PRIMARY KEY,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL CHECK (status IN ('lead', 'evidence_missing', 'rejected', 'experiment_ready')),
    blockers_json TEXT NOT NULL DEFAULT '[]',
    reason TEXT NOT NULL DEFAULT '',
    evaluated_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_demand_verdict_case_id ON demand_verdict(demand_case_id);
