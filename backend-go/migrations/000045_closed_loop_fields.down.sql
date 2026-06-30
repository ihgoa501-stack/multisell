-- Remove closed-loop MVP fields (rollback)
ALTER TABLE listing_task DROP COLUMN IF EXISTS agent_feedback_status;
ALTER TABLE listing_task DROP COLUMN IF EXISTS agent_feedback_note;
ALTER TABLE listing_recommendation DROP COLUMN IF EXISTS status;
