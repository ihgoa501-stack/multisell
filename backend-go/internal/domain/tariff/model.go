package tariff

import (
	"time"
)

// TariffRule maps to "tariff_rule".
type TariffRule struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CountryCode       string     `gorm:"column:country_code;not null" json:"country_code"`
	HSCode            string     `gorm:"column:hs_code" json:"hs_code,omitempty"`
	HSCodePrefix      string     `gorm:"column:hs_code_prefix" json:"hs_code_prefix,omitempty"`
	DutyRatePct       float64    `gorm:"column:duty_rate_pct;default:0" json:"duty_rate_pct"`
	VatRatePct        float64    `gorm:"column:vat_rate_pct;default:0" json:"vat_rate_pct"`
	OtherTaxRatePct   float64    `gorm:"column:other_tax_rate_pct;default:0" json:"other_tax_rate_pct"`
	MinThresholdUSD   float64    `gorm:"column:min_threshold_usd;default:0" json:"min_threshold_usd"`
	MaxThresholdUSD   float64    `gorm:"column:max_threshold_usd;default:0" json:"max_threshold_usd"`
	Incoterm          string     `gorm:"column:incoterm;not null;default:DDU" json:"incoterm"` // DDP / DDU
	Priority          int        `gorm:"column:priority;default:0" json:"priority"`
	EffectiveFrom     *time.Time `gorm:"column:effective_from" json:"effective_from,omitempty"`
	EffectiveTo       *time.Time `gorm:"column:effective_to" json:"effective_to,omitempty"`
	Status            string     `gorm:"column:status;default:active" json:"status"` // active / inactive
	Remark            string     `gorm:"column:remark" json:"remark"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TariffRule) TableName() string { return "tariff_rule" }

// ---------- Input / DTO structs ----------

// CreateRuleInput is the payload for POST /tariff.
type CreateRuleInput struct {
	CountryCode     string     `json:"country_code" binding:"required"`
	HSCode          string     `json:"hs_code"`
	HSCodePrefix    string     `json:"hs_code_prefix"`
	DutyRatePct     *float64   `json:"duty_rate_pct"`
	VatRatePct      *float64   `json:"vat_rate_pct"`
	OtherTaxRatePct *float64   `json:"other_tax_rate_pct"`
	MinThresholdUSD *float64   `json:"min_threshold_usd"`
	MaxThresholdUSD *float64   `json:"max_threshold_usd"`
	Incoterm        string     `json:"incoterm"`
	Priority        *int       `json:"priority"`
	EffectiveFrom   *time.Time `json:"effective_from"`
	EffectiveTo     *time.Time `json:"effective_to"`
	Status          string     `json:"status"`
	Remark          string     `json:"remark"`
}

// UpdateRuleInput allows partial updates.
type UpdateRuleInput struct {
	CountryCode     *string    `json:"country_code"`
	HSCode          *string    `json:"hs_code"`
	HSCodePrefix    *string    `json:"hs_code_prefix"`
	DutyRatePct     *float64   `json:"duty_rate_pct"`
	VatRatePct      *float64   `json:"vat_rate_pct"`
	OtherTaxRatePct *float64   `json:"other_tax_rate_pct"`
	MinThresholdUSD *float64   `json:"min_threshold_usd"`
	MaxThresholdUSD *float64   `json:"max_threshold_usd"`
	Incoterm        *string    `json:"incoterm"`
	Priority        *int       `json:"priority"`
	EffectiveFrom   *time.Time `json:"effective_from"`
	EffectiveTo     *time.Time `json:"effective_to"`
	Status          *string    `json:"status"`
	Remark          *string    `json:"remark"`
}

// RuleListFilter captures query parameters.
type RuleListFilter struct {
	CountryCode string
	HSCode      string
	Incoterm    string
	Status      string
}

// DecisionRequest is the payload for POST /tariff/decide.
type DecisionRequest struct {
	DestinationCountry string  `json:"destination_country" binding:"required"`
	ProductValueUSD    float64 `json:"product_value_usd" binding:"required"`
	HSCode             string  `json:"hs_code"`
	Quantity           int     `json:"quantity"`
	CargoType          string  `json:"cargo_type"`
}

// DecisionResult is the response for POST /tariff/decide.
type DecisionResult struct {
	Incoterm         string          `json:"incoterm"` // DDP / DDU
	TotalDutyTaxUSD  float64         `json:"total_duty_tax_usd"`
	DutyAmountUSD    float64         `json:"duty_amount_usd"`
	VatAmountUSD     float64         `json:"vat_amount_usd"`
	OtherTaxAmountUSD float64        `json:"other_tax_amount_usd"`
	RulesMatched     []RuleMatchItem `json:"rules_matched"`
	IncotermReason   string          `json:"incoterm_reason"`
	TotalValueUSD    float64         `json:"total_value_usd"`
}

// RuleMatchItem describes a single matched tariff rule and its contribution.
type RuleMatchItem struct {
	RuleID          int64   `json:"rule_id"`
	CountryCode     string  `json:"country_code"`
	HSCode          string  `json:"hs_code,omitempty"`
	HSCodePrefix    string  `json:"hs_code_prefix,omitempty"`
	DutyRatePct     float64 `json:"duty_rate_pct"`
	VatRatePct      float64 `json:"vat_rate_pct"`
	OtherTaxRatePct float64 `json:"other_tax_rate_pct"`
	Incoterm        string  `json:"incoterm"`
	Priority        int     `json:"priority"`
	DutyAmountUSD   float64 `json:"duty_amount_usd"`
	VatAmountUSD    float64 `json:"vat_amount_usd"`
	OtherTaxAmount  float64 `json:"other_tax_amount_usd"`
}
