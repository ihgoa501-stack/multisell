-- =============================================================================
-- LingMirror Forward Repair: Apply Missing Migrations (000015, 000017-000031)
-- =============================================================================
--
-- Purpose: Applies all migrations that were NOT applied in production
-- (confirmed by DB audit on 2026-06-29).
--
-- All DDL uses IF NOT EXISTS / IF EXISTS guards so this file is idempotent
-- and safe to rerun.
--
-- Covers these gaps:
--   000015 (metabolism)     — NOT applied, table absent
--   000016 (missing)        — gap in numbering, no file exists
--   000017-000031           — NOT applied, all tables absent
--
-- Duplicate versions (each is a separate migration sharing a version number):
--   000026: data_freshness + tariff_rule   (both applied here)
--   000028: landed_cost + sku_return_stats (both applied here)
--   000029: identical to 000030, skipped in favor of 000030 (which has down.sql)
--
-- DEPENDENCY ORDER: 000025 supply_chain_flow MUST be created before
-- 000027 supply_chain_tracking which FK-references it.
--
-- BEGIN TRANSACTION
-- =============================================================================

BEGIN;

-- =============================================================================
-- 000015 — Metabolism M1 Phase 1
-- =============================================================================

CREATE TABLE IF NOT EXISTS metabolism_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL,
    source VARCHAR(100) NOT NULL,
    total_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    impact_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    ref_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    semantic_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    sem_skipped BOOLEAN NOT NULL DEFAULT false,
    raw_score_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS excreted_at TIMESTAMPTZ;
ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS excretion_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_metabolism_log_created_at ON metabolism_log(created_at);
CREATE INDEX IF NOT EXISTS idx_metabolism_log_source ON metabolism_log(source);
CREATE INDEX IF NOT EXISTS idx_metabolism_log_high_score ON metabolism_log(total_score) WHERE total_score >= 0.70;

-- =============================================================================
-- 000017 — LLM Cost Logs
-- =============================================================================

