ALTER TABLE order_profit_record
    DROP COLUMN IF EXISTS missing_costs,
    DROP COLUMN IF EXISTS profit_status;
