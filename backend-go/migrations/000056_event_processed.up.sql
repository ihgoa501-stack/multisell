-- Event idempotency tracking: prevents duplicate events from double-applying
-- business-state mutations (inventory, order, financial).
--
-- The publisher sets an idempotency_key (e.g. "purchase_order_received:PO-2024-001")
-- via context.WithIdempotencyKey. The bus workerLoop claims the key atomically
-- before dispatch; if another event with the same key was already processed,
-- the duplicate is skipped.
--
-- State machine:
--   processing → handler is executing (INSERT claimed before dispatch)
--   succeeded  → handler completed successfully
--   failed     → handler exhausted retries, event in DLQ (reclaimable for replay)
CREATE TABLE IF NOT EXISTS event_processed (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    topic VARCHAR(100) NOT NULL,
    event_id VARCHAR(36) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'processing',
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_event_processed_topic ON event_processed(topic);
CREATE INDEX IF NOT EXISTS idx_event_processed_event_id ON event_processed(event_id);
CREATE INDEX IF NOT EXISTS idx_event_processed_state ON event_processed(state);
