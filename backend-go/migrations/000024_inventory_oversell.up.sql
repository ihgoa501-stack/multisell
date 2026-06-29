-- Cross-platform inventory sync / oversell prevention
-- Tracks oversell detections where total committed stock across all
-- e-commerce platforms exceeds available inventory.

CREATE TABLE IF NOT EXISTS inventory_oversell_log (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    available_stock INT NOT NULL,
    total_committed INT NOT NULL,
    oversell_by INT NOT NULL DEFAULT 0,
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    status TEXT DEFAULT 'open'
);

CREATE INDEX IF NOT EXISTS idx_inventory_oversell_product_id ON inventory_oversell_log(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_oversell_status ON inventory_oversell_log(status);
CREATE INDEX IF NOT EXISTS idx_inventory_oversell_detected_at ON inventory_oversell_log(detected_at);
