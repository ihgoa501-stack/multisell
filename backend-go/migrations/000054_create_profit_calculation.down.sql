-- +migrate Down
-- +migrate StatementBegin

DROP INDEX IF EXISTS idx_profit_order_sku;
DROP TABLE IF EXISTS profit_calculation;

-- +migrate StatementEnd
