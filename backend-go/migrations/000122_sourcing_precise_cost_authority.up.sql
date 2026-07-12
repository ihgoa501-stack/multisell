CREATE TABLE sourcing_cost_version (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id) ON DELETE RESTRICT,
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT,
    product_opportunity_id BIGINT NOT NULL REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    opportunity_decision_id BIGINT NOT NULL REFERENCES product_opportunity_decision(id) ON DELETE RESTRICT,
    source_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id) ON DELETE RESTRICT,
    sku_mapping_id BIGINT NOT NULL REFERENCES sourcing_sku_mapping(id) ON DELETE RESTRICT,
    version BIGINT NOT NULL CHECK (version > 0),
    target_currency VARCHAR(8) NOT NULL CHECK (target_currency ~ '^[A-Z]{3,8}$'),
    total_minor BIGINT NOT NULL CHECK (total_minor >= 0),
    content_hash CHAR(64) NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    created_by BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, task_link_id, sku_mapping_id, version),
    UNIQUE (content_hash)
);

CREATE TABLE sourcing_cost_line (
    id BIGSERIAL PRIMARY KEY,
    cost_version_id BIGINT NOT NULL REFERENCES sourcing_cost_version(id) ON DELETE RESTRICT,
    cost_type VARCHAR(64) NOT NULL CHECK (cost_type IN ('purchase','domestic_shipping','packaging','cross_border_shipping','platform_fee','payment_fee','advertising','tax','duty','return_loss')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency VARCHAR(8) NOT NULL CHECK (currency ~ '^[A-Z]{3,8}$'),
    normalized_amount_minor BIGINT NOT NULL CHECK (normalized_amount_minor >= 0),
    exchange_rate_decimal VARCHAR(80),
    exchange_rate_source_uri TEXT,
    exchange_rate_observed_at TIMESTAMPTZ,
    truth_status VARCHAR(16) NOT NULL CHECK (truth_status IN ('actual','quoted','estimated')),
    source_uri TEXT NOT NULL CHECK (BTRIM(source_uri) <> ''),
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cost_version_id, cost_type),
    CHECK ((exchange_rate_decimal IS NULL AND exchange_rate_source_uri IS NULL AND exchange_rate_observed_at IS NULL) OR
           (exchange_rate_decimal IS NOT NULL AND exchange_rate_source_uri IS NOT NULL AND BTRIM(exchange_rate_source_uri) <> '' AND exchange_rate_observed_at IS NOT NULL))
);

CREATE INDEX idx_sourcing_cost_version_source ON sourcing_cost_version(owner_id, sourcing_product_id, task_link_id);
CREATE INDEX idx_sourcing_cost_line_version ON sourcing_cost_line(cost_version_id);

CREATE OR REPLACE FUNCTION validate_sourcing_cost_line_currency() RETURNS trigger AS $$
DECLARE target VARCHAR(8);
BEGIN
    SELECT target_currency INTO target FROM sourcing_cost_version WHERE id = NEW.cost_version_id;
    IF NEW.currency = target AND (NEW.exchange_rate_decimal IS NOT NULL OR NEW.exchange_rate_source_uri IS NOT NULL OR NEW.exchange_rate_observed_at IS NOT NULL OR NEW.amount_minor <> NEW.normalized_amount_minor) THEN
        RAISE EXCEPTION 'same-currency cost line must preserve exact minor units without exchange rate';
    END IF;
    IF NEW.currency <> target AND (NEW.exchange_rate_decimal IS NULL OR NEW.exchange_rate_source_uri IS NULL OR NEW.exchange_rate_observed_at IS NULL) THEN
        RAISE EXCEPTION 'cross-currency cost line requires exchange-rate evidence';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_cost_line_currency
BEFORE INSERT ON sourcing_cost_line
FOR EACH ROW EXECUTE FUNCTION validate_sourcing_cost_line_currency();

CREATE OR REPLACE FUNCTION reject_sourcing_cost_authority_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing cost authority is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_cost_version_immutable
BEFORE UPDATE OR DELETE ON sourcing_cost_version
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_cost_authority_mutation();

CREATE TRIGGER trg_sourcing_cost_line_immutable
BEFORE UPDATE OR DELETE ON sourcing_cost_line
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_cost_authority_mutation();

CREATE OR REPLACE FUNCTION validate_sourcing_cost_version_complete() RETURNS trigger AS $$
DECLARE
    line_count INTEGER;
    line_total BIGINT;
BEGIN
    SELECT COUNT(*), COALESCE(SUM(normalized_amount_minor), 0)
      INTO line_count, line_total
      FROM sourcing_cost_line WHERE cost_version_id = NEW.id;
    IF line_count <> 10 OR line_total <> NEW.total_minor THEN
        RAISE EXCEPTION 'sourcing cost version must contain exactly 10 lines matching total_minor';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_sourcing_cost_version_complete
AFTER INSERT ON sourcing_cost_version
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_sourcing_cost_version_complete();
