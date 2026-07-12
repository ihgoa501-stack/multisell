ALTER TABLE sourcing_1688_product
    ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    ADD COLUMN IF NOT EXISTS lifecycle_actor_id BIGINT REFERENCES "user"(id),
    ADD COLUMN IF NOT EXISTS lifecycle_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lifecycle_updated_at TIMESTAMPTZ;

UPDATE sourcing_1688_product
SET lifecycle_status = CASE status
    WHEN 'rejected' THEN 'rejected'
    WHEN 'reviewed' THEN 'ready_for_product'
    WHEN 'draft_created' THEN 'editing'
    ELSE 'pending_review'
END,
lifecycle_updated_at = COALESCE(updated_at, now())
WHERE lifecycle_updated_at IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_sourcing_1688_lifecycle_status'
          AND conrelid = 'sourcing_1688_product'::regclass
    ) THEN
        ALTER TABLE sourcing_1688_product
            ADD CONSTRAINT ck_sourcing_1688_lifecycle_status CHECK (lifecycle_status IN (
                'capture_failed', 'pending_review', 'rejected', 'ready_for_product',
                'editing', 'pending_approval', 'approved_draft'
            ));
    END IF;
END;
$$;

ALTER TABLE sourcing_listing_draft
    ADD COLUMN IF NOT EXISTS approval_id BIGINT REFERENCES approval_request(id),
    ADD COLUMN IF NOT EXISTS approval_status VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS approval_rejection_reason TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS ux_sourcing_listing_draft_approval
    ON sourcing_listing_draft(approval_id) WHERE approval_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sourcing_1688_lifecycle
    ON sourcing_1688_product(lifecycle_status);

CREATE OR REPLACE FUNCTION sync_sourcing_1688_lifecycle_from_legacy_status()
RETURNS trigger AS $$
BEGIN
    IF NEW.status = 'pending_review' AND OLD.status IS DISTINCT FROM NEW.status THEN
        NEW.lifecycle_status := 'pending_review';
        NEW.lifecycle_actor_id := NULL;
        NEW.lifecycle_reason := '';
        NEW.lifecycle_updated_at := now();
    ELSIF NEW.status = 'reviewed' AND OLD.status IS DISTINCT FROM NEW.status THEN
        NEW.lifecycle_status := 'ready_for_product';
        NEW.lifecycle_updated_at := now();
    ELSIF NEW.status = 'draft_created' AND OLD.status IS DISTINCT FROM NEW.status THEN
        NEW.lifecycle_status := 'editing';
        NEW.lifecycle_updated_at := now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sourcing_1688_lifecycle_sync ON sourcing_1688_product;
CREATE TRIGGER trg_sourcing_1688_lifecycle_sync
BEFORE UPDATE OF status ON sourcing_1688_product
FOR EACH ROW EXECUTE FUNCTION sync_sourcing_1688_lifecycle_from_legacy_status();

-- Keep the canonical approval endpoint safe as well as the sourcing-specific
-- endpoint. A generic approval review may close this request only when the
-- reviewer is the selected market Owner and the target is still an internal
-- draft; it can never promote product_listing out of draft.
CREATE OR REPLACE FUNCTION apply_sourcing_1688_draft_approval()
RETURNS trigger AS $$
DECLARE
    linked_draft sourcing_listing_draft%ROWTYPE;
    selected_owner BIGINT;
    current_listing_status VARCHAR(32);
BEGIN
    IF NEW.request_type <> 'sourcing_1688_draft'
       OR OLD.status IS NOT DISTINCT FROM NEW.status THEN
        RETURN NEW;
    END IF;
    IF OLD.status <> 'pending' OR NEW.status NOT IN ('approved', 'rejected') THEN
        RAISE EXCEPTION 'invalid sourcing 1688 draft approval transition';
    END IF;

    SELECT * INTO STRICT linked_draft
    FROM sourcing_listing_draft
    WHERE id = NEW.target_id
      AND NEW.target_type = 'sourcing_listing_draft'
      AND approval_id = NEW.id
      AND product_id = NEW.product_id;

    SELECT dc.owner_id, pl.status
    INTO STRICT selected_owner, current_listing_status
    FROM sourcing_1688_product sp
    JOIN demand_case dc ON dc.id = sp.demand_case_id
    JOIN product_listing pl ON pl.id = linked_draft.listing_id
    WHERE sp.id = linked_draft.sourcing_product_id;

    IF NEW.reviewer_user_id IS NULL OR NEW.reviewer_user_id <> selected_owner THEN
        RAISE EXCEPTION 'only the selected market Owner may review this draft';
    END IF;
    IF current_listing_status <> 'draft' THEN
        RAISE EXCEPTION 'sourcing approval target is no longer an internal draft';
    END IF;

    UPDATE sourcing_listing_draft
    SET approval_status = NEW.status,
        approval_rejection_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
    WHERE id = linked_draft.id;

    UPDATE sourcing_1688_product
    SET lifecycle_status = CASE WHEN NEW.status = 'approved' THEN 'approved_draft' ELSE 'editing' END,
        lifecycle_actor_id = NEW.reviewer_user_id,
        lifecycle_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END,
        lifecycle_updated_at = now()
    WHERE id = linked_draft.sourcing_product_id
      AND lifecycle_status = 'pending_approval';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'sourcing draft is not pending approval';
    END IF;

    INSERT INTO operation_log (
        module, action, resource_id, content, operator, user_id, result,
        trigger_type, approval_id, entity_type, entity_id, created_at
    ) VALUES (
        'sourcing1688', 'approval.review', linked_draft.id::text,
        COALESCE(NEW.review_note, ''), COALESCE(NEW.reviewer, NEW.reviewer_user_id::text),
        NEW.reviewer_user_id, 'success', 'owner_approval', NEW.id,
        'sourcing_listing_draft', linked_draft.id, now()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_apply_sourcing_1688_draft_approval ON approval_request;
CREATE TRIGGER trg_apply_sourcing_1688_draft_approval
AFTER UPDATE OF status ON approval_request
FOR EACH ROW EXECUTE FUNCTION apply_sourcing_1688_draft_approval();
