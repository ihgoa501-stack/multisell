package dashboard

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides dashboard business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new dashboard service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Overview returns the aggregated dashboard overview.
func (s *Service) Overview() (*DashboardOverview, error) {
	o := &DashboardOverview{OrderByStatus: map[string]int64{}}

	// Order stats from sales_order
	if err := s.db.Table("sales_order").Count(&o.OrderTotal).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Table("sales_order").
		Select("status, COUNT(*) AS cnt").
		Group("status").
		Scan(&scs).Error; err != nil {
		return nil, err
	}
	for _, sc := range scs {
		o.OrderByStatus[sc.Status] = sc.Cnt
	}
	var rev struct {
		Revenue float64
		Profit  float64
	}
	if err := s.db.Table("sales_order").
		Select("COALESCE(SUM(pay_amount),0) AS revenue, COALESCE(SUM(profit_amount),0) AS profit").
		Scan(&rev).Error; err != nil {
		return nil, err
	}
	o.OrderRevenue = rev.Revenue
	o.OrderProfit = rev.Profit

	// SKU stats from sku + inventory
	if err := s.db.Table("sku").Count(&o.SkuTotal).Error; err != nil {
		return nil, err
	}
	// Low stock: stock <= warning_stock AND warning_stock > 0
	if err := s.db.Table("sku").
		Where("warning_stock > 0 AND stock <= warning_stock").
		Count(&o.LowStockCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("sku").
		Where("stock <= 0").
		Count(&o.OutOfStockCount).Error; err != nil {
		return nil, err
	}

	// Active listings
	if err := s.db.Table("product_listing").
		Where("status = ?", "online").
		Count(&o.ListingActiveCount).Error; err != nil {
		return nil, err
	}

	// Aftersales pending
	if err := s.db.Table("after_sales_order").
		Where("status IN ?", []string{"pending", "open"}).
		Count(&o.AftersalesPendingCount).Error; err != nil {
		return nil, err
	}

	// Open exceptions
	if err := s.db.Table("exception_item").
		Where("status = ?", "open").
		Count(&o.ExceptionOpenCount).Error; err != nil {
		return nil, err
	}

	// Finance ledger for current month
	now := time.Now()
	start, end := monthRange(now)
	var fin struct {
		Revenue float64
		Cost    float64
	}
	if err := s.db.Table("finance_ledger_entry").
		Select("COALESCE(SUM(CASE WHEN entry_type = 'revenue' THEN amount ELSE 0 END),0) AS revenue, "+
			"COALESCE(SUM(CASE WHEN entry_type IN ('product_cost','shipping_cost','platform_fee','payment_fee','other_fee') THEN amount ELSE 0 END),0) AS cost").
		Where("created_at >= ? AND created_at < ?", start, end).
		Scan(&fin).Error; err != nil {
		return nil, err
	}
	o.MonthRevenue = fin.Revenue
	o.MonthCost = fin.Cost

	// Platform connections
	type pRow struct {
		PlatformID   int64
		PlatformCode string
		PlatformName string
		StoreName    string
		Status       string
		SyncStatus   string
		LastSyncAt   *time.Time
		LastError    string
	}
	var prs []pRow
	s.db.Table("platform_integration_account AS a").
		Joins("JOIN platform AS p ON p.id = a.platform_id").
		Select("a.platform_id, p.code AS platform_code, p.name AS platform_name, a.store_name, a.status, a.sync_status, a.last_sync_at, a.last_error").
		Scan(&prs)
	for _, pr := range prs {
		var lastSync *string
		if pr.LastSyncAt != nil {
			s := pr.LastSyncAt.Format("2006-01-02T15:04:05Z")
			lastSync = &s
		}
		o.PlatformConnections = append(o.PlatformConnections, PlatformConnectionStatus{
			PlatformID:   pr.PlatformID,
			PlatformCode: pr.PlatformCode,
			PlatformName: pr.PlatformName,
			StoreName:    pr.StoreName,
			Status:       pr.Status,
			SyncStatus:   pr.SyncStatus,
			LastSyncAt:   lastSync,
			LastError:    pr.LastError,
		})
	}
	if o.PlatformConnections == nil {
		o.PlatformConnections = []PlatformConnectionStatus{}
	}

	// Agent statuses (read from unified action / decision tables)
	type aRow struct {
		AgentID string
		Cnt     int64
		LastAct *time.Time
	}
	var ars []aRow
	s.db.Table("unified_action").
		Select("agent_id, COUNT(*) AS cnt, MAX(created_at) AS last_act").
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Group("agent_id").
		Scan(&ars)
	agentNames := map[string]string{
		"A4": "客服助手", "A5": "库存预警", "A6": "利润监控",
		"A8": "结算对账", "A10": "物流运营", "A9": "批量运营",
		"G0": "协调仲裁", "G1": "驾驶舱", "G3": "折扣风控",
	}
	for _, ar := range ars {
		var lastAct *string
		if ar.LastAct != nil {
			s := ar.LastAct.Format("2006-01-02T15:04:05Z")
			lastAct = &s
		}
		name := agentNames[ar.AgentID]
		if name == "" {
			name = ar.AgentID
		}
		status := "active"
		if ar.Cnt == 0 {
			status = "idle"
		}
		o.AgentStatuses = append(o.AgentStatuses, AgentStatusEntry{
			AgentID:      ar.AgentID,
			Name:         name,
			Status:       status,
			LastActivity: lastAct,
		})
	}
	if o.AgentStatuses == nil {
		o.AgentStatuses = []AgentStatusEntry{}
	}

	return o, nil
}

// OrdersTrend returns the per-day order count + revenue for the last `days` days.
func (s *Service) OrdersTrend(days int) ([]OrderTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	end := time.Now()
	start := end.AddDate(0, 0, -days)
	type row struct {
		Day      time.Time
		OrderCnt int64
		Revenue  float64
	}
	var rows []row
	if err := s.db.Table("sales_order").
		Select("DATE(created_at) AS day, COUNT(*) AS order_cnt, COALESCE(SUM(pay_amount),0) AS revenue").
		Where("created_at >= ? AND created_at < ?", start, end).
		Group("day").
		Order("day ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]OrderTrendPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, OrderTrendPoint{
			Date:     r.Day.Format("2006-01-02"),
			OrderCnt: r.OrderCnt,
			Revenue:  r.Revenue,
		})
	}
	return out, nil
}

// InventoryHealth returns top 20 low-stock SKUs.
func (s *Service) InventoryHealth(limit int) ([]LowStockSku, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []LowStockSku
	if err := s.db.Table("sku AS s").
		Select("s.id AS sku_id, s.product_id, s.code, s.spec_desc, s.stock, s.warning_stock, COALESCE(i.warehouse,'') AS warehouse").
		Joins("LEFT JOIN inventory AS i ON i.sku_id = s.id").
		Where("s.warning_stock > 0 AND s.stock <= s.warning_stock").
		Order("s.stock ASC, s.warning_stock DESC").
		Limit(limit).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ExceptionDistribution returns exception counts grouped by severity + source_module.
func (s *Service) ExceptionDistribution() ([]ExceptionDistribution, error) {
	var items []ExceptionDistribution
	if err := s.db.Table("exception_item").
		Select("severity, source_module, COUNT(*) AS cnt").
		Group("severity, source_module").
		Order("cnt DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
