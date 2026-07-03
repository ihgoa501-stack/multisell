-- 000057_fulfillment_phase3_tracking.up.sql
-- Phase 3: Fulfillment tracking — tracking numbers, events, lost/returned/damaged flags

BEGIN;

CREATE TABLE IF NOT EXISTS fulfillment_tracking (
    id               BIGSERIAL PRIMARY KEY,
    order_id         BIGINT NOT NULL,
    tracking_number  VARCHAR(128) NOT NULL,
    carrier_code     VARCHAR(64),
    carrier_name     VARCHAR(128),
    status           VARCHAR(32) DEFAULT 'pending',
    tracking_events  JSONB DEFAULT '[]',
    estimated_delivery TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    is_lost          BOOLEAN DEFAULT FALSE,
    is_returned      BOOLEAN DEFAULT FALSE,
    is_damaged       BOOLEAN DEFAULT FALSE,
    note             TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tracking_order ON fulfillment_tracking (order_id);
CREATE INDEX IF NOT EXISTS idx_tracking_number ON fulfillment_tracking (tracking_number);

COMMIT;
