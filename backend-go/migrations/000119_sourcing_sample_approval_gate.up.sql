ALTER TABLE sourcing_1688_task_link
    ADD COLUMN sample_policy VARCHAR(16) NOT NULL DEFAULT 'required',
    ADD COLUMN sample_waiver_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN sample_waived_by BIGINT REFERENCES "user"(id) ON DELETE RESTRICT,
    ADD COLUMN sample_waived_at TIMESTAMPTZ;

UPDATE sourcing_1688_task_link
SET sample_policy = 'waived',
    sample_waiver_reason = 'legacy_before_sample_gate',
    sample_waived_by = owner_id,
    sample_waived_at = now();

ALTER TABLE sourcing_1688_task_link
    ADD CONSTRAINT ck_sourcing_task_sample_policy
    CHECK (
        (sample_policy = 'required' AND sample_waiver_reason = '' AND sample_waived_by IS NULL AND sample_waived_at IS NULL)
        OR
        (sample_policy = 'waived' AND length(trim(sample_waiver_reason)) > 0 AND sample_waived_by IS NOT NULL AND sample_waived_at IS NOT NULL)
    );

COMMENT ON COLUMN sourcing_1688_task_link.sample_policy IS
    'required means an accepted physical sample is mandatory before draft approval; waived is an explicit immutable Owner exception.';
