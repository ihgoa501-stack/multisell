ALTER TABLE dispute_case ADD COLUMN IF NOT EXISTS order_id BIGINT REFERENCES sales_order(id);
CREATE INDEX IF NOT EXISTS idx_dispute_case_order_id ON dispute_case(order_id);
