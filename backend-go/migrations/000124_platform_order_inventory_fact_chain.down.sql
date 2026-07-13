DROP TRIGGER IF EXISTS trg_platform_order_ingest_identity_immutable ON platform_order_ingest;
DROP FUNCTION IF EXISTS protect_platform_order_ingest_identity();
DROP TRIGGER IF EXISTS trg_order_inventory_ledger_immutable ON order_inventory_ledger;
DROP TRIGGER IF EXISTS trg_platform_order_ingest_item_immutable ON platform_order_ingest_item;
DROP FUNCTION IF EXISTS prevent_platform_order_fact_mutation();
DROP TABLE IF EXISTS order_inventory_ledger;
DROP TABLE IF EXISTS platform_order_ingest_item;
DROP TABLE IF EXISTS platform_order_ingest;
DROP TABLE IF EXISTS owner_platform_account_authority;
