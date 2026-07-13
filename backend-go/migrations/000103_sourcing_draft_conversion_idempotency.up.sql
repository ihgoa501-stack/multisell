ALTER TABLE sourcing_listing_draft
    ADD COLUMN conversion_request_id VARCHAR(100),
    ADD COLUMN conversion_request_sha256 VARCHAR(64) NOT NULL DEFAULT '';

UPDATE sourcing_listing_draft
SET conversion_request_id = 'legacy_convert_' || id
WHERE conversion_request_id IS NULL;

ALTER TABLE sourcing_listing_draft ALTER COLUMN conversion_request_id SET NOT NULL;
CREATE UNIQUE INDEX ux_sourcing_draft_conversion_request ON sourcing_listing_draft(conversion_request_id);

COMMENT ON COLUMN sourcing_listing_draft.conversion_request_id IS
    'Owner conversion idempotency key. Reusing the key with different request content is rejected.';
