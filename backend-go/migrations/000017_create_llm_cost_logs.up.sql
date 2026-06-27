CREATE TABLE llm_cost_logs (
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

CREATE INDEX idx_llm_cost_logs_date   ON llm_cost_logs(window_date);
CREATE INDEX idx_llm_cost_logs_user   ON llm_cost_logs(user_id, window_date);
CREATE INDEX idx_llm_cost_logs_agent  ON llm_cost_logs(agent_id, window_date);
