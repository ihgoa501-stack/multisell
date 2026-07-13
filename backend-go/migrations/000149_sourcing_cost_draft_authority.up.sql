ALTER TABLE sourcing_cost_version
    ADD COLUMN revenue_minor BIGINT NOT NULL DEFAULT 0 CHECK (revenue_minor >= 0),
    ADD COLUMN contribution_profit_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN pricing_basis VARCHAR(48) NOT NULL DEFAULT 'legacy_unspecified'
        CHECK (pricing_basis IN ('legacy_unspecified', 'owner_confirmed_listing_price')),
    ADD COLUMN quantity_tier_min BIGINT NOT NULL DEFAULT 1 CHECK (quantity_tier_min > 0),
    ADD COLUMN quantity_tier_max BIGINT,
    ADD COLUMN purchase_line_owner_confirmed BOOLEAN NOT NULL DEFAULT FALSE;

-- Existing cost authority is immutable at runtime, but this migration must
-- backfill its newly introduced derived column before restoring that guard.
DROP TRIGGER IF EXISTS trg_sourcing_cost_version_immutable ON sourcing_cost_version;

UPDATE sourcing_cost_version
SET contribution_profit_minor = revenue_minor - total_minor;

CREATE TRIGGER trg_sourcing_cost_version_immutable
BEFORE UPDATE OR DELETE ON sourcing_cost_version
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_cost_authority_mutation();

ALTER TABLE sourcing_cost_version
    ADD CONSTRAINT ck_sourcing_cost_profit_math CHECK (contribution_profit_minor = revenue_minor - total_minor),
    ADD CONSTRAINT ck_sourcing_cost_quantity_tier CHECK (quantity_tier_max IS NULL OR quantity_tier_max >= quantity_tier_min);

ALTER TABLE sourcing_listing_draft
    ADD COLUMN cost_version_id BIGINT REFERENCES sourcing_cost_version(id) ON DELETE RESTRICT,
    ADD COLUMN cost_version_content_hash CHAR(64),
    ADD CONSTRAINT ck_sourcing_draft_cost_hash_pair CHECK (
        (cost_version_id IS NULL AND cost_version_content_hash IS NULL) OR
        (cost_version_id IS NOT NULL AND cost_version_content_hash ~ '^[0-9a-f]{64}$')
    );

CREATE OR REPLACE FUNCTION validate_sourcing_draft_cost_authority() RETURNS trigger AS $$
DECLARE
    expected_hash CHAR(64);
    expected_source BIGINT;
    expected_task BIGINT;
BEGIN
    IF NEW.approval_status IN ('pending', 'approved') THEN
        IF NEW.cost_version_id IS NULL OR NEW.cost_version_content_hash IS NULL THEN
            RAISE EXCEPTION 'draft approval requires an exact precise cost version';
        END IF;
        SELECT content_hash, sourcing_product_id, task_link_id
          INTO expected_hash, expected_source, expected_task
          FROM sourcing_cost_version WHERE id = NEW.cost_version_id;
        IF expected_hash IS NULL OR expected_hash <> NEW.cost_version_content_hash OR
           expected_source <> NEW.sourcing_product_id OR expected_task <> NEW.task_link_id THEN
            RAISE EXCEPTION 'draft precise cost version authority mismatch';
        END IF;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.approval_status IN ('pending', 'approved') AND
       (NEW.cost_version_id IS DISTINCT FROM OLD.cost_version_id OR
        NEW.cost_version_content_hash IS DISTINCT FROM OLD.cost_version_content_hash) THEN
        RAISE EXCEPTION 'approved or pending draft cost authority is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_draft_cost_authority
BEFORE INSERT OR UPDATE ON sourcing_listing_draft
FOR EACH ROW EXECUTE FUNCTION validate_sourcing_draft_cost_authority();
