-- 000035_rbac_seed.up.sql
-- Seed RBAC baseline data: roles, permissions, role-permission mappings, and
-- admin user assignments to existing admin-role users.
-- Uses NOT EXISTS / DO NOTHING for idempotent replay safety.

-- ==========================================================================
-- Roles
-- ==========================================================================
INSERT INTO role (name, code, description, status) VALUES
    ('Administrator', 'admin', '系统管理员，拥有全部权限', 1),
    ('Operator', 'ops', '运营人员，拥有运营和只读权限', 1),
    ('Viewer', 'viewer', '只读用户，仅可查看数据', 1)
ON CONFLICT (code) DO NOTHING;

-- ==========================================================================
-- Permissions
-- ==========================================================================
INSERT INTO permission (name, code, module, description) VALUES
    ('产品读取',   'product.read',      'product',     '查看产品信息'),
    ('产品写入',   'product.write',     'product',     '创建和编辑产品'),
    ('订单读取',   'order.read',        'order',       '查看订单信息'),
    ('订单写入',   'order.write',       'order',       '创建和编辑订单'),
    ('财务读取',   'finance.read',      'finance',     '查看财务数据'),
    ('RBAC 管理',  'rbac.manage',       'rbac',        '管理角色和权限'),
    ('Agent 读取', 'agent.read',        'agent',       '查看Agent信息'),
    ('Agent 写入', 'agent.write',       'agent',       '配置和操作Agent'),
    ('设置读取',   'settings.read',     'settings',    '查看系统设置'),
    ('设置写入',   'settings.write',    'settings',    '修改系统设置'),
    ('审计日志读取', 'audit.read',      'audit',       '查看审计日志'),
    ('报表读取',   'report.read',       'report',      '查看报表'),
    ('用户管理',   'user.manage',       'user',        '管理用户账号'),
    ('供应链读取', 'supplychain.read',  'supplychain', '查看供应链数据'),
    ('供应链写入', 'supplychain.write', 'supplychain', '编辑供应链数据')
ON CONFLICT (code) DO NOTHING;

-- ==========================================================================
-- Role-Permission mappings
-- ==========================================================================

-- Admin: all permissions
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'admin'
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- Operator: operational + read-only permissions
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'ops'
  AND p.code IN (
    'product.read', 'product.write',
    'order.read', 'order.write',
    'finance.read',
    'agent.read',
    'settings.read',
    'audit.read',
    'report.read',
    'supplychain.read'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- Viewer: read-only permissions
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r, permission p
WHERE r.code = 'viewer'
  AND p.code IN (
    'product.read',
    'order.read',
    'finance.read',
    'agent.read',
    'settings.read',
    'audit.read',
    'report.read',
    'supplychain.read'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- ==========================================================================
-- Link existing admin-role users to the admin RBAC role
-- ==========================================================================
INSERT INTO user_role (user_id, role_id)
SELECT u.id, r.id
FROM "user" u, role r
WHERE u.role = 'admin'
  AND r.code = 'admin'
  AND NOT EXISTS (
    SELECT 1 FROM user_role ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
  );
