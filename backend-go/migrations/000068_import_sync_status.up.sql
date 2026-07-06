CREATE TABLE IF NOT EXISTS import_sync_status (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_sync_result VARCHAR(20) NOT NULL DEFAULT '',
    order_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    pending_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_import_sync_platform ON import_sync_status(platform);
