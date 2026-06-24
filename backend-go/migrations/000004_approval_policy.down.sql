-- ⚠️ Reverse 000004: drops approval_policy_rule table.
-- WARNING: This also destroys all policy rules created at runtime.
-- Guard: refuse if any actions reference the table's data.
-- If you need to preserve runtime data, manually back up first.
DROP TABLE IF EXISTS approval_policy_rule;