CREATE TABLE IF NOT EXISTS llm_cost_logs (
    id            BIGSERIAL       PRIMARY KEY,
    user_id       BIGINT          NOT NULL DEFAULT 0,
    agent_id      VARCHAR(50)     NOT NULL DEFAULT '',
    model         VARCHAR(50)     NOT NULL DEFAULT '',
    tokens_in     INT             NOT NULL DEFAULT 0,
    tokens_out    INT             NOT NULL DEFAULT 0,
    cost_usd      NUMERIC(12,6)   NOT NULL DEFAULT 0,
    request_hash  VARCHAR(64)     NOT NULL DEFAULT '',
    cached        BOOLEAN         NOT NULL DEFAULT FALSE,
    window_date   DATE            NOT NULL DEFAULT CURRENT_DATE,
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_cost_logs_date   ON llm_cost_logs(window_date);
CREATE INDEX IF NOT EXISTS idx_llm_cost_logs_user   ON llm_cost_logs(user_id, window_date);
CREATE INDEX IF NOT EXISTS idx_llm_cost_logs_agent  ON llm_cost_logs(agent_id, window_date);

-- =============================================================================
-- 000018 — Approval Request
-- =============================================================================

CREATE TABLE IF NOT EXISTS approval_request (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    request_type VARCHAR(32) NOT NULL,
    requester VARCHAR(64) NOT NULL,
    reviewer VARCHAR(64),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    old_value TEXT,
    new_value TEXT,
    reason TEXT,
    review_note TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_approval_status ON approval_request(status);
CREATE INDEX IF NOT EXISTS idx_approval_request_type ON approval_request(request_type);
CREATE INDEX IF NOT EXISTS idx_approval_product_id ON approval_request(product_id);
CREATE INDEX IF NOT EXISTS idx_approval_requester ON approval_request(requester);
CREATE INDEX IF NOT EXISTS idx_approval_created_at ON approval_request(created_at);

-- =============================================================================
-- 000019 — Product Relation Graph
-- =============================================================================

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

-- =============================================================================
-- 000020 — Product Version History
-- =============================================================================

CREATE TABLE IF NOT EXISTS product_version (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    version_data JSONB,
    snapshot JSONB,
    agent_id VARCHAR(255),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_version_product_id ON product_version(product_id);
CREATE INDEX IF NOT EXISTS idx_product_version_created_at ON product_version(created_at);

-- =============================================================================
-- 000021 — Supplier Score
-- =============================================================================

CREATE TABLE IF NOT EXISTS supplier_score (
    id                      BIGSERIAL       PRIMARY KEY,
    supplier_id             BIGINT          NOT NULL REFERENCES supplier(id),
    on_time_delivery_rate   NUMERIC(5,1)    NOT NULL DEFAULT 0,
    quality_pass_rate       NUMERIC(5,1)    NOT NULL DEFAULT 0,
    communication_score     NUMERIC(5,1)    NOT NULL DEFAULT 0,
    order_fulfillment_pct   NUMERIC(5,1)    NOT NULL DEFAULT 0,
    avg_lead_time_days      NUMERIC(6,1)    NOT NULL DEFAULT 0,
    reliability_score       NUMERIC(5,1)    NOT NULL DEFAULT 0,
    data_freshness          INT             NOT NULL DEFAULT 0,
    last_order_date         TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_score_supplier_id ON supplier_score(supplier_id);
CREATE INDEX IF NOT EXISTS idx_supplier_score_reliability ON supplier_score(reliability_score DESC);

-- =============================================================================
-- 000022 — Agent Learning
-- =============================================================================

CREATE TABLE IF NOT EXISTS decision_evaluation (
    id BIGSERIAL PRIMARY KEY,
    decision_trace_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    agent_id VARCHAR(20) NOT NULL,
    predicted_outcome TEXT,
    actual_outcome TEXT,
    score NUMERIC(6,4) NOT NULL DEFAULT 0,
    evaluated_at TIMESTAMPTZ,
    evaluation_type VARCHAR(10) NOT NULL DEFAULT 'T+30',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_decision_evaluation_trace ON decision_evaluation(decision_trace_id);
CREATE INDEX IF NOT EXISTS idx_decision_evaluation_product ON decision_evaluation(product_id);
CREATE INDEX IF NOT EXISTS idx_decision_evaluation_agent ON decision_evaluation(agent_id);
CREATE INDEX IF NOT EXISTS idx_decision_evaluation_created ON decision_evaluation(created_at);

CREATE TABLE IF NOT EXISTS agent_accuracy (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(20) NOT NULL,
    period VARCHAR(5) NOT NULL,
    total_decisions INT NOT NULL DEFAULT 0,
    correct_decisions INT NOT NULL DEFAULT 0,
    accuracy_pct NUMERIC(6,2) NOT NULL DEFAULT 0,
    trend VARCHAR(20) NOT NULL DEFAULT 'stable',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_period ON agent_accuracy(agent_id, period);
CREATE INDEX IF NOT EXISTS idx_agent_accuracy_agent ON agent_accuracy(agent_id);

-- =============================================================================
-- 000023 — Webhook Event Log
-- =============================================================================

CREATE TABLE IF NOT EXISTS webhook_event_log (
    id BIGSERIAL PRIMARY KEY,
    platform TEXT NOT NULL,
    event_type TEXT NOT NULL,
    raw_payload JSONB,
    status TEXT DEFAULT 'received',
    mapped_event TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- 000024 — Inventory Oversell Log
-- =============================================================================

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

-- =============================================================================
-- 000025 — Supply Chain Flow
-- IMPORTANT: Must exist before 000027 which FK-references this table.
-- =============================================================================

CREATE TABLE IF NOT EXISTS supply_chain_flow (
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

-- =============================================================================
-- 000026a — Data Freshness
-- =============================================================================

CREATE TABLE IF NOT EXISTS data_freshness (
    id              BIGSERIAL PRIMARY KEY,
    product_id      BIGINT NOT NULL REFERENCES product(id) ON DELETE CASCADE,
    dimension       VARCHAR(50) NOT NULL,
    last_verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_check_at   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    freshness_days  INTEGER NOT NULL DEFAULT 30,
    status          VARCHAR(20) NOT NULL DEFAULT 'fresh',
    drift_detected  BOOLEAN NOT NULL DEFAULT FALSE,
    last_value      TEXT,
    current_value   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_data_freshness_product_id ON data_freshness(product_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_data_freshness_product_dimension ON data_freshness(product_id, dimension);
CREATE INDEX IF NOT EXISTS idx_data_freshness_next_check_at ON data_freshness(next_check_at);
CREATE INDEX IF NOT EXISTS idx_data_freshness_status ON data_freshness(status);

-- =============================================================================
-- 000026b — Tariff Rule
-- =============================================================================

CREATE TABLE IF NOT EXISTS tariff_rule (
    id BIGSERIAL PRIMARY KEY,
    country_code VARCHAR(10) NOT NULL,
    hs_code VARCHAR(20),
    hs_code_prefix VARCHAR(10),
    duty_rate_pct NUMERIC(10,4) DEFAULT 0,
    vat_rate_pct NUMERIC(10,4) DEFAULT 0,
    other_tax_rate_pct NUMERIC(10,4) DEFAULT 0,
    min_threshold_usd NUMERIC(14,2) DEFAULT 0,
    max_threshold_usd NUMERIC(14,2) DEFAULT 0,
    incoterm VARCHAR(10) NOT NULL DEFAULT 'DDU',
    priority INTEGER DEFAULT 0,
    effective_from TIMESTAMPTZ,
    effective_to TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'active',
    remark TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tariff_rule_country ON tariff_rule(country_code);
CREATE INDEX IF NOT EXISTS idx_tariff_rule_status ON tariff_rule(status);
CREATE INDEX IF NOT EXISTS idx_tariff_rule_country_status ON tariff_rule(country_code, status);

-- =============================================================================
-- 000027 — Supply Chain Tracking
-- =============================================================================

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

CREATE INDEX IF NOT EXISTS idx_tracking_flow_id ON supply_chain_tracking(flow_id);
CREATE INDEX IF NOT EXISTS idx_tracking_order_id ON supply_chain_tracking(order_id);
CREATE INDEX IF NOT EXISTS idx_tracking_tracking_no ON supply_chain_tracking(tracking_no);
CREATE INDEX IF NOT EXISTS idx_tracking_status ON supply_chain_tracking(status);
CREATE INDEX IF NOT EXISTS idx_tracking_carrier_code ON supply_chain_tracking(carrier_code);

-- =============================================================================
-- 000028a — Landed Cost
-- =============================================================================

CREATE TABLE IF NOT EXISTS landed_cost (
    id              BIGSERIAL       PRIMARY KEY,
    product_id      BIGINT          NOT NULL,
    platform_id     BIGINT          NOT NULL,
    unit_cost_cny       DECIMAL(12,2) NOT NULL DEFAULT 0,
    freight_cny         DECIMAL(12,2) NOT NULL DEFAULT 0,
    insurance_cny       DECIMAL(12,2) NOT NULL DEFAULT 0,
    duty_rate           DECIMAL(5,2)  NOT NULL DEFAULT 0,
    duty_cny            DECIMAL(12,2) NOT NULL DEFAULT 0,
    vat_rate            DECIMAL(5,2)  NOT NULL DEFAULT 0,
    vat_cny             DECIMAL(12,2) NOT NULL DEFAULT 0,
    platform_fee_pct    DECIMAL(5,2)  NOT NULL DEFAULT 0,
    platform_fee_cny    DECIMAL(12,2) NOT NULL DEFAULT 0,
    clearing_fee_cny    DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_cost_cny      DECIMAL(12,2) NOT NULL DEFAULT 0,
    exchange_rate       DECIMAL(10,4) NOT NULL DEFAULT 0,
    total_cost_local    DECIMAL(12,2) NOT NULL DEFAULT 0,
    target_price        DECIMAL(12,2) NOT NULL DEFAULT 0,
    calculated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_landed_cost_product_id ON landed_cost (product_id);
CREATE INDEX IF NOT EXISTS idx_landed_cost_product_platform ON landed_cost (product_id, platform_id);

-- =============================================================================
-- 000028b — SKU Return Stats
-- =============================================================================

CREATE TABLE IF NOT EXISTS sku_return_stats (
    sku_id BIGINT NOT NULL PRIMARY KEY,
    total_orders BIGINT NOT NULL DEFAULT 0,
    total_returns BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- 000030 — Product Lifecycle Orchestration
-- NOTE: 000029 (identical content) is skipped; 000030 used as canonical version
-- because it has a proper down.sql while 000029 does not.
-- =============================================================================

CREATE TABLE IF NOT EXISTS lifecycle_step (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    step VARCHAR(50) NOT NULL,
    agent_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_step_product_id ON lifecycle_step(product_id);
CREATE INDEX IF NOT EXISTS idx_lifecycle_step_status ON lifecycle_step(status);

CREATE TABLE IF NOT EXISTS orchestration_config (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    steps JSONB NOT NULL DEFAULT '[]',
    failure_action VARCHAR(20) NOT NULL DEFAULT 'stop',
    auto_approve_pct NUMERIC(5,2) NOT NULL DEFAULT 80,
    auto_retry_count INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- 000031 — Product Sentiment
-- =============================================================================

CREATE TABLE IF NOT EXISTS product_sentiment (
    id              BIGSERIAL       PRIMARY KEY,
    product_id      BIGINT          NOT NULL UNIQUE,
    avg_rating      DECIMAL(3,2)    NOT NULL DEFAULT 0,
    review_count    INTEGER         NOT NULL DEFAULT 0,
    positive_pct    DECIMAL(5,2)    NOT NULL DEFAULT 0,
    negative_pct    DECIMAL(5,2)    NOT NULL DEFAULT 0,
    return_rate     DECIMAL(5,2)    NOT NULL DEFAULT 0,
    top_complaints  TEXT            NOT NULL DEFAULT '[]',
    sentiment_score DECIMAL(5,2)    NOT NULL DEFAULT 0,
    last_updated    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_sentiment_score ON product_sentiment (sentiment_score);
CREATE INDEX IF NOT EXISTS idx_product_sentiment_product ON product_sentiment (product_id);

-- =============================================================================
-- Initialize schema_migrations tracking table
-- =============================================================================
-- Enables golang-migrate compatibility for future migrations.
-- Sets version=31 (last applied), dirty=false.
-- NOTE: Versions 000026 and 000028 have duplicate files, and 000029 was
-- skipped. The golang-migrate tool will not handle these correctly without
-- the file rename fix described in the plan.
-- =============================================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

INSERT INTO schema_migrations (version, dirty)
SELECT 31, false
WHERE NOT EXISTS (SELECT 1 FROM schema_migrations);

COMMIT;
