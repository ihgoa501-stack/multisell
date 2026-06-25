-- ⚠️ Reverse 000005: drops agent_trust_score table.
-- WARNING: Destroys all trust scores accumulated at runtime.
-- Guard: none — this is the owner migration for agent_trust_score.
DROP TABLE IF EXISTS agent_trust_score;
