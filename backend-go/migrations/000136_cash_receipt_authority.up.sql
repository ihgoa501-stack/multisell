ALTER TABLE finance_account ADD COLUMN IF NOT EXISTS owner_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_finance_account_owner ON finance_account(owner_id, id);

CREATE TABLE cash_receipt (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    finance_account_id BIGINT NOT NULL REFERENCES finance_account(id) ON DELETE RESTRICT,
    source_type VARCHAR(16) NOT NULL CHECK (source_type IN ('bank','payment')),
    external_receipt_id VARCHAR(200) NOT NULL,
    idempotency_key VARCHAR(200) NOT NULL,
    request_sha256 CHAR(64) NOT NULL CHECK (request_sha256 ~ '^[0-9a-f]{64}$'),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    observed_at TIMESTAMPTZ NOT NULL,
    value_date DATE,
    raw_payload BYTEA NOT NULL,
    raw_payload_sha256 CHAR(64) NOT NULL CHECK (raw_payload_sha256 ~ '^[0-9a-f]{64}$'),
    truth_status VARCHAR(24) NOT NULL DEFAULT 'external_observed' CHECK (truth_status = 'external_observed'),
    reconciliation_status VARCHAR(16) NOT NULL DEFAULT 'unmatched' CHECK (reconciliation_status IN ('unmatched','partial','reconciled','conflict')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key),
    UNIQUE(owner_id, source_type, finance_account_id, external_receipt_id)
);
CREATE INDEX idx_cash_receipt_owner_created ON cash_receipt(owner_id, created_at DESC);

CREATE TABLE cash_reconciliation (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    cash_receipt_id BIGINT NOT NULL REFERENCES cash_receipt(id) ON DELETE RESTRICT,
    platform_settlement_ingest_id BIGINT NOT NULL REFERENCES platform_settlement_ingest(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(200) NOT NULL,
    request_sha256 CHAR(64) NOT NULL CHECK (request_sha256 ~ '^[0-9a-f]{64}$'),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    expected_receivable_minor BIGINT NOT NULL CHECK (expected_receivable_minor > 0),
    status VARCHAR(16) NOT NULL CHECK (status IN ('partial','reconciled','conflict')),
    conflict_reason TEXT NOT NULL DEFAULT '',
    reconciled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key)
);
CREATE INDEX idx_cash_reconciliation_settlement ON cash_reconciliation(owner_id, platform_settlement_ingest_id, status);
CREATE INDEX idx_cash_reconciliation_receipt ON cash_reconciliation(owner_id, cash_receipt_id, status);

CREATE OR REPLACE FUNCTION guard_cash_receipt_immutable() RETURNS trigger AS $$
BEGIN
    IF ROW(OLD.owner_id, OLD.finance_account_id, OLD.source_type, OLD.external_receipt_id,
           OLD.idempotency_key, OLD.request_sha256, OLD.amount_minor, OLD.currency,
           OLD.observed_at, OLD.value_date, OLD.raw_payload, OLD.raw_payload_sha256, OLD.truth_status)
       IS DISTINCT FROM
       ROW(NEW.owner_id, NEW.finance_account_id, NEW.source_type, NEW.external_receipt_id,
           NEW.idempotency_key, NEW.request_sha256, NEW.amount_minor, NEW.currency,
           NEW.observed_at, NEW.value_date, NEW.raw_payload, NEW.raw_payload_sha256, NEW.truth_status) THEN
        RAISE EXCEPTION 'cash receipt fact fields are immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_cash_receipt_immutable BEFORE UPDATE ON cash_receipt
FOR EACH ROW EXECUTE FUNCTION guard_cash_receipt_immutable();

CREATE OR REPLACE FUNCTION guard_cash_reconciliation_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'cash reconciliation records are immutable';
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_cash_reconciliation_immutable BEFORE UPDATE OR DELETE ON cash_reconciliation
FOR EACH ROW EXECUTE FUNCTION guard_cash_reconciliation_immutable();
