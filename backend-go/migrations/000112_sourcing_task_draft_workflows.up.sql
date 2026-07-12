-- One Owner-private 1688 source can serve several approved sourcing tasks.
-- The source observation stays shared, while every task owns its conversion
-- identity, status and resulting internal draft.
ALTER TABLE sourcing_1688_task_link
    ADD COLUMN workflow_status VARCHAR(32) NOT NULL DEFAULT 'needs_review',
    ADD COLUMN draft_id BIGINT,
    ADD COLUMN workflow_updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE sourcing_listing_draft
    DROP CONSTRAINT IF EXISTS sourcing_listing_draft_sourcing_product_id_key;

ALTER TABLE sourcing_listing_draft
    ADD COLUMN task_link_id BIGINT REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT;

-- Preserve every historical single-chain draft by binding it to the exact
-- matching task. Rows predating task links remain legacy-readable with NULL.
UPDATE sourcing_listing_draft d
SET task_link_id = l.id
FROM sourcing_1688_task_link l
WHERE d.task_link_id IS NULL
  AND l.sourcing_product_id = d.sourcing_product_id
  AND l.experiment_id = d.experiment_id
  AND l.owner_id = d.created_by;

CREATE UNIQUE INDEX ux_sourcing_draft_task_link
    ON sourcing_listing_draft(task_link_id)
    WHERE task_link_id IS NOT NULL;
CREATE UNIQUE INDEX ux_sourcing_draft_legacy_source
    ON sourcing_listing_draft(sourcing_product_id)
    WHERE task_link_id IS NULL;

ALTER TABLE sourcing_1688_task_link
    ADD CONSTRAINT fk_sourcing_task_link_draft
    FOREIGN KEY (draft_id) REFERENCES sourcing_listing_draft(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX ux_sourcing_task_link_draft
    ON sourcing_1688_task_link(draft_id)
    WHERE draft_id IS NOT NULL;

UPDATE sourcing_1688_task_link l
SET draft_id = d.id,
    workflow_status = 'converted_to_draft',
    workflow_updated_at = now()
FROM sourcing_listing_draft d
WHERE d.task_link_id = l.id;

UPDATE sourcing_1688_task_link l
SET workflow_status = CASE
        WHEN p.status IN ('reviewed', 'draft_created') AND p.reviewed_by IS NOT NULL
            THEN 'ready_for_draft'
        ELSE 'needs_review'
    END,
    workflow_updated_at = now()
FROM sourcing_1688_product p
WHERE p.id = l.sourcing_product_id
  AND l.draft_id IS NULL;

ALTER TABLE sourcing_1688_task_link
    ADD CONSTRAINT ck_sourcing_task_link_workflow_status CHECK (workflow_status IN (
        'needs_review','ready_for_draft','converting','converted_to_draft',
        'blocked','conversion_failed','reconcile_required','archived'
    ));

COMMENT ON COLUMN sourcing_listing_draft.task_link_id IS
    'Exact Owner sourcing task that owns this draft conversion; NULL is reserved for historical rows without a recoverable task link.';
COMMENT ON COLUMN sourcing_1688_task_link.workflow_status IS
    'Independent per-task sourcing-to-draft workflow state. It must not be inferred from the product primary link.';
