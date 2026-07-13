INSERT INTO permission (name, code, module, description) VALUES
    ('Owner 采购权威链', 'purchase.owner', 'purchase', '仅 Owner 管理绑定经营决定和外部回执的采购、收货与库存账本'),
    ('Owner 经营行动反馈', 'business_feedback.owner', 'business_feedback', '仅 Owner 创建、执行和观察绑定经营决定的受控行动'),
    ('Owner 售后处置权威链', 'aftersales.owner', 'aftersales', '仅 Owner 决定并核对外部售后处置与终局回执')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM role r CROSS JOIN permission p
WHERE r.code IN ('owner', 'admin') AND r.status = 1
  AND p.code IN ('purchase.owner', 'business_feedback.owner', 'aftersales.owner')
  AND NOT EXISTS (
      SELECT 1 FROM role_permission rp WHERE rp.role_id=r.id AND rp.permission_id=p.id
  );
