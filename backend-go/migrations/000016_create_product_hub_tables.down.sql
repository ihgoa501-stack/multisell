-- Migration: 000016_create_product_hub_tables
-- Down
DROP TABLE IF EXISTS product_variant CASCADE;
DROP TABLE IF EXISTS product_master CASCADE;
