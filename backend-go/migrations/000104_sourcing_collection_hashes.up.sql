ALTER TABLE sourcing_1688_snapshot
    ADD COLUMN schema_version VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN structured_data_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN request_envelope_sha256 VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN sourcing_1688_snapshot.raw_sha256 IS 'Hash of the submitted finite raw evidence bytes.';
COMMENT ON COLUMN sourcing_1688_snapshot.structured_data_sha256 IS 'Hash of canonical structured collection fields.';
COMMENT ON COLUMN sourcing_1688_snapshot.request_envelope_sha256 IS 'Hash binding Owner, request id, page identity, versions, observation time and evidence hashes.';
