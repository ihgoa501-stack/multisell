-- 000064_link_operator_users_to_ops_role.down.sql
-- Remove operator user → ops RBAC role links added by the up migration.

DELETE FROM user_role
WHERE role_id IN (SELECT id FROM role WHERE code = 'ops')
  AND user_id IN (SELECT id FROM "user" WHERE role = 'operator');
