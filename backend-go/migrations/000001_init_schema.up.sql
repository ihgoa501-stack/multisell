-- ============================================================
-- LingMirror init schema — business tables
-- Source: SQLAlchemy models
--   - app/models.py
--   - app/agent/models.py
--   - app/agentos/models.py
--   - app/order_import/models.py
-- Target: PostgreSQL 13+
-- Note: AI trace tables (ai_trace / ai_trace_event / ai_evidence_ref /
--       unified_action) are defined in 000002_ai_tables.up.sql
--       and are NOT duplicated here.
-- ============================================================


-- ============================================================
-- Section 1: Reference / root tables (no FK dependencies)
-- ============================================================

CREATE TABLE IF NOT EXISTS category (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id BIGINT DEFAULT 0,
    level INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS brand (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    logo VARCHAR(500),
    description TEXT,
    status SMALLINT DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS platform (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    api_base_url VARCHAR(500),
    api_key VARCHAR(500),
    client_id VARCHAR(200),
    extra_config JSONB,
    status SMALLINT DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS shipping_provider (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50),
    contact VARCHAR(100),
    phone VARCHAR(50),
    remark TEXT,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS supplier (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    contact_person VARCHAR(100),
    contact_phone VARCHAR(50),
    email VARCHAR(200),
    address VARCHAR(500),
    status SMALLINT DEFAULT 1,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS warehouse (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50),
    address VARCHAR(500),
    contact VARCHAR(100),
    phone VARCHAR(50),
    is_default SMALLINT DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS "user" (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(500) NOT NULL,
    display_name VARCHAR(200),
    role VARCHAR(50) DEFAULT 'user',
    email VARCHAR(200),
    status SMALLINT DEFAULT 1,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (username)
);

CREATE TABLE IF NOT EXISTS role (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS permission (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    module VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS finance_account (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    platform_id BIGINT REFERENCES platform(id),
    currency VARCHAR(3) DEFAULT 'CNY',
    balance NUMERIC(14, 2) DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS alert_rule (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    enabled SMALLINT DEFAULT 1,
    config JSONB,
    description VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (alert_type)
);

CREATE TABLE IF NOT EXISTS system_config (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL,
    config_value TEXT,
    config_json JSONB,
    is_secret SMALLINT DEFAULT 0,
    description VARCHAR(500),
    updated_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (config_key)
);

CREATE TABLE IF NOT EXISTS exchange_rate (
    id BIGSERIAL PRIMARY KEY,
    from_currency VARCHAR(10) NOT NULL,
    to_currency VARCHAR(10) NOT NULL,
    rate NUMERIC(14, 6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS operation_log (
    id BIGSERIAL PRIMARY KEY,
    module VARCHAR(50),
    action VARCHAR(50),
    resource_id VARCHAR(100),
    content TEXT,
    operator VARCHAR(100),
    ip VARCHAR(50),
    duration INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exception_item (
    id BIGSERIAL PRIMARY KEY,
    source_module VARCHAR(50) NOT NULL,
    source_type VARCHAR(50),
    source_id BIGINT,
    severity VARCHAR(20) DEFAULT 'medium',
    status VARCHAR(20) DEFAULT 'open',
    title VARCHAR(300) NOT NULL,
    description TEXT,
    recommended_action VARCHAR(500),
    assigned_to VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(100),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_episode (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    episode_number INTEGER NOT NULL,
    decision_count INTEGER NOT NULL,
    episode_summary TEXT,
    key_insights JSONB,
    improvement_suggestions JSONB,
    acceptance_rate NUMERIC(4, 3),
    avg_confidence NUMERIC(4, 3),
    avg_response_ms INTEGER,
    total_tokens INTEGER,
    nudge_triggered INTEGER DEFAULT 0,
    nudge_topics JSONB,
    nudge_response TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rule_mark_change (
    id BIGSERIAL PRIMARY KEY,
    target_type VARCHAR(30) NOT NULL,
    target_id BIGINT NOT NULL,
    field_path VARCHAR(200) NOT NULL,
    old_value JSONB,
    new_value JSONB NOT NULL,
    source_type VARCHAR(30) NOT NULL,
    source_id VARCHAR(100),
    change_summary TEXT NOT NULL,
    parent_change_id BIGINT,
    related_decision_ids JSONB,
    context_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agentos_operation_log (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    item_id VARCHAR(128) NOT NULL,
    action VARCHAR(32) NOT NULL,
    source_type VARCHAR(32),
    previous_status VARCHAR(32),
    new_status VARCHAR(32),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_settlement_batch (
    id BIGSERIAL PRIMARY KEY,
    platform_name VARCHAR(100),
    filename VARCHAR(500) NOT NULL,
    row_count INTEGER DEFAULT 0,
    matched_count INTEGER DEFAULT 0,
    unmatched_count INTEGER DEFAULT 0,
    import_status VARCHAR(30) DEFAULT 'imported',
    status VARCHAR(30) DEFAULT 'imported',
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cost_allocation_batch (
    id BIGSERIAL PRIMARY KEY,
    allocation_type VARCHAR(50) NOT NULL,
    allocation_method VARCHAR(30) NOT NULL,
    total_amount NUMERIC(14, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    source_filename VARCHAR(500),
    row_count INTEGER DEFAULT 0,
    status VARCHAR(30) DEFAULT 'imported',
    posted_count INTEGER DEFAULT 0,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS import_batch (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(30) NOT NULL,
    file_name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_rows INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    error_summary TEXT,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_import_batch (
    id BIGSERIAL PRIMARY KEY,
    adapter_code VARCHAR(50) NOT NULL,
    platform VARCHAR(100),
    store_name VARCHAR(200),
    source_filename VARCHAR(500) NOT NULL,
    row_count INTEGER DEFAULT 0,
    created_order_count INTEGER DEFAULT 0,
    skipped_duplicate_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    imported_by VARCHAR(100),
    chain_status VARCHAR(50) DEFAULT 'chain_pending',
    ledger_rebuilt_count INTEGER DEFAULT 0,
    exception_generated_count INTEGER DEFAULT 0,
    chain_failure_count INTEGER DEFAULT 0,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS prompt_template (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description VARCHAR(500),
    prompt VARCHAR(2000) NOT NULL,
    negative_prompt VARCHAR(1000) DEFAULT '',
    style VARCHAR(50) DEFAULT 'product_white',
    size VARCHAR(20) DEFAULT '1024x1024',
    platform_code VARCHAR(50),
    is_shared SMALLINT DEFAULT 1,
    usage_count INTEGER DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agentos_action_proposal (
    id BIGSERIAL PRIMARY KEY,
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(100),
    agent_id VARCHAR(20),
    squad_id VARCHAR(50),
    action_type VARCHAR(100) NOT NULL,
    business_object_type VARCHAR(50),
    business_object_id VARCHAR(100),
    title VARCHAR(300) NOT NULL,
    description TEXT,
    proposed_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    before_snapshot JSONB,
    after_snapshot JSONB,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(30) NOT NULL DEFAULT 'suggested',
    confidence NUMERIC(5, 4),
    proposed_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,
    rejected_by VARCHAR(100),
    rejected_at TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_agentos_action_proposal_risk CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT ck_agentos_action_proposal_status CHECK (status IN ('suggested', 'pending_approval', 'approved', 'executing', 'executed', 'reviewed', 'rejected', 'expired', 'blocked_by_policy', 'failed', 'cancelled'))
);


-- ============================================================
-- Section 2: Tables with FK to root tables
-- ============================================================

CREATE TABLE IF NOT EXISTS product (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    subtitle VARCHAR(500),
    description TEXT,
    brand_id BIGINT DEFAULT 0,
    category_id BIGINT REFERENCES category(id),
    unit VARCHAR(20) DEFAULT '件',
    status SMALLINT DEFAULT 0,
    main_image VARCHAR(500),
    images JSONB,
    product_length_cm NUMERIC(10, 2),
    product_width_cm NUMERIC(10, 2),
    product_height_cm NUMERIC(10, 2),
    product_weight_kg NUMERIC(10, 2),
    package_length_cm NUMERIC(10, 2),
    package_width_cm NUMERIC(10, 2),
    package_height_cm NUMERIC(10, 2),
    package_weight_kg NUMERIC(10, 2),
    cargo_type VARCHAR(50) DEFAULT 'normal',
    ai_title VARCHAR(500),
    ai_description TEXT,
    seo_keywords JSONB,
    ai_status VARCHAR(50) DEFAULT 'pending',
    platform_statuses JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shipping_channel (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES shipping_provider(id),
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50),
    volumetric_divisor INTEGER NOT NULL DEFAULT 6000,
    cargo_types JSONB,
    estimated_delivery_min INTEGER,
    estimated_delivery_max INTEGER,
    currency VARCHAR(10) DEFAULT 'CNY',
    sort_order INTEGER DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS honcho_profile (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    risk_tolerance VARCHAR(30) DEFAULT 'moderate',
    communication_style VARCHAR(20) DEFAULT 'balanced',
    notification_prefs JSONB,
    agent_profiles JSONB NOT NULL DEFAULT '{}'::jsonb,
    hypothesis_count INTEGER DEFAULT 0,
    confirmed_count INTEGER DEFAULT 0,
    last_dialectic_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);

CREATE TABLE IF NOT EXISTS agent_decision (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    context_json JSONB NOT NULL,
    agent_output JSONB NOT NULL,
    final_decision JSONB NOT NULL,
    user_action VARCHAR(20) NOT NULL,
    user_overrides JSONB,
    user_feedback TEXT,
    rules_applied JSONB,
    rule_overrides INTEGER DEFAULT 0,
    evolution_stage VARCHAR(20) NOT NULL,
    confidence NUMERIC(4, 3),
    response_time_ms INTEGER,
    token_count INTEGER,
    session_id VARCHAR(100) NOT NULL,
    episode_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS personal_rule (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    rule_type VARCHAR(20) NOT NULL,
    rule_name VARCHAR(100) NOT NULL,
    rule_condition JSONB NOT NULL,
    rule_action JSONB NOT NULL,
    priority INTEGER DEFAULT 100,
    source VARCHAR(20) NOT NULL,
    source_decisions JSONB,
    status VARCHAR(20) DEFAULT 'active',
    confidence NUMERIC(4, 3) DEFAULT 0,
    times_applied INTEGER DEFAULT 0,
    times_overridden INTEGER DEFAULT 0,
    last_applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS spc_control_limit (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    baseline_mean NUMERIC(10, 4) NOT NULL,
    baseline_stddev NUMERIC(10, 4) NOT NULL,
    baseline_samples INTEGER NOT NULL,
    ucl NUMERIC(10, 4) NOT NULL,
    lcl NUMERIC(10, 4) NOT NULL,
    uwl NUMERIC(10, 4) NOT NULL,
    lwl NUMERIC(10, 4) NOT NULL,
    consecutive_same_side INTEGER DEFAULT 0,
    last_breach_at TIMESTAMPTZ,
    baseline_recalc_at TIMESTAMPTZ NOT NULL,
    next_recalc_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_spc_metric UNIQUE (user_id, agent_id, decision_point, metric_name)
);

CREATE TABLE IF NOT EXISTS agent_evolution_config (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    current_stage VARCHAR(20) NOT NULL DEFAULT 'observation',
    stage_updated_at TIMESTAMPTZ,
    stage_updated_by VARCHAR(20) DEFAULT 'system',
    trust_score NUMERIC(6, 2) DEFAULT 0,
    decision_count INTEGER DEFAULT 0,
    adoption_rate NUMERIC(5, 4),
    avg_confidence NUMERIC(5, 4),
    consistency_score NUMERIC(5, 4),
    stability_score NUMERIC(5, 4),
    last_calculated_at TIMESTAMPTZ,
    nudge_last_shown_at TIMESTAMPTZ,
    nudge_dismissed_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_agent_evolution UNIQUE (user_id, agent_id, decision_point)
);

CREATE TABLE IF NOT EXISTS agent_nudge (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    target_stage VARCHAR(20) NOT NULL,
    trust_score_at_time NUMERIC(6, 2) NOT NULL,
    score_components JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    responded_at TIMESTAMPTZ,
    cooling_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    alert_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    link_url VARCHAR(500),
    severity VARCHAR(20) DEFAULT 'info',
    is_read SMALLINT DEFAULT 0,
    source_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS stores (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    name VARCHAR(200) NOT NULL,
    platform_id BIGINT REFERENCES platform(id),
    platform_account_id VARCHAR(100),
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_role (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    role_id BIGINT NOT NULL REFERENCES role(id)
);

CREATE TABLE IF NOT EXISTS role_permission (
    id BIGSERIAL PRIMARY KEY,
    role_id BIGINT NOT NULL REFERENCES role(id),
    permission_id BIGINT NOT NULL REFERENCES permission(id)
);

CREATE TABLE IF NOT EXISTS platform_integration_account (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    adapter_code VARCHAR(50) NOT NULL,
    account_name VARCHAR(200) NOT NULL,
    status VARCHAR(30) DEFAULT 'draft',
    credential_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_category_mapping (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    adapter_code VARCHAR(50) NOT NULL,
    local_category_id BIGINT NOT NULL REFERENCES category(id),
    platform_category_id VARCHAR(200) NOT NULL,
    platform_category_name VARCHAR(500),
    platform_category_path VARCHAR(1000),
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_attribute_mapping (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    adapter_code VARCHAR(50) NOT NULL,
    local_attribute VARCHAR(100) NOT NULL,
    platform_attribute VARCHAR(200) NOT NULL,
    default_value VARCHAR(500),
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_fee_rule (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    country_code VARCHAR(10),
    category_id BIGINT REFERENCES category(id),
    fee_type VARCHAR(30) NOT NULL,
    fee_rate_pct NUMERIC(10, 4) DEFAULT 0,
    fixed_amount NUMERIC(12, 2) DEFAULT 0,
    min_amount NUMERIC(12, 2),
    max_amount NUMERIC(12, 2),
    currency VARCHAR(3) DEFAULT 'CNY',
    effective_from TIMESTAMPTZ,
    effective_to TIMESTAMPTZ,
    priority INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_import (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT REFERENCES platform(id),
    source_type VARCHAR(50) NOT NULL,
    file_name VARCHAR(255),
    total_rows INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    error_detail JSONB,
    status VARCHAR(20) DEFAULT 'pending',
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS settlement (
    id BIGSERIAL PRIMARY KEY,
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    settlement_no VARCHAR(100) NOT NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    currency VARCHAR(3) DEFAULT 'CNY',
    total_revenue NUMERIC(12, 2) DEFAULT 0,
    total_fee NUMERIC(12, 2) DEFAULT 0,
    total_refund NUMERIC(12, 2) DEFAULT 0,
    total_net NUMERIC(12, 2) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    raw_data JSONB,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shipping_bill_batch (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT REFERENCES shipping_provider(id),
    source_filename VARCHAR(500) NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    row_count INTEGER DEFAULT 0,
    matched_count INTEGER DEFAULT 0,
    mismatch_count INTEGER DEFAULT 0,
    unmatched_count INTEGER DEFAULT 0,
    status VARCHAR(30) DEFAULT 'imported',
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_action (
    id BIGSERIAL PRIMARY KEY,
    source_module VARCHAR(50),
    source_type VARCHAR(50),
    source_id BIGINT,
    exception_id BIGINT REFERENCES exception_item(id),
    action_type VARCHAR(100) NOT NULL,
    title VARCHAR(300) NOT NULL,
    description TEXT,
    proposed_payload JSONB,
    before_snapshot JSONB,
    after_snapshot JSONB,
    status VARCHAR(30) DEFAULT 'proposed',
    proposed_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,
    rejected_by VARCHAR(100),
    rejected_at TIMESTAMPTZ,
    rejection_reason TEXT,
    executed_by VARCHAR(100),
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS allocation_rule (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    priority INTEGER DEFAULT 0,
    rule_type VARCHAR(50) NOT NULL,
    warehouse_id BIGINT NOT NULL REFERENCES warehouse(id),
    allocation_pct NUMERIC(5, 2) DEFAULT 100,
    allocation_qty INTEGER DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- ============================================================
-- Section 3: Tables with FK to Section 2 tables
-- ============================================================

CREATE TABLE IF NOT EXISTS spec_name (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sku (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    code VARCHAR(100),
    barcode VARCHAR(100),
    spec_desc VARCHAR(500),
    spec_values JSONB,
    price NUMERIC(10, 2) DEFAULT 0,
    cost_price NUMERIC(10, 2) DEFAULT 0,
    market_price NUMERIC(10, 2) DEFAULT 0,
    stock INTEGER DEFAULT 0,
    lock_stock INTEGER DEFAULT 0,
    warning_stock INTEGER DEFAULT 0,
    weight NUMERIC(10, 2) DEFAULT 0,
    sku_length_cm NUMERIC(10, 2),
    sku_width_cm NUMERIC(10, 2),
    sku_height_cm NUMERIC(10, 2),
    sku_weight_kg NUMERIC(10, 2),
    image VARCHAR(500),
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS product_supplier (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    supplier_id BIGINT NOT NULL REFERENCES supplier(id),
    supply_price NUMERIC(10, 2),
    min_order_qty INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sourcing_1688_product (
    id BIGSERIAL PRIMARY KEY,
    source_url VARCHAR(1000) NOT NULL,
    title VARCHAR(500),
    price NUMERIC(10, 2),
    moq INTEGER DEFAULT 1,
    supplier_name VARCHAR(200),
    shop_url VARCHAR(1000),
    shop_location VARCHAR(200),
    images JSONB,
    attributes JSONB,
    sku_variants JSONB,
    description TEXT,
    package_length_cm NUMERIC(10, 2),
    package_width_cm NUMERIC(10, 2),
    package_height_cm NUMERIC(10, 2),
    package_weight_kg NUMERIC(10, 2),
    raw_data JSONB,
    status VARCHAR(50) DEFAULT 'collected',
    product_id BIGINT REFERENCES product(id),
    supplier_id BIGINT REFERENCES supplier(id),
    collected_by VARCHAR(100),
    imported_by VARCHAR(100),
    imported_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_url)
);

CREATE TABLE IF NOT EXISTS product_listing (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    platform_product_id VARCHAR(200),
    platform_sku VARCHAR(200),
    status VARCHAR(50) DEFAULT 'draft',
    platform_url VARCHAR(500),
    sync_message TEXT,
    published_data JSONB,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS product_image_gen (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    prompt VARCHAR(2000) NOT NULL,
    style VARCHAR(50) DEFAULT 'product_white',
    negative_prompt VARCHAR(1000) DEFAULT '',
    size VARCHAR(20) DEFAULT '1024x1024',
    requested_count INTEGER DEFAULT 1,
    status VARCHAR(20) DEFAULT 'pending',
    image_urls JSONB,
    error_message VARCHAR(1000),
    created_by BIGINT,
    batch_id VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS product_canvases (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    name VARCHAR(200) DEFAULT '未命名画布',
    layers JSONB,
    thumbnail TEXT,
    created_by BIGINT REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shipping_zone (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES shipping_channel(id),
    country_code VARCHAR(10) NOT NULL,
    postal_code_from VARCHAR(20),
    postal_code_to VARCHAR(20),
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_pending_action (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    agent_id VARCHAR(20) NOT NULL,
    decision_id BIGINT REFERENCES agent_decision(id),
    action_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    summary VARCHAR(500) NOT NULL,
    action_payload JSONB,
    execution_result JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rule_conflict (
    id BIGSERIAL PRIMARY KEY,
    decision_id BIGINT NOT NULL REFERENCES agent_decision(id),
    conflicting_rules JSONB NOT NULL,
    winner_rule_id BIGINT NOT NULL,
    resolution VARCHAR(20) NOT NULL,
    nudge_sent INTEGER DEFAULT 0,
    nudge_resolved INTEGER DEFAULT 0,
    user_choice BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- ============================================================
-- Section 4: Tables with FK to Section 3 tables
-- ============================================================

CREATE TABLE IF NOT EXISTS spec_value (
    id BIGSERIAL PRIMARY KEY,
    spec_name_id BIGINT NOT NULL REFERENCES spec_name(id),
    product_id BIGINT NOT NULL REFERENCES product(id),
    value VARCHAR(100) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS price (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    price_type VARCHAR(50) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS price_change_log (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    old_price NUMERIC(10, 2),
    new_price NUMERIC(10, 2),
    price_type VARCHAR(50),
    change_type VARCHAR(50),
    operator VARCHAR(100),
    remark VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inventory (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    warehouse VARCHAR(100) DEFAULT '默认仓库',
    location VARCHAR(200),
    quantity INTEGER DEFAULT 0,
    locked_quantity INTEGER NOT NULL DEFAULT 0,
    safety_stock INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (sku_id)
);

CREATE TABLE IF NOT EXISTS inventory_log (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    change_type VARCHAR(50) NOT NULL,
    change_qty INTEGER NOT NULL,
    before_qty INTEGER,
    after_qty INTEGER,
    remark VARCHAR(500),
    operator VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inventory_warehouse (
    id BIGSERIAL PRIMARY KEY,
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    warehouse_id BIGINT NOT NULL REFERENCES warehouse(id),
    quantity INTEGER DEFAULT 0,
    locked_quantity INTEGER DEFAULT 0,
    safety_stock INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sku_warehouse UNIQUE (sku_id, warehouse_id)
);

CREATE TABLE IF NOT EXISTS shipping_quote_rule (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES shipping_channel(id),
    zone_id BIGINT REFERENCES shipping_zone(id),
    rule_type VARCHAR(50) NOT NULL,
    priority INTEGER DEFAULT 0,
    min_weight_kg NUMERIC(10, 3) DEFAULT 0,
    max_weight_kg NUMERIC(10, 3),
    first_kg NUMERIC(10, 3) DEFAULT 0,
    first_price NUMERIC(10, 2) DEFAULT 0,
    additional_kg NUMERIC(10, 3) DEFAULT 0,
    additional_price NUMERIC(10, 2) DEFAULT 0,
    fixed_fee NUMERIC(10, 2) DEFAULT 0,
    per_kg_price NUMERIC(10, 2) DEFAULT 0,
    minimum_charge NUMERIC(10, 2),
    tier_config JSONB,
    surcharge_fixed NUMERIC(10, 2) DEFAULT 0,
    fuel_surcharge_pct NUMERIC(5, 2) DEFAULT 0,
    rounding_increment NUMERIC(10, 3) DEFAULT 0.1,
    remark TEXT,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sales_order (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(100) NOT NULL,
    platform_id BIGINT REFERENCES platform(id),
    status VARCHAR(50) DEFAULT 'pending',
    tracking_number VARCHAR(200),
    recipient_name VARCHAR(100),
    recipient_phone VARCHAR(50),
    shipping_address VARCHAR(500),
    total_amount NUMERIC(10, 2) DEFAULT 0,
    shipping_fee NUMERIC(10, 2) DEFAULT 0,
    pay_amount NUMERIC(10, 2) DEFAULT 0,
    platform_fee NUMERIC(10, 2) DEFAULT 0,
    payment_fee NUMERIC(10, 2) DEFAULT 0,
    other_fee NUMERIC(10, 2) DEFAULT 0,
    product_cost NUMERIC(10, 2) DEFAULT 0,
    profit_amount NUMERIC(10, 2) DEFAULT 0,
    profit_margin NUMERIC(10, 4) DEFAULT 0,
    payment_method VARCHAR(50),
    remark TEXT,
    paid_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_no)
);

CREATE TABLE IF NOT EXISTS listing_task (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product(id),
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    sku_id BIGINT REFERENCES sku(id),
    product_listing_id BIGINT REFERENCES product_listing(id),
    source_type VARCHAR(50) NOT NULL DEFAULT 'decision',
    source_item_key VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'blocked',
    missing_requirements JSONB NOT NULL DEFAULT '[]'::jsonb,
    decision_snapshot JSONB,
    target_sale_price NUMERIC(12, 2),
    target_profit_margin NUMERIC(8, 2),
    destination_country VARCHAR(10),
    last_error TEXT,
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- ============================================================
-- Section 5: Tables with FK to Section 4 tables
-- ============================================================

CREATE TABLE IF NOT EXISTS sales_order_item (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    product_id BIGINT NOT NULL REFERENCES product(id),
    product_name VARCHAR(200) NOT NULL,
    sku_code VARCHAR(100),
    spec_desc VARCHAR(500),
    unit_price NUMERIC(10, 2) NOT NULL,
    quantity INTEGER NOT NULL,
    subtotal NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sales_order_status_log (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    from_status VARCHAR(50),
    to_status VARCHAR(50) NOT NULL,
    operator VARCHAR(100),
    remark VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sales_order_shipping_snapshot (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    quantity INTEGER NOT NULL,
    destination_country VARCHAR(10) NOT NULL,
    postal_code VARCHAR(20),
    cargo_type VARCHAR(50) DEFAULT 'normal',
    package_source VARCHAR(20),
    package_length_cm NUMERIC(10, 2) NOT NULL,
    package_width_cm NUMERIC(10, 2) NOT NULL,
    package_height_cm NUMERIC(10, 2) NOT NULL,
    package_weight_kg NUMERIC(10, 3) NOT NULL,
    provider_id BIGINT NOT NULL REFERENCES shipping_provider(id),
    provider_name VARCHAR(200) NOT NULL,
    channel_id BIGINT NOT NULL REFERENCES shipping_channel(id),
    channel_name VARCHAR(200) NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    actual_weight_kg NUMERIC(10, 4) NOT NULL,
    volumetric_weight_kg NUMERIC(10, 4) NOT NULL,
    chargeable_weight_kg NUMERIC(10, 4) NOT NULL,
    base_shipping_fee NUMERIC(10, 2) NOT NULL,
    surcharge_fee NUMERIC(10, 2) DEFAULT 0,
    fuel_surcharge_fee NUMERIC(10, 2) DEFAULT 0,
    total_shipping_fee NUMERIC(10, 2) NOT NULL,
    calculation_detail TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id)
);

CREATE TABLE IF NOT EXISTS after_sales_order (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    item_id BIGINT REFERENCES sales_order_item(id),
    sku_id BIGINT NOT NULL REFERENCES sku(id),
    return_quantity INTEGER NOT NULL,
    reason VARCHAR(200) NOT NULL,
    status VARCHAR(30) DEFAULT 'pending',
    refund_amount NUMERIC(12, 2),
    inspection_result TEXT,
    rejection_reason VARCHAR(500),
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,
    rejected_by VARCHAR(100),
    rejected_at TIMESTAMPTZ,
    received_by VARCHAR(100),
    received_at TIMESTAMPTZ,
    refunded_by VARCHAR(100),
    refunded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS settlement_item (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES settlement(id),
    transaction_type VARCHAR(30) NOT NULL,
    transaction_id VARCHAR(100),
    order_no VARCHAR(100),
    order_id BIGINT REFERENCES sales_order(id),
    sku_id BIGINT REFERENCES sku(id),
    amount NUMERIC(12, 2) DEFAULT 0,
    fee NUMERIC(12, 2) DEFAULT 0,
    net NUMERIC(12, 2) DEFAULT 0,
    quantity INTEGER DEFAULT 0,
    occurred_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reconciliation_status VARCHAR(20) DEFAULT 'pending',
    reconciliation_note TEXT,
    reconciled_at TIMESTAMPTZ,
    reconciled_by VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS finance_ledger_entry (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_order(id),
    entry_type VARCHAR(50) NOT NULL,
    amount NUMERIC(14, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    cost_layer VARCHAR(30) NOT NULL,
    source_type VARCHAR(50),
    source_id BIGINT,
    description VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_transaction (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES finance_account(id),
    transaction_type VARCHAR(50) NOT NULL,
    amount NUMERIC(14, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'CNY',
    order_id BIGINT REFERENCES sales_order(id),
    settlement_id BIGINT REFERENCES settlement(id),
    platform_id BIGINT REFERENCES platform(id),
    description VARCHAR(500),
    transaction_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cost_allocation_item (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES cost_allocation_batch(id),
    row_number INTEGER NOT NULL,
    sku_id BIGINT REFERENCES sku(id),
    sku_code VARCHAR(100),
    order_id BIGINT REFERENCES sales_order(id),
    quantity INTEGER DEFAULT 0,
    weight_kg NUMERIC(10, 3),
    volume_m3 NUMERIC(10, 4),
    item_value NUMERIC(14, 2),
    allocation_factor NUMERIC(14, 4),
    allocated_amount NUMERIC(14, 2) DEFAULT 0,
    cost_layer VARCHAR(30) DEFAULT 'allocated',
    posted_to_ledger INTEGER DEFAULT 0,
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_settlement_item (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES platform_settlement_batch(id),
    row_number INTEGER NOT NULL,
    platform VARCHAR(100),
    store_name VARCHAR(200),
    platform_order_no VARCHAR(200),
    order_no VARCHAR(200),
    transaction_type VARCHAR(50) NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    amount NUMERIC(14, 2) DEFAULT 0,
    settled_at TIMESTAMPTZ,
    description TEXT,
    match_status VARCHAR(30) DEFAULT 'unmatched',
    matched_order_id BIGINT REFERENCES sales_order(id),
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shipping_bill_item (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES shipping_bill_batch(id),
    row_number INTEGER NOT NULL,
    reconciliation_status VARCHAR(30) DEFAULT 'unmatched_bill',
    tracking_number VARCHAR(200),
    order_no VARCHAR(200),
    provider_name VARCHAR(200),
    channel_name VARCHAR(200),
    destination_country VARCHAR(10),
    billed_weight_kg NUMERIC(10, 3),
    currency VARCHAR(10) DEFAULT 'CNY',
    actual_shipping_fee NUMERIC(12, 2),
    surcharge_fee NUMERIC(12, 2) DEFAULT 0,
    total_actual_fee NUMERIC(12, 2),
    billed_at TIMESTAMPTZ,
    matched_order_id BIGINT REFERENCES sales_order(id),
    matched_snapshot_id BIGINT REFERENCES sales_order_shipping_snapshot(id),
    snapshot_shipping_fee NUMERIC(12, 2),
    variance_amount NUMERIC(12, 2),
    raw_payload JSONB,
    note TEXT,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS import_batch_row (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES import_batch(id),
    row_index INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    raw_data JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_import_item (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES order_import_batch(id),
    row_number INTEGER NOT NULL,
    platform VARCHAR(100),
    store_name VARCHAR(200),
    platform_order_no VARCHAR(200),
    order_no VARCHAR(200),
    order_id BIGINT REFERENCES sales_order(id),
    sku_code VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DOUBLE PRECISION,
    currency VARCHAR(10) DEFAULT 'CNY',
    recipient_name VARCHAR(100),
    recipient_phone VARCHAR(50),
    country_code VARCHAR(10),
    shipping_address VARCHAR(500),
    shipping_fee DOUBLE PRECISION DEFAULT 0,
    tracking_number VARCHAR(200),
    paid_at VARCHAR(50),
    status VARCHAR(50) NOT NULL,
    failure_reason VARCHAR(500),
    chain_status VARCHAR(50) DEFAULT 'chain_pending',
    chain_failure_reason VARCHAR(500),
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS listing_task_item (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES listing_task(id),
    product_id BIGINT NOT NULL REFERENCES product(id),
    platform_id BIGINT NOT NULL REFERENCES platform(id),
    status VARCHAR(50) DEFAULT 'pending',
    result JSONB,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    executed_at TIMESTAMPTZ,
    CONSTRAINT uq_task_product_platform UNIQUE (task_id, product_id, platform_id)
);

CREATE TABLE IF NOT EXISTS agentos_approval_request (
    id BIGSERIAL PRIMARY KEY,
    proposal_id BIGINT NOT NULL REFERENCES agentos_action_proposal(id),
    requester VARCHAR(100),
    approver VARCHAR(100),
    decision VARCHAR(30) NOT NULL DEFAULT 'pending',
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ,
    CONSTRAINT ck_agentos_approval_request_decision CHECK (decision IN ('pending', 'approved', 'rejected'))
);

CREATE TABLE IF NOT EXISTS agentos_command_execution (
    id BIGSERIAL PRIMARY KEY,
    proposal_id BIGINT NOT NULL REFERENCES agentos_action_proposal(id),
    command_name VARCHAR(100) NOT NULL,
    executor VARCHAR(100),
    status VARCHAR(30) NOT NULL DEFAULT 'started',
    input_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_payload JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    CONSTRAINT ck_agentos_command_execution_status CHECK (status IN ('started', 'succeeded', 'failed'))
);

CREATE TABLE IF NOT EXISTS agentos_outcome_review (
    id BIGSERIAL PRIMARY KEY,
    proposal_id BIGINT NOT NULL REFERENCES agentos_action_proposal(id),
    outcome VARCHAR(30) NOT NULL,
    business_metric VARCHAR(100),
    metric_delta NUMERIC(14, 4),
    notes TEXT,
    reviewed_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_agentos_outcome_review_outcome CHECK (outcome IN ('positive', 'neutral', 'negative'))
);


-- ============================================================
-- Section 6: Indexes (FK columns + explicit indexes)
-- ============================================================

-- product
CREATE INDEX idx_product_category_id ON product(category_id);
CREATE INDEX idx_product_brand_id ON product(brand_id);

-- spec_name
CREATE INDEX idx_spec_name_product_id ON spec_name(product_id);

-- spec_value
CREATE INDEX idx_spec_value_spec_name_id ON spec_value(spec_name_id);
CREATE INDEX idx_spec_value_product_id ON spec_value(product_id);

-- sku
CREATE INDEX idx_sku_product_id ON sku(product_id);

-- price
CREATE INDEX idx_price_sku_id ON price(sku_id);

-- price_change_log
CREATE INDEX idx_price_change_log_sku_id ON price_change_log(sku_id);

-- inventory
-- sku_id already has UNIQUE index from constraint

-- inventory_log
CREATE INDEX idx_inventory_log_sku_id ON inventory_log(sku_id);

-- inventory_warehouse
CREATE INDEX idx_inventory_warehouse_sku_id ON inventory_warehouse(sku_id);
CREATE INDEX idx_inventory_warehouse_warehouse_id ON inventory_warehouse(warehouse_id);

-- product_supplier
CREATE INDEX idx_product_supplier_product_id ON product_supplier(product_id);
CREATE INDEX idx_product_supplier_supplier_id ON product_supplier(supplier_id);

-- sourcing_1688_product
CREATE INDEX idx_sourcing_1688_product_product_id ON sourcing_1688_product(product_id);
CREATE INDEX idx_sourcing_1688_product_supplier_id ON sourcing_1688_product(supplier_id);

-- shipping_channel
CREATE INDEX idx_shipping_channel_provider_id ON shipping_channel(provider_id);

-- shipping_zone
CREATE INDEX idx_shipping_zone_channel_id ON shipping_zone(channel_id);

-- shipping_quote_rule
CREATE INDEX idx_shipping_quote_rule_channel_id ON shipping_quote_rule(channel_id);
CREATE INDEX idx_shipping_quote_rule_zone_id ON shipping_quote_rule(zone_id);

-- product_listing
CREATE INDEX idx_product_listing_product_id ON product_listing(product_id);
CREATE INDEX idx_product_listing_platform_id ON product_listing(platform_id);

-- listing_task
CREATE INDEX idx_listing_task_product_id ON listing_task(product_id);
CREATE INDEX idx_listing_task_platform_id ON listing_task(platform_id);
CREATE INDEX idx_listing_task_sku_id ON listing_task(sku_id);
CREATE INDEX idx_listing_task_product_listing_id ON listing_task(product_listing_id);

-- listing_task_item
CREATE INDEX idx_listing_task_item_task_id ON listing_task_item(task_id);
CREATE INDEX idx_listing_task_item_product_id ON listing_task_item(product_id);
CREATE INDEX idx_listing_task_item_platform_id ON listing_task_item(platform_id);

-- shipping_bill_batch
CREATE INDEX idx_shipping_bill_batch_provider_id ON shipping_bill_batch(provider_id);

-- shipping_bill_item
CREATE INDEX idx_shipping_bill_item_batch_id ON shipping_bill_item(batch_id);
CREATE INDEX idx_shipping_bill_item_matched_order_id ON shipping_bill_item(matched_order_id);
CREATE INDEX idx_shipping_bill_item_matched_snapshot_id ON shipping_bill_item(matched_snapshot_id);

-- platform_settlement_batch: no FK indexes needed

-- platform_settlement_item
CREATE INDEX idx_platform_settlement_item_batch_id ON platform_settlement_item(batch_id);
CREATE INDEX idx_platform_settlement_item_matched_order_id ON platform_settlement_item(matched_order_id);

-- finance_ledger_entry
CREATE INDEX idx_finance_ledger_entry_order_id ON finance_ledger_entry(order_id);

-- exception_item: no FK

-- agent_action
CREATE INDEX idx_agent_action_exception_id ON agent_action(exception_id);

-- platform_integration_account
CREATE INDEX idx_platform_integration_account_platform_id ON platform_integration_account(platform_id);

-- platform_category_mapping
CREATE INDEX idx_platform_category_mapping_platform_id ON platform_category_mapping(platform_id);
CREATE INDEX idx_platform_category_mapping_local_category_id ON platform_category_mapping(local_category_id);

-- platform_attribute_mapping
CREATE INDEX idx_platform_attribute_mapping_platform_id ON platform_attribute_mapping(platform_id);

-- cost_allocation_batch: no FK indexes needed

-- cost_allocation_item
CREATE INDEX idx_cost_allocation_item_batch_id ON cost_allocation_item(batch_id);
CREATE INDEX idx_cost_allocation_item_sku_id ON cost_allocation_item(sku_id);
CREATE INDEX idx_cost_allocation_item_order_id ON cost_allocation_item(order_id);

-- user_role
CREATE INDEX idx_user_role_user_id ON user_role(user_id);
CREATE INDEX idx_user_role_role_id ON user_role(role_id);

-- role_permission
CREATE INDEX idx_role_permission_role_id ON role_permission(role_id);
CREATE INDEX idx_role_permission_permission_id ON role_permission(permission_id);

-- sales_order
CREATE INDEX idx_sales_order_platform_id ON sales_order(platform_id);

-- sales_order_item
CREATE INDEX idx_sales_order_item_order_id ON sales_order_item(order_id);
CREATE INDEX idx_sales_order_item_sku_id ON sales_order_item(sku_id);
CREATE INDEX idx_sales_order_item_product_id ON sales_order_item(product_id);

-- sales_order_status_log
CREATE INDEX idx_sales_order_status_log_order_id ON sales_order_status_log(order_id);

-- sales_order_shipping_snapshot
-- order_id already has UNIQUE index from constraint
CREATE INDEX idx_sales_order_shipping_snapshot_sku_id ON sales_order_shipping_snapshot(sku_id);
CREATE INDEX idx_sales_order_shipping_snapshot_provider_id ON sales_order_shipping_snapshot(provider_id);
CREATE INDEX idx_sales_order_shipping_snapshot_channel_id ON sales_order_shipping_snapshot(channel_id);

-- import_batch_row
CREATE INDEX idx_import_batch_row_batch_id ON import_batch_row(batch_id);

-- platform_fee_rule
CREATE INDEX idx_platform_fee_rule_platform_id ON platform_fee_rule(platform_id);
CREATE INDEX idx_platform_fee_rule_category_id ON platform_fee_rule(category_id);

-- settlement
CREATE INDEX idx_settlement_platform_id ON settlement(platform_id);

-- settlement_item
CREATE INDEX idx_settlement_item_settlement_id ON settlement_item(settlement_id);
CREATE INDEX idx_settlement_item_order_id ON settlement_item(order_id);
CREATE INDEX idx_settlement_item_sku_id ON settlement_item(sku_id);

-- order_import
CREATE INDEX idx_order_import_platform_id ON order_import(platform_id);

-- warehouse: no FK

-- allocation_rule
CREATE INDEX idx_allocation_rule_warehouse_id ON allocation_rule(warehouse_id);

-- finance_account
CREATE INDEX idx_finance_account_platform_id ON finance_account(platform_id);

-- finance_transaction
CREATE INDEX idx_finance_transaction_account_id ON finance_transaction(account_id);
CREATE INDEX idx_finance_transaction_order_id ON finance_transaction(order_id);
CREATE INDEX idx_finance_transaction_settlement_id ON finance_transaction(settlement_id);
CREATE INDEX idx_finance_transaction_platform_id ON finance_transaction(platform_id);

-- notification
CREATE INDEX idx_notification_user_id ON notification(user_id);

-- stores
CREATE INDEX idx_stores_user_id ON stores(user_id);
CREATE INDEX idx_stores_platform_id ON stores(platform_id);

-- product_image_gen
CREATE INDEX idx_product_image_gen_product_id ON product_image_gen(product_id);

-- product_canvases
CREATE INDEX idx_product_canvases_product_id ON product_canvases(product_id);
CREATE INDEX idx_product_canvases_created_by ON product_canvases(created_by);

-- honcho_profile: user_id has UNIQUE

-- agent_decision
CREATE INDEX idx_agent_decision_user_id ON agent_decision(user_id);

-- personal_rule
CREATE INDEX idx_personal_rule_user_id ON personal_rule(user_id);

-- agent_episode
CREATE INDEX idx_agent_episode_user_id ON agent_episode(user_id);

-- agent_pending_action
CREATE INDEX idx_agent_pending_action_user_id ON agent_pending_action(user_id);
CREATE INDEX idx_agent_pending_action_decision_id ON agent_pending_action(decision_id);

-- rule_conflict
CREATE INDEX idx_rule_conflict_decision_id ON rule_conflict(decision_id);

-- spc_control_limit
CREATE INDEX idx_spc_control_limit_user_id ON spc_control_limit(user_id);

-- agent_evolution_config
CREATE INDEX idx_agent_evolution_config_user_id ON agent_evolution_config(user_id);

-- agent_nudge
CREATE INDEX idx_agent_nudge_user_id ON agent_nudge(user_id);

-- after_sales_order
CREATE INDEX idx_after_sales_order_order_id ON after_sales_order(order_id);
CREATE INDEX idx_after_sales_order_item_id ON after_sales_order(item_id);
CREATE INDEX idx_after_sales_order_sku_id ON after_sales_order(sku_id);

-- agentos_operation_log
CREATE INDEX idx_agentos_operation_log_user_id ON agentos_operation_log(user_id);

-- agentos_action_proposal
CREATE INDEX idx_agentos_action_proposal_source_type ON agentos_action_proposal(source_type);
CREATE INDEX idx_agentos_action_proposal_source_id ON agentos_action_proposal(source_id);
CREATE INDEX idx_agentos_action_proposal_agent_id ON agentos_action_proposal(agent_id);
CREATE INDEX idx_agentos_action_proposal_squad_id ON agentos_action_proposal(squad_id);
CREATE INDEX idx_agentos_action_proposal_action_type ON agentos_action_proposal(action_type);
CREATE INDEX idx_agentos_action_proposal_business_object_type ON agentos_action_proposal(business_object_type);
CREATE INDEX idx_agentos_action_proposal_business_object_id ON agentos_action_proposal(business_object_id);
CREATE INDEX idx_agentos_action_proposal_risk_level ON agentos_action_proposal(risk_level);
CREATE INDEX idx_agentos_action_proposal_status ON agentos_action_proposal(status);

-- agentos_approval_request
CREATE INDEX idx_agentos_approval_request_proposal_id ON agentos_approval_request(proposal_id);

-- agentos_command_execution
CREATE INDEX idx_agentos_command_execution_proposal_id ON agentos_command_execution(proposal_id);

-- agentos_outcome_review
CREATE INDEX idx_agentos_outcome_review_proposal_id ON agentos_outcome_review(proposal_id);

-- order_import_batch: no FK
-- order_import_item
CREATE INDEX idx_order_import_item_batch_id ON order_import_item(batch_id);
CREATE INDEX idx_order_import_item_order_id ON order_import_item(order_id);

-- End of init schema
