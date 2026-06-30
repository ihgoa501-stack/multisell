-- listing_task: add agent_feedback fields for closed-loop MVP
ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS agent_feedback_status VARCHAR(20);
ALTER TABLE listing_task ADD COLUMN IF NOT EXISTS agent_feedback_note TEXT DEFAULT '';

-- listing_recommendation: add lifecycle status
ALTER TABLE listing_recommendation ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'pending';
