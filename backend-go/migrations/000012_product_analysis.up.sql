-- 000012_product_analysis

-- Product analysis results
CREATE TABLE IF NOT EXISTS product_analysis (
    id BIGSERIAL PRIMARY KEY,
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    target_sale_price DECIMAL(12,2) NOT NULL,
    estimated_cost DECIMAL(12,2) NOT NULL DEFAULT 0,
    estimated_profit_margin DECIMAL(5,2),
    demand_score DECIMAL(5,2),
    demand_score_status VARCHAR(20) NOT NULL DEFAULT 'no_data',
    competition_index DECIMAL(5,2),
    competition_status VARCHAR(20) NOT NULL DEFAULT 'no_data',
    analysis_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    analyzed_by VARCHAR(255) NOT NULL,
    analyzed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_analysis_sourcing_product ON product_analysis(sourcing_product_id);
CREATE INDEX IF NOT EXISTS idx_product_analysis_analyzed_by ON product_analysis(analyzed_by);

-- Immutable audit log for user feedback
CREATE TABLE IF NOT EXISTS analysis_feedback (
    id BIGSERIAL PRIMARY KEY,
    product_analysis_id BIGINT NOT NULL REFERENCES product_analysis(id),
    user_id VARCHAR(255) NOT NULL,
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('imported', 'abandoned')),
    actual_profit_margin DECIMAL(5,2),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analysis_feedback_analysis ON analysis_feedback(product_analysis_id);
