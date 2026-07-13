ALTER TABLE product_image_tasks
    ADD COLUMN IF NOT EXISTS provider_environment VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS max_cost VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sandbox BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS watermarked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS non_publishable BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE product_image_tasks
    ADD CONSTRAINT ck_product_image_task_sandbox_restrictions
    CHECK (NOT sandbox OR (watermarked AND non_publishable));

CREATE INDEX IF NOT EXISTS idx_product_image_tasks_non_publishable
    ON product_image_tasks(owner_id, non_publishable, id);
