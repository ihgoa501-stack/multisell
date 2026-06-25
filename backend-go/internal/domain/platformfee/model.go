package platformfee

import (
	"time"
)

// PlatformFeeRule maps to "platform_fee_rule".
type PlatformFeeRule struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID   *int64     `gorm:"column:platform_id;index" json:"platform_id,omitempty"`
	CountryCode  string     `gorm:"column:country_code" json:"country_code"`
	CategoryID   *int64     `gorm:"column:category_id" json:"category_id,omitempty"`
	FeeType      string     `gorm:"column:fee_type;not null" json:"fee_type"` // commission/fixed/payment/storage/other
	FeeRatePct   float64    `gorm:"column:fee_rate_pct;default:0" json:"fee_rate_pct"`
	FixedAmount  float64    `gorm:"column:fixed_amount;default:0" json:"fixed_amount"`
	MinAmount    float64    `gorm:"column:min_amount;default:0" json:"min_amount"`
	MaxAmount    float64    `gorm:"column:max_amount;default:0" json:"max_amount"`
	Currency     string     `gorm:"column:currency;default:CNY" json:"currency"`
	EffectiveFrom *time.Time `gorm:"column:effective_from" json:"effective_from,omitempty"`
	EffectiveTo   *time.Time `gorm:"column:effective_to" json:"effective_to,omitempty"`
	Priority     int        `gorm:"column:priority;default:0" json:"priority"`
	Status       string     `gorm:"column:status;default:active" json:"status"` // active/inactive
	Remark       string     `gorm:"column:remark" json:"remark"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PlatformFeeRule) TableName() string { return "platform_fee_rule" }

// ---------- Input / DTO structs ----------

// CreateRuleInput is the payload for POST /platform-fee.
type CreateRuleInput struct {
	PlatformID    *int64     `json:"platform_id"`
	CountryCode   string     `json:"country_code"`
	CategoryID    *int64     `json:"category_id"`
	FeeType       string     `json:"fee_type" binding:"required"`
	FeeRatePct    *float64   `json:"fee_rate_pct"`
	FixedAmount   *float64   `json:"fixed_amount"`
	MinAmount     *float64   `json:"min_amount"`
	MaxAmount     *float64   `json:"max_amount"`
	Currency      string     `json:"currency"`
	EffectiveFrom *time.Time `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to"`
	Priority      *int       `json:"priority"`
	Status        string     `json:"status"`
	Remark        string     `json:"remark"`
}

// UpdateRuleInput allows partial updates.
type UpdateRuleInput struct {
	PlatformID    *int64     `json:"platform_id"`
	CountryCode   *string    `json:"country_code"`
	CategoryID    *int64     `json:"category_id"`
	FeeType       *string    `json:"fee_type"`
	FeeRatePct    *float64   `json:"fee_rate_pct"`
	FixedAmount   *float64   `json:"fixed_amount"`
	MinAmount     *float64   `json:"min_amount"`
	MaxAmount     *float64   `json:"max_amount"`
	Currency      *string    `json:"currency"`
	EffectiveFrom *time.Time `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to"`
	Priority      *int       `json:"priority"`
	Status        *string    `json:"status"`
	Remark        *string    `json:"remark"`
}

// RuleListFilter captures query parameters.
type RuleListFilter struct {
	PlatformID *int64
	FeeType    string
	Status     string
}

// CalculateRequest is the payload for POST /platform-fee/calculate.
type CalculateRequest struct {
	PlatformID  int64   `json:"platform_id" binding:"required"`
	CategoryID  *int64  `json:"category_id"`
	CountryCode string  `json:"country_code"`
	Amount      float64 `json:"amount" binding:"required"`
}

// CalculateResult is the response for POST /platform-fee/calculate.
type CalculateResult struct {
	RuleID        int64   `json:"rule_id"`
	FeeType       string  `json:"fee_type"`
	FeeRatePct    float64 `json:"fee_rate_pct"`
	FixedAmount   float64 `json:"fixed_amount"`
	CalculatedFee float64 `json:"calculated_fee"`
	MinAmount     float64 `json:"min_amount"`
	MaxAmount     float64 `json:"max_amount"`
	Currency      string  `json:"currency"`
	Matched       bool    `json:"matched"`
}
