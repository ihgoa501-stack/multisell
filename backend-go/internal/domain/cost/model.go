package cost

// DailyCostVO shows daily LLM cost aggregation.
type DailyCostVO struct {
	Date    string  `json:"date"`
	CostUSD float64 `json:"cost_usd"`
	Calls   int     `json:"calls"`
}

// AgentCostVO shows per-agent LLM cost aggregation.
type AgentCostVO struct {
	AgentID string  `json:"agent_id"`
	CostUSD float64 `json:"cost_usd"`
	Calls   int     `json:"calls"`
}

// CostDashboardVO is the full cost dashboard response.
type CostDashboardVO struct {
	Today       DailyCostVO   `json:"today"`
	Last7Days   []DailyCostVO `json:"last_7_days"`
	ByAgent     []AgentCostVO `json:"by_agent"`
	DailyBudget float64       `json:"daily_budget_usd"`
	BudgetUsed  float64       `json:"budget_used_usd"`
	BudgetPct   float64       `json:"budget_used_pct"`
}
