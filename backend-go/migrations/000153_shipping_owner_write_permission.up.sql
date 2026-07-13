-- Shipping mutations are authoritative Owner operations. Historical ops grants
-- are removed; enabled owner/admin roles retain the existing shipping.write capability.
DELETE FROM role_permission rp
USING role r, permission p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND p.code = 'shipping.write'
  AND (r.code NOT IN ('owner', 'admin') OR r.status <> 1);

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r CROSS JOIN permission p
WHERE r.code IN ('owner', 'admin')
  AND r.status = 1
  AND p.code = 'shipping.write'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
