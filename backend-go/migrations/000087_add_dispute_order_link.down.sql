DROP INDEX IF EXISTS idx_dispute_case_order_id;
ALTER TABLE dispute_case DROP COLUMN IF EXISTS order_id;
