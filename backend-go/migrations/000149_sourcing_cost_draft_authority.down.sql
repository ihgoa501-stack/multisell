DROP TRIGGER IF EXISTS trg_sourcing_draft_cost_authority ON sourcing_listing_draft;
DROP FUNCTION IF EXISTS validate_sourcing_draft_cost_authority();
ALTER TABLE sourcing_listing_draft
    DROP CONSTRAINT IF EXISTS ck_sourcing_draft_cost_hash_pair,
    DROP COLUMN IF EXISTS cost_version_content_hash,
    DROP COLUMN IF EXISTS cost_version_id;
ALTER TABLE sourcing_cost_version
    DROP CONSTRAINT IF EXISTS ck_sourcing_cost_quantity_tier,
    DROP CONSTRAINT IF EXISTS ck_sourcing_cost_profit_math,
    DROP COLUMN IF EXISTS purchase_line_owner_confirmed,
    DROP COLUMN IF EXISTS quantity_tier_max,
    DROP COLUMN IF EXISTS quantity_tier_min,
    DROP COLUMN IF EXISTS pricing_basis,
    DROP COLUMN IF EXISTS contribution_profit_minor,
    DROP COLUMN IF EXISTS revenue_minor;
