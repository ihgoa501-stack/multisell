-- Reverse of 000010: undo the ALTER TABLE ADD COLUMN changes.
-- The agent_nudge table itself is owned by 000001 and is NOT dropped here.
ALTER TABLE agent_nudge DROP COLUMN IF EXISTS trust_score;
ALTER TABLE agent_nudge DROP COLUMN IF EXISTS decided_at;
