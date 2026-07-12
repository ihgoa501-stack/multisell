-- ADR-001 unit 2: keep research evaluation separate from immutable Owner
-- decisions, then create Owner-scoped product opportunities.
CREATE TABLE market_owner_decision (
    id BIGSERIAL PRIMARY KEY,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE RESTRICT,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    verdict_id BIGINT NOT NULL REFERENCES demand_verdict(id) ON DELETE RESTRICT,
    decision VARCHAR(32) NOT NULL CHECK (decision IN ('selected','rejected','paused','request_more_evidence')),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    evidence_hash VARCHAR(64) NOT NULL CHECK (length(evidence_hash) = 64),
    idempotency_key VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key)
);
CREATE INDEX idx_market_owner_decision_case ON market_owner_decision(demand_case_id, id DESC);

CREATE TABLE product_opportunity (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    demand_case_id BIGINT NOT NULL REFERENCES demand_case(id) ON DELETE RESTRICT,
    market_decision_id BIGINT NOT NULL REFERENCES market_owner_decision(id) ON DELETE RESTRICT,
    title VARCHAR(240) NOT NULL,
    consumer_problem TEXT NOT NULL,
    product_thesis TEXT NOT NULL,
    target_channel VARCHAR(160) NOT NULL,
    value_hypothesis TEXT NOT NULL,
    price_hypothesis TEXT NOT NULL,
    source_uri TEXT NOT NULL,
    truth_status VARCHAR(16) NOT NULL CHECK (truth_status IN ('quoted','estimated')),
    strongest_counterevidence TEXT NOT NULL,
    unknowns_json TEXT NOT NULL DEFAULT '[]',
    stop_condition TEXT NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft','evidence_missing','ready_for_owner','approved','rejected','paused')),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    content_hash VARCHAR(64) NOT NULL CHECK (length(content_hash) = 64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_product_opportunity_owner_status ON product_opportunity(owner_id, status, id DESC);
CREATE INDEX idx_product_opportunity_case ON product_opportunity(demand_case_id, id DESC);

CREATE TABLE product_opportunity_decision (
    id BIGSERIAL PRIMARY KEY,
    opportunity_id BIGINT NOT NULL REFERENCES product_opportunity(id) ON DELETE RESTRICT,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    version BIGINT NOT NULL CHECK (version > 0),
    content_hash VARCHAR(64) NOT NULL CHECK (length(content_hash) = 64),
    decision VARCHAR(16) NOT NULL CHECK (decision IN ('approved','rejected','paused')),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    idempotency_key VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, idempotency_key),
    UNIQUE(opportunity_id, version)
);

INSERT INTO permission (name, code, module, description) VALUES
    ('市场与机会读取', 'market.read', 'market', '查看 Owner 候选市场、裁决和商品机会'),
    ('市场研究写入', 'market.write', 'market', '创建研究案件、导入证据并生成商品机会草案'),
    ('Owner 市场与机会裁决', 'market.decide', 'market', '由 Owner 选择市场并批准或淘汰商品机会')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role r CROSS JOIN permission p
WHERE r.code = 'admin' AND p.code IN ('market.read','market.write','market.decide')
  AND NOT EXISTS (SELECT 1 FROM role_permission rp WHERE rp.role_id=r.id AND rp.permission_id=p.id);
