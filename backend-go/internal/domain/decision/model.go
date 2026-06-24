package decision

import (
	"time"
)

// PreListingDecision maps to "pre_listing_decision".
type PreListingDecision struct {
	ID                   int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID                int64      `gorm:"column:sku_id;not null;index" json:"sku_id"`
	PlatformID           *int64     `gorm:"column:platform_id" json:"platform_id,omitempty"`
	CountryCode          string     `gorm:"column:country_code" json:"country_code"`
	DecisionPoint        string     `gorm:"column:decision_point;default:pre_listing" json:"decision_point"`
	EstimatedRevenue     float64    `gorm:"column:estimated_revenue;default:0" json:"estimated_revenue"`
	EstimatedProductCost float64    `gorm:"column:estimated_product_cost;default:0" json:"estimated_product_cost"`
	EstimatedShippingCost float64   `gorm:"column:estimated_shipping_cost;default:0" json:"estimated_shipping_cost"`
	EstimatedPlatformFee float64    `gorm:"column:estimated_platform_fee;default:0" json:"estimated_platform_fee"`
	EstimatedPaymentFee  float64    `gorm:"column:estimated_payment_fee;default:0" json:"estimated_payment_fee"`
	EstimatedOtherFee    float64    `gorm:"column:estimated_other_fee;default:0" json:"estimated_other_fee"`
	EstimatedProfit      float64    `gorm:"column:estimated_profit;default:0" json:"estimated_profit"`
	ProfitMargin         float64    `gorm:"column:profit_margin;default:0" json:"profit_margin"`
	ConfidenceScore      float64    `gorm:"column:confidence_score;default:0" json:"confidence_score"`
	RiskLevel            string     `gorm:"column:risk_level;default:medium" json:"risk_level"`
	Recommendation       string     `gorm:"column:recommendation" json:"recommendation"`
	Reasoning            string     `gorm:"column:reasoning" json:"reasoning"`
	Status               string     `gorm:"column:status;default:pending" json:"status"`
	DecidedBy            string     `gorm:"column:decided_by" json:"decided_by"`
	DecidedAt            *time.Time `gorm:"column:decided_at" json:"decided_at,omitempty"`
	TraceID              string     `gorm:"column:trace_id" json:"trace_id"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PreListingDecision) TableName() string { return "pre_listing_decision" }

// CreateInput is the payload for POST /decision.
type CreateInput struct {
	SkuID                int64   `json:"sku_id" binding:"required"`
	PlatformID           *int64  `json:"platform_id"`
	CountryCode          string  `json:"country_code"`
	DecisionPoint        string  `json:"decision_point"`
	EstimatedRevenue     *float64 `json:"estimated_revenue"`
	EstimatedProductCost *float64 `json:"estimated_product_cost"`
	EstimatedShippingCost *float64 `json:"estimated_shipping_cost"`
	EstimatedPlatformFee *float64 `json:"estimated_platform_fee"`
	EstimatedPaymentFee  *float64 `json:"estimated_payment_fee"`
	EstimatedOtherFee    *float64 `json:"estimated_other_fee"`
	EstimatedProfit      *float64 `json:"estimated_profit"`
	ProfitMargin         *float64 `json:"profit_margin"`
	ConfidenceScore      *float64 `json:"confidence_score"`
	RiskLevel            string  `json:"risk_level"`
	Recommendation       string  `json:"recommendation"`
	Reasoning            string  `json:"reasoning"`
	Status               string  `json:"status"`
	TraceID              string  `json:"trace_id"`
}

// UpdateInput allows partial updates.
type UpdateInput struct {
	PlatformID            *int64   `json:"platform_id"`
	CountryCode           *string  `json:"country_code"`
	EstimatedRevenue      *float64 `json:"estimated_revenue"`
	EstimatedProductCost  *float64 `json:"estimated_product_cost"`
	EstimatedShippingCost *float64 `json:"estimated_shipping_cost"`
	EstimatedPlatformFee  *float64 `json:"estimated_platform_fee"`
	EstimatedPaymentFee   *float64 `json:"estimated_payment_fee"`
	EstimatedOtherFee     *float64 `json:"estimated_other_fee"`
	EstimatedProfit       *float64 `json:"estimated_profit"`
	ProfitMargin          *float64 `json:"profit_margin"`
	ConfidenceScore       *float64 `json:"confidence_score"`
	RiskLevel             *string  `json:"risk_level"`
	Recommendation        *string  `json:"recommendation"`
	Reasoning             *string  `json:"reasoning"`
	Status                *string  `json:"status"`
	TraceID               *string  `json:"trace_id"`
}

// ListFilter captures query parameters.
type ListFilter struct {
	Search     string
	SkuID      *int64
	PlatformID *int64
	Status     string
	RiskLevel  string
}

// ApproveInput is the body for POST /decision/:id/approve.
type ApproveInput struct {
	DecidedBy string `json:"decided_by"`
}

// RejectInput is the body for POST /decision/:id/reject.
type RejectInput struct {
	DecidedBy string `json:"decided_by"`
	Reason    string `json:"reason"`
}

// Summary is the aggregation payload.
type Summary struct {
	Total             int64            `json:"total"`
	ByRecommendation  map[string]int64 `json:"by_recommendation"`
	ByRiskLevel       map[string]int64 `json:"by_risk_level"`
	ByStatus          map[string]int64 `json:"by_status"`
}
