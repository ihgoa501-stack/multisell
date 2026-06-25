-- ============================================================
-- Rollback: sku_module
-- Moves product / SKU tables back to public schema and drops schema.
-- ============================================================

-- Move tables back to public schema.
ALTER TABLE IF EXISTS sku_module.product SET SCHEMA public;
ALTER TABLE IF EXISTS sku_module.spec_name SET SCHEMA public;
ALTER TABLE IF EXISTS sku_module.spec_value SET SCHEMA public;
ALTER TABLE IF EXISTS sku_module.sku SET SCHEMA public;

-- Drop the schema.
DROP SCHEMA IF EXISTS sku_module;
