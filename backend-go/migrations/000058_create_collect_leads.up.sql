CREATE TABLE IF NOT EXISTS collect_leads (
    id              BIGSERIAL PRIMARY KEY,
    title           TEXT NOT NULL DEFAULT '',
    price_range     VARCHAR(128) NOT NULL DEFAULT '',
    detail_url      VARCHAR(2048) NOT NULL DEFAULT '',
    image_url       VARCHAR(2048) NOT NULL DEFAULT '',
    shop_hint       VARCHAR(256) NOT NULL DEFAULT '',
    source_page_url VARCHAR(2048) NOT NULL DEFAULT '',
    status          VARCHAR(64) NOT NULL DEFAULT 'pending_detail_collect',
    collected_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_collect_leads_status ON collect_leads(status);
CREATE INDEX IF NOT EXISTS idx_collect_leads_detail_url ON collect_leads(detail_url);
