ALTER TABLE order_profit_record
    ADD COLUMN IF NOT EXISTS profit_status VARCHAR(20) NOT NULL DEFAULT 'provisional',
    ADD COLUMN IF NOT EXISTS missing_costs TEXT NOT NULL DEFAULT '';
