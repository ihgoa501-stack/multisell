-- ============================================================
-- Rollback: inventory_module
-- Moves inventory tables back to public schema and drops schema.
-- ============================================================

-- Move tables back to public schema.
ALTER TABLE IF EXISTS inventory_module.inventory SET SCHEMA public;
ALTER TABLE IF EXISTS inventory_module.inventory_log SET SCHEMA public;
ALTER TABLE IF EXISTS inventory_module.inventory_warehouse SET SCHEMA public;
ALTER TABLE IF EXISTS inventory_module.warehouse SET SCHEMA public;

-- Drop the schema.
DROP SCHEMA IF EXISTS inventory_module;
