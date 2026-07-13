DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM product_image_budget_charges WHERE kind IN ('no_charge','charged_no_output')) THEN
        RAISE EXCEPTION 'cannot roll back migration 151 while terminal reconciliation evidence exists';
    END IF;
END $$;

DROP TRIGGER IF EXISTS trg_product_image_budget_charges_append_only ON product_image_budget_charges;
DROP FUNCTION IF EXISTS reject_product_image_budget_charge_mutation();

ALTER TABLE product_image_budget_reservations
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_state,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_released_at,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_claimed_at;

ALTER TABLE product_image_budget_reservations
    ADD CONSTRAINT product_image_budget_reservations_state_check
        CHECK (state IN ('reserved','claimed','spent','released')),
    ADD CONSTRAINT product_image_budget_reservations_check
        CHECK ((state='released')=(released_at IS NOT NULL)),
    ADD CONSTRAINT product_image_budget_reservations_check1
        CHECK ((state IN ('claimed','spent'))=(claimed_at IS NOT NULL));

ALTER TABLE product_image_budget_charges
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_charge_no_charge,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_charge_amount,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_charge_kind;

ALTER TABLE product_image_budget_charges
    ADD CONSTRAINT product_image_budget_charges_amount_check CHECK (amount > 0),
    ADD CONSTRAINT product_image_budget_charges_kind_check CHECK (kind IN ('actual','late_fee'));

ALTER TABLE product_image_budget_charges ALTER COLUMN kind TYPE VARCHAR(16);
