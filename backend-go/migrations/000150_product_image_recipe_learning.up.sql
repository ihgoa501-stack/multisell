ALTER TABLE product_image_tasks
    ADD COLUMN IF NOT EXISTS sku_id BIGINT REFERENCES sku(id),
    ADD COLUMN IF NOT EXISTS recipe_key VARCHAR(100),
    ADD COLUMN IF NOT EXISTS recipe_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS recipe_manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS recipe_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS parent_task_id BIGINT REFERENCES product_image_tasks(id),
    ADD COLUMN IF NOT EXISTS candidate_round INTEGER NOT NULL DEFAULT 1;

ALTER TABLE product_image_tasks
    ADD CONSTRAINT ck_product_image_recipe_version CHECK (recipe_version > 0),
    ADD CONSTRAINT ck_product_image_candidate_round CHECK (candidate_round > 0),
    ADD CONSTRAINT ck_product_image_recipe_hash CHECK (recipe_hash IS NULL OR recipe_hash ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT ck_product_image_recipe_new_fields CHECK (
        recipe_key IS NULL OR (sku_id IS NOT NULL AND recipe_hash IS NOT NULL AND jsonb_typeof(recipe_manifest) = 'object')
    );

CREATE INDEX IF NOT EXISTS idx_product_image_tasks_owner_recipe
    ON product_image_tasks(owner_id, recipe_key, recipe_version, candidate_round, id)
    WHERE recipe_key IS NOT NULL;

ALTER TABLE product_image_reviews
    ADD COLUMN IF NOT EXISTS outcome VARCHAR(32),
    ADD COLUMN IF NOT EXISTS reason_codes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS error_regions JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS rework_instruction TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS review_seconds INTEGER NOT NULL DEFAULT 0;

ALTER TABLE product_image_reviews
    ADD CONSTRAINT ck_product_image_review_outcome CHECK (
        outcome IS NULL OR outcome IN ('selected', 'rejected', 'rework_requested')
    ),
    ADD CONSTRAINT ck_product_image_review_reason_codes CHECK (jsonb_typeof(reason_codes) = 'array'),
    ADD CONSTRAINT ck_product_image_review_error_regions CHECK (jsonb_typeof(error_regions) = 'array'),
    ADD CONSTRAINT ck_product_image_review_seconds CHECK (review_seconds BETWEEN 0 AND 86400);

CREATE INDEX IF NOT EXISTS idx_product_image_reviews_task_feedback
    ON product_image_reviews(owner_id, task_id, outcome, id)
    WHERE decision = 'candidate_feedback';
