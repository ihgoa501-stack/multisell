package agentlearning

import "time"

// DecisionEvaluation records the comparison between an agent's predicted
// outcome and the actual result after a specified observation period.
type DecisionEvaluation struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DecisionTraceID  int64      `gorm:"column:decision_trace_id;not null;index" json:"decision_trace_id"`
	ProductID        int64      `gorm:"column:product_id;not null;index" json:"product_id"`
	AgentID          string     `gorm:"column:agent_id;not null;index" json:"agent_id"`
	PredictedOutcome string     `gorm:"column:predicted_outcome;type:text" json:"predicted_outcome"`
	ActualOutcome    string     `gorm:"column:actual_outcome;type:text" json:"actual_outcome,omitempty"`
	Score            float64    `gorm:"column:score;default:0" json:"score"` // how accurate was the prediction (0.0 - 1.0)
	EvaluatedAt      *time.Time `gorm:"column:evaluated_at" json:"evaluated_at"`
	EvaluationType   string     `gorm:"column:evaluation_type;default:T+30" json:"evaluation_type"` // T+30, T+60, T+90
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (DecisionEvaluation) TableName() string { return "decision_evaluation" }

// AgentAccuracy stores aggregate accuracy metrics for an agent over a period.
type AgentAccuracy struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AgentID          string    `gorm:"column:agent_id;not null;uniqueIndex:idx_agent_period" json:"agent_id"`
	Period           string    `gorm:"column:period;not null;uniqueIndex:idx_agent_period" json:"period"` // 7d, 30d, 90d
	TotalDecisions   int       `gorm:"column:total_decisions;default:0" json:"total_decisions"`
	CorrectDecisions int       `gorm:"column:correct_decisions;default:0" json:"correct_decisions"`
	AccuracyPct      float64   `gorm:"column:accuracy_pct;default:0" json:"accuracy_pct"`
	Trend            string    `gorm:"column:trend;default:stable" json:"trend"` // improving, stable, declining
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AgentAccuracy) TableName() string { return "agent_accuracy" }
