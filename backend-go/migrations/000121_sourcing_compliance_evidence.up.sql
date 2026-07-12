CREATE TABLE sourcing_compliance_evidence (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id) ON DELETE RESTRICT,
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT,
    product_opportunity_id BIGINT NOT NULL REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    source_snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id) ON DELETE RESTRICT,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE RESTRICT,
    internal_sku_id BIGINT REFERENCES sku(id) ON DELETE RESTRICT,
    country_code VARCHAR(16) NOT NULL,
    channel_code VARCHAR(64) NOT NULL,
    requirement_code VARCHAR(120) NOT NULL,
    requirement_text TEXT NOT NULL,
    evidence_source TEXT NOT NULL,
    truth_status VARCHAR(24) NOT NULL CHECK (truth_status IN ('actual','quoted','estimated','unknown','mock','inferred')),
    scope TEXT NOT NULL,
    issued_at TIMESTAMPTZ,
    observed_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by BIGINT,
    revocation_reason TEXT NOT NULL DEFAULT '',
    review_status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending','approved','rejected')),
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at IS NULL OR expires_at > observed_at),
    CHECK ((review_status = 'pending' AND reviewed_by IS NULL AND reviewed_at IS NULL) OR
           (review_status IN ('approved','rejected') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)),
    CHECK ((revoked_at IS NULL AND revoked_by IS NULL AND revocation_reason = '') OR
           (revoked_at IS NOT NULL AND revoked_by IS NOT NULL AND revocation_reason <> ''))
);

CREATE INDEX idx_sourcing_compliance_owner_task ON sourcing_compliance_evidence(owner_id, sourcing_product_id, task_link_id);
CREATE INDEX idx_sourcing_compliance_gate ON sourcing_compliance_evidence(owner_id, task_link_id, requirement_code, review_status, truth_status, expires_at) WHERE revoked_at IS NULL;

CREATE OR REPLACE FUNCTION protect_sourcing_compliance_evidence()
RETURNS trigger AS $$
BEGIN
    IF OLD.owner_id IS DISTINCT FROM NEW.owner_id OR
       OLD.sourcing_product_id IS DISTINCT FROM NEW.sourcing_product_id OR
       OLD.task_link_id IS DISTINCT FROM NEW.task_link_id OR
       OLD.product_opportunity_id IS DISTINCT FROM NEW.product_opportunity_id OR
       OLD.source_snapshot_id IS DISTINCT FROM NEW.source_snapshot_id OR
       OLD.product_id IS DISTINCT FROM NEW.product_id OR
       OLD.internal_sku_id IS DISTINCT FROM NEW.internal_sku_id OR
       OLD.country_code IS DISTINCT FROM NEW.country_code OR
       OLD.channel_code IS DISTINCT FROM NEW.channel_code OR
       OLD.requirement_code IS DISTINCT FROM NEW.requirement_code OR
       OLD.requirement_text IS DISTINCT FROM NEW.requirement_text OR
       OLD.evidence_source IS DISTINCT FROM NEW.evidence_source OR
       OLD.truth_status IS DISTINCT FROM NEW.truth_status OR
       OLD.scope IS DISTINCT FROM NEW.scope OR
       OLD.issued_at IS DISTINCT FROM NEW.issued_at OR
       OLD.observed_at IS DISTINCT FROM NEW.observed_at OR
       OLD.expires_at IS DISTINCT FROM NEW.expires_at OR
       OLD.created_by IS DISTINCT FROM NEW.created_by OR
       OLD.created_at IS DISTINCT FROM NEW.created_at THEN
        RAISE EXCEPTION 'sourcing compliance fact identity and evidence are immutable';
    END IF;
    IF OLD.review_status <> 'pending' AND (OLD.review_status IS DISTINCT FROM NEW.review_status OR OLD.reviewed_by IS DISTINCT FROM NEW.reviewed_by OR OLD.reviewed_at IS DISTINCT FROM NEW.reviewed_at OR OLD.review_notes IS DISTINCT FROM NEW.review_notes) THEN
        RAISE EXCEPTION 'sourcing compliance review is immutable once decided';
    END IF;
    IF OLD.revoked_at IS NOT NULL AND (OLD.revoked_at IS DISTINCT FROM NEW.revoked_at OR OLD.revoked_by IS DISTINCT FROM NEW.revoked_by OR OLD.revocation_reason IS DISTINCT FROM NEW.revocation_reason) THEN
        RAISE EXCEPTION 'sourcing compliance revocation is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_protect_sourcing_compliance_evidence
BEFORE UPDATE ON sourcing_compliance_evidence
FOR EACH ROW EXECUTE FUNCTION protect_sourcing_compliance_evidence();

CREATE OR REPLACE FUNCTION validate_sourcing_compliance_authority()
RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM sourcing_1688_task_link tl
        JOIN sourcing_1688_product sp ON sp.id = tl.sourcing_product_id
        JOIN demand_case dc ON dc.id = tl.demand_case_id
        WHERE tl.id = NEW.task_link_id AND tl.sourcing_product_id = NEW.sourcing_product_id
          AND tl.owner_id = NEW.owner_id AND sp.owner_id = NEW.owner_id
          AND tl.product_opportunity_id = NEW.product_opportunity_id
          AND sp.snapshot_id = NEW.source_snapshot_id AND sp.product_id = NEW.product_id
          AND lower(dc.region) = lower(NEW.country_code)
          AND lower(dc.sales_channel) = lower(NEW.channel_code)
    ) THEN RAISE EXCEPTION 'compliance evidence authority chain mismatch'; END IF;
    IF NEW.internal_sku_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM sku WHERE id = NEW.internal_sku_id AND product_id = NEW.product_id) THEN
        RAISE EXCEPTION 'compliance evidence SKU does not belong to product';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_validate_sourcing_compliance_authority
BEFORE INSERT ON sourcing_compliance_evidence
FOR EACH ROW EXECUTE FUNCTION validate_sourcing_compliance_authority();
