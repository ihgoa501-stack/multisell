package report

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides report business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new report service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Daily returns the daily report for the given date (defaults to today).
func (s *Service) Daily(dateStr string) (*DailyReport, error) {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	nextDay := date.AddDate(0, 0, 1)
	r := &DailyReport{Date: dateStr}

	// Sales, Orders, Profit from sales_order
	type salesRow struct {
		Sales  float64
		Orders int64
		Profit float64
	}
	var sr salesRow
	if err := s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", date, nextDay).
		Select("COALESCE(SUM(pay_amount),0) AS sales, COUNT(*) AS orders, COALESCE(SUM(profit_amount),0) AS profit").
		Scan(&sr).Error; err != nil {
		return nil, err
	}
	r.Sales = sr.Sales
	r.Orders = sr.Orders
	r.Profit = sr.Profit

	// NewListings from listing_task
	type cnt struct {
		C int64
	}
	var lc cnt
	if err := s.db.Table("listing_task").
		Where("created_at >= ? AND created_at < ?", date, nextDay).
		Select("COUNT(*) AS c").Scan(&lc).Error; err != nil {
		return nil, err
	}
	r.NewListings = lc.C

	// Anomalies from exception_item
	var ac cnt
	if err := s.db.Table("exception_item").
		Where("created_at >= ? AND created_at < ?", date, nextDay).
		Select("COUNT(*) AS c").Scan(&ac).Error; err != nil {
		return nil, err
	}
	r.Anomalies = ac.C

	// Approvals from approval_request
	var apc cnt
	if err := s.db.Table("approval_request").
		Where("created_at >= ? AND created_at < ?", date, nextDay).
		Select("COUNT(*) AS c").Scan(&apc).Error; err != nil {
		return nil, err
	}
	r.Approvals = apc.C

	// AgentProposals from unified_action
	var uc cnt
	if err := s.db.Table("unified_action").
		Where("created_at >= ? AND created_at < ?", date, nextDay).
		Select("COUNT(*) AS c").Scan(&uc).Error; err != nil {
		return nil, err
	}
	r.AgentProposals = uc.C

	// LLMCost from llm_cost_logs
	type costRow struct {
		Total float64
	}
	var lcst costRow
	if err := s.db.Table("llm_cost_logs").
		Where("window_date = ?", dateStr).
		Select("COALESCE(SUM(cost_usd),0) AS total").
		Scan(&lcst).Error; err != nil {
		return nil, err
	}
	r.LLMCost = lcst.Total

	return r, nil
}

// Weekly returns the weekly report starting from the given Monday.
func (s *Service) Weekly(weekStartStr string) (*WeeklyReport, error) {
	if weekStartStr == "" {
		now := time.Now()
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		weekStart := now.AddDate(0, 0, -int(weekday-time.Monday))
		weekStartStr = weekStart.Format("2006-01-02")
	}
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		return nil, err
	}
	weekEnd := weekStart.AddDate(0, 0, 6)

	r := &WeeklyReport{
		WeekStart: weekStartStr,
		WeekEnd:   weekEnd.Format("2006-01-02"),
	}

	for i := 0; i < 7; i++ {
		d := weekStart.AddDate(0, 0, i)
		dr, err := s.Daily(d.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		r.DailyReports = append(r.DailyReports, *dr)
		r.SalesTotal += dr.Sales
		r.ProfitTotal += dr.Profit
		r.OrdersTotal += dr.Orders
		r.AnomaliesTotal += dr.Anomalies
	}

	return r, nil
}

// parseRange parses from/to query strings. Missing values default to wide bounds.
func parseRange(fromStr, toStr string) (time.Time, time.Time) {
	var zero time.Time
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		from = zero
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		to = time.Now()
	} else {
		// include the whole `to` day
		to = to.AddDate(0, 0, 1)
	}
	return from, to
}

// applyPlatform applies an optional platform_id filter on the given column.
func applyPlatform(q *gorm.DB, col string, platformID *int64) *gorm.DB {
	if platformID == nil {
		return q
	}
	return q.Where(col+" = ?", *platformID)
}

