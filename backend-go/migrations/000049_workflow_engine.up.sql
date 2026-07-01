-- Generic Workflow Engine
-- Defines, runs, and tracks multi-step agent pipelines.

CREATE TABLE IF NOT EXISTS workflow_def (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_run (
    id BIGSERIAL PRIMARY KEY,
    workflow_def_id BIGINT REFERENCES workflow_def(id),
    name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    context JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_run_status ON workflow_run(status);

CREATE TABLE IF NOT EXISTS workflow_step_run (
    id BIGSERIAL PRIMARY KEY,
    workflow_run_id BIGINT NOT NULL REFERENCES workflow_run(id) ON DELETE CASCADE,
    step_name VARCHAR(200) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    parent_id BIGINT REFERENCES workflow_step_run(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT,
    attempt INT NOT NULL DEFAULT 1,
    max_attempts INT NOT NULL DEFAULT 1,
    timeout_seconds INT NOT NULL DEFAULT 300,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_step_run_run_id ON workflow_step_run(workflow_run_id);
CREATE INDEX IF NOT EXISTS idx_workflow_step_run_status ON workflow_step_run(status);
