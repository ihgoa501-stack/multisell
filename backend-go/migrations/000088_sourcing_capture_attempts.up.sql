CREATE TABLE IF NOT EXISTS sourcing_1688_capture_attempt (
    id BIGSERIAL PRIMARY KEY,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id),
    experiment_id VARCHAR(40) NOT NULL REFERENCES experiment_case(experiment_id),
    source_url TEXT NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL,
    driver VARCHAR(40) NOT NULL,
    parser_version VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_code VARCHAR(80) NOT NULL,
    error_message TEXT NOT NULL,
    attempted_by BIGINT NOT NULL REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_sourcing_capture_attempt_status CHECK (status IN ('capture_failed','retry_scheduled','resolved'))
);
CREATE INDEX IF NOT EXISTS idx_sourcing_capture_attempt_workflow ON sourcing_1688_capture_attempt(demand_case_id, experiment_id, status);
