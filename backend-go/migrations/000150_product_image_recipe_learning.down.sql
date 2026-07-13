DROP INDEX IF EXISTS idx_product_image_reviews_task_feedback;
ALTER TABLE product_image_reviews
    DROP CONSTRAINT IF EXISTS ck_product_image_review_seconds,
    DROP CONSTRAINT IF EXISTS ck_product_image_review_error_regions,
    DROP CONSTRAINT IF EXISTS ck_product_image_review_reason_codes,
    DROP CONSTRAINT IF EXISTS ck_product_image_review_outcome,
    DROP COLUMN IF EXISTS review_seconds,
    DROP COLUMN IF EXISTS rework_instruction,
    DROP COLUMN IF EXISTS error_regions,
    DROP COLUMN IF EXISTS reason_codes,
    DROP COLUMN IF EXISTS outcome;

DROP INDEX IF EXISTS idx_product_image_tasks_owner_recipe;
ALTER TABLE product_image_tasks
    DROP CONSTRAINT IF EXISTS ck_product_image_recipe_new_fields,
    DROP CONSTRAINT IF EXISTS ck_product_image_recipe_hash,
    DROP CONSTRAINT IF EXISTS ck_product_image_candidate_round,
    DROP CONSTRAINT IF EXISTS ck_product_image_recipe_version,
    DROP COLUMN IF EXISTS candidate_round,
    DROP COLUMN IF EXISTS parent_task_id,
    DROP COLUMN IF EXISTS recipe_hash,
    DROP COLUMN IF EXISTS recipe_manifest,
    DROP COLUMN IF EXISTS recipe_version,
    DROP COLUMN IF EXISTS recipe_key,
    DROP COLUMN IF EXISTS sku_id;
