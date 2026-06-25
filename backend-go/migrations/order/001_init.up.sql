-- ============================================================
-- Schema isolation: order_module
-- Moves order tables from public schema to order_module.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS order_module;

-- Move existing tables into the order_module schema.
ALTER TABLE IF EXISTS public.sales_order SET SCHEMA order_module;
ALTER TABLE IF EXISTS public.sales_order_item SET SCHEMA order_module;
ALTER TABLE IF EXISTS public.sales_order_status_log SET SCHEMA order_module;
