-- Approved product opportunities are the authority for entering sourcing.
-- Existing experiment-only links remain visible as legacy trace records but
-- are deliberately not upgraded because approval cannot be inferred.
ALTER TABLE sourcing_1688_task_link
    ADD COLUMN product_opportunity_id BIGINT REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    ADD COLUMN opportunity_decision_id BIGINT REFERENCES product_opportunity_decision(id) ON DELETE RESTRICT,
    ADD COLUMN authority_kind VARCHAR(24) NOT NULL DEFAULT 'legacy_experiment'
        CHECK (authority_kind IN ('legacy_experiment','product_opportunity'));

CREATE INDEX idx_sourcing_task_link_opportunity
    ON sourcing_1688_task_link(owner_id, product_opportunity_id)
    WHERE product_opportunity_id IS NOT NULL;
CREATE UNIQUE INDEX ux_sourcing_task_link_product_opportunity
    ON sourcing_1688_task_link(sourcing_product_id, product_opportunity_id)
    WHERE product_opportunity_id IS NOT NULL;

ALTER TABLE sourcing_1688_task_link ADD CONSTRAINT ck_sourcing_task_link_authority
CHECK (
    (authority_kind = 'legacy_experiment' AND product_opportunity_id IS NULL AND opportunity_decision_id IS NULL)
 OR (authority_kind = 'product_opportunity' AND product_opportunity_id IS NOT NULL AND opportunity_decision_id IS NOT NULL)
);
