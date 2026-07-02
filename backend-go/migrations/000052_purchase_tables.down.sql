-- Drop purchase module tables and indexes.

DROP INDEX IF EXISTS idx_purchase_suggestion_sku_id;
DROP INDEX IF EXISTS idx_purchase_order_item_order_id;
DROP INDEX IF EXISTS idx_purchase_order_order_no;

DROP TABLE IF EXISTS purchase_suggestion;
DROP TABLE IF EXISTS purchase_order_item;
DROP TABLE IF EXISTS purchase_order;
