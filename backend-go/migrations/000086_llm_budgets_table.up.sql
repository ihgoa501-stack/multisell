CREATE TABLE IF NOT EXISTS llm_budgets (
    id BIGSERIAL PRIMARY KEY,
    monthly_limit_usd DOUBLE PRECISION NOT NULL,
    current_month_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    budget_month VARCHAR(7) NOT NULL DEFAULT '',
    is_paused BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the initial row so the first query always finds it.
INSERT INTO llm_budgets (id, monthly_limit_usd, current_month_usd, budget_month, is_paused)
VALUES (1, 0, 0, to_char(CURRENT_DATE, 'YYYY-MM'), false)
ON CONFLICT (id) DO NOTHING;
