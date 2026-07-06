-- 000064_link_operator_users_to_ops_role.up.sql
-- Link existing operator-role users (user.role = 'operator') to the
-- RBAC ops role. Migration 000035 only linked admin users; operator
-- users were left without any RBAC role assignment, causing all
-- RequirePermission middleware checks to return 403.

INSERT INTO user_role (user_id, role_id)
SELECT u.id, r.id
FROM "user" u, role r
WHERE u.role = 'operator'
  AND r.code = 'ops'
  AND NOT EXISTS (
    SELECT 1 FROM user_role ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
  );
