-- ============================================================
-- Rollback: settlement_module
-- Moves settlement tables back to public schema and drops schema.
-- ============================================================

-- Move tables back to public schema.
ALTER TABLE IF EXISTS settlement_module.settlement SET SCHEMA public;
ALTER TABLE IF EXISTS settlement_module.settlement_item SET SCHEMA public;
ALTER TABLE IF EXISTS settlement_module.platform_settlement_batch SET SCHEMA public;
ALTER TABLE IF EXISTS settlement_module.platform_settlement_item SET SCHEMA public;

-- Drop the schema.
DROP SCHEMA IF EXISTS settlement_module;
