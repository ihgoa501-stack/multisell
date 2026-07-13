ALTER TABLE demand_verdict
    ADD COLUMN IF NOT EXISTS evidence_max_id BIGINT NOT NULL DEFAULT 0;

UPDATE demand_verdict v
SET evidence_max_id = COALESCE((
    SELECT MAX(e.id)
    FROM demand_evidence e
    WHERE e.demand_case_id = v.demand_case_id
      AND e.created_at <= v.created_at
), 0)
WHERE evidence_max_id = 0;
