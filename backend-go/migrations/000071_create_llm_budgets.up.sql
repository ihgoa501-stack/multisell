CREATE TABLE IF NOT EXISTS llm_budgets (
    id SERIAL PRIMARY KEY,
    monthly_limit_usd NUMERIC(10,2) NOT NULL DEFAULT 200.00,
    current_month_usd NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    budget_month VARCHAR(7) DEFAULT to_char(now(), 'YYYY-MM'),
    is_paused BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT now()
);

INSERT INTO llm_budgets (monthly_limit_usd, budget_month)
VALUES (200.00, to_char(now(), 'YYYY-MM'))
ON CONFLICT DO NOTHING;
