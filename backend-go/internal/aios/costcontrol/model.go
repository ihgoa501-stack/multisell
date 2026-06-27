// Package costcontrol provides LLM budget management: daily caps, burst detection,
// and audit logging. Integrates into the AI orchestrator to gate LLM calls.
package costcontrol

import (
	"time"

	"gorm.io/gorm"
)

// CostLog records one LLM invocation for cost auditing.
type CostLog struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement"`
	UserID      int64           `gorm:"column:user_id;not null;default:0"`
	AgentID     string          `gorm:"column:agent_id;size:50;not null;default:''"`
	Model       string          `gorm:"column:model;size:50;not null;default:''"`
	TokensIn    int             `gorm:"column:tokens_in;not null;default:0"`
	TokensOut   int             `gorm:"column:tokens_out;not null;default:0"`
	CostUSD     float64         `gorm:"column:cost_usd;type:numeric(12,6);not null;default:0"`
	RequestHash string          `gorm:"column:request_hash;size:64;not null;default:''"`
	Cached      bool            `gorm:"column:cached;not null;default:false"`
	WindowDate  string          `gorm:"column:window_date;type:date;not null;default:CURRENT_DATE"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides GORM's default table name.
func (CostLog) TableName() string {
	return "llm_cost_logs"
}

// CostLogSummary is the daily rollup from a SUM query.
type CostLogSummary struct {
	WindowDate string  `json:"window_date"`
	TotalCost  float64 `json:"total_cost_usd"`
	TotalCalls int     `json:"total_calls"`
}

// DailySpend returns total cost in USD for today from the DB.
func DailySpend(db *gorm.DB) (*CostLogSummary, error) {
	var s CostLogSummary
	err := db.Model(&CostLog{}).
		Select("window_date, SUM(cost_usd) AS total_cost_usd, COUNT(*) AS total_calls").
		Where("window_date = CURRENT_DATE").
		Group("window_date").
		Take(&s).Error
	if err != nil {
		return &CostLogSummary{}, nil // empty result when no rows
	}
	return &s, nil
}
