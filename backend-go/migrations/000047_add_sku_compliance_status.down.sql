-- Remove compliance_status column from sku table.

ALTER TABLE sku DROP COLUMN IF EXISTS compliance_status;
