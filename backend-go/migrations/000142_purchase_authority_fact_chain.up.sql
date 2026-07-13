CREATE TABLE purchase_authority (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES supplier(id) ON DELETE RESTRICT,
    sku_mapping_id BIGINT NOT NULL REFERENCES sourcing_sku_mapping(id) ON DELETE RESTRICT,
    internal_sku_id BIGINT NOT NULL REFERENCES sku(id) ON DELETE RESTRICT,
    cost_version_id BIGINT NOT NULL REFERENCES sourcing_cost_version(id) ON DELETE RESTRICT,
    inventory_id BIGINT NOT NULL REFERENCES inventory(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_amount_minor BIGINT NOT NULL CHECK (unit_amount_minor >= 0),
    total_amount_minor BIGINT NOT NULL CHECK (total_amount_minor >= 0),
    currency VARCHAR(8) NOT NULL CHECK (currency ~ '^[A-Z]{3,8}$'),
    status VARCHAR(32) NOT NULL CHECK (status IN ('requested','owner_approved','external_submitted','ordered','failed','partially_received','fully_received')),
    request_sha256 CHAR(64) NOT NULL CHECK (request_sha256 ~ '^[0-9a-f]{64}$'),
    idempotency_key VARCHAR(160) NOT NULL,
    owner_decision_id BIGINT REFERENCES business_owner_decision(id) ON DELETE RESTRICT,
    approved_at TIMESTAMPTZ,
    external_submitted_at TIMESTAMPTZ,
    external_ordered_at TIMESTAMPTZ,
    external_failed_at TIMESTAMPTZ,
    received_quantity INTEGER NOT NULL DEFAULT 0 CHECK (received_quantity >= 0 AND received_quantity <= quantity),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, idempotency_key),
    UNIQUE(id, owner_id),
    CHECK (total_amount_minor = unit_amount_minor * quantity)
);

CREATE TABLE purchase_external_fact (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    purchase_id BIGINT NOT NULL,
    event_type VARCHAR(24) NOT NULL CHECK (event_type IN ('submitted','ordered','failed','received')),
    external_event_id VARCHAR(200) NOT NULL,
    external_order_id VARCHAR(200) NOT NULL CHECK (BTRIM(external_order_id) <> ''),
    received_quantity INTEGER NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
    truth_status VARCHAR(24) NOT NULL CHECK (truth_status = 'external_observed'),
    raw_payload JSONB NOT NULL,
    payload_sha256 CHAR(64) NOT NULL CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (purchase_id, owner_id) REFERENCES purchase_authority(id, owner_id) ON DELETE RESTRICT,
    UNIQUE(owner_id, external_event_id),
    CHECK ((event_type = 'received' AND received_quantity > 0) OR (event_type <> 'received' AND received_quantity = 0))
);

