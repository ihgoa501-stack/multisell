DROP TABLE IF EXISTS product_image_execution_approvals;
ALTER TABLE product_image_tasks DROP COLUMN IF EXISTS version;
ALTER TABLE product_image_tasks DROP COLUMN IF EXISTS processor;
