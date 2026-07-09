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

	// M6.1: Today sales (pay_amount for orders created today)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	if err := s.db.Table("sales_order").
		Select("COALESCE(SUM(pay_amount),0)").
		Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).
		Scan(&o.TodaySales).Error; err != nil {
		s.logger.Error("failed to query today sales", zap.Error(err))
	}

	// M6.1: Pending approval requests
	if err := s.db.Table("approval_request").
		Where("status = ?", "pending").
		Count(&o.PendingApprovals).Error; err != nil {
		s.logger.Error("failed to count pending approvals", zap.Error(err))
	}

	// M6.1: Anomaly count from exception_item (same as ExceptionOpenCount)
	o.AnomalyCount = o.ExceptionOpenCount

	// M6.1: Agent suggestions (unified_action with status 'suggested')
	if err := s.db.Table("unified_action").
		Where("status = ?", "suggested").
		Count(&o.AgentSuggestions).Error; err != nil {
		s.logger.Error("failed to count agent suggestions", zap.Error(err))
	}

	// M6.1: Recent alerts (top 5 open exceptions)
	type alertRow struct {
		ID        int64
		Severity  string
		Title     string
		CreatedAt time.Time
	}
	var alertRows []alertRow
	if err := s.db.Table("exception_item").
		Select("id, severity, COALESCE(title,'') AS title, created_at").
		Where("status = ?", "open").
		Order("created_at DESC").
		Limit(5).
		Scan(&alertRows).Error; err != nil {
		s.logger.Error("failed to query recent alerts", zap.Error(err))
	}
	for _, r := range alertRows {
		o.RecentAlerts = append(o.RecentAlerts, AlertBrief{
			ID:        r.ID,
			Severity:  r.Severity,
			Title:     r.Title,
			CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	if o.RecentAlerts == nil {
		o.RecentAlerts = []AlertBrief{}
	}

	// Platform connections
	o.PlatformConnections = s.getPlatformConnections()


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

// GetDailyBrief returns the daily composite brief for the seller's workspace.
func (s *Service) GetDailyBrief() (*DailyBrief, error) {
	b := &DailyBrief{
		LowStockSkus:       make([]LowStockSkuBrief, 0),
		NegativeMarginSkus: make([]NegativeMarginSkuBrief, 0),
		RecentExceptions:   make([]ExceptionBrief, 0),
		UrgentConversations: make([]UrgentConversationBrief, 0),
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	monthStart, monthEnd := monthRange(now)

	// Today profit/revenue from profit_summary
	var todayAgg struct {
		Profit  float64
		Revenue float64
	}
	if err := s.db.Table("profit_summary").
		Select("COALESCE(SUM(estimated_profit),0) AS profit, COALESCE(SUM(target_revenue),0) AS revenue").
		Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).
		Scan(&todayAgg).Error; err != nil {
		s.logger.Error("failed to query today profit", zap.Error(err))
	}
	b.TodayProfit = todayAgg.Profit
	b.TodayRevenue = todayAgg.Revenue

	// Month profit/revenue from profit_summary
	var monthAgg struct {
		Profit  float64
		Revenue float64
	}
	if err := s.db.Table("profit_summary").
		Select("COALESCE(SUM(estimated_profit),0) AS profit, COALESCE(SUM(target_revenue),0) AS revenue").
		Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).
		Scan(&monthAgg).Error; err != nil {
		s.logger.Error("failed to query month profit", zap.Error(err))
	}
	b.MonthProfit = monthAgg.Profit
	b.MonthRevenue = monthAgg.Revenue

	// Month cost from finance_ledger_entry (same pattern as Overview)
	var costAgg struct {
		Cost float64
	}
	if err := s.db.Table("finance_ledger_entry").
		Select("COALESCE(SUM(CASE WHEN entry_type IN ('product_cost','shipping_cost','platform_fee','payment_fee','other_fee') THEN amount ELSE 0 END),0) AS cost").
		Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).
		Scan(&costAgg).Error; err != nil {
		s.logger.Error("failed to query month cost", zap.Error(err))
	}
	b.MonthCost = costAgg.Cost

	// Counts
	if err := s.db.Table("exception_item").Where("status = ?", "open").Count(&b.OpenExceptionCount).Error; err != nil {
		s.logger.Error("failed to count exceptions", zap.Error(err))
	}
	if err := s.db.Table("sku").Where("warning_stock > 0 AND stock <= warning_stock").Count(&b.LowStockCount).Error; err != nil {
		s.logger.Error("failed to count low stock", zap.Error(err))
	}
	if err := s.db.Table("sku").Where("stock <= 0").Count(&b.OutOfStockCount).Error; err != nil {
		s.logger.Error("failed to count out of stock", zap.Error(err))
	}
	if err := s.db.Table("profit_summary").Where("status = ?", "unprofitable").Count(&b.NegativeMarginCount).Error; err != nil {
		s.logger.Error("failed to count negative margin", zap.Error(err))
	}
	if err := s.db.Table("customer_conversations").Where("status = ? AND priority IN ?", "open", []string{"high", "urgent"}).Count(&b.PendingSupportCount).Error; err != nil {
		s.logger.Error("failed to count pending support", zap.Error(err))
	}
	if err := s.db.Table("after_sales_order").Where("status IN ?", []string{"pending", "open"}).Count(&b.PendingAftersalesCount).Error; err != nil {
		s.logger.Error("failed to count pending aftersales", zap.Error(err))
	}

	// Top 5 low stock SKUs
	if err := s.db.Table("sku").
		Select("id AS sku_id, product_id, code, spec_desc, stock, warning_stock").
		Where("warning_stock > 0 AND stock <= warning_stock").
		Order("stock ASC, warning_stock DESC").
		Limit(5).
		Scan(&b.LowStockSkus).Error; err != nil {
		s.logger.Error("failed to query low stock SKUs", zap.Error(err))
	}

	// Top 5 negative margin SKUs (distinct by product_id, newest calculation)
	var negRows []struct {
		ProductID       int64
		SkuCode         string
		Title           string
		ProfitMargin    float64
		EstimatedProfit float64
	}
	if err := s.db.Raw(`
		SELECT ps.product_id,
			COALESCE((SELECT s.code FROM sku s WHERE s.product_id = ps.product_id LIMIT 1),'') AS sku_code,
			COALESCE(cp.title,'') AS title,
			ps.profit_margin, ps.estimated_profit
		FROM profit_summary ps
		INNER JOIN (
			SELECT product_id, MAX(id) AS max_id
			FROM profit_summary
			WHERE status = ?
			GROUP BY product_id
		) latest ON latest.max_id = ps.id
		LEFT JOIN candidate_product cp ON cp.id = ps.product_id
		ORDER BY ps.profit_margin ASC
		LIMIT 5
	`, "unprofitable").Scan(&negRows).Error; err != nil {
		s.logger.Error("failed to query negative margin SKUs", zap.Error(err))
	}
	for _, r := range negRows {
		b.NegativeMarginSkus = append(b.NegativeMarginSkus, NegativeMarginSkuBrief{
			ProductID:       r.ProductID,
			SkuCode:         r.SkuCode,
			Title:           r.Title,
			ProfitMargin:    r.ProfitMargin,
			EstimatedProfit: r.EstimatedProfit,
		})
	}

	// Top 5 recent open exceptions
	type excRow struct {
		ID           int64
		Severity     string
		SourceModule string
		Message      string
		Status       string
		CreatedAt    time.Time
	}
	var excRows []excRow
	if err := s.db.Table("exception_item").
		Select("id, severity, source_module, COALESCE(description,'') AS message, status, created_at").
		Where("status = ?", "open").
		Order("created_at DESC").
		Limit(5).
		Scan(&excRows).Error; err != nil {
		s.logger.Error("failed to query recent exceptions", zap.Error(err))
	}
	for _, r := range excRows {
		b.RecentExceptions = append(b.RecentExceptions, ExceptionBrief{
			ID:           r.ID,
			Severity:     r.Severity,
			SourceModule: r.SourceModule,
			Message:      r.Message,
			Status:       r.Status,
			CreatedAt:    r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// Top 5 urgent conversations
	type convRow struct {
		ID            int64
		CustomerName  string
		Subject       string
		Priority      string
		Platform      string
		LastMessageAt *time.Time
	}
	var convRows []convRow
	if err := s.db.Table("customer_conversations").
		Where("status = ? AND priority IN ?", "open", []string{"high", "urgent"}).
		Order("priority DESC, last_message_at ASC").
		Limit(5).
		Scan(&convRows).Error; err != nil {
		s.logger.Error("failed to query urgent conversations", zap.Error(err))
	}
	for _, r := range convRows {
		var lastMsg *string
		if r.LastMessageAt != nil {
			s := r.LastMessageAt.Format("2006-01-02T15:04:05Z")
			lastMsg = &s
		}
		b.UrgentConversations = append(b.UrgentConversations, UrgentConversationBrief{
			ID:            r.ID,
			CustomerName:  r.CustomerName,
			Subject:       r.Subject,
			Priority:      r.Priority,
			Platform:      r.Platform,
			LastMessageAt: lastMsg,
		})
	}

	// Platform connections (same query as Overview)
	b.PlatformConnections = s.getPlatformConnections()

	return b, nil
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

// GetRejectionReasonStats returns rejection counts grouped by agent and reason in the last 30 days.
func (s *Service) GetRejectionReasonStats() ([]RejectionReasonStat, error) {
	var items []RejectionReasonStat
	if err := s.db.Table("unified_action").
		Select("agent_id, rejection_reason, COUNT(*) AS count").
		Where("status = ? AND created_at > ?", "rejected", time.Now().Add(-30*24*time.Hour)).
		Group("agent_id, rejection_reason").
		Order("agent_id ASC, count DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// getPlatformConnections returns platform connection statuses, shared by Overview and GetDailyBrief.
func (s *Service) getPlatformConnections() []PlatformConnectionStatus {
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
	out := make([]PlatformConnectionStatus, 0, len(prs))
	for _, pr := range prs {
		var lastSync *string
		if pr.LastSyncAt != nil {
			s := pr.LastSyncAt.Format("2006-01-02T15:04:05Z")
			lastSync = &s
		}
		out = append(out, PlatformConnectionStatus{
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
	return out
}
