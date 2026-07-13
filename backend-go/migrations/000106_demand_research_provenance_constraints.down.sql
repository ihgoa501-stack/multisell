DROP TRIGGER IF EXISTS trg_demand_research_snapshot_append_only ON demand_research_snapshot;
DROP FUNCTION IF EXISTS block_demand_research_snapshot_mutation();

DROP TRIGGER IF EXISTS trg_demand_evidence_snapshot_provenance ON demand_evidence;
DROP FUNCTION IF EXISTS enforce_demand_evidence_snapshot_provenance();

DROP TRIGGER IF EXISTS trg_demand_research_snapshot_scope ON demand_research_snapshot;
DROP FUNCTION IF EXISTS enforce_demand_research_snapshot_scope();

DROP TRIGGER IF EXISTS trg_demand_research_batch_owner_immutable ON demand_research_batch;
DROP TRIGGER IF EXISTS trg_demand_case_research_owner_immutable ON demand_case;
DROP FUNCTION IF EXISTS block_demand_research_scope_owner_mutation();

ALTER TABLE demand_evidence
    DROP CONSTRAINT IF EXISTS fk_demand_evidence_research_snapshot,
    DROP CONSTRAINT IF EXISTS ck_demand_evidence_snapshot_positive;
