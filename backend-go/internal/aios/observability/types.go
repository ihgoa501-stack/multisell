package observability

import "time"

// DecisionPointStats holds aggregated statistics for a single decision point
// across multiple executions.
type DecisionPointStats struct {
	DecisionPoint     string         `json:"decision_point"`
	TotalExecutions   int            `json:"total_executions"`
	AverageConfidence float64        `json:"avg_confidence"`
	AverageLatencyMs  int            `json:"avg_latency_ms"`
	SuccessRate       float64        `json:"success_rate"`
	FailureBreakdown  map[string]int `json:"failure_breakdown"`
}

// CostRow is a single row in a cost-breakdown report, grouped by a dimension
// such as agent, squad, or decision point.
type CostRow struct {
	Dimension   string  `json:"dimension"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens int64   `json:"total_tokens"`
	CallCount   int     `json:"call_count"`
}

// AnomalyReport is automatically generated when agent behavior deviates from
// historical patterns beyond the configured threshold.
type AnomalyReport struct {
	AgentID     string    `json:"agent_id"`
	Type        string    `json:"type"`     // "confidence_drop" | "risk_spike" | "latency_spike" | "failure_burst"
	Severity    string    `json:"severity"` // "warning" | "critical"
	TriggeredAt time.Time `json:"triggered_at"`
	Details     string    `json:"details"`
}
