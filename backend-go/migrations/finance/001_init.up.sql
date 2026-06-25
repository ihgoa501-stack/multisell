-- ============================================================
-- Schema isolation: finance_module
-- Moves finance tables from public schema to finance_module.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS finance_module;

-- Move existing tables into the finance_module schema.
ALTER TABLE IF EXISTS public.finance_account SET SCHEMA finance_module;
ALTER TABLE IF EXISTS public.finance_transaction SET SCHEMA finance_module;
ALTER TABLE IF EXISTS public.finance_ledger_entry SET SCHEMA finance_module;
