ALTER TABLE problem_case
ADD COLUMN IF NOT EXISTS residual_barrier_status VARCHAR(24) NOT NULL DEFAULT 'unknown'
CHECK (residual_barrier_status IN ('unknown', 'confirmed', 'not_confirmed'));
