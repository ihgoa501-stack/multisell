DELETE FROM role_permission
WHERE permission_id IN (SELECT id FROM permission WHERE code IN ('market.read','market.write','market.decide'));
DELETE FROM permission WHERE code IN ('market.read','market.write','market.decide');
DROP TABLE IF EXISTS product_opportunity_decision;
DROP TABLE IF EXISTS product_opportunity;
DROP TABLE IF EXISTS market_owner_decision;
