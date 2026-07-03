-- 000055_add_rbac_permissions.up.sql
-- Add RBAC permissions for high-risk commerce modules: shipping, settlement,
-- inventory, and listing. The product and order permissions already exist
-- from migration 000035.

-- ==========================================================================
-- New Permissions
-- ==========================================================================
INSERT INTO permission (name, code, module, description) VALUES
    ('物流读取',   'shipping.read',     'shipping',    '查看物流/发货信息'),
    ('物流写入',   'shipping.write',    'shipping',    '创建和编辑物流/发货信息'),
    ('结算读取',   'settlement.read',   'settlement',  '查看结算数据'),
    ('结算写入',   'settlement.write',  'settlement',  '创建和编辑结算数据'),
    ('库存读取',   'inventory.read',    'inventory',   '查看库存信息'),
    ('库存写入',   'inventory.write',   'inventory',   '创建和编辑库存信息'),
    ('刊登读取',   'listing.read',      'listing',     '查看刊登信息'),
    ('刊登写入',   'listing.write',     'listing',     '创建和编辑刊登信息')
ON CONFLICT (code) DO NOTHING;

-- ==========================================================================
-- Admin: grant all new permissions (idempotent cross-join)
-- ==========================================================================
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'admin'
  AND p.code IN (
    'shipping.read', 'shipping.write',
    'settlement.read', 'settlement.write',
    'inventory.read', 'inventory.write',
    'listing.read', 'listing.write'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- ==========================================================================
-- Operator: add read+write for shipping, settlement, inventory, listing
-- ==========================================================================
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'ops'
  AND p.code IN (
    'shipping.read', 'shipping.write',
    'settlement.read', 'settlement.write',
    'inventory.read', 'inventory.write',
    'listing.read', 'listing.write'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- ==========================================================================
-- Viewer: add read-only for shipping, settlement, inventory, listing
-- ==========================================================================
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'viewer'
  AND p.code IN (
    'shipping.read',
    'settlement.read',
    'inventory.read',
    'listing.read'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
