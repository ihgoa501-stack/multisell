-- 000009_personal_rule
-- ⚠️ personal_rule table already created in 000001_init_schema.up.sql.
-- This migration adds any columns not present in the original schema.
-- Using ALTER TABLE ADD COLUMN IF NOT EXISTS for idempotency.

ALTER TABLE personal_rule ADD COLUMN IF NOT EXISTS decision_point VARCHAR(80);
ALTER TABLE personal_rule ADD COLUMN IF NOT EXISTS description TEXT;

CREATE INDEX IF NOT EXISTS idx_personal_rule_user ON personal_rule(user_id, agent_id);
