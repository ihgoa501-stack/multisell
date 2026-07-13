-- Safe failure telemetry for Owner-triggered private 1688 collection.
-- This table intentionally has no raw payload, HTML, arbitrary error text or
-- credential columns.
CREATE TABLE sourcing_1688_private_capture_failure (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    request_id VARCHAR(80) NOT NULL,
    source_url TEXT NOT NULL,
    error_code VARCHAR(80) NOT NULL CHECK (error_code IN (
        'invalid_source_url', 'title_parse_failed', 'sku_parse_failed',
        'invalid_payload', 'network_error'
    )),
    safe_message TEXT NOT NULL,
    schema_version VARCHAR(40) NOT NULL,
    extension_version VARCHAR(40) NOT NULL,
    parser_version VARCHAR(40) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ux_private_capture_failure UNIQUE (owner_id, request_id, error_code)
);

CREATE INDEX idx_private_capture_failure_owner_time
    ON sourcing_1688_private_capture_failure(owner_id, occurred_at DESC, id DESC);

COMMENT ON TABLE sourcing_1688_private_capture_failure IS
    'Owner-isolated safe operational failures for private 1688 collection; never stores page source or credentials.';
