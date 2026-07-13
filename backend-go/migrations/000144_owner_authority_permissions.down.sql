DELETE FROM role_permission
WHERE permission_id IN (
    SELECT id FROM permission WHERE code IN ('purchase.owner','business_feedback.owner','aftersales.owner')
);
DELETE FROM permission WHERE code IN ('purchase.owner','business_feedback.owner','aftersales.owner');
