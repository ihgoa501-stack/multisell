CREATE TABLE product_image_budget_policies (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, currency VARCHAR(3) NOT NULL,
 period_start TIMESTAMPTZ NOT NULL, period_end TIMESTAMPTZ NOT NULL,
 total_amount NUMERIC(20,4) NOT NULL CHECK (total_amount > 0),
 idempotency_key VARCHAR(100) NOT NULL, request_hash VARCHAR(64) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(owner_id,idempotency_key), CHECK(period_end > period_start),
 CHECK(currency IN ('USD','EUR','CNY','GBP','JPY'))
);
CREATE INDEX idx_product_image_budget_policy_lookup ON product_image_budget_policies(owner_id,currency,period_start,period_end);

CREATE TABLE product_image_budget_reservations (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL,
 policy_id BIGINT NOT NULL REFERENCES product_image_budget_policies(id),
 approval_id BIGINT NOT NULL UNIQUE REFERENCES product_image_execution_approvals(id),
 task_id BIGINT NOT NULL REFERENCES product_image_tasks(id), task_version BIGINT NOT NULL,
 manifest_hash VARCHAR(64) NOT NULL, provider VARCHAR(64) NOT NULL, currency VARCHAR(3) NOT NULL,
 reserved_amount NUMERIC(20,4) NOT NULL CHECK(reserved_amount > 0),
 state VARCHAR(16) NOT NULL CHECK(state IN ('reserved','claimed','spent','released')),
 claimed_at TIMESTAMPTZ, released_at TIMESTAMPTZ, release_reason TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(owner_id,task_id,task_version,manifest_hash,provider),
 CHECK((state='released')=(released_at IS NOT NULL)),
 CHECK((state IN ('claimed','spent'))=(claimed_at IS NOT NULL))
);
CREATE INDEX idx_product_image_budget_reservation_total ON product_image_budget_reservations(policy_id,state);

CREATE TABLE product_image_budget_charges (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL,
 reservation_id BIGINT NOT NULL REFERENCES product_image_budget_reservations(id),
 amount NUMERIC(20,4) NOT NULL CHECK(amount > 0), delta_amount NUMERIC(20,4) NOT NULL, currency VARCHAR(3) NOT NULL,
 kind VARCHAR(16) NOT NULL CHECK(kind IN ('actual','late_fee')), over_budget BOOLEAN NOT NULL,
 evidence_sha VARCHAR(64) NOT NULL CHECK(evidence_sha ~ '^[0-9a-f]{64}$'),
 observed_at TIMESTAMPTZ NOT NULL, idempotency_key VARCHAR(100) NOT NULL,
 request_hash VARCHAR(64) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(owner_id,idempotency_key)
);
CREATE INDEX idx_product_image_budget_charge_reservation ON product_image_budget_charges(reservation_id,created_at);
