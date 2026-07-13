ALTER TABLE sourcing_1688_snapshot
    DROP COLUMN IF EXISTS request_envelope_sha256,
    DROP COLUMN IF EXISTS structured_data_sha256,
    DROP COLUMN IF EXISTS schema_version;
