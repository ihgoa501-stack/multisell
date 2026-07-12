CREATE TABLE sourcing_sample (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id) ON DELETE RESTRICT,
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT,
    product_opportunity_id BIGINT NOT NULL REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    opportunity_decision_id BIGINT NOT NULL REFERENCES product_opportunity_decision(id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES supplier(id) ON DELETE RESTRICT,
    snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id) ON DELETE RESTRICT,
    supplier_sku VARCHAR(160),
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    status VARCHAR(32) NOT NULL DEFAULT 'request'
        CHECK (status IN ('request','approved_to_order','ordered','received','evaluated','accepted','rejected')),
    order_amount NUMERIC(14,2),
    currency VARCHAR(8),
    external_credential_uri TEXT,
    observed_at TIMESTAMPTZ,
    truth_status VARCHAR(16) NOT NULL DEFAULT 'unknown'
        CHECK (truth_status IN ('actual','quoted','estimated','unknown')),
    evaluation TEXT NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, sourcing_product_id, task_link_id, supplier_id, snapshot_id, supplier_sku)
);

CREATE INDEX idx_sourcing_sample_owner_status ON sourcing_sample(owner_id, status, id DESC);
CREATE INDEX idx_sourcing_sample_source ON sourcing_sample(owner_id, sourcing_product_id, id DESC);

CREATE TABLE sourcing_sample_event (
    id BIGSERIAL PRIMARY KEY,
    sample_id BIGINT NOT NULL REFERENCES sourcing_sample(id) ON DELETE RESTRICT,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    order_amount NUMERIC(14,2),
    currency VARCHAR(8),
    external_credential_uri TEXT,
    observed_at TIMESTAMPTZ,
    truth_status VARCHAR(16) NOT NULL CHECK (truth_status IN ('actual','quoted','estimated','unknown')),
    note TEXT NOT NULL DEFAULT '',
    actor_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sourcing_sample_event_chain ON sourcing_sample_event(owner_id, sample_id, id);

CREATE OR REPLACE FUNCTION reject_sourcing_sample_event_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing_sample_event is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_sample_event_append_only
BEFORE UPDATE OR DELETE ON sourcing_sample_event
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_sample_event_mutation();
