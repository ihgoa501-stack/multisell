ALTER TABLE product_image_tasks ADD COLUMN IF NOT EXISTS processor VARCHAR(64) NOT NULL DEFAULT 'deterministic';
ALTER TABLE product_image_tasks ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS product_image_execution_approvals (
    id BIGSERIAL PRIMARY KEY,
    execution_id VARCHAR(64) NOT NULL UNIQUE,
    owner_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL REFERENCES product_image_tasks(id),
    task_version BIGINT NOT NULL,
    manifest_hash VARCHAR(64) NOT NULL,
    operation VARCHAR(64) NOT NULL,
    processor VARCHAR(64) NOT NULL,
    max_cost VARCHAR(32) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    nonce VARCHAR(64) NOT NULL UNIQUE,
    approved_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    CONSTRAINT ck_product_image_approval_task_version CHECK (task_version > 0),
    CONSTRAINT ck_product_image_approval_processor CHECK (processor <> '' AND processor <> 'deterministic'),
    CONSTRAINT ck_product_image_approval_currency CHECK (currency IN ('USD', 'EUR', 'CNY', 'GBP', 'JPY')),
    CONSTRAINT ck_product_image_approval_max_cost CHECK (max_cost ~ '^(0\.[0-9]{0,3}[1-9]|[1-9][0-9]{0,9}(\.[0-9]{1,4})?)$'),
    CONSTRAINT ck_product_image_approval_expiry CHECK (expires_at > approved_at AND expires_at <= approved_at + INTERVAL '5 minutes'),
    CONSTRAINT ck_product_image_approval_consumed CHECK (consumed_at IS NULL OR consumed_at >= approved_at)
);
CREATE INDEX IF NOT EXISTS idx_product_image_execution_approval_owner_task ON product_image_execution_approvals(owner_id, task_id, id DESC);
