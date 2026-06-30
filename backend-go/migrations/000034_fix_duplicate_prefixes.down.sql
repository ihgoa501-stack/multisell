-- Rollback: drop tables created by the forward repair migration.

DROP TABLE IF EXISTS tariff_rule;
DROP TABLE IF EXISTS sku_return_stats;
