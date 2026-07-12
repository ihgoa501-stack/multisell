-- ADR-001 unit 3: freeze the canonical supplier -> internal -> channel SKU
-- identity chain used by one exact Owner sourcing task and listing draft.
-- The JSON embedded in product_listing is presentation evidence; this table is
-- the authoritative, queryable and append-only identity record.
CREATE TABLE sourcing_sku_mapping (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id) ON DELETE RESTRICT,
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id) ON DELETE RESTRICT,
    snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id) ON DELETE RESTRICT,
    product_opportunity_id BIGINT NOT NULL REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE RESTRICT,
    supplier_sku VARCHAR(240) NOT NULL CHECK (length(trim(supplier_sku)) > 0),
    internal_sku_id BIGINT NOT NULL REFERENCES sku(id) ON DELETE RESTRICT,
    internal_sku VARCHAR(240) NOT NULL CHECK (length(trim(internal_sku)) > 0),
    channel_sku VARCHAR(240) NOT NULL CHECK (length(trim(channel_sku)) > 0),
    platform_id BIGINT NOT NULL REFERENCES platform(id) ON DELETE RESTRICT,
    listing_id BIGINT NOT NULL REFERENCES product_listing(id) ON DELETE RESTRICT,
    version BIGINT NOT NULL CHECK (version > 0),
    content_hash VARCHAR(64) NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    created_by BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, task_link_id, version, supplier_sku),
    UNIQUE (owner_id, task_link_id, version, internal_sku_id),
    UNIQUE (owner_id, listing_id, version, channel_sku),
    UNIQUE (owner_id, task_link_id, version, content_hash)
);

CREATE INDEX idx_sourcing_sku_mapping_source
    ON sourcing_sku_mapping(owner_id, sourcing_product_id, task_link_id, version);
CREATE INDEX idx_sourcing_sku_mapping_internal
    ON sourcing_sku_mapping(owner_id, internal_sku_id);
CREATE INDEX idx_sourcing_sku_mapping_channel
    ON sourcing_sku_mapping(owner_id, platform_id, channel_sku);

CREATE OR REPLACE FUNCTION validate_sourcing_sku_mapping_binding()
RETURNS trigger AS $$
DECLARE
    linked_owner BIGINT;
    linked_source BIGINT;
    linked_opportunity BIGINT;
    linked_snapshot_source BIGINT;
    linked_sku_product BIGINT;
    linked_sku_code TEXT;
    linked_listing_product BIGINT;
    linked_listing_platform BIGINT;
BEGIN
    SELECT owner_id, sourcing_product_id, product_opportunity_id
      INTO linked_owner, linked_source, linked_opportunity
      FROM sourcing_1688_task_link WHERE id = NEW.task_link_id;
    SELECT sourcing_product_id INTO linked_snapshot_source
      FROM sourcing_1688_snapshot WHERE id = NEW.snapshot_id;
    SELECT product_id, code INTO linked_sku_product, linked_sku_code
      FROM sku WHERE id = NEW.internal_sku_id;
    SELECT product_id, platform_id INTO linked_listing_product, linked_listing_platform
      FROM product_listing WHERE id = NEW.listing_id;

    IF linked_owner IS NULL
       OR linked_owner <> NEW.owner_id
       OR linked_source <> NEW.sourcing_product_id
       OR linked_opportunity IS DISTINCT FROM NEW.product_opportunity_id
       OR linked_snapshot_source <> NEW.sourcing_product_id
       OR linked_sku_product <> NEW.product_id
       OR trim(coalesce(linked_sku_code, '')) <> NEW.internal_sku
       OR linked_listing_product <> NEW.product_id
       OR linked_listing_platform <> NEW.platform_id
       OR NOT EXISTS (
            SELECT 1 FROM sourcing_1688_product p
            WHERE p.id = NEW.sourcing_product_id AND p.owner_id = NEW.owner_id
       )
       OR NOT EXISTS (
            SELECT 1 FROM product_opportunity o
            WHERE o.id = NEW.product_opportunity_id AND o.owner_id = NEW.owner_id
       )
    THEN
        RAISE EXCEPTION 'sourcing SKU mapping authority binding mismatch';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_validate_sourcing_sku_mapping_binding
BEFORE INSERT ON sourcing_sku_mapping
FOR EACH ROW EXECUTE FUNCTION validate_sourcing_sku_mapping_binding();

CREATE OR REPLACE FUNCTION reject_sourcing_sku_mapping_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'sourcing_sku_mapping is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sourcing_sku_mapping_immutable
BEFORE UPDATE OR DELETE ON sourcing_sku_mapping
FOR EACH ROW EXECUTE FUNCTION reject_sourcing_sku_mapping_mutation();

COMMENT ON TABLE sourcing_sku_mapping IS
    'Immutable Owner-scoped supplier -> internal -> channel SKU identity chain, frozen per sourcing task version.';
