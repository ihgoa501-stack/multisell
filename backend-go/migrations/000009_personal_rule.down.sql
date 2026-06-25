-- Reverse of 000009: undo the ALTER TABLE ADD COLUMN changes.
-- The personal_rule table itself is owned by 000001 and is NOT dropped here.
ALTER TABLE personal_rule DROP COLUMN IF EXISTS decision_point;
ALTER TABLE personal_rule DROP COLUMN IF EXISTS description;
