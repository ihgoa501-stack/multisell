-- Add new columns to platform_integration_account for token-based auth and Ozon support.
-- The existing schema (adapter_code, account_name, credential_metadata) remains for
-- backward compatibility; the new columns let adapters use the Go model's encrypted
-- token fields and structured config.

ALTER TABLE platform_integration_account
    ADD COLUMN IF NOT EXISTS store_name     VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS account_id     VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS access_token   TEXT         NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS refresh_token  TEXT         NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS token_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sync_status    VARCHAR(30) NOT NULL DEFAULT 'idle',
    ADD COLUMN IF NOT EXISTS last_error     TEXT         NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS config         JSONB        NOT NULL DEFAULT '{}'::jsonb;
