ALTER TABLE sourcing_1688_snapshot
    DROP CONSTRAINT IF EXISTS ck_sourcing_snapshot_capture_mode;

ALTER TABLE sourcing_1688_snapshot
    DROP COLUMN IF EXISTS capture_mode;
