-- Owner-private 1688 bookmarks are collected before any market/experiment link.
-- They remain unverified leads until a later governed promotion.
ALTER TABLE sourcing_1688_product
    ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES "user"(id);

UPDATE sourcing_1688_product sp
SET owner_id = COALESCE(
    (SELECT dc.owner_id FROM demand_case dc WHERE dc.id = sp.demand_case_id),
    (SELECT ss.collected_by
     FROM sourcing_1688_snapshot ss
     WHERE ss.sourcing_product_id = sp.id
     ORDER BY ss.id DESC LIMIT 1)
)
WHERE sp.owner_id IS NULL;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sourcing_1688_product WHERE owner_id IS NULL) THEN
        RAISE EXCEPTION 'cannot assign owner_id to every sourcing_1688_product row';
    END IF;
END $$;

ALTER TABLE sourcing_1688_product ALTER COLUMN owner_id SET NOT NULL;
ALTER TABLE sourcing_1688_product DROP CONSTRAINT IF EXISTS sourcing_1688_product_source_url_key;
DROP INDEX IF EXISTS ux_sourcing_1688_offer_id;
CREATE UNIQUE INDEX ux_sourcing_1688_owner_offer
    ON sourcing_1688_product(owner_id, source_offer_id)
    WHERE source_offer_id IS NOT NULL AND source_offer_id <> '';
CREATE INDEX idx_sourcing_1688_owner_status
    ON sourcing_1688_product(owner_id, status);

CREATE TABLE sourcing_1688_task_link (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id),
    experiment_id VARCHAR(40) NOT NULL REFERENCES experiment_case(experiment_id),
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
	status VARCHAR(24) NOT NULL DEFAULT 'linked' CHECK (status IN ('linked','active_workflow','blocked','archived')),
	is_primary BOOLEAN NOT NULL DEFAULT FALSE,
	blocked_reason VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (sourcing_product_id, experiment_id)
);
CREATE INDEX idx_sourcing_1688_task_link_owner
    ON sourcing_1688_task_link(owner_id, created_at DESC);
UPDATE sourcing_1688_task_link l
SET is_primary = TRUE, status = 'active_workflow'
FROM sourcing_1688_product p
WHERE p.id = l.sourcing_product_id AND p.experiment_id = l.experiment_id;
CREATE UNIQUE INDEX ux_sourcing_1688_primary_task
    ON sourcing_1688_task_link(sourcing_product_id) WHERE is_primary;

ALTER TABLE sourcing_1688_snapshot
    ADD COLUMN IF NOT EXISTS extension_version VARCHAR(40) NOT NULL DEFAULT '';

-- A deliberate recapture is a new observation even when page bytes are equal.
ALTER TABLE sourcing_1688_snapshot
    DROP CONSTRAINT IF EXISTS sourcing_1688_snapshot_sourcing_product_id_raw_sha256_key;
DROP INDEX IF EXISTS ux_sourcing_snapshot_hash;
CREATE INDEX IF NOT EXISTS idx_sourcing_snapshot_hash
    ON sourcing_1688_snapshot(sourcing_product_id, raw_sha256);

ALTER TABLE sourcing_1688_snapshot
    DROP CONSTRAINT IF EXISTS ck_sourcing_snapshot_capture_mode;
ALTER TABLE sourcing_1688_snapshot
    ADD CONSTRAINT ck_sourcing_snapshot_capture_mode
    CHECK (capture_mode IN ('controlled_fetch', 'extension_click', 'manual_import', 'legacy_unknown'));

COMMENT ON COLUMN sourcing_1688_snapshot.capture_mode IS
    'Server-assigned provenance. extension_click is an Owner-private quoted page observation; only controlled_fetch can satisfy the governed controlled-collection gate.';

ALTER TABLE sourcing_1688_product
    DROP CONSTRAINT IF EXISTS ck_sourcing_1688_lifecycle_status;
ALTER TABLE sourcing_1688_product
    ADD CONSTRAINT ck_sourcing_1688_lifecycle_status CHECK (lifecycle_status IN (
		'unverified_lead', 'needs_review', 'capture_failed', 'pending_review', 'rejected', 'ready_for_product',
        'editing', 'pending_approval', 'approved_draft'
    ));
