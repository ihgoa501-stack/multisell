ALTER TABLE sourcing_1688_snapshot
    ADD COLUMN IF NOT EXISTS capture_mode VARCHAR(24) NOT NULL DEFAULT 'legacy_unknown';

ALTER TABLE sourcing_1688_snapshot
    DROP CONSTRAINT IF EXISTS ck_sourcing_snapshot_capture_mode;

ALTER TABLE sourcing_1688_snapshot
    ADD CONSTRAINT ck_sourcing_snapshot_capture_mode
    CHECK (capture_mode IN ('controlled_fetch', 'manual_import', 'legacy_unknown'));

COMMENT ON COLUMN sourcing_1688_snapshot.capture_mode IS
    'Server-assigned provenance. Only controlled_fetch can prove real controlled collection; legacy rows are never upgraded implicitly.';
