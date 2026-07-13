ALTER TABLE product_image_assets
    ADD COLUMN IF NOT EXISTS source_kind VARCHAR(32) NOT NULL DEFAULT 'upload',
    ADD COLUMN IF NOT EXISTS parent_asset_id BIGINT REFERENCES product_image_assets(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS parent_asset_sha VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS channel_restriction VARCHAR(64) NOT NULL DEFAULT '*';

ALTER TABLE product_image_assets
    ADD CONSTRAINT ck_product_image_asset_parent_lineage
    CHECK ((parent_asset_id IS NULL AND parent_asset_sha = '') OR
           (parent_asset_id IS NOT NULL AND parent_asset_sha ~ '^[0-9a-f]{64}$'));
ALTER TABLE product_image_assets
    ADD CONSTRAINT ck_product_image_asset_channel_restriction CHECK (channel_restriction <> '');
CREATE INDEX IF NOT EXISTS idx_product_image_asset_parent ON product_image_assets(owner_id, parent_asset_id);

ALTER TABLE product_image_tasks
    ADD COLUMN IF NOT EXISTS purpose VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS channel VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS region VARCHAR(64) NOT NULL DEFAULT '';

-- Existing tasks predate the processing-rights contract. They remain readable,
-- but their empty contract fields make them fail closed at every new image-set
-- or release gate; they are not silently grandfathered into authorization.

INSERT INTO permission (name, code, module, description)
VALUES ('Owner 商品图片系统', 'product_image.owner', 'product_image', '仅 Owner 可管理商品图片、权利、审核和发布证明')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r CROSS JOIN permission p
WHERE r.code IN ('owner', 'admin') AND r.status = 1 AND p.code = 'product_image.owner'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
