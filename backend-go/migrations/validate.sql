-- ============================================================
-- LingMirror data migration validation
-- Run after 000003_data_migration.up.sql to verify row counts
-- and sample checksums match between legacy and new tables.
--
-- Exit code strategy: this script uses \set ON_ERROR_STOP off and
-- raises NOTICEs for each mismatch. A shell wrapper can grep for
-- "MISMATCH" to determine pass/fail.
--
-- Usage: psql -d multisell_new -f validate.sql -v ON_ERROR_STOP=0
-- ============================================================

\set ON_ERROR_STOP off

\echo '============================================================'
\echo 'LingMirror data migration validation'
\echo '============================================================'
\echo ''

-- Helper: compare row counts for one table pair.
-- We expect legacy_<table> (source) and <table> (target) to have equal counts.
CREATE TEMP TABLE IF NOT EXISTS _parity_results (
    table_name  TEXT,
    legacy_count BIGINT,
    new_count   BIGINT,
    status      TEXT
);

DO $$
DECLARE
    pairs text[] := ARRAY[
        'category','brand','platform','user','warehouse',
        'product','spec_name','spec_value','sku','price','inventory',
        'supplier','sourcing_1688_product',
        'shipping_provider','shipping_channel','shipping_zone','shipping_quote_rule',
        'sales_order','sales_order_item','sales_order_status_log','sales_order_shipping_snapshot',
        'product_listing','listing_task',
        'settlement','settlement_item',
        'finance_account','finance_transaction','finance_ledger_entry',
        'platform_fee_rule','after_sales_order',
        'import_batch','order_import',
        'notification','alert_rule','exception_item','operation_log',
        'platform_integration_account',
        'allocation_rule','cost_allocation_batch',
        'product_image_gen'
    ];
    t text;
    legacy_n bigint;
    new_n bigint;
    status text;
BEGIN
    FOREACH t IN ARRAY pairs LOOP
        EXECUTE format('SELECT COUNT(*) FROM %I', 'legacy_' || t) INTO legacy_n;
        EXECUTE format('SELECT COUNT(*) FROM %I', t) INTO new_n;
        IF legacy_n = new_n THEN
            status := 'OK';
        ELSE
            status := 'MISMATCH';
        END IF;
        INSERT INTO _parity_results VALUES (t, legacy_n, new_n, status);
    END LOOP;
END $$;

\echo '--- Row count parity ---'
SELECT table_name, legacy_count, new_count, status
FROM _parity_results
ORDER BY table_name;

\echo ''
\echo '--- Mismatches (if any) ---'
SELECT table_name, legacy_count, new_count
FROM _parity_results
WHERE status = 'MISMATCH';

\echo ''
\echo '--- Sample checksum (top 100 IDs per critical table) ---'
\echo ''

-- Checksum critical tables by hashing their id list.
DO $$
DECLARE
    checks text[] := ARRAY['sku','sales_order','settlement','finance_ledger_entry','product_listing'];
    t text;
    legacy_h text;
    new_h text;
BEGIN
    FOREACH t IN ARRAY checks LOOP
        EXECUTE format('SELECT md5(string_agg(id::text, '','' ORDER BY id)) FROM (SELECT id FROM %I ORDER BY id LIMIT 100) s', 'legacy_' || t) INTO legacy_h;
        EXECUTE format('SELECT md5(string_agg(id::text, '','' ORDER BY id)) FROM (SELECT id FROM %I ORDER BY id LIMIT 100) s', t) INTO new_h;
        IF (legacy_h IS NULL AND new_h IS NULL) OR legacy_h = new_h THEN
            RAISE NOTICE 'CHECKSUM OK: %', t;
        ELSE
            RAISE NOTICE 'CHECKSUM MISMATCH: % (legacy=%, new=%)', t, legacy_h, new_h;
        END IF;
    END LOOP;
END $$;

\echo ''
\echo '--- FK integrity check (orphans) ---'

-- sales_order_item.order_id must exist in sales_order
SELECT 'sales_order_item orphans' AS check,
       COUNT(*) AS orphan_count
FROM sales_order_item soi
LEFT JOIN sales_order so ON soi.order_id = so.id
WHERE so.id IS NULL;

-- settlement_item.settlement_id must exist in settlement
SELECT 'settlement_item orphans' AS check,
       COUNT(*) AS orphan_count
FROM settlement_item si
LEFT JOIN settlement s ON si.settlement_id = s.id
WHERE s.id IS NULL;

-- finance_transaction.account_id must exist in finance_account
SELECT 'finance_transaction orphans' AS check,
       COUNT(*) AS orphan_count
FROM finance_transaction ft
LEFT JOIN finance_account fa ON ft.account_id = fa.id
WHERE fa.id IS NULL;

-- sku.product_id must exist in product
SELECT 'sku orphans' AS check,
       COUNT(*) AS orphan_count
FROM sku
LEFT JOIN product p ON sku.product_id = p.id
WHERE p.id IS NULL;

\echo ''
\echo '============================================================'
\echo 'Validation complete. Review MISMATCH rows above.'
\echo '============================================================'
