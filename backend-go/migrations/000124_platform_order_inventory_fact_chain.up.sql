CREATE TABLE platform_order_ingest (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL REFERENCES platform_integration_account(id),
    platform_code VARCHAR(32) NOT NULL,
    external_event_id VARCHAR(200) NOT NULL,
    external_order_id VARCHAR(200) NOT NULL,
    event_action VARCHAR(16) NOT NULL CHECK (event_action IN ('reserve', 'commit', 'release')),
    truth_status VARCHAR(32) NOT NULL CHECK (truth_status IN ('external_observed', 'mock')),
    raw_payload JSONB NOT NULL,
    payload_sha256 CHAR(64) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    normalized_order_id BIGINT REFERENCES sales_order(id),
    processing_status VARCHAR(24) NOT NULL DEFAULT 'received' CHECK (processing_status IN ('received', 'applied', 'failed')),
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, account_id, external_event_id)
);

CREATE TABLE owner_platform_account_authority (
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    account_id BIGINT NOT NULL REFERENCES platform_integration_account(id),
    platform_code VARCHAR(32) NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    verified_by BIGINT NOT NULL REFERENCES "user"(id),
    PRIMARY KEY (owner_id, account_id)
);

CREATE UNIQUE INDEX uq_platform_order_identity
    ON platform_order_ingest(owner_id, account_id, external_order_id)
    WHERE event_action = 'reserve';

CREATE TABLE platform_order_ingest_item (
    id BIGSERIAL PRIMARY KEY,
    ingest_id BIGINT NOT NULL REFERENCES platform_order_ingest(id),
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    external_sku_code VARCHAR(100) NOT NULL,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ingest_id, line_number),
    UNIQUE (ingest_id, external_sku_code)
);

CREATE TABLE order_inventory_ledger (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    ingest_id BIGINT NOT NULL REFERENCES platform_order_ingest(id),
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    order_item_id BIGINT NOT NULL REFERENCES sales_order_item(id),
    inventory_id BIGINT NOT NULL REFERENCES inventory(id),
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    action VARCHAR(16) NOT NULL CHECK (action IN ('reserve', 'commit', 'release')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    before_quantity INTEGER NOT NULL,
    after_quantity INTEGER NOT NULL,
    before_locked_quantity INTEGER NOT NULL,
    after_locked_quantity INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ingest_id, order_item_id, action)
);

CREATE OR REPLACE FUNCTION prevent_platform_order_fact_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'platform order facts are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_platform_order_ingest_item_immutable
BEFORE UPDATE OR DELETE ON platform_order_ingest_item
FOR EACH ROW EXECUTE FUNCTION prevent_platform_order_fact_mutation();

CREATE TRIGGER trg_order_inventory_ledger_immutable
BEFORE UPDATE OR DELETE ON order_inventory_ledger
FOR EACH ROW EXECUTE FUNCTION prevent_platform_order_fact_mutation();

CREATE OR REPLACE FUNCTION protect_platform_order_ingest_identity()
RETURNS trigger AS $$
BEGIN
    IF NEW.owner_id <> OLD.owner_id
       OR NEW.account_id <> OLD.account_id
       OR NEW.platform_code <> OLD.platform_code
       OR NEW.external_event_id <> OLD.external_event_id
       OR NEW.external_order_id <> OLD.external_order_id
       OR NEW.event_action <> OLD.event_action
       OR NEW.truth_status <> OLD.truth_status
       OR NEW.raw_payload <> OLD.raw_payload
       OR NEW.payload_sha256 <> OLD.payload_sha256
       OR NEW.observed_at <> OLD.observed_at THEN
        RAISE EXCEPTION 'platform order ingest identity is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_platform_order_ingest_identity_immutable
BEFORE UPDATE ON platform_order_ingest
FOR EACH ROW EXECUTE FUNCTION protect_platform_order_ingest_identity();
