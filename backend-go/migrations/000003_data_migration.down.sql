-- Reverse of 000003_data_migration.up.sql
-- ⚠️ DANGER: TRUNCATE CASCADE destroys ALL migrated data in 37 tables.
-- Only safe during pre-cutover rehearsals when no production data exists.
-- Guard: refuse if the migrate tool's schema_migrations table shows
-- any migration > 000003 has been applied (means app is running).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM schema_migrations WHERE version > '000003') THEN
        RAISE EXCEPTION 'Refusing to roll back 000003: later migrations have been applied (version > 000003). This TRUNCATE would destroy production data.';
    END IF;
END;
$$;

BEGIN;

TRUNCATE TABLE
    product_image_gen,
    cost_allocation_batch,
    allocation_rule,
    platform_integration_account,
    operation_log,
    exception_item,
    alert_rule,
    notification,
    order_import,
    import_batch,
    after_sales_order,
    platform_fee_rule,
    finance_ledger_entry,
    finance_transaction,
    finance_account,
    settlement_item,
    settlement,
    listing_task,
    product_listing,
    sales_order_shipping_snapshot,
    sales_order_status_log,
    sales_order_item,
    sales_order,
    shipping_quote_rule,
    shipping_zone,
    shipping_channel,
    shipping_provider,
    sourcing_1688_product,
    supplier,
    inventory,
    price,
    sku,
    spec_value,
    spec_name,
    product,
    warehouse,
    "user",
    platform,
    brand,
    category
RESTART IDENTITY CASCADE;

COMMIT;