CREATE TABLE purchase_inventory_receipt_ledger (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    purchase_id BIGINT NOT NULL,
    external_fact_id BIGINT NOT NULL REFERENCES purchase_external_fact(id) ON DELETE RESTRICT,
    inventory_id BIGINT NOT NULL REFERENCES inventory(id) ON DELETE RESTRICT,
    sku_id BIGINT NOT NULL REFERENCES sku(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    before_quantity INTEGER NOT NULL CHECK (before_quantity >= 0),
    after_quantity INTEGER NOT NULL CHECK (after_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (purchase_id, owner_id) REFERENCES purchase_authority(id, owner_id) ON DELETE RESTRICT,
    UNIQUE(external_fact_id),
    CHECK (after_quantity = before_quantity + quantity)
);

CREATE INDEX idx_purchase_authority_owner_status ON purchase_authority(owner_id, status, id DESC);
CREATE INDEX idx_purchase_external_fact_purchase ON purchase_external_fact(owner_id, purchase_id, id);
CREATE INDEX idx_purchase_receipt_ledger_purchase ON purchase_inventory_receipt_ledger(owner_id, purchase_id, id);

CREATE OR REPLACE FUNCTION protect_purchase_authority_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.owner_id <> OLD.owner_id OR NEW.supplier_id <> OLD.supplier_id
       OR NEW.sku_mapping_id <> OLD.sku_mapping_id OR NEW.internal_sku_id <> OLD.internal_sku_id
       OR NEW.cost_version_id <> OLD.cost_version_id OR NEW.inventory_id <> OLD.inventory_id
       OR NEW.quantity <> OLD.quantity OR NEW.unit_amount_minor <> OLD.unit_amount_minor
       OR NEW.total_amount_minor <> OLD.total_amount_minor OR NEW.currency <> OLD.currency
       OR NEW.request_sha256 <> OLD.request_sha256 OR NEW.idempotency_key <> OLD.idempotency_key
       OR NEW.created_at <> OLD.created_at THEN
        RAISE EXCEPTION 'purchase authority identity is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_purchase_authority_identity_immutable BEFORE UPDATE ON purchase_authority FOR EACH ROW EXECUTE FUNCTION protect_purchase_authority_identity();

CREATE OR REPLACE FUNCTION reject_purchase_fact_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'purchase external facts and receipt ledger are immutable';
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_purchase_external_fact_immutable BEFORE UPDATE OR DELETE ON purchase_external_fact FOR EACH ROW EXECUTE FUNCTION reject_purchase_fact_mutation();
CREATE TRIGGER trg_purchase_receipt_ledger_immutable BEFORE UPDATE OR DELETE ON purchase_inventory_receipt_ledger FOR EACH ROW EXECUTE FUNCTION reject_purchase_fact_mutation();
CREATE TRIGGER trg_purchase_authority_no_delete BEFORE DELETE ON purchase_authority FOR EACH ROW EXECUTE FUNCTION reject_purchase_fact_mutation();

CREATE OR REPLACE FUNCTION validate_purchase_authority_transition() RETURNS trigger AS $$
DECLARE fact_count BIGINT; fact_quantity BIGINT; ledger_quantity BIGINT; decision_ok BOOLEAN;
BEGIN
    IF NEW.status = OLD.status THEN RETURN NEW; END IF;
    IF OLD.status = 'requested' AND NEW.status = 'owner_approved' THEN
        SELECT EXISTS(SELECT 1 FROM business_owner_decision d WHERE d.id=NEW.owner_decision_id AND d.owner_id=NEW.owner_id
          AND d.decision='selected' AND d.capability_id='purchase.authority.execute' AND d.command_type='purchase.submit'
          AND d.target_type='purchase_authority' AND d.target_id=NEW.id::text AND d.input_sha256=NEW.request_sha256) INTO decision_ok;
        IF NOT decision_ok OR NEW.approved_at IS NULL THEN RAISE EXCEPTION 'exact selected Owner decision required'; END IF;
    ELSIF OLD.status = 'owner_approved' AND NEW.status = 'external_submitted' THEN
        SELECT COUNT(*) INTO fact_count FROM purchase_external_fact WHERE purchase_id=NEW.id AND owner_id=NEW.owner_id AND event_type='submitted';
        IF fact_count <> 1 OR NEW.external_submitted_at IS NULL THEN RAISE EXCEPTION 'external submission fact required'; END IF;
    ELSIF OLD.status = 'external_submitted' AND NEW.status IN ('ordered','failed') THEN
        SELECT COUNT(*) INTO fact_count FROM purchase_external_fact WHERE purchase_id=NEW.id AND owner_id=NEW.owner_id AND event_type=NEW.status;
        IF fact_count <> 1 THEN RAISE EXCEPTION 'external terminal order receipt required'; END IF;
    ELSIF OLD.status IN ('ordered','partially_received') AND NEW.status IN ('partially_received','fully_received') THEN
        SELECT COALESCE(SUM(received_quantity),0) INTO fact_quantity FROM purchase_external_fact WHERE purchase_id=NEW.id AND owner_id=NEW.owner_id AND event_type='received';
        SELECT COALESCE(SUM(quantity),0) INTO ledger_quantity FROM purchase_inventory_receipt_ledger WHERE purchase_id=NEW.id AND owner_id=NEW.owner_id;
        IF NEW.received_quantity <> fact_quantity OR NEW.received_quantity <> ledger_quantity
           OR (NEW.status='fully_received' AND NEW.received_quantity<>NEW.quantity)
           OR (NEW.status='partially_received' AND (NEW.received_quantity<=0 OR NEW.received_quantity>=NEW.quantity)) THEN
            RAISE EXCEPTION 'receiving status requires matching external facts and inventory ledger';
        END IF;
    ELSE
        RAISE EXCEPTION 'invalid purchase authority transition % -> %', OLD.status, NEW.status;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_purchase_authority_transition BEFORE UPDATE ON purchase_authority FOR EACH ROW EXECUTE FUNCTION validate_purchase_authority_transition();

CREATE OR REPLACE FUNCTION validate_purchase_authority_links() RETURNS trigger AS $$
DECLARE mapping_owner BIGINT; mapping_sku BIGINT; cost_owner BIGINT; cost_mapping BIGINT; inventory_sku BIGINT; supplier_owner BIGINT;
BEGIN
    SELECT owner_id INTO supplier_owner FROM supplier WHERE id=NEW.supplier_id;
    SELECT owner_id, internal_sku_id INTO mapping_owner, mapping_sku FROM sourcing_sku_mapping WHERE id=NEW.sku_mapping_id;
    SELECT owner_id, sku_mapping_id INTO cost_owner, cost_mapping FROM sourcing_cost_version WHERE id=NEW.cost_version_id;
    SELECT sku_id INTO inventory_sku FROM inventory WHERE id=NEW.inventory_id;
    IF supplier_owner <> NEW.owner_id OR mapping_owner <> NEW.owner_id OR cost_owner <> NEW.owner_id
       OR mapping_sku <> NEW.internal_sku_id OR cost_mapping <> NEW.sku_mapping_id OR inventory_sku <> NEW.internal_sku_id THEN
        RAISE EXCEPTION 'purchase authority cross-object identity mismatch';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_purchase_authority_links BEFORE INSERT ON purchase_authority FOR EACH ROW EXECUTE FUNCTION validate_purchase_authority_links();
