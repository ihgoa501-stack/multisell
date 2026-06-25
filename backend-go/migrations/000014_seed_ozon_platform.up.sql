-- Seed the Ozon platform record if it doesn't exist.

INSERT INTO platform (name, code, api_base_url, sort_order, status)
SELECT 'Ozon', 'ozon', 'https://api-seller.ozon.ru', 10, 1
WHERE NOT EXISTS (SELECT 1 FROM platform WHERE code = 'ozon');
