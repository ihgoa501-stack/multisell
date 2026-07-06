DROP INDEX IF EXISTS idx_listing_recommendation_feedback_status;
ALTER TABLE listing_recommendation DROP COLUMN IF EXISTS feedback_note;
ALTER TABLE listing_recommendation DROP COLUMN IF EXISTS feedback_status;