// Sales returns the sales report for a date range + optional platform_id.
func (s *Service) Sales(from, to time.Time, platformID *int64) (*SalesReport, error) {
	r := &SalesReport{ByStatus: map[string]int64{}}

	base := s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to)
	if err := applyPlatform(base, "platform_id", platformID).
		Count(&r.TotalOrders).Error; err != nil {
		return nil, err
	}

	var tot struct {
		Revenue float64
		Profit  float64
	}
	if err := applyPlatform(s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("COALESCE(SUM(pay_amount),0) AS revenue, COALESCE(SUM(profit_amount),0) AS profit").
		Scan(&tot).Error; err != nil {
		return nil, err
	}
	r.TotalRevenue = tot.Revenue
	r.TotalProfit = tot.Profit

	// by platform
	type pRow struct {
		PlatformID *int64
		Cnt        int64
		Revenue    float64
		Profit     float64
	}
	var prs []pRow
	if err := applyPlatform(s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("platform_id, COUNT(*) AS cnt, COALESCE(SUM(pay_amount),0) AS revenue, COALESCE(SUM(profit_amount),0) AS profit").
		Group("platform_id").
		Scan(&prs).Error; err != nil {
		return nil, err
	}
	for _, p := range prs {
		r.ByPlatform = append(r.ByPlatform, PlatformBreakdown{
			PlatformID: p.PlatformID,
			Count:      p.Cnt,
			Revenue:    p.Revenue,
			Profit:     p.Profit,
		})
	}

	// by status
	type sRow struct {
		Status string
		Cnt    int64
	}
	var srs []sRow
	if err := applyPlatform(s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("status, COUNT(*) AS cnt").
		Group("status").
		Scan(&srs).Error; err != nil {
		return nil, err
	}
	for _, sr := range srs {
		r.ByStatus[sr.Status] = sr.Cnt
	}
	return r, nil
}

// Profit returns the profit report.
func (s *Service) Profit(from, to time.Time, platformID *int64) (*ProfitReport, error) {
	r := &ProfitReport{}
	var tot struct {
		Revenue float64
		Profit  float64
	}
	if err := applyPlatform(s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("COALESCE(SUM(pay_amount),0) AS revenue, COALESCE(SUM(profit_amount),0) AS profit").
		Scan(&tot).Error; err != nil {
		return nil, err
	}
	r.TotalProfit = tot.Profit
	if tot.Revenue > 0 {
		r.ProfitMargin = tot.Profit / tot.Revenue
	}

	// by platform
	type pRow struct {
		PlatformID *int64
		Profit     float64
		Revenue    float64
	}
	var prs []pRow
	if err := applyPlatform(s.db.Table("sales_order").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("platform_id, COALESCE(SUM(profit_amount),0) AS profit, COALESCE(SUM(pay_amount),0) AS revenue").
		Group("platform_id").
		Scan(&prs).Error; err != nil {
		return nil, err
	}
	for _, p := range prs {
		margin := 0.0
		if p.Revenue > 0 {
			margin = p.Profit / p.Revenue
		}
		r.ByPlatform = append(r.ByPlatform, PlatformProfit{
			PlatformID:   p.PlatformID,
			Profit:       p.Profit,
			Revenue:      p.Revenue,
			ProfitMargin: margin,
		})
	}

	// by category: join sales_order_item -> sku -> product
	type cRow struct {
		CategoryID int64
		Profit     float64
		Revenue    float64
	}
	var crs []cRow
	if err := applyPlatform(s.db.Table("sales_order AS o").
		Joins("JOIN sales_order_item AS oi ON oi.order_id = o.id").
		Joins("JOIN sku AS s ON s.id = oi.sku_id").
		Joins("JOIN product AS p ON p.id = s.product_id").
		Where("o.created_at >= ? AND o.created_at < ?", from, to), "o.platform_id", platformID).
		Select("p.category_id AS category_id, COALESCE(SUM(oi.subtotal),0) AS revenue, COALESCE(SUM(oi.subtotal - s.cost_price * oi.quantity),0) AS profit").
		Group("p.category_id").
		Scan(&crs).Error; err != nil {
		return nil, err
	}
	for _, c := range crs {
		margin := 0.0
		if c.Revenue > 0 {
			margin = c.Profit / c.Revenue
		}
		r.ByCategory = append(r.ByCategory, CategoryProfit{
			CategoryID:   c.CategoryID,
			Profit:       c.Profit,
			Revenue:      c.Revenue,
			ProfitMargin: margin,
		})
	}
	return r, nil
}

// Inventory returns the inventory report.
func (s *Service) Inventory() (*InventoryReport, error) {
	r := &InventoryReport{}
	if err := s.db.Table("sku").Count(&r.SkuTotal).Error; err != nil {
		return nil, err
	}
	// total stock value = SUM(stock * cost_price)
	var sv struct {
		Total float64
	}
	if err := s.db.Table("sku").
		Select("COALESCE(SUM(stock * cost_price),0) AS total").
		Scan(&sv).Error; err != nil {
		return nil, err
	}
	r.TotalStockValue = sv.Total

	// by warehouse (via inventory table)
	type wRow struct {
		Warehouse  string
		SkuCount   int64
		StockValue float64
	}
	var wrs []wRow
	if err := s.db.Table("inventory AS i").
		Joins("JOIN sku AS s ON s.id = i.sku_id").
		Select("i.warehouse AS warehouse, COUNT(DISTINCT i.sku_id) AS sku_count, COALESCE(SUM(i.quantity * s.cost_price),0) AS stock_value").
		Group("i.warehouse").
		Scan(&wrs).Error; err != nil {
		return nil, err
	}
	for _, w := range wrs {
		r.ByWarehouse = append(r.ByWarehouse, WarehouseStock{
			Warehouse:  w.Warehouse,
			SkuCount:   w.SkuCount,
			StockValue: w.StockValue,
		})
	}

	// low stock top 20
	type lRow struct {
		SkuID        int64
		ProductID    int64
		Code         string
		SpecDesc     string
		Stock        int
		WarningStock int
		CostPrice    float64
	}
	var lrs []lRow
	if err := s.db.Table("sku").
		Select("id AS sku_id, product_id, code, spec_desc, stock, warning_stock, cost_price").
		Where("warning_stock > 0 AND stock <= warning_stock").
		Order("stock ASC, warning_stock DESC").
		Limit(20).
		Scan(&lrs).Error; err != nil {
		return nil, err
	}
	for _, l := range lrs {
		r.LowStockTop20 = append(r.LowStockTop20, LowStockRow{
			SkuID:        l.SkuID,
			ProductID:    l.ProductID,
			Code:         l.Code,
			SpecDesc:     l.SpecDesc,
			Stock:        l.Stock,
			WarningStock: l.WarningStock,
			CostPrice:    l.CostPrice,
		})
	}
	return r, nil
}

// Settlement returns the settlement report.
func (s *Service) Settlement(from, to time.Time, platformID *int64) (*SettlementReport, error) {
	r := &SettlementReport{ReconciliationDist: map[string]int64{}}
	if err := applyPlatform(s.db.Table("settlement").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Count(&r.TotalSettlements).Error; err != nil {
		return nil, err
	}
	var tot struct {
		Net float64
	}
	if err := applyPlatform(s.db.Table("settlement").
		Where("created_at >= ? AND created_at < ?", from, to), "platform_id", platformID).
		Select("COALESCE(SUM(total_net),0) AS net").
		Scan(&tot).Error; err != nil {
		return nil, err
	}
	r.TotalNet = tot.Net

	// reconciliation status distribution from settlement_item
	type rRow struct {
		Status string
		Cnt    int64
	}
	var rrs []rRow
	if err := applyPlatform(s.db.Table("settlement_item AS si").
		Joins("JOIN settlement AS s ON s.id = si.settlement_id").
		Where("s.created_at >= ? AND s.created_at < ?", from, to), "s.platform_id", platformID).
		Select("si.reconciliation_status AS status, COUNT(*) AS cnt").
		Group("si.reconciliation_status").
		Scan(&rrs).Error; err != nil {
		return nil, err
	}
	for _, rr := range rrs {
		r.ReconciliationDist[rr.Status] = rr.Cnt
	}
	return r, nil
}

// PlatformFee returns the platform fee report.
// Aggregates actual fee amounts from finance_ledger_entry (entry_type in fee types)
// joined with platform_id, plus platform_fee_rule counts by fee_type.
func (s *Service) PlatformFee(from, to time.Time, platformID *int64) (*PlatformFeeReport, error) {
	r := &PlatformFeeReport{}

	// by platform: sum amounts from finance_ledger_entry for fee-like entry types
	type pRow struct {
		PlatformID *int64
		TotalFee   float64
		Cnt        int64
	}
	var prs []pRow
	if err := applyPlatform(s.db.Table("finance_ledger_entry").
		Where("created_at >= ? AND created_at < ?", from, to).
		Where("entry_type IN ?", []string{"platform_fee", "payment_fee"}), "platform_id", platformID).
		Select("platform_id, COALESCE(SUM(amount),0) AS total_fee, COUNT(*) AS cnt").
		Group("platform_id").
		Scan(&prs).Error; err != nil {
		return nil, err
	}
	for _, p := range prs {
		r.ByPlatform = append(r.ByPlatform, PlatformFeeRow{
			PlatformID: p.PlatformID,
			TotalFee:   p.TotalFee,
			Count:      p.Cnt,
		})
	}

	// by fee_type: count rules from platform_fee_rule
	type fRow struct {
		FeeType  string
		TotalFee float64
		Cnt      int64
	}
	var frs []fRow
	if err := s.db.Table("platform_fee_rule").
		Select("fee_type, COALESCE(SUM(fee_rate_pct),0) AS total_fee, COUNT(*) AS cnt").
		Group("fee_type").
		Scan(&frs).Error; err != nil {
		return nil, err
	}
	for _, f := range frs {
		r.ByFeeType = append(r.ByFeeType, FeeTypeRow{
			FeeType:  f.FeeType,
			TotalFee: f.TotalFee,
			Count:    f.Cnt,
		})
	}
	return r, nil
}
