-- ============================================================
-- Rollback: finance_module
-- Moves finance tables back to public schema and drops schema.
-- ============================================================

-- Move tables back to public schema.
ALTER TABLE IF EXISTS finance_module.finance_account SET SCHEMA public;
ALTER TABLE IF EXISTS finance_module.finance_transaction SET SCHEMA public;
ALTER TABLE IF EXISTS finance_module.finance_ledger_entry SET SCHEMA public;

-- Drop the schema.
DROP SCHEMA IF EXISTS finance_module;
