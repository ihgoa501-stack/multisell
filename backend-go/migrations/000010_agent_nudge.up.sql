-- 000010_agent_nudge
-- ⚠️ agent_nudge table already created in 000001_init_schema.up.sql.
-- This migration adds any columns not present in the original schema.
-- Using ALTER TABLE ADD COLUMN IF NOT EXISTS for idempotency.

ALTER TABLE agent_nudge ADD COLUMN IF NOT EXISTS trust_score NUMERIC(5,4);
ALTER TABLE agent_nudge ADD COLUMN IF NOT EXISTS decided_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_nudge_user ON agent_nudge(user_id);
CREATE INDEX IF NOT EXISTS idx_nudge_agent ON agent_nudge(agent_id);
CREATE INDEX IF NOT EXISTS idx_nudge_status ON agent_nudge(status);
