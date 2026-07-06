ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(255);
ALTER TABLE unified_action ADD COLUMN IF NOT EXISTS block_reason VARCHAR(100);
