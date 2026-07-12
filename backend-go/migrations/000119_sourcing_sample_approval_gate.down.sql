ALTER TABLE sourcing_1688_task_link
    DROP CONSTRAINT IF EXISTS ck_sourcing_task_sample_policy,
    DROP COLUMN IF EXISTS sample_waived_at,
    DROP COLUMN IF EXISTS sample_waived_by,
    DROP COLUMN IF EXISTS sample_waiver_reason,
    DROP COLUMN IF EXISTS sample_policy;
