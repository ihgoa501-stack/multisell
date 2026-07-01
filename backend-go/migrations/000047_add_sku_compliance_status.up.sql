-- Add compliance_status column to sku table for AI compliance flagging.
-- The FlagNonCompliantHandler in command/handlers.go writes compliance
-- flags like "flagged_high", "flagged_critical" here.

ALTER TABLE sku ADD COLUMN IF NOT EXISTS compliance_status VARCHAR(30) NOT NULL DEFAULT '';
