-- 000035_rbac_seed.down.sql
-- Remove RBAC seed data previously inserted by the up migration.

-- Remove admin user role assignments for existing admin-role users
DELETE FROM user_role
WHERE role_id IN (SELECT id FROM role WHERE code = 'admin')
  AND user_id IN (SELECT id FROM "user" WHERE role = 'admin');

-- Remove role-permission mappings for seeded roles
DELETE FROM role_permission
WHERE role_id IN (SELECT id FROM role WHERE code IN ('admin', 'ops', 'viewer'));

-- Remove seeded permissions
DELETE FROM permission WHERE code IN (
    'product.read',
    'product.write',
    'order.read',
    'order.write',
    'finance.read',
    'rbac.manage',
    'agent.read',
    'agent.write',
    'settings.read',
    'settings.write',
    'audit.read',
    'report.read',
    'user.manage',
    'supplychain.read',
    'supplychain.write'
);

-- Remove seeded roles
DELETE FROM role WHERE code IN ('admin', 'ops', 'viewer');
