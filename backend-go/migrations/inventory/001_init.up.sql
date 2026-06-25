-- ============================================================
-- Schema isolation: inventory_module
-- Moves inventory tables from public schema to inventory_module.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS inventory_module;

-- Move existing tables into the inventory_module schema.
ALTER TABLE IF EXISTS public.inventory SET SCHEMA inventory_module;
ALTER TABLE IF EXISTS public.inventory_log SET SCHEMA inventory_module;
ALTER TABLE IF EXISTS public.inventory_warehouse SET SCHEMA inventory_module;
ALTER TABLE IF EXISTS public.warehouse SET SCHEMA inventory_module;
