package dashboard

import "time"

// DashboardOverview is the aggregated payload for GET /dashboard/overview.
type DashboardOverview struct {
	OrderTotal            int64            `json:"order_total"`
	OrderByStatus         map[string]int64 `json:"order_by_status"`
	OrderRevenue          float64          `json:"order_revenue"`
	OrderProfit           float64          `json:"order_profit"`
	SkuTotal              int64            `json:"sku_total"`
	LowStockCount         int64            `json:"low_stock_count"`
	OutOfStockCount       int64            `json:"out_of_stock_count"`
	ListingActiveCount    int64            `json:"listing_active_count"`
	AftersalesPendingCount int64           `json:"aftersales_pending_count"`
	ExceptionOpenCount    int64            `json:"exception_open_count"`
	MonthRevenue          float64          `json:"month_revenue"`
	MonthCost             float64          `json:"month_cost"`
}

// OrderTrendPoint is one day in the 30-day order trend.
type OrderTrendPoint struct {
	Date     string  `json:"date"`
	OrderCnt int64   `json:"order_cnt"`
	Revenue  float64 `json:"revenue"`
}

// LowStockSku is one row in the low-stock SKU list.
type LowStockSku struct {
	SkuID         int64  `json:"sku_id"`
	ProductID     int64  `json:"product_id"`
	Code          string `json:"code"`
	SpecDesc      string `json:"spec_desc"`
	Stock         int    `json:"stock"`
	WarningStock  int    `json:"warning_stock"`
	Warehouse     string `json:"warehouse"`
}

// ExceptionDistribution is one row in the exception aggregation.
type ExceptionDistribution struct {
	Severity      string `json:"severity"`
	SourceModule  string `json:"source_module"`
	Cnt           int64  `json:"cnt"`
}

// monthRange returns the [start, end) range for the month containing t.
func monthRange(t time.Time) (time.Time, time.Time) {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end := start.AddDate(0, 1, 0)
	return start, end
}
