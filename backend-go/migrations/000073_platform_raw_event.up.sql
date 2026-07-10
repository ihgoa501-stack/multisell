CREATE TABLE IF NOT EXISTS platform_raw_event (
    id              BIGSERIAL PRIMARY KEY,
    platform_code   VARCHAR(32) NOT NULL,
    account_id      BIGINT NOT NULL REFERENCES platform_integration_account(id),
    event_type      VARCHAR(64) NOT NULL,
    raw_payload     JSONB NOT NULL,
    mapped_result   JSONB,
    mapping_status  VARCHAR(16) NOT NULL DEFAULT 'pending',
    confidence      REAL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    mapped_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_raw_event_platform ON platform_raw_event (platform_code, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_raw_event_status ON platform_raw_event (mapping_status);
