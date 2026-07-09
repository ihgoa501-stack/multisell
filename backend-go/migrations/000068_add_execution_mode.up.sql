ALTER TABLE platform_integration_account
    ADD COLUMN IF NOT EXISTS execution_mode SMALLINT NOT NULL DEFAULT 0;
