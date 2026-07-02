-- 000055_add_rbac_permissions.down.sql
-- Revert the RBAC permissions added in the up migration.

-- Remove role-permission mappings for new permissions
DELETE FROM role_permission
WHERE permission_id IN (
  SELECT id FROM permission WHERE code IN (
    'shipping.read', 'shipping.write',
    'settlement.read', 'settlement.write',
    'inventory.read', 'inventory.write',
    'listing.read', 'listing.write'
  )
);

-- Remove new permissions
DELETE FROM permission WHERE code IN (
  'shipping.read', 'shipping.write',
  'settlement.read', 'settlement.write',
  'inventory.read', 'inventory.write',
  'listing.read', 'listing.write'
);
