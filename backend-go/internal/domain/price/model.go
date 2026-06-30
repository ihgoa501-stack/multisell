package price

import (
	"time"

	"github.com/shopspring/decimal"
)

// Price maps to the PostgreSQL "price" table.
type Price struct {
	ID        int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID     int64           `gorm:"column:sku_id;not null" json:"sku_id"`
	PriceType string          `gorm:"column:price_type;not null" json:"price_type"`
	Price     decimal.Decimal `gorm:"column:price;type:numeric(10,2);not null" json:"price"`
	StartTime *time.Time      `gorm:"column:start_time" json:"start_time,omitempty"`
	EndTime   *time.Time      `gorm:"column:end_time" json:"end_time,omitempty"`
	Status    int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Price) TableName() string { return "price" }

// PriceChangeLog maps to the PostgreSQL "price_change_log" table.
type PriceChangeLog struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID      int64           `gorm:"column:sku_id;not null" json:"sku_id"`
	OldPrice   *decimal.Decimal `gorm:"column:old_price;type:numeric(10,2)" json:"old_price,omitempty"`
	NewPrice   *decimal.Decimal `gorm:"column:new_price;type:numeric(10,2)" json:"new_price,omitempty"`
	PriceType  string          `gorm:"column:price_type" json:"price_type"`
	ChangeType string          `gorm:"column:change_type" json:"change_type"`
	Operator   string          `gorm:"column:operator" json:"operator"`
	Remark     string          `gorm:"column:remark" json:"remark"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (PriceChangeLog) TableName() string { return "price_change_log" }

// ---------------------------------------------------------------------------
// Competitor Price Monitoring
// ---------------------------------------------------------------------------

// CompetitorPrice maps to the PostgreSQL "competitor_prices" table.
type CompetitorPrice struct {
	ID             int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID          int64           `gorm:"column:sku_id;not null;index" json:"sku_id"`
	Platform       string          `gorm:"column:platform;size:50" json:"platform"`
	CompetitorName string          `gorm:"column:competitor_name;size:200;not null" json:"competitor_name"`
	Price          decimal.Decimal `gorm:"column:price;type:numeric(12,2);not null" json:"price"`
	Currency       string          `gorm:"column:currency;size:3;default:'USD'" json:"currency"`
	CapturedAt     time.Time       `gorm:"column:captured_at;not null" json:"captured_at"`
	SourceURL      string          `gorm:"column:source_url;size:500" json:"source_url,omitempty"`
	CreatedAt      time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CompetitorPrice) TableName() string { return "competitor_prices" }

// ---------------------------------------------------------------------------
// Pricing Strategy
// ---------------------------------------------------------------------------

// StrategyParameters holds configurable parameters for a pricing strategy.
type StrategyParameters struct {
	BuyBoxDiscount  float64 `json:"buy_box_discount,omitempty"`
	MinProfitMargin float64 `json:"min_profit_margin,omitempty"`
	MinPrice        float64 `json:"min_price,omitempty"`
	MaxPrice        float64 `json:"max_price,omitempty"`
}

// PricingStrategy maps to the PostgreSQL "pricing_strategies" table.
type PricingStrategy struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID        *int64    `gorm:"column:sku_id" json:"sku_id,omitempty"`
	StrategyType string    `gorm:"column:strategy_type;size:30;not null" json:"strategy_type"`
	Parameters   string    `gorm:"column:parameters;type:text;default:'{}'" json:"parameters"`
	Active       bool      `gorm:"column:active;default:true" json:"active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PricingStrategy) TableName() string { return "pricing_strategies" }

// ---------------------------------------------------------------------------
// Pricing Recommendation
// ---------------------------------------------------------------------------

// PricingRecommendation maps to the PostgreSQL "pricing_recommendations" table.
type PricingRecommendation struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID            int64           `gorm:"column:sku_id;not null;index" json:"sku_id"`
	CurrentPrice     decimal.Decimal `gorm:"column:current_price;type:numeric(10,2);not null" json:"current_price"`
	RecommendedPrice decimal.Decimal `gorm:"column:recommended_price;type:numeric(10,2);not null" json:"recommended_price"`
	StrategyUsed     string          `gorm:"column:strategy_used;size:30" json:"strategy_used"`
	Reason           string          `gorm:"column:reason;size:500" json:"reason"`
	RiskLevel        string          `gorm:"column:risk_level;size:10" json:"risk_level"`
	CompetitorCount  int             `gorm:"column:competitor_count;default:0" json:"competitor_count"`
	Applied          bool            `gorm:"column:applied;default:false" json:"applied"`
	AppliedAt        *time.Time      `gorm:"column:applied_at" json:"applied_at,omitempty"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PricingRecommendation) TableName() string { return "pricing_recommendations" }

// ---------------------------------------------------------------------------
// Strategy constants and request types
// ---------------------------------------------------------------------------

const (
	StrategyBuyBoxFirst    = "buy_box_first"
	StrategyProfitPriority = "profit_priority"
	StrategyMatch          = "match"
)

// GenerateRecommendationInput is the API request for generating a pricing recommendation.
type GenerateRecommendationInput struct {
	SkuID           int64           `json:"sku_id" binding:"required"`
	StrategyType    string          `json:"strategy_type"`
	Cost            decimal.Decimal `json:"cost,omitempty"`
	PlatformFeeRate float64         `json:"platform_fee_rate,omitempty"`
}
