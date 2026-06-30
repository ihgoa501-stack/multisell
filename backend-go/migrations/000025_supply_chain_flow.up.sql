CREATE TABLE supply_chain_flow (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    source_type VARCHAR(50),
    source_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'pending',
    context JSONB,
    carrier_summary JSONB,
    financial_summary JSONB,
    error_log JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_supply_chain_flow_status ON supply_chain_flow(status);
CREATE INDEX idx_supply_chain_flow_source_type ON supply_chain_flow(source_type);
CREATE INDEX idx_supply_chain_flow_created_at ON supply_chain_flow(created_at DESC);
