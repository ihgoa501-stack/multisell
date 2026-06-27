-- Product relation graph — tracks variant, replacement, bundle, cross_sell,
-- alternative, and accessory relationships between products.
-- Used by the Product360 page to display related products grouped by type.

CREATE TABLE IF NOT EXISTS product_relation (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    target_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    relation_type VARCHAR(50) NOT NULL,
    weight NUMERIC(3,2) NOT NULL DEFAULT 0,
    auto_discovered BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_relation_source_id ON product_relation(source_id);
CREATE INDEX IF NOT EXISTS idx_product_relation_target_id ON product_relation(target_id);
CREATE INDEX IF NOT EXISTS idx_product_relation_type ON product_relation(relation_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_relation_unique_pair ON product_relation(
    LEAST(source_id, target_id),
    GREATEST(source_id, target_id),
    relation_type
);
