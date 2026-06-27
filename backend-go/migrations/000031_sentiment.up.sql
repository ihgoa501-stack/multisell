CREATE TABLE IF NOT EXISTS product_sentiment (
    id              BIGSERIAL       PRIMARY KEY,
    product_id      BIGINT          NOT NULL UNIQUE,

    -- Ratings & reviews
    avg_rating      DECIMAL(3,2)    NOT NULL DEFAULT 0,
    review_count    INTEGER         NOT NULL DEFAULT 0,
    positive_pct    DECIMAL(5,2)    NOT NULL DEFAULT 0,
    negative_pct    DECIMAL(5,2)    NOT NULL DEFAULT 0,

    -- Returns
    return_rate     DECIMAL(5,2)    NOT NULL DEFAULT 0,

    -- Analysis
    top_complaints  TEXT            NOT NULL DEFAULT '[]',
    sentiment_score DECIMAL(5,2)    NOT NULL DEFAULT 0,

    last_updated    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_sentiment_score ON product_sentiment (sentiment_score);
CREATE INDEX idx_product_sentiment_product ON product_sentiment (product_id);
