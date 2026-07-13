DELETE FROM role_permission
WHERE permission_id = (SELECT id FROM permission WHERE code = 'product_image.owner');
DELETE FROM permission WHERE code = 'product_image.owner';

ALTER TABLE product_image_tasks
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS channel,
    DROP COLUMN IF EXISTS purpose;

DROP INDEX IF EXISTS idx_product_image_asset_parent;
ALTER TABLE product_image_assets
    DROP CONSTRAINT IF EXISTS ck_product_image_asset_channel_restriction,
    DROP CONSTRAINT IF EXISTS ck_product_image_asset_parent_lineage,
    DROP COLUMN IF EXISTS channel_restriction,
    DROP COLUMN IF EXISTS parent_asset_sha,
    DROP COLUMN IF EXISTS parent_asset_id,
    DROP COLUMN IF EXISTS source_kind;
