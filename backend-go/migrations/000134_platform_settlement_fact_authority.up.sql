CREATE TABLE platform_settlement_ingest (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    account_id BIGINT NOT NULL REFERENCES platform_integration_account(id) ON DELETE RESTRICT,
    platform_code VARCHAR(32) NOT NULL,
    external_event_id VARCHAR(200) NOT NULL,
    external_settlement_id VARCHAR(200) NOT NULL,
    truth_status VARCHAR(32) NOT NULL CHECK (truth_status IN ('external_observed', 'mock')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    raw_payload BYTEA NOT NULL,
    payload_sha256 CHAR(64) NOT NULL CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    content_sha256 CHAR(64) NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, account_id, external_event_id)
);

CREATE TABLE platform_settlement_fact_line (
    id BIGSERIAL PRIMARY KEY,
    ingest_id BIGINT NOT NULL REFERENCES platform_settlement_ingest(id) ON DELETE RESTRICT,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    external_line_id VARCHAR(200) NOT NULL,
    external_order_id VARCHAR(200) NOT NULL,
    order_id BIGINT NOT NULL REFERENCES sales_order(id) ON DELETE RESTRICT,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('sale', 'fee', 'refund', 'commission')),
    fee_code VARCHAR(32) NOT NULL DEFAULT '' CHECK (
        (kind IN ('fee', 'commission') AND fee_code IN ('platform_fee', 'payment_fee', 'tax_fee', 'fulfillment_fee', 'advertising_fee', 'other_fee'))
        OR (kind IN ('sale', 'refund') AND fee_code = '')
    ),
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    external_transaction_id VARCHAR(200) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ingest_id, line_number),
    UNIQUE (ingest_id, external_line_id),
    UNIQUE (ingest_id, external_transaction_id)
);

CREATE OR REPLACE FUNCTION validate_platform_settlement_ingest_authority()
RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM owner_platform_account_authority a
        WHERE a.owner_id = NEW.owner_id
          AND a.account_id = NEW.account_id
          AND a.platform_code = NEW.platform_code
    ) THEN
        RAISE EXCEPTION 'platform settlement account is not authoritative for owner';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_platform_settlement_ingest_authority
BEFORE INSERT ON platform_settlement_ingest
FOR EACH ROW EXECUTE FUNCTION validate_platform_settlement_ingest_authority();

CREATE OR REPLACE FUNCTION validate_platform_settlement_fact_line_authority()
RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM platform_settlement_ingest s
        JOIN platform_order_ingest o
          ON o.owner_id = s.owner_id
         AND o.account_id = s.account_id
         AND o.external_order_id = NEW.external_order_id
         AND o.normalized_order_id = NEW.order_id
         AND o.truth_status = s.truth_status
         AND o.event_action = 'reserve'
         AND o.processing_status = 'applied'
        WHERE s.id = NEW.ingest_id
          AND s.currency = NEW.currency
    ) THEN
        RAISE EXCEPTION 'settlement line is not bound to the same Owner/account/order/currency/truth authority';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_platform_settlement_fact_line_authority
BEFORE INSERT ON platform_settlement_fact_line
FOR EACH ROW EXECUTE FUNCTION validate_platform_settlement_fact_line_authority();

CREATE OR REPLACE FUNCTION prevent_platform_settlement_fact_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'platform settlement facts are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_platform_settlement_ingest_immutable
BEFORE UPDATE OR DELETE ON platform_settlement_ingest
FOR EACH ROW EXECUTE FUNCTION prevent_platform_settlement_fact_mutation();

CREATE TRIGGER trg_platform_settlement_fact_line_immutable
BEFORE UPDATE OR DELETE ON platform_settlement_fact_line
FOR EACH ROW EXECUTE FUNCTION prevent_platform_settlement_fact_mutation();
