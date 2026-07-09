ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS execution_mode SMALLINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN listing_task.execution_mode IS '0=dry_run 1=sandbox 2=approval_required 3=production';

ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS external_reference_id VARCHAR(255) NOT NULL DEFAULT '';
COMMENT ON COLUMN listing_task.external_reference_id IS 'Platform-returned product ID after publish';

ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS external_reference_url TEXT NOT NULL DEFAULT '';
COMMENT ON COLUMN listing_task.external_reference_url IS 'Platform product URL after publish';
