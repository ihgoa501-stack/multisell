-- ============================================================
-- Schema isolation: settlement_module
-- Moves settlement tables from public schema to settlement_module.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS settlement_module;

-- Move existing tables into the settlement_module schema.
ALTER TABLE IF EXISTS public.settlement SET SCHEMA settlement_module;
ALTER TABLE IF EXISTS public.settlement_item SET SCHEMA settlement_module;
ALTER TABLE IF EXISTS public.platform_settlement_batch SET SCHEMA settlement_module;
ALTER TABLE IF EXISTS public.platform_settlement_item SET SCHEMA settlement_module;
