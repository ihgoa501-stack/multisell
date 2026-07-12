DROP INDEX IF EXISTS idx_listing_recommendation_approval;

ALTER TABLE listing_recommendation
    DROP COLUMN IF EXISTS approval_id;
