-- 000056_fulfillment_phase1.up.sql
-- Phase 1 of Fulfillment Intelligence OS:
-- Rule versioning, snapshot enhancements, bill reconciliation fields
--
-- This migration adds:
-- 1. Version fields to shipping_quote_rule (effective_start_time, effective_end_time, rule_version, import_batch)
-- 2. Version+source fields to sales_order_shipping_snapshot (rule_version, quoted_by, source_trigger)
-- 3. Anomaly fields to shipping_bill_item (variance_pct, anomaly_type, review_status)

BEGIN;

-- ── 1. shipping_quote_rule versioning ───────────────────────────────────
ALTER TABLE shipping_quote_rule
    ADD COLUMN IF NOT EXISTS effective_start_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS effective_end_time   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rule_version         INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS import_batch         VARCHAR(64);

CREATE INDEX idx_quote_rule_version ON shipping_quote_rule (channel_id, rule_version DESC);

-- ── 2. sales_order_shipping_snapshot ───────────────────────────────────
ALTER TABLE sales_order_shipping_snapshot
    ADD COLUMN IF NOT EXISTS rule_version_id      BIGINT,
    ADD COLUMN IF NOT EXISTS rule_version         INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS quoted_by            VARCHAR(128),
    ADD COLUMN IF NOT EXISTS source_trigger       VARCHAR(64) DEFAULT 'manual';

-- Drop the UpdatedAt auto-update so snapshots become immutable.
-- We keep the column for backward compat but GORM will no longer auto-set it.
-- (GORM autoUpdateTime must be removed in the model; migration is advisory.)

CREATE INDEX IF NOT EXISTS idx_snapshot_rule_version ON sales_order_shipping_snapshot (rule_version_id);

-- ── 3. shipping_bill_item reconciliation fields ───────────────────────
ALTER TABLE shipping_bill_item
    ADD COLUMN IF NOT EXISTS variance_pct          DECIMAL(10,2),
    ADD COLUMN IF NOT EXISTS anomaly_type          VARCHAR(32),
    ADD COLUMN IF NOT EXISTS review_status         VARCHAR(32) DEFAULT 'pending';

COMMIT;
