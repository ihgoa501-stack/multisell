-- Purchase Module — procurement tables
-- purchase_order, purchase_order_item, purchase_suggestion

CREATE TABLE IF NOT EXISTS purchase_order (
    id               BIGSERIAL           PRIMARY KEY,
    order_no         VARCHAR(255)        NOT NULL,
    supplier_id      BIGINT              NOT NULL,
    status           VARCHAR(20)         NOT NULL DEFAULT 'draft',
    total_amount     DECIMAL(12,2)       NOT NULL DEFAULT 0,
    expected_delivery VARCHAR(50),
    remark           TEXT                NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_purchase_order_order_no ON purchase_order(order_no);

CREATE TABLE IF NOT EXISTS purchase_order_item (
    id                BIGSERIAL           PRIMARY KEY,
    purchase_order_id BIGINT              NOT NULL,
    sku_id            BIGINT              NOT NULL,
    quantity          INTEGER             NOT NULL,
    received_qty      INTEGER             NOT NULL DEFAULT 0,
    unit_price        DECIMAL(12,2)       NOT NULL,
    subtotal          DECIMAL(12,2)       NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_item_order_id ON purchase_order_item(purchase_order_id);

CREATE TABLE IF NOT EXISTS purchase_suggestion (
    id             BIGSERIAL           PRIMARY KEY,
    sku_id         BIGINT              NOT NULL,
    suggested_qty  INTEGER             NOT NULL,
    reason         TEXT                NOT NULL,
    status         VARCHAR(20)         NOT NULL DEFAULT 'pending',
    generated_at   TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_suggestion_sku_id ON purchase_suggestion(sku_id);
