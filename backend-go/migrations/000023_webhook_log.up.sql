CREATE TABLE IF NOT EXISTS webhook_event_log (
    id BIGSERIAL PRIMARY KEY,
    platform TEXT NOT NULL,
    event_type TEXT NOT NULL,
    raw_payload JSONB,
    status TEXT DEFAULT 'received',
    mapped_event TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
