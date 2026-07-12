ALTER TABLE sourcing_listing_draft
    ADD COLUMN IF NOT EXISTS approval_content_sha256 CHAR(64) NOT NULL DEFAULT '';

CREATE OR REPLACE FUNCTION guard_sourcing_draft_approval_hash()
RETURNS trigger AS $$
DECLARE
    stored_hash TEXT;
    request_hash TEXT;
BEGIN
    IF NEW.request_type <> 'sourcing_1688_draft'
       OR OLD.status IS NOT DISTINCT FROM NEW.status THEN
        RETURN NEW;
    END IF;
    SELECT approval_content_sha256 INTO STRICT stored_hash
    FROM sourcing_listing_draft
    WHERE id = NEW.target_id
      AND approval_id = NEW.id
      AND product_id = NEW.product_id;
    request_hash := NEW.new_value::jsonb ->> 'content_sha256';
    IF length(stored_hash) <> 64 OR request_hash IS DISTINCT FROM stored_hash THEN
        RAISE EXCEPTION 'sourcing draft approval content fingerprint is missing or mismatched';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_sourcing_draft_approval_hash ON approval_request;
CREATE TRIGGER trg_guard_sourcing_draft_approval_hash
BEFORE UPDATE OF status ON approval_request
FOR EACH ROW EXECUTE FUNCTION guard_sourcing_draft_approval_hash();

-- Once an approval is pending, the exact content under review cannot change.
-- The Go service locks these rows while calculating the hash; these triggers
-- close the race after that transaction commits.
CREATE OR REPLACE FUNCTION block_pending_sourcing_draft_content_mutation()
RETURNS trigger AS $$
DECLARE
    affected_product_id BIGINT;
    is_pending BOOLEAN;
BEGIN
    IF TG_TABLE_NAME = 'product' THEN
        IF TG_OP = 'DELETE' THEN affected_product_id := OLD.id; ELSE affected_product_id := NEW.id; END IF;
    ELSIF TG_TABLE_NAME = 'product_listing' THEN
        SELECT product_id INTO affected_product_id FROM sourcing_listing_draft
        WHERE listing_id = CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END;
    ELSE
        IF TG_OP = 'DELETE' THEN affected_product_id := OLD.product_id; ELSE affected_product_id := NEW.product_id; END IF;
    END IF;
    SELECT EXISTS (
        SELECT 1 FROM sourcing_listing_draft
        WHERE product_id = affected_product_id AND approval_status = 'pending'
    ) INTO is_pending;
    IF is_pending THEN
        RAISE EXCEPTION 'sourcing draft content is frozen while Owner approval is pending';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_block_pending_sourcing_product ON product;
CREATE TRIGGER trg_block_pending_sourcing_product
BEFORE UPDATE OR DELETE ON product
FOR EACH ROW EXECUTE FUNCTION block_pending_sourcing_draft_content_mutation();

DROP TRIGGER IF EXISTS trg_block_pending_sourcing_listing ON product_listing;
CREATE TRIGGER trg_block_pending_sourcing_listing
BEFORE UPDATE OR DELETE ON product_listing
FOR EACH ROW EXECUTE FUNCTION block_pending_sourcing_draft_content_mutation();

DROP TRIGGER IF EXISTS trg_block_pending_sourcing_sku ON sku;
CREATE TRIGGER trg_block_pending_sourcing_sku
BEFORE INSERT OR UPDATE OR DELETE ON sku
FOR EACH ROW EXECUTE FUNCTION block_pending_sourcing_draft_content_mutation();

DROP TRIGGER IF EXISTS trg_block_pending_sourcing_media ON product_media_asset;
CREATE TRIGGER trg_block_pending_sourcing_media
BEFORE INSERT OR UPDATE OR DELETE ON product_media_asset
FOR EACH ROW EXECUTE FUNCTION block_pending_sourcing_draft_content_mutation();

DROP TRIGGER IF EXISTS trg_block_pending_sourcing_cost ON product_cost_input;
CREATE TRIGGER trg_block_pending_sourcing_cost
BEFORE INSERT OR UPDATE OR DELETE ON product_cost_input
FOR EACH ROW EXECUTE FUNCTION block_pending_sourcing_draft_content_mutation();
