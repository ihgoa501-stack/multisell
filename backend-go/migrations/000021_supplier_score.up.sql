CREATE TABLE IF NOT EXISTS supplier_score (
    id                      BIGSERIAL       PRIMARY KEY,
    supplier_id             BIGINT          NOT NULL REFERENCES supplier(id),
    on_time_delivery_rate   NUMERIC(5,1)    NOT NULL DEFAULT 0,
    quality_pass_rate       NUMERIC(5,1)    NOT NULL DEFAULT 0,
    communication_score     NUMERIC(5,1)    NOT NULL DEFAULT 0,
    order_fulfillment_pct   NUMERIC(5,1)    NOT NULL DEFAULT 0,
    avg_lead_time_days      NUMERIC(6,1)    NOT NULL DEFAULT 0,
    reliability_score       NUMERIC(5,1)    NOT NULL DEFAULT 0,
    data_freshness          INT             NOT NULL DEFAULT 0,
    last_order_date         TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_score_supplier_id ON supplier_score(supplier_id);
CREATE INDEX IF NOT EXISTS idx_supplier_score_reliability ON supplier_score(reliability_score DESC);
