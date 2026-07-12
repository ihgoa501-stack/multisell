ALTER TABLE agent_trust_score
    DROP COLUMN IF EXISTS listing_feedback_rate,
    DROP COLUMN IF EXISTS feedback_rejected,
    DROP COLUMN IF EXISTS feedback_adopted;
