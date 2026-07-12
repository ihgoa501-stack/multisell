CREATE TABLE IF NOT EXISTS image_jobs (
    id text PRIMARY KEY,
    owner_id bigint NOT NULL CHECK (owner_id > 0),
    lingmirror_task_id text NOT NULL DEFAULT '',
    lingmirror_task_version bigint NOT NULL DEFAULT 0 CHECK (lingmirror_task_version >= 0),
    idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 256),
    manifest_hash text NOT NULL CHECK (manifest_hash ~ '^[0-9a-f]{64}$'),
    operation text NOT NULL,
    processor text NOT NULL,
    prompt text NOT NULL DEFAULT '',
    input_blob_id text NOT NULL CHECK (input_blob_id ~ '^[0-9a-f]{64}$'),
    output_blob_id text NOT NULL DEFAULT '',
    width integer NOT NULL CHECK (width BETWEEN 1 AND 4000),
    height integer NOT NULL CHECK (height BETWEEN 1 AND 4000),
    format text NOT NULL CHECK (format IN ('png', 'jpeg')),
    status text NOT NULL CHECK (status IN ('QUEUED', 'RUNNING', 'READY', 'FAILED')),
    error_code text NOT NULL DEFAULT '',
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (owner_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS image_attempts (
    id text PRIMARY KEY,
    job_id text NOT NULL REFERENCES image_jobs(id) ON DELETE RESTRICT,
    idempotency_key text NOT NULL UNIQUE CHECK (length(idempotency_key) BETWEEN 1 AND 256),
    number integer NOT NULL CHECK (number > 0),
    status text NOT NULL CHECK (status IN ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED')),
    lease_owner text NOT NULL DEFAULT '',
    lease_until timestamptz,
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    started_at timestamptz,
    completed_at timestamptz,
    UNIQUE (job_id, number),
    CHECK ((status = 'RUNNING' AND lease_owner <> '' AND lease_until IS NOT NULL)
        OR (status <> 'RUNNING' AND lease_owner = '' AND lease_until IS NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS image_attempts_one_active_per_job
    ON image_attempts(job_id) WHERE status IN ('QUEUED', 'RUNNING');
CREATE INDEX IF NOT EXISTS image_attempts_claim_order
    ON image_attempts(created_at, id) WHERE status IN ('QUEUED', 'RUNNING');

CREATE TABLE IF NOT EXISTS image_consumed_nonces (
    nonce text PRIMARY KEY CHECK (length(nonce) BETWEEN 1 AND 512),
    attempt_id text NOT NULL UNIQUE REFERENCES image_attempts(id) ON DELETE RESTRICT,
    consumed_at timestamptz NOT NULL
);
