DROP INDEX IF EXISTS idx_listing_recommendation_approval_id;

ALTER TABLE listing_recommendation
    DROP COLUMN IF EXISTS approval_id;
