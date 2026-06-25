// Package observability provides agent metrics collection, cross-agent trace
// linkage, and anomaly detection for the AIOS platform.
//
// This package implements Section 10 of the AIOS architecture specification.
// It is the in-memory metrics aggregation layer that feeds the DB-backed
// Observer queries described in the architecture doc.
package observability

import "time"

// AgentMetrics is a per-agent metrics snapshot over a time period.
//
// It captures volume, quality, risk, and cost dimensions for one agent.
// Each snapshot covers PeriodStart to PeriodEnd and is recorded by the
// Collector for aggregation and anomaly scanning.
type AgentMetrics struct {
	AgentID     string    `json:"agent_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	// Volume
	DecisionsMade   int `json:"decisions_made"`
	ActionsCreated  int `json:"actions_created"`
	ActionsApproved int `json:"actions_approved"`
	ActionsRejected int `json:"actions_rejected"`
	ActionsExecuted int `json:"actions_executed"`

	// Quality
	AverageConfidence float64 `json:"avg_confidence"`
	AvgLatencyMs      int     `json:"avg_latency_ms"`
	SuccessRate       float64 `json:"success_rate"`

	// Risk
	HighRiskActions int `json:"high_risk_actions"`

	// Cost
	TokensUsed       int64   `json:"tokens_used"`
	EstimatedCostUsd float64 `json:"estimated_cost_usd"`
	ToolCallsMade    int     `json:"tool_calls_made"`
}
