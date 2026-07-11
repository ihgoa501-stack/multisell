DROP INDEX IF EXISTS idx_settlement_source_type;
ALTER TABLE settlement DROP COLUMN IF EXISTS source_type;
