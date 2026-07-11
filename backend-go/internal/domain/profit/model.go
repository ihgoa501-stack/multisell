package profit

import (
	"time"
)

// ProfitSummary maps to the "profit_summary" table.
type ProfitSummary struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID       int64     `gorm:"column:product_id;not null;index" json:"product_id"`
	PurchaseCost    float64   `gorm:"column:purchase_cost;default:0" json:"purchase_cost"`
	ShippingCost    float64   `gorm:"column:shipping_cost;default:0" json:"shipping_cost"`
	PlatformFee     float64   `gorm:"column:platform_fee;default:0" json:"platform_fee"`
	TariffCost      float64   `gorm:"column:tariff_cost;default:0" json:"tariff_cost"`
	OtherCost       float64   `gorm:"column:other_cost;default:0" json:"other_cost"`
	TotalCost       float64   `gorm:"column:total_cost;default:0" json:"total_cost"`
	TargetRevenue   float64   `gorm:"column:target_revenue;default:0" json:"target_revenue"`
	EstimatedProfit float64   `gorm:"column:estimated_profit;default:0" json:"estimated_profit"`
	ProfitMargin    float64   `gorm:"column:profit_margin;default:0" json:"profit_margin"`
	Status          string    `gorm:"column:status;default:unknown" json:"status"` // profitable, marginal, unprofitable, unknown
	Currency        string    `gorm:"column:currency;default:USD" json:"currency"`
	CalculatedBy    string    `gorm:"column:calculated_by" json:"calculated_by"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProfitSummary) TableName() string { return "profit_summary" }

// ProfitResult is the API response for a profit calculation.
type ProfitResult struct {
	ProductID       int64   `json:"product_id"`
	Title           string  `json:"title"`
	PurchaseCost    float64 `json:"purchase_cost"`
	ShippingCost    float64 `json:"shipping_cost"`
	PlatformFee     float64 `json:"platform_fee"`
	TariffCost      float64 `json:"tariff_cost"`
	OtherCost       float64 `json:"other_cost"`
	TotalCost       float64 `json:"total_cost"`
	TargetRevenue   float64 `json:"target_revenue"`
	EstimatedProfit float64 `json:"estimated_profit"`
	ProfitMargin    float64 `json:"profit_margin"`
	Status          string  `json:"status"` // profitable, marginal, unprofitable
	Currency        string  `json:"currency"`
}

// SummaryInput is optional payload for triggering profit calculation.
type SummaryInput struct {
	CalculatedBy string `json:"calculated_by"`
}

// OrderProfitRecord maps to "order_profit_record".
// Stores one row per order capturing all cost components and computed profit.
type OrderProfitRecord struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID      int64     `gorm:"column:order_id;not null;uniqueIndex" json:"order_id"`
	Revenue      float64   `gorm:"column:revenue;default:0" json:"revenue"`
	Cost         float64   `gorm:"column:cost;default:0" json:"cost"`
	ShippingCost float64   `gorm:"column:shipping_cost;default:0" json:"shipping_cost"`
	PlatformFee  float64   `gorm:"column:platform_fee;default:0" json:"platform_fee"`
	PaymentFee   float64   `gorm:"column:payment_fee;default:0" json:"payment_fee"`
	TariffCost   float64   `gorm:"column:tariff_cost;default:0" json:"tariff_cost"`
	TotalCost    float64   `gorm:"column:total_cost;default:0" json:"total_cost"`
	Profit       float64   `gorm:"column:profit;default:0" json:"profit"`
	Margin       float64   `gorm:"column:margin;default:0" json:"margin"`
	ProfitStatus string    `gorm:"column:profit_status;type:varchar(20);not null;default:provisional" json:"profit_status"`
	MissingCosts string    `gorm:"column:missing_costs;type:text;not null;default:''" json:"missing_costs"`
	CalculatedAt time.Time `gorm:"column:calculated_at;autoCreateTime" json:"calculated_at"`
}

func (OrderProfitRecord) TableName() string { return "order_profit_record" }

// OrderProfit is the API response for order-level profit calculation.
type OrderProfit struct {
	OrderID      uint    `json:"order_id"`
	Revenue      float64 `json:"revenue"`
	Cost         float64 `json:"cost"`
	ShippingCost float64 `json:"shipping_cost"`
	PlatformFee  float64 `json:"platform_fee"`
	PaymentFee   float64 `json:"payment_fee"`
	TariffCost   float64 `json:"tariff_cost"`
	TotalCost    float64 `json:"total_cost"`
	Profit       float64 `json:"profit"`
	Margin       float64 `json:"margin"`
}
