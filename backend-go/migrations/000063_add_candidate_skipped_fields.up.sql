ALTER TABLE candidate_product ADD COLUMN IF NOT EXISTS skipped_fields jsonb DEFAULT '[]'::jsonb;
