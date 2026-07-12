DROP TRIGGER IF EXISTS trg_block_pending_sourcing_cost ON product_cost_input;
DROP TRIGGER IF EXISTS trg_block_pending_sourcing_media ON product_media_asset;
DROP TRIGGER IF EXISTS trg_block_pending_sourcing_sku ON sku;
DROP TRIGGER IF EXISTS trg_block_pending_sourcing_listing ON product_listing;
DROP TRIGGER IF EXISTS trg_block_pending_sourcing_product ON product;
DROP FUNCTION IF EXISTS block_pending_sourcing_draft_content_mutation();

DROP TRIGGER IF EXISTS trg_guard_sourcing_draft_approval_hash ON approval_request;
DROP FUNCTION IF EXISTS guard_sourcing_draft_approval_hash();

ALTER TABLE sourcing_listing_draft
    DROP COLUMN IF EXISTS approval_content_sha256;
