-- Durable, Owner-isolated receipt for extension collection reconciliation.
-- It stores only request identity and safe outcome metadata, never page data.
CREATE TABLE sourcing_1688_private_collection_request (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    request_id VARCHAR(80) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('receiving','saved','not_saved','reconcile_required')),
    request_envelope_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    failure_code VARCHAR(80) NOT NULL DEFAULT '',
    safe_message VARCHAR(500) NOT NULL DEFAULT '',
    record_id BIGINT REFERENCES sourcing_1688_product(id),
    snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT ux_private_collection_request UNIQUE (owner_id, request_id),
    CONSTRAINT ck_private_collection_request_saved_refs CHECK (
        status <> 'saved' OR (record_id IS NOT NULL AND snapshot_id IS NOT NULL)
    ),
    CONSTRAINT ck_private_collection_request_failure_safe CHECK (
        status NOT IN ('not_saved','reconcile_required') OR (failure_code <> '' AND safe_message <> '')
    )
);

CREATE INDEX idx_private_collection_request_owner_time
    ON sourcing_1688_private_collection_request(owner_id, updated_at DESC, id DESC);

COMMENT ON TABLE sourcing_1688_private_collection_request IS
    'Owner-isolated reconciliation receipt for one extension request; contains no page payload, HTML, or credentials.';
