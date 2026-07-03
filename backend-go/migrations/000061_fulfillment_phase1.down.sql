-- 000061_fulfillment_phase1.down.sql

BEGIN;

ALTER TABLE shipping_quote_rule
    DROP COLUMN IF EXISTS effective_start_time,
    DROP COLUMN IF EXISTS effective_end_time,
    DROP COLUMN IF EXISTS rule_version,
    DROP COLUMN IF EXISTS import_batch;

ALTER TABLE sales_order_shipping_snapshot
    DROP COLUMN IF EXISTS rule_version_id,
    DROP COLUMN IF EXISTS rule_version,
    DROP COLUMN IF EXISTS quoted_by,
    DROP COLUMN IF EXISTS source_trigger;

ALTER TABLE shipping_bill_item
    DROP COLUMN IF EXISTS variance_pct,
    DROP COLUMN IF EXISTS anomaly_type,
    DROP COLUMN IF EXISTS review_status;

COMMIT;
