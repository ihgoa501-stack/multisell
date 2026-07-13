ALTER TABLE product_image_budget_reservations
    DROP CONSTRAINT IF EXISTS product_image_budget_reservations_state_check,
    DROP CONSTRAINT IF EXISTS product_image_budget_reservations_check,
    DROP CONSTRAINT IF EXISTS product_image_budget_reservations_check1,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_state,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_released_at,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_reservation_claimed_at;

ALTER TABLE product_image_budget_reservations
    ADD CONSTRAINT ck_product_image_budget_reservation_state
        CHECK (state IN ('reserved','claimed','spent','released','no_charge')),
    ADD CONSTRAINT ck_product_image_budget_reservation_released_at
        CHECK ((state IN ('released','no_charge')) = (released_at IS NOT NULL)),
    ADD CONSTRAINT ck_product_image_budget_reservation_claimed_at
        CHECK ((state IN ('claimed','spent','no_charge')) = (claimed_at IS NOT NULL));

ALTER TABLE product_image_budget_charges
    DROP CONSTRAINT IF EXISTS product_image_budget_charges_amount_check,
    DROP CONSTRAINT IF EXISTS product_image_budget_charges_kind_check,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_charge_amount,
    DROP CONSTRAINT IF EXISTS ck_product_image_budget_charge_kind;

ALTER TABLE product_image_budget_charges ALTER COLUMN kind TYPE VARCHAR(24);

ALTER TABLE product_image_budget_charges
    ADD CONSTRAINT ck_product_image_budget_charge_amount
        CHECK (amount >= 0),
    ADD CONSTRAINT ck_product_image_budget_charge_kind
        CHECK (kind IN ('actual','late_fee','no_charge','charged_no_output')),
    ADD CONSTRAINT ck_product_image_budget_charge_no_charge
        CHECK ((kind = 'no_charge') = (amount = 0));

CREATE OR REPLACE FUNCTION reject_product_image_budget_charge_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'product image budget charges are append-only';
END;
$$;

CREATE TRIGGER trg_product_image_budget_charges_append_only
BEFORE UPDATE OR DELETE ON product_image_budget_charges
FOR EACH ROW EXECUTE FUNCTION reject_product_image_budget_charge_mutation();
