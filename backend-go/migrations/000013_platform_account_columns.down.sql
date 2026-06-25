ALTER TABLE platform_integration_account
    DROP COLUMN IF EXISTS store_name,
    DROP COLUMN IF EXISTS account_id,
    DROP COLUMN IF EXISTS access_token,
    DROP COLUMN IF EXISTS refresh_token,
    DROP COLUMN IF EXISTS token_expires_at,
    DROP COLUMN IF EXISTS last_sync_at,
    DROP COLUMN IF EXISTS sync_status,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS config;
