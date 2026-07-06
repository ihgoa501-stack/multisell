ALTER TABLE platform_integration_account ADD COLUMN IF NOT EXISTS execution_mode SMALLINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN platform_integration_account.execution_mode IS '0=dry_run 1=sandbox 2=approval_required 3=production';
