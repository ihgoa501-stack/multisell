ALTER TABLE sourcing_publish_attempt
    ADD COLUMN task_link_id BIGINT REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT;
ALTER TABLE sourcing_listing_draft
    ADD COLUMN editable_version BIGINT NOT NULL DEFAULT 1;

UPDATE sourcing_publish_attempt a
SET task_link_id = d.task_link_id
FROM sourcing_listing_draft d
WHERE d.id = a.draft_id AND d.task_link_id IS NOT NULL;

DROP INDEX IF EXISTS ux_sourcing_publish_one_active;
CREATE UNIQUE INDEX ux_sourcing_publish_task_one_active
    ON sourcing_publish_attempt(task_link_id)
    WHERE task_link_id IS NOT NULL
      AND status IN ('pending_approval','approved','executing','submitted','reconcile_required');
CREATE UNIQUE INDEX ux_sourcing_publish_legacy_one_active
    ON sourcing_publish_attempt(sourcing_product_id)
    WHERE task_link_id IS NULL
      AND status IN ('pending_approval','approved','executing','submitted','reconcile_required');

ALTER TABLE sourcing_1688_task_link DROP CONSTRAINT IF EXISTS ck_sourcing_task_link_workflow_status;
ALTER TABLE sourcing_1688_task_link ADD CONSTRAINT ck_sourcing_task_link_workflow_status CHECK (workflow_status IN (
    'needs_review','ready_for_draft','converting','converted_to_draft','editing',
    'pending_approval','approved_draft','publish_pending','publish_approved',
    'publishing','submitted','succeeded','blocked','conversion_failed',
    'publish_failed','reconcile_required','archived'
));

-- Draft approval now advances only the task that owns the exact draft. The
-- product-level lifecycle is updated only for the primary link as a legacy
-- compatibility projection and is no longer the authority for other tasks.
CREATE OR REPLACE FUNCTION apply_sourcing_1688_draft_approval()
RETURNS trigger AS $$
DECLARE
    linked_draft sourcing_listing_draft%ROWTYPE;
    linked_task sourcing_1688_task_link%ROWTYPE;
    selected_owner BIGINT;
    current_listing_status VARCHAR(32);
BEGIN
    IF NEW.request_type <> 'sourcing_1688_draft' OR OLD.status IS NOT DISTINCT FROM NEW.status THEN RETURN NEW; END IF;
    IF OLD.status <> 'pending' OR NEW.status NOT IN ('approved','rejected') THEN
        RAISE EXCEPTION 'invalid sourcing 1688 draft approval transition';
    END IF;
    SELECT * INTO STRICT linked_draft FROM sourcing_listing_draft
      WHERE id = NEW.target_id AND NEW.target_type = 'sourcing_listing_draft'
        AND approval_id = NEW.id AND product_id = NEW.product_id;
    IF linked_draft.task_link_id IS NULL THEN
        RAISE EXCEPTION 'task-bound sourcing draft approval is required';
    END IF;
    SELECT * INTO STRICT linked_task FROM sourcing_1688_task_link
      WHERE id = linked_draft.task_link_id AND sourcing_product_id = linked_draft.sourcing_product_id
        AND draft_id = linked_draft.id;
    SELECT dc.owner_id, pl.status INTO STRICT selected_owner, current_listing_status
      FROM demand_case dc JOIN product_listing pl ON pl.id = linked_draft.listing_id
      WHERE dc.id = linked_task.demand_case_id;
    IF NEW.reviewer_user_id IS NULL OR NEW.reviewer_user_id <> selected_owner THEN
        RAISE EXCEPTION 'only the task Owner may review this draft';
    END IF;
    IF current_listing_status <> 'draft' THEN RAISE EXCEPTION 'sourcing approval target is no longer an internal draft'; END IF;

    UPDATE sourcing_listing_draft SET approval_status = NEW.status,
      approval_rejection_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
      WHERE id = linked_draft.id;
    UPDATE sourcing_1688_task_link SET
      workflow_status = CASE WHEN NEW.status = 'approved' THEN 'approved_draft' ELSE 'editing' END,
      blocked_reason = CASE WHEN NEW.status = 'rejected' THEN COALESCE(NEW.review_note,'') ELSE '' END,
      workflow_updated_at = now()
      WHERE id = linked_task.id AND workflow_status = 'pending_approval';
    IF NOT FOUND THEN RAISE EXCEPTION 'task draft is not pending approval'; END IF;

    IF linked_task.is_primary THEN
      UPDATE sourcing_1688_product SET
        lifecycle_status = CASE WHEN NEW.status = 'approved' THEN 'approved_draft' ELSE 'editing' END,
        lifecycle_actor_id = NEW.reviewer_user_id,
        lifecycle_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END,
        lifecycle_updated_at = now()
        WHERE id = linked_draft.sourcing_product_id;
    END IF;
    INSERT INTO operation_log(module,action,resource_id,content,operator,user_id,result,trigger_type,approval_id,entity_type,entity_id,created_at)
    VALUES ('sourcing1688','approval.review',linked_draft.id::text,COALESCE(NEW.review_note,''),
      COALESCE(NEW.reviewer,NEW.reviewer_user_id::text),NEW.reviewer_user_id,'success','owner_approval',NEW.id,
      'sourcing_listing_draft',linked_draft.id,now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON COLUMN sourcing_publish_attempt.task_link_id IS
  'Exact task/draft workflow that owns this publish approval and external-write attempt.';
COMMENT ON COLUMN sourcing_listing_draft.editable_version IS
  'Optimistic-lock version for exact task draft editing; incremented on every successful PUT.';
