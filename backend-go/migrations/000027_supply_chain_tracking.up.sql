CREATE TABLE IF NOT EXISTS supply_chain_tracking (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    flow_id UUID REFERENCES supply_chain_flow(id) ON DELETE CASCADE,
    order_id VARCHAR(100) NOT NULL DEFAULT '',
    carrier_code VARCHAR(50) NOT NULL DEFAULT '',
    tracking_no VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    estimated_delivery TIMESTAMPTZ,
    actual_delivery TIMESTAMPTZ,
    status_history JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tracking_flow_id ON supply_chain_tracking(flow_id);
CREATE INDEX idx_tracking_order_id ON supply_chain_tracking(order_id);
CREATE INDEX idx_tracking_tracking_no ON supply_chain_tracking(tracking_no);
CREATE INDEX idx_tracking_status ON supply_chain_tracking(status);
CREATE INDEX idx_tracking_carrier_code ON supply_chain_tracking(carrier_code);
