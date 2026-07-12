ALTER TABLE demand_case
    ADD COLUMN IF NOT EXISTS target_locale VARCHAR(35) NOT NULL DEFAULT '';

-- Preserve the exact collector response bytes. JSONB normalizes whitespace and
-- key order, which breaks the original-byte SHA-256 evidence contract.
ALTER TABLE sourcing_1688_snapshot
    ALTER COLUMN raw_payload TYPE BYTEA
    USING convert_to(raw_payload::text, 'UTF8');

ALTER TABLE sourcing_1688_snapshot
    ADD COLUMN IF NOT EXISTS collection_request_id VARCHAR(80) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX ux_sourcing_snapshot_collection_request
    ON sourcing_1688_snapshot(collection_request_id)
    WHERE collection_request_id <> '';

-- One unresolved external write is allowed per source. A different
-- idempotency key must never create a parallel approval/execution path.
CREATE UNIQUE INDEX ux_sourcing_publish_one_active
    ON sourcing_publish_attempt(sourcing_product_id)
    WHERE status IN ('pending_approval', 'approved', 'executing', 'submitted', 'reconcile_required');

-- Freeze both sides of product_id-changing UPDATEs while approval is pending.
CREATE OR REPLACE FUNCTION block_pending_sourcing_draft_content_mutation()
RETURNS trigger AS $$
DECLARE
    old_product_id BIGINT;
    new_product_id BIGINT;
    is_pending BOOLEAN;
BEGIN
    IF TG_TABLE_NAME = 'product' THEN
        old_product_id := OLD.id;
        new_product_id := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE NEW.id END;
    ELSIF TG_TABLE_NAME = 'product_listing' THEN
        SELECT product_id INTO old_product_id FROM sourcing_listing_draft WHERE listing_id = OLD.id;
        IF TG_OP <> 'DELETE' THEN
            SELECT product_id INTO new_product_id FROM sourcing_listing_draft WHERE listing_id = NEW.id;
        END IF;
    ELSE
        old_product_id := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE OLD.product_id END;
        new_product_id := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE NEW.product_id END;
    END IF;
    SELECT EXISTS (
        SELECT 1 FROM sourcing_listing_draft
        WHERE product_id IN (old_product_id, new_product_id) AND approval_status = 'pending'
    ) INTO is_pending;
    IF is_pending THEN
        RAISE EXCEPTION 'sourcing draft content is frozen while Owner approval is pending';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END;
$$ LANGUAGE plpgsql;

-- Historical cross-Owner duplicate candidates must not expose another Owner's
-- internal source ID. New candidates are also filtered in the service.
DELETE FROM sourcing_1688_duplicate_candidate c
USING sourcing_1688_product source_product,
      sourcing_1688_product matched_product,
      demand_case source_case,
      demand_case matched_case
WHERE c.source_product_id = source_product.id
  AND c.matched_product_id = matched_product.id
  AND source_product.demand_case_id = source_case.id
  AND matched_product.demand_case_id = matched_case.id
  AND source_case.owner_id <> matched_case.owner_id;

INSERT INTO permission (name, code, module, description)
VALUES ('真实渠道发布', 'listing.publish', 'listing', '审批并执行真实渠道发布')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role r CROSS JOIN permission p
WHERE r.code = 'admin' AND p.code = 'listing.publish'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
