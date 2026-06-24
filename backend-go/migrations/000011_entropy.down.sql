-- Reverse of 000011: undo the ALTER TABLE ADD COLUMN changes.
-- rule_conflict and spc_control_limit are owned by 000001 and are NOT dropped here.
ALTER TABLE rule_conflict DROP COLUMN IF EXISTS status;
ALTER TABLE rule_conflict DROP COLUMN IF EXISTS resolution;
ALTER TABLE rule_conflict DROP COLUMN IF EXISTS payload;
ALTER TABLE rule_conflict DROP COLUMN IF EXISTS created_at;
ALTER TABLE rule_conflict DROP COLUMN IF EXISTS resolved_at;

ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS decision_point;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS mean;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS std_dev;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS upper_control;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS lower_control;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS upper_warning;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS lower_warning;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS sample_size;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS out_of_control;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS last_value;
ALTER TABLE spc_control_limit DROP COLUMN IF EXISTS calculated_at;
