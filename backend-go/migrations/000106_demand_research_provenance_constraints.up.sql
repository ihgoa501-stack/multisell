-- Demand research provenance is only trustworthy when every evidence row is
-- bound to one immutable snapshot from the same Owner, case, run and source.
-- Fail before changing the schema when historical rows violate that contract;
-- operators can audit the rows without this migration deleting or rewriting
-- business data.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM demand_evidence WHERE snapshot_id = 0) THEN
        RAISE EXCEPTION '000106 preflight: demand_evidence contains snapshot_id=0';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM demand_evidence e
        LEFT JOIN demand_research_snapshot s ON s.id = e.snapshot_id
        WHERE s.id IS NULL
    ) THEN
        RAISE EXCEPTION '000106 preflight: demand_evidence contains orphan snapshot references';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM demand_research_snapshot s
        LEFT JOIN demand_case c ON c.id = s.demand_case_id
        LEFT JOIN demand_research_batch b ON b.id = s.batch_id
        WHERE c.id IS NULL OR b.id IS NULL
           OR s.owner_id <> c.owner_id
           OR s.owner_id <> b.owner_id
    ) THEN
        RAISE EXCEPTION '000106 preflight: demand research snapshot Owner/case/batch mismatch';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM demand_evidence e
        JOIN demand_research_snapshot s ON s.id = e.snapshot_id
        WHERE e.demand_case_id <> s.demand_case_id
           OR e.run_id IS DISTINCT FROM s.run_id
           OR e.source_uri IS DISTINCT FROM s.source_uri
           OR e.observed_at IS DISTINCT FROM s.collected_at
    ) THEN
        RAISE EXCEPTION '000106 preflight: demand evidence provenance differs from its snapshot';
    END IF;
END;
$$;

ALTER TABLE demand_evidence
    ADD CONSTRAINT ck_demand_evidence_snapshot_positive CHECK (snapshot_id > 0),
    ADD CONSTRAINT fk_demand_evidence_research_snapshot
    FOREIGN KEY (snapshot_id)
    REFERENCES demand_research_snapshot(id)
    ON DELETE RESTRICT;

CREATE OR REPLACE FUNCTION enforce_demand_research_snapshot_scope()
RETURNS trigger AS $$
DECLARE
    case_owner BIGINT;
    batch_owner BIGINT;
BEGIN
    SELECT owner_id INTO case_owner FROM demand_case WHERE id = NEW.demand_case_id;
    SELECT owner_id INTO batch_owner FROM demand_research_batch WHERE id = NEW.batch_id;

    IF case_owner IS NULL OR batch_owner IS NULL
       OR NEW.owner_id <> case_owner
       OR NEW.owner_id <> batch_owner THEN
        RAISE EXCEPTION 'demand research snapshot Owner, case and batch must match';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_demand_research_snapshot_scope
BEFORE INSERT OR UPDATE ON demand_research_snapshot
FOR EACH ROW EXECUTE FUNCTION enforce_demand_research_snapshot_scope();

-- Once a snapshot exists, changing the Owner of its case or batch would make
-- the already-recorded provenance false. Reject only that identity mutation;
-- ordinary case and batch lifecycle updates remain allowed.
CREATE OR REPLACE FUNCTION block_demand_research_scope_owner_mutation()
RETURNS trigger AS $$
BEGIN
    IF NEW.owner_id IS DISTINCT FROM OLD.owner_id THEN
        IF TG_TABLE_NAME = 'demand_case' AND EXISTS (
            SELECT 1 FROM demand_research_snapshot WHERE demand_case_id = OLD.id
        ) THEN
            RAISE EXCEPTION 'cannot change demand case Owner after research snapshots exist';
        END IF;
        IF TG_TABLE_NAME = 'demand_research_batch' AND EXISTS (
            SELECT 1 FROM demand_research_snapshot WHERE batch_id = OLD.id
        ) THEN
            RAISE EXCEPTION 'cannot change research batch Owner after snapshots exist';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_demand_case_research_owner_immutable
BEFORE UPDATE OF owner_id ON demand_case
FOR EACH ROW EXECUTE FUNCTION block_demand_research_scope_owner_mutation();

CREATE TRIGGER trg_demand_research_batch_owner_immutable
BEFORE UPDATE OF owner_id ON demand_research_batch
FOR EACH ROW EXECUTE FUNCTION block_demand_research_scope_owner_mutation();

CREATE OR REPLACE FUNCTION enforce_demand_evidence_snapshot_provenance()
RETURNS trigger AS $$
DECLARE
    snapshot_row demand_research_snapshot%ROWTYPE;
BEGIN
    SELECT * INTO snapshot_row
    FROM demand_research_snapshot
    WHERE id = NEW.snapshot_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'demand evidence must reference an existing research snapshot';
    END IF;
    IF NEW.demand_case_id <> snapshot_row.demand_case_id
       OR NEW.run_id IS DISTINCT FROM snapshot_row.run_id
       OR NEW.source_uri IS DISTINCT FROM snapshot_row.source_uri
       OR NEW.observed_at IS DISTINCT FROM snapshot_row.collected_at THEN
        RAISE EXCEPTION 'demand evidence case, run, source and observed time must match its snapshot';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_demand_evidence_snapshot_provenance
BEFORE INSERT OR UPDATE ON demand_evidence
FOR EACH ROW EXECUTE FUNCTION enforce_demand_evidence_snapshot_provenance();

CREATE OR REPLACE FUNCTION block_demand_research_snapshot_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'demand research snapshots are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_demand_research_snapshot_append_only
BEFORE UPDATE OR DELETE ON demand_research_snapshot
FOR EACH ROW EXECUTE FUNCTION block_demand_research_snapshot_mutation();
