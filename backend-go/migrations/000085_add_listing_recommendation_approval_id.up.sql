ALTER TABLE listing_recommendation
    ADD COLUMN IF NOT EXISTS approval_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_listing_recommendation_approval_id
    ON listing_recommendation(approval_id);
