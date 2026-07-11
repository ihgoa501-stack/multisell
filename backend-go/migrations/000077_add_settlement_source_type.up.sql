ALTER TABLE settlement
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(32) NOT NULL DEFAULT 'manual';

CREATE INDEX IF NOT EXISTS idx_settlement_source_type ON settlement(source_type);
