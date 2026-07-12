DELETE FROM user_role
WHERE role_id IN (SELECT id FROM role WHERE code = 'ops')
  AND user_id IN (SELECT id FROM "user" WHERE role = 'operator');
