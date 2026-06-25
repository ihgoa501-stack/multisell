-- ============================================================
-- Schema isolation: sku_module
-- Moves product / SKU tables from public schema to sku_module.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS sku_module;

-- Move existing tables into the sku_module schema.
ALTER TABLE IF EXISTS public.product SET SCHEMA sku_module;
ALTER TABLE IF EXISTS public.spec_name SET SCHEMA sku_module;
ALTER TABLE IF EXISTS public.spec_value SET SCHEMA sku_module;
ALTER TABLE IF EXISTS public.sku SET SCHEMA sku_module;
