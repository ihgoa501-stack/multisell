-- ============================================================
-- Rollback: order_module
-- Moves order tables back to public schema and drops schema.
-- ============================================================

-- Move tables back to public schema.
ALTER TABLE IF EXISTS order_module.sales_order SET SCHEMA public;
ALTER TABLE IF EXISTS order_module.sales_order_item SET SCHEMA public;
ALTER TABLE IF EXISTS order_module.sales_order_status_log SET SCHEMA public;

-- Drop the schema.
DROP SCHEMA IF EXISTS order_module;
