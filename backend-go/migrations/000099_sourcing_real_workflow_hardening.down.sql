DELETE FROM role_permission
WHERE permission_id = (SELECT id FROM permission WHERE code = 'listing.publish');
DELETE FROM permission WHERE code = 'listing.publish';

ALTER TABLE demand_case DROP COLUMN IF EXISTS target_locale;

DROP INDEX IF EXISTS ux_sourcing_snapshot_collection_request;
ALTER TABLE sourcing_1688_snapshot DROP COLUMN IF EXISTS collection_request_id;

DROP INDEX IF EXISTS ux_sourcing_publish_one_active;

ALTER TABLE sourcing_1688_snapshot
    ALTER COLUMN raw_payload TYPE JSONB
    USING convert_from(raw_payload, 'UTF8')::jsonb;

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
