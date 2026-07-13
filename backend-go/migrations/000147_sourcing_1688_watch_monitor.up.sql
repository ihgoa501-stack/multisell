CREATE TABLE sourcing_1688_watch_subscription (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disabled_at TIMESTAMPTZ,
    CONSTRAINT ux_sourcing_watch_owner_source UNIQUE (owner_id, sourcing_product_id)
);

CREATE TABLE sourcing_1688_watch_refresh_run (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    request_id VARCHAR(80) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending_browser', 'evaluated', 'failed')),
    previous_snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    current_snapshot_id BIGINT REFERENCES sourcing_1688_snapshot(id),
    alert_count INTEGER NOT NULL DEFAULT 0 CHECK (alert_count >= 0),
    failure_code VARCHAR(60) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT ux_sourcing_watch_run_request UNIQUE (owner_id, request_id)
);

CREATE INDEX idx_sourcing_watch_run_source ON sourcing_1688_watch_refresh_run(owner_id, sourcing_product_id, created_at DESC);

CREATE TABLE sourcing_1688_watch_alert (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    refresh_run_id BIGINT NOT NULL REFERENCES sourcing_1688_watch_refresh_run(id),
    previous_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    current_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    change_type VARCHAR(40) NOT NULL CHECK (change_type IN ('price', 'moq', 'supplier', 'sku_set', 'quoted_inventory', 'offer_state')),
    before_value JSONB NOT NULL,
    after_value JSONB NOT NULL,
    content_hash VARCHAR(64) NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_sourcing_watch_alert_run_type UNIQUE (refresh_run_id, change_type)
);

CREATE INDEX idx_sourcing_watch_alert_owner_source ON sourcing_1688_watch_alert(owner_id, sourcing_product_id, id DESC);

CREATE FUNCTION prevent_sourcing_watch_alert_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing watch alerts are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sourcing_watch_alert_no_update
BEFORE UPDATE ON sourcing_1688_watch_alert
FOR EACH ROW EXECUTE FUNCTION prevent_sourcing_watch_alert_mutation();

CREATE TRIGGER sourcing_watch_alert_no_delete
BEFORE DELETE ON sourcing_1688_watch_alert
FOR EACH ROW EXECUTE FUNCTION prevent_sourcing_watch_alert_mutation();
