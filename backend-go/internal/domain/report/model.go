package report

// SalesReport aggregates sales figures for a date range + optional platform filter.
type SalesReport struct {
	TotalOrders  int64              `json:"total_orders"`
	TotalRevenue float64            `json:"total_revenue"`
	TotalProfit  float64            `json:"total_profit"`
	ByPlatform   []PlatformBreakdown `json:"by_platform"`
	ByStatus     map[string]int64   `json:"by_status"`
}

// PlatformBreakdown is one row in ByPlatform aggregations.
type PlatformBreakdown struct {
	PlatformID *int64  `json:"platform_id,omitempty"`
	Count      int64   `json:"count"`
	Revenue    float64 `json:"revenue"`
	Profit     float64 `json:"profit"`
}

// ProfitReport aggregates profit figures.
type ProfitReport struct {
	TotalProfit  float64            `json:"total_profit"`
	ProfitMargin float64            `json:"profit_margin"`
	ByPlatform   []PlatformProfit   `json:"by_platform"`
	ByCategory   []CategoryProfit   `json:"by_category"`
}

// PlatformProfit is one row in ProfitReport.ByPlatform.
type PlatformProfit struct {
	PlatformID   *int64  `json:"platform_id,omitempty"`
	Profit       float64 `json:"profit"`
	Revenue      float64 `json:"revenue"`
	ProfitMargin float64 `json:"profit_margin"`
}

// CategoryProfit is one row in ProfitReport.ByCategory.
type CategoryProfit struct {
	CategoryID   int64   `json:"category_id"`
	Profit       float64 `json:"profit"`
	Revenue      float64 `json:"revenue"`
	ProfitMargin float64 `json:"profit_margin"`
}

// InventoryReport aggregates inventory health.
type InventoryReport struct {
	SkuTotal         int64             `json:"sku_total"`
	TotalStockValue  float64           `json:"total_stock_value"`
	ByWarehouse      []WarehouseStock  `json:"by_warehouse"`
	LowStockTop20    []LowStockRow     `json:"low_stock_top_20"`
}

// WarehouseStock is one row in InventoryReport.ByWarehouse.
type WarehouseStock struct {
	Warehouse   string  `json:"warehouse"`
	SkuCount    int64   `json:"sku_count"`
	StockValue  float64 `json:"stock_value"`
}

// LowStockRow is one row in InventoryReport.LowStockTop20.
type LowStockRow struct {
	SkuID        int64   `json:"sku_id"`
	ProductID    int64   `json:"product_id"`
	Code         string  `json:"code"`
	SpecDesc     string  `json:"spec_desc"`
	Stock        int     `json:"stock"`
	WarningStock int     `json:"warning_stock"`
	CostPrice    float64 `json:"cost_price"`
}

// SettlementReport aggregates settlement figures.
type SettlementReport struct {
	TotalSettlements   int64              `json:"total_settlements"`
	TotalNet           float64            `json:"total_net"`
	ReconciliationDist map[string]int64   `json:"reconciliation_dist"`
}

// PlatformFeeReport aggregates platform fees.
type PlatformFeeReport struct {
	ByPlatform []PlatformFeeRow `json:"by_platform"`
	ByFeeType  []FeeTypeRow     `json:"by_fee_type"`
}

// PlatformFeeRow is one row in PlatformFeeReport.ByPlatform.
type PlatformFeeRow struct {
	PlatformID *int64  `json:"platform_id,omitempty"`
	TotalFee   float64 `json:"total_fee"`
	Count      int64   `json:"count"`
}

// FeeTypeRow is one row in PlatformFeeReport.ByFeeType.
type FeeTypeRow struct {
	FeeType   string  `json:"fee_type"`
	TotalFee  float64 `json:"total_fee"`
	Count     int64   `json:"count"`
}

// DailyReport is a single-day executive summary.
type DailyReport struct {
	Date           string  `json:"date"`
	Sales          float64 `json:"sales"`
	Orders         int64   `json:"orders"`
	Profit         float64 `json:"profit"`
	NewListings    int64   `json:"new_listings"`
	Anomalies      int64   `json:"anomalies"`
	Approvals      int64   `json:"approvals"`
	AgentProposals int64   `json:"agent_proposals"`
	LLMCost        float64 `json:"llm_cost"`
}

// WeeklyReport is a 7-day summary with per-day breakdown.
type WeeklyReport struct {
	WeekStart      string        `json:"week_start"`
	WeekEnd        string        `json:"week_end"`
	DailyReports   []DailyReport `json:"daily_reports"`
	SalesTotal     float64       `json:"sales_total"`
	ProfitTotal    float64       `json:"profit_total"`
	OrdersTotal    int64         `json:"orders_total"`
	AnomaliesTotal int64         `json:"anomalies_total"`
}
