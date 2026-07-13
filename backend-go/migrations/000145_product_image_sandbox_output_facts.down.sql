DROP INDEX IF EXISTS idx_product_image_tasks_non_publishable;
ALTER TABLE product_image_tasks DROP CONSTRAINT IF EXISTS ck_product_image_task_sandbox_restrictions;
ALTER TABLE product_image_tasks
    DROP COLUMN IF EXISTS non_publishable,
    DROP COLUMN IF EXISTS watermarked,
    DROP COLUMN IF EXISTS sandbox,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS max_cost,
    DROP COLUMN IF EXISTS provider_environment;
