ALTER TABLE listing_recommendation ADD COLUMN IF NOT EXISTS feedback_status VARCHAR(20) NOT NULL DEFAULT 'pending';
ALTER TABLE listing_recommendation ADD COLUMN IF NOT EXISTS feedback_note TEXT;
CREATE INDEX IF NOT EXISTS idx_listing_recommendation_feedback_status ON listing_recommendation(feedback_status);
