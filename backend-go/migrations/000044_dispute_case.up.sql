-- +migrate Up
-- +migrate StatementBegin

CREATE TABLE IF NOT EXISTS dispute_case (
    id              BIGSERIAL    PRIMARY KEY,
    transaction_id  VARCHAR(255) NOT NULL,
    platform        VARCHAR(50)  NOT NULL,
    claim_type      VARCHAR(50)  NOT NULL,
    amount          NUMERIC(12,2) NOT NULL DEFAULT 0,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    evidence        TEXT         NOT NULL DEFAULT '',
    decision_score  NUMERIC(5,2) NOT NULL DEFAULT 0,
    ai_reason       TEXT         NOT NULL DEFAULT '',
    decision_source VARCHAR(20)  NOT NULL DEFAULT 'rule',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dispute_case_transaction_id ON dispute_case(transaction_id);
CREATE INDEX idx_dispute_case_platform      ON dispute_case(platform);
CREATE INDEX idx_dispute_case_status        ON dispute_case(status);

-- +migrate StatementEnd
