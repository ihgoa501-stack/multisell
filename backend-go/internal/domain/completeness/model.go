package completeness

import (
	"time"
)

// CompletenessCheck maps to the "completeness_check" table.
type CompletenessCheck struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID       int64           `gorm:"column:product_id;not null;index" json:"product_id"`
	Score           float64         `gorm:"column:score;default:0" json:"score"`                   // 0-100
	MissingItems    string          `gorm:"column:missing_items;type:text" json:"missing_items"`    // JSON array of missing dimensions
	ScoreBreakdown  string          `gorm:"column:score_breakdown;type:text" json:"score_breakdown"` // JSON detail per dimension
	Status          string          `gorm:"column:status;default:incomplete" json:"status"`         // complete, incomplete
	TriggeredBy     string          `gorm:"column:triggered_by" json:"triggered_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CompletenessCheck) TableName() string { return "completeness_check" }

// CompletenessDimension is a scored dimension in the completeness check.
type CompletenessDimension struct {
	Dimension string `json:"dimension"`
	Label     string `json:"label"`
	Score     float64 `json:"score"`     // 0-100
	Weight    float64 `json:"weight"`    // contribution to total score
	Complete  bool    `json:"complete"`
	Reason    string  `json:"reason,omitempty"`
}

// CheckResult is the API response for a completeness check.
type CheckResult struct {
	ProductID    int64                    `json:"product_id"`
	Score        float64                  `json:"score"`
	Status       string                   `json:"status"` // complete, incomplete
	Dimensions   []CompletenessDimension  `json:"dimensions"`
	MissingItems []string                 `json:"missing_items"`
	Breakdown    string                   `json:"breakdown,omitempty"`
}

// CheckInput is the payload for requesting a completeness check.
type CheckInput struct {
	TriggeredBy string `json:"triggered_by"`
}

// CompletenessReport is the enhanced API response with economic estimates
// from profit, logistics, and platform fee services.
type CompletenessReport struct {
	CandidateID          int64    `json:"candidate_id"`
	BaseInfoScore        float64  `json:"base_info_score"`          // 基础信息完整度 0-1
	CostScore            float64  `json:"cost_score"`               // 成本完整度
	LogisticsScore       float64  `json:"logistics_score"`         // 物流完整度
	PlatformFeeScore     float64  `json:"platform_fee_score"`      // 平台费完整度
	ProfitScore          float64  `json:"profit_score"`            // 利润可测算
	OverallScore         float64  `json:"overall_score"`           // 综合
	MissingFields        []string `json:"missing_fields"`
	EstimatedProfit      *float64 `json:"estimated_profit,omitempty"`
	EstimatedMargin      *float64 `json:"estimated_margin,omitempty"`
	EstimatedLogistics   *float64 `json:"estimated_logistics,omitempty"`
	EstimatedPlatformFee *float64 `json:"estimated_platform_fee,omitempty"`
}
