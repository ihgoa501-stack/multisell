DROP TRIGGER IF EXISTS trg_product_image_execution_rights_snapshot_immutable
    ON product_image_execution_rights_snapshots;
DROP FUNCTION IF EXISTS reject_product_image_execution_rights_snapshot_mutation();
DROP TABLE IF EXISTS product_image_execution_rights_snapshots;

ALTER TABLE product_image_budget_reservations
    DROP CONSTRAINT IF EXISTS product_image_budget_reservations_reserved_amount_check;
ALTER TABLE product_image_budget_reservations
    ADD CONSTRAINT product_image_budget_reservations_reserved_amount_check CHECK (reserved_amount > 0);

ALTER TABLE product_image_execution_approvals
    DROP CONSTRAINT IF EXISTS ck_product_image_approval_max_cost;
ALTER TABLE product_image_execution_approvals
    ADD CONSTRAINT ck_product_image_approval_max_cost CHECK (
        max_cost ~ '^(0\.[0-9]{0,3}[1-9]|[1-9][0-9]{0,9}(\.[0-9]{1,4})?)$'
    );
