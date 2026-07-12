-- Link existing operator-role users to the RBAC ops role.
INSERT INTO user_role (user_id, role_id)
SELECT u.id, r.id
FROM "user" u, role r
WHERE u.role = 'operator'
  AND r.code = 'ops'
  AND NOT EXISTS (
    SELECT 1 FROM user_role ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
  );
