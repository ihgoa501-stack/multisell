DROP INDEX IF EXISTS ux_sourcing_draft_conversion_request;
ALTER TABLE sourcing_listing_draft
    DROP COLUMN IF EXISTS conversion_request_sha256,
    DROP COLUMN IF EXISTS conversion_request_id;
