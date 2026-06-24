-- Reverse of 000003_data_migration.up.sql
-- This does NOT drop the new tables (those are owned by 000001 + 000002).
-- It only removes the migrated rows so the migration can be re-run cleanly.
-- Use with extreme caution — only during pre-cutover rehearsals.

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
