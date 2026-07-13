-- The separately gated Photoroom sandbox canary is a zero-dollar external
-- execution. Keep zero forbidden for every other provider at the database
-- boundary while allowing this exact processor contract.
ALTER TABLE product_image_execution_approvals
    DROP CONSTRAINT IF EXISTS ck_product_image_approval_max_cost;
ALTER TABLE product_image_execution_approvals
    ADD CONSTRAINT ck_product_image_approval_max_cost CHECK (
        max_cost ~ '^(0|0\.[0-9]{0,3}[1-9]|[1-9][0-9]{0,9}(\.[0-9]{1,4})?)$'
        AND (max_cost <> '0' OR (
            processor = 'photoroom'
            AND operation IN ('PHOTOROOM_REMOVE_BACKGROUND_SANDBOX', 'PHOTOROOM_WHITE_BACKGROUND_SANDBOX', 'PHOTOROOM_AI_SHADOW_SANDBOX')
        ))
    );

ALTER TABLE product_image_budget_reservations
    DROP CONSTRAINT IF EXISTS product_image_budget_reservations_reserved_amount_check;
ALTER TABLE product_image_budget_reservations
    ADD CONSTRAINT product_image_budget_reservations_reserved_amount_check CHECK (
        reserved_amount > 0 OR (reserved_amount = 0 AND provider = 'photoroom')
    );

CREATE TABLE IF NOT EXISTS product_image_execution_rights_snapshots (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    approval_id BIGINT NOT NULL UNIQUE REFERENCES product_image_execution_approvals(id),
    approval_execution_id VARCHAR(64) NOT NULL UNIQUE,
    task_id BIGINT NOT NULL REFERENCES product_image_tasks(id),
    task_version BIGINT NOT NULL,
    manifest_hash VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    grant_id BIGINT NOT NULL REFERENCES product_image_rights_grants(id),
    grant_version BIGINT NOT NULL,
    asset_sha VARCHAR(64) NOT NULL,
    evidence_sha VARCHAR(64) NOT NULL,
    grant_request_hash VARCHAR(64) NOT NULL,
    can_copy BOOLEAN NOT NULL,
    can_modify BOOLEAN NOT NULL,
    can_third_party_ai BOOLEAN NOT NULL,
    can_cross_border BOOLEAN NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    claimed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_product_image_execution_rights_hashes CHECK (
        manifest_hash ~ '^[0-9a-f]{64}$' AND asset_sha ~ '^[0-9a-f]{64}$'
        AND evidence_sha ~ '^[0-9a-f]{64}$' AND grant_request_hash ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT ck_product_image_execution_rights_permissions CHECK (
        can_copy AND can_modify AND can_third_party_ai AND can_cross_border
    )
);

CREATE INDEX IF NOT EXISTS idx_product_image_execution_rights_owner_task
    ON product_image_execution_rights_snapshots(owner_id, task_id, id);
CREATE INDEX IF NOT EXISTS idx_product_image_execution_rights_grant
    ON product_image_execution_rights_snapshots(grant_id, grant_version, id);

CREATE OR REPLACE FUNCTION reject_product_image_execution_rights_snapshot_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'product image execution rights snapshots are immutable';
END;
$$;

DROP TRIGGER IF EXISTS trg_product_image_execution_rights_snapshot_immutable
    ON product_image_execution_rights_snapshots;
CREATE TRIGGER trg_product_image_execution_rights_snapshot_immutable
BEFORE UPDATE OR DELETE ON product_image_execution_rights_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_product_image_execution_rights_snapshot_mutation();
