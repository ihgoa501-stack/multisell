package landedcost

import (
	"time"
)

// LandedCost maps to the "landed_cost" table.
// Each row records a complete landed cost calculation for a product on a platform.
type LandedCost struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID int64 `gorm:"column:product_id;not null;index" json:"product_id"`
	PlatformID int64 `gorm:"column:platform_id;not null" json:"platform_id"`

	// Cost components
	UnitCostCNY    float64 `gorm:"column:unit_cost_cny;type:decimal(12,2)" json:"unit_cost_cny"`         // 采购成本 (CNY)
	FreightCNY     float64 `gorm:"column:freight_cny;type:decimal(12,2)" json:"freight_cny"`             // 运费 (CNY)
	InsuranceCNY   float64 `gorm:"column:insurance_cny;type:decimal(12,2)" json:"insurance_cny"`         // 保险费
	DutyRate       float64 `gorm:"column:duty_rate;type:decimal(5,2)" json:"duty_rate"`                  // 关税率 %
	DutyCNY        float64 `gorm:"column:duty_cny;type:decimal(12,2)" json:"duty_cny"`                   // 关税金额
	VatRate        float64 `gorm:"column:vat_rate;type:decimal(5,2)" json:"vat_rate"`                    // VAT率 %
	VatCNY         float64 `gorm:"column:vat_cny;type:decimal(12,2)" json:"vat_cny"`                     // VAT金额
	PlatformFeePct float64 `gorm:"column:platform_fee_pct;type:decimal(5,2)" json:"platform_fee_pct"`    // 平台佣金率 %
	PlatformFeeCNY float64 `gorm:"column:platform_fee_cny;type:decimal(12,2)" json:"platform_fee_cny"`   // 平台佣金
	ClearingFeeCNY float64 `gorm:"column:clearing_fee_cny;type:decimal(12,2)" json:"clearing_fee_cny"`   // 清关费

	// Results
	TotalCostCNY   float64   `gorm:"column:total_cost_cny;type:decimal(12,2)" json:"total_cost_cny"`       // 总成本 (CNY)
	ExchangeRate   float64   `gorm:"column:exchange_rate;type:decimal(10,4)" json:"exchange_rate"`          // 汇率
	TotalCostLocal float64   `gorm:"column:total_cost_local;type:decimal(12,2)" json:"total_cost_local"`   // 当地货币总成本
	TargetPrice    float64   `gorm:"column:target_price;type:decimal(12,2)" json:"target_price"`           // 建议售价

	CalculatedAt time.Time `gorm:"column:calculated_at" json:"calculated_at"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (LandedCost) TableName() string { return "landed_cost" }

// ---------- Input / Output DTOs ----------

// CalculateRequest is the payload for POST /landed-cost/calculate.
type CalculateRequest struct {
	ProductID   int64   `json:"product_id" binding:"required"`
	PlatformID  int64   `json:"platform_id" binding:"required"`
	UnitCostCNY float64 `json:"unit_cost_cny"`                       // 0 → look up from sourcing
	FreightCNY  float64 `json:"freight_cny"`                         // 0 → estimate if available
	InsuranceCNY float64 `json:"insurance_cny"`                      // 0 → default 0
	DutyRate    *float64 `json:"duty_rate,omitempty"`                // nil → look up
	VatRate     *float64 `json:"vat_rate,omitempty"`                 // nil → look up default
	PlatformFeePct *float64 `json:"platform_fee_pct,omitempty"`      // nil → look up from platformfee
	ClearingFeeCNY *float64 `json:"clearing_fee_cny,omitempty"`      // nil → default 0
	TargetMarginPct float64 `json:"target_margin_pct"`               // desired margin %, e.g. 15 for 15%
	CountryCode string `json:"country_code,omitempty"`               // for duty/VAT lookup
	CategoryID  *int64 `json:"category_id,omitempty"`                // for platform fee matching
}

// CalculateResult is the response for POST /landed-cost/calculate.
type CalculateResult struct {
	LandedCost

	ProfitMarginPct  float64 `json:"profit_margin_pct"`   // actual margin achieved
	RecommendedPrice float64 `json:"recommended_price"`    // same as TargetPrice
}

// LandedCostVO is a view object for API responses (omit sensitive fields).
type LandedCostVO struct {
	LandedCost
	ProfitMarginPct float64 `json:"profit_margin_pct"`
}

// CompareItem is one platform entry in the compare response.
type CompareItem struct {
	PlatformID      int64   `json:"platform_id"`
	TotalCostLocal  float64 `json:"total_cost_local"`
	TargetPrice     float64 `json:"target_price"`
	ProfitMarginPct float64 `json:"profit_margin_pct"`
}
