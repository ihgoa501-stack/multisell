package finance

import "time"

// ---------- Model ----------

// ProfitCalculation maps to "profit_calculation".
// Stores one row per SKU per order, capturing all cost components.
type ProfitCalculation struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID         int64     `gorm:"column:order_id;not null;index:idx_profit_order_sku,unique" json:"order_id"`
	SkuID           int64     `gorm:"column:sku_id;not null;index:idx_profit_order_sku,unique" json:"sku_id"`
	Revenue         float64   `gorm:"column:revenue;default:0" json:"revenue"`
	PlatformFee     float64   `gorm:"column:platform_fee;default:0" json:"platform_fee"`
	LogisticsFee    float64   `gorm:"column:logistics_fee;default:0" json:"logistics_fee"`
	AdvertisingCost float64   `gorm:"column:advertising_cost;default:0" json:"advertising_cost"`
	PurchaseCost    float64   `gorm:"column:purchase_cost;default:0" json:"purchase_cost"`
	OtherCost       float64   `gorm:"column:other_cost;default:0" json:"other_cost"`
	NetProfit       float64   `gorm:"column:net_profit;default:0" json:"net_profit"`
	ProfitMargin    float64   `gorm:"column:profit_margin;default:0" json:"profit_margin"`
	CalculatedAt    time.Time `gorm:"column:calculated_at;autoCreateTime" json:"calculated_at"`
	PeriodStart     time.Time `gorm:"column:period_start;type:date" json:"period_start"`
	PeriodEnd       time.Time `gorm:"column:period_end;type:date" json:"period_end"`
}

func (ProfitCalculation) TableName() string { return "profit_calculation" }

// ---------- Input / DTO structs ----------

// CalculateProfitInput is the payload for POST /finance/profit/calculate.
type CalculateProfitInput struct {
	OrderID int64 `json:"order_id" binding:"required"`
}

// BatchCalculateInput is the payload for POST /finance/profit/batch-calculate.
type BatchCalculateInput struct {
	Since string `json:"since" binding:"required"`
	Until string `json:"until" binding:"required"`
}

// ProfitSummaryResult is the response for GET /finance/profit/summary.
type ProfitSummaryResult struct {
	TotalProfit  float64 `json:"total_profit"`
	AvgMargin    float64 `json:"avg_margin"`
	LossSKUCount int64   `json:"loss_sku_count"`
	PeriodCount  int64   `json:"period_count"`
	TotalRevenue float64 `json:"total_revenue"`
	TotalCost    float64 `json:"total_cost"`
}
