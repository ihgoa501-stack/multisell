package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides finance business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	orderReader OrderFinanceReader
	bankAdapter BankAdapter
}

// NewService creates a new finance service.
func NewService(db *gorm.DB, logger *zap.Logger, orderReader OrderFinanceReader) *Service {
	return &Service{db: db, logger: logger, orderReader: orderReader}
}

// WithBankAdapter sets the bank adapter on the service.
func (s *Service) WithBankAdapter(ba BankAdapter) *Service {
	s.bankAdapter = ba
	return s
}

// ---------- FinanceAccount ----------

// ListAccounts returns paginated accounts with optional filter.
func (s *Service) ListAccounts(p *common.Pagination, f *AccountListFilter) ([]FinanceAccount, int64, error) {
	q := s.db.Model(&FinanceAccount{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("name ILIKE ?", like)
		}
		if f.AccountType != "" {
			q = q.Where("account_type = ?", f.AccountType)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FinanceAccount
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetAccount returns a single account by id.
func (s *Service) GetAccount(id int64) (*FinanceAccount, error) {
	var a FinanceAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateAccount inserts a new account.
func (s *Service) CreateAccount(in *CreateAccountInput) (*FinanceAccount, error) {
	a := FinanceAccount{
		Name:        in.Name,
		AccountType: in.AccountType,
		PlatformID:  in.PlatformID,
	}
	if in.Currency != "" {
		a.Currency = in.Currency
	} else {
		a.Currency = "CNY"
	}
	if in.Balance != nil {
		a.Balance = *in.Balance
	}
	if in.Status != "" {
		a.Status = in.Status
	} else {
		a.Status = "active"
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateAccount applies partial updates to an account.
func (s *Service) UpdateAccount(id int64, in *UpdateAccountInput) (*FinanceAccount, error) {
	var a FinanceAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.AccountType != nil {
		updates["account_type"] = *in.AccountType
	}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.Currency != nil {
		updates["currency"] = *in.Currency
	}
	if in.Balance != nil {
		updates["balance"] = *in.Balance
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &a, nil
	}
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteAccount removes an account by id.
func (s *Service) DeleteAccount(id int64) error {
	res := s.db.Delete(&FinanceAccount{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- FinanceTransaction ----------

// ListTransactions returns paginated transactions with optional filter.
func (s *Service) ListTransactions(p *common.Pagination, f *TransactionListFilter) ([]FinanceTransaction, int64, error) {
	q := s.db.Model(&FinanceTransaction{})
	if f != nil {
		if f.AccountID != nil {
			q = q.Where("account_id = ?", *f.AccountID)
		}
		if f.TransactionType != "" {
			q = q.Where("transaction_type = ?", f.TransactionType)
		}
		if f.OrderID != nil {
			q = q.Where("order_id = ?", *f.OrderID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FinanceTransaction
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreateTransaction inserts a new transaction.
func (s *Service) CreateTransaction(in *CreateTransactionInput) (*FinanceTransaction, error) {
	t := FinanceTransaction{
		AccountID:       in.AccountID,
		TransactionType: in.TransactionType,
		Amount:          in.Amount,
		OrderID:         in.OrderID,
		SettlementID:    in.SettlementID,
		PlatformID:      in.PlatformID,
		Description:     in.Description,
		TransactionDate: in.TransactionDate,
	}
	if in.Currency != "" {
		t.Currency = in.Currency
	} else {
		t.Currency = "CNY"
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		// Adjust account balance based on transaction type
		delta := in.Amount
		switch in.TransactionType {
		case "cost", "fee", "refund", "transfer":
			delta = -in.Amount
		}
		if err := tx.Model(&FinanceAccount{}).
			Where("id = ?", in.AccountID).
			UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ---------- FinanceLedgerEntry ----------

// ListLedgerEntries returns paginated ledger entries with optional filter.
func (s *Service) ListLedgerEntries(p *common.Pagination, f *LedgerListFilter) ([]FinanceLedgerEntry, int64, error) {
	q := s.db.Model(&FinanceLedgerEntry{})
	if f != nil {
		if f.OrderID != nil {
			q = q.Where("order_id = ?", *f.OrderID)
		}
		if f.EntryType != "" {
			q = q.Where("entry_type = ?", f.EntryType)
		}
		if f.CostLayer != "" {
			q = q.Where("cost_layer = ?", f.CostLayer)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FinanceLedgerEntry
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ---------- Summary ----------

// Summary returns finance aggregation for dashboard.
func (s *Service) Summary() (*FinanceSummary, error) {
	// Total balance across all active accounts
	var totalBal struct {
		Total float64
	}
	if err := s.db.Model(&FinanceAccount{}).
		Where("status = ?", "active").
		Select("COALESCE(SUM(balance),0) AS total").
		Scan(&totalBal).Error; err != nil {
		return nil, err
	}

	// Balance by account type
	type typeBal struct {
		AccountType string
		Total       float64
	}
	var tbs []typeBal
	if err := s.db.Model(&FinanceAccount{}).
		Where("status = ?", "active").
		Select("account_type, COALESCE(SUM(balance),0) AS total").
		Group("account_type").
		Scan(&tbs).Error; err != nil {
		return nil, err
	}
	balanceByType := make(map[string]float64, len(tbs))
	for _, tb := range tbs {
		balanceByType[tb.AccountType] = tb.Total
	}

	// Ledger amount by entry type
	type entrySum struct {
		EntryType string
		Total     float64
	}
	var ess []entrySum
	if err := s.db.Model(&FinanceLedgerEntry{}).
		Select("entry_type, COALESCE(SUM(amount),0) AS total").
		Group("entry_type").
		Scan(&ess).Error; err != nil {
		return nil, err
	}
	ledgerByEntryType := make(map[string]float64, len(ess))
	for _, es := range ess {
		ledgerByEntryType[es.EntryType] = es.Total
	}

	// Current month revenue / cost from transactions
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	type monthSum struct {
		Revenue float64
		Cost    float64
	}
	var ms monthSum
	if err := s.db.Model(&FinanceTransaction{}).
		Where("transaction_date >= ?", monthStart).
		Select("COALESCE(SUM(CASE WHEN transaction_type = 'revenue' THEN amount ELSE 0 END),0) AS revenue, " +
			"COALESCE(SUM(CASE WHEN transaction_type IN ('cost','fee','refund') THEN amount ELSE 0 END),0) AS cost").
		Scan(&ms).Error; err != nil {
		return nil, err
	}

	return &FinanceSummary{
		TotalBalance:      totalBal.Total,
		BalanceByType:     balanceByType,
		LedgerByEntryType: ledgerByEntryType,
		MonthRevenue:      ms.Revenue,
		MonthCost:         ms.Cost,
	}, nil
}

// ---------- Profit Summary / Order Ledger ----------

// profitSummaryQuery returns a fresh query with the filter applied.
func (s *Service) profitSummaryQuery(f *ProfitSummaryFilter) *gorm.DB {
	q := s.db.Table("finance_ledger_entry").
		Joins("LEFT JOIN sales_order ON sales_order.id = finance_ledger_entry.order_id")
	if f != nil {
		if f.From != "" {
			q = q.Where("sales_order.created_at >= ?", f.From)
		}
		if f.To != "" {
			q = q.Where("sales_order.created_at <= ?", f.To)
		}
		if f.PlatformID != nil {
			q = q.Where("sales_order.platform_id = ?", *f.PlatformID)
		}
	}
	return q
}

// ProfitSummary returns aggregated profit across orders in a date range.
func (s *Service) ProfitSummary(f *ProfitSummaryFilter) (*ProfitSummary, error) {
	revenueExpr := "COALESCE(SUM(CASE WHEN finance_ledger_entry.entry_type = 'revenue' THEN finance_ledger_entry.amount ELSE 0 END),0)"
	costExpr := "COALESCE(SUM(CASE WHEN finance_ledger_entry.entry_type IN ('product_cost','shipping_cost','platform_fee','payment_fee','other_fee','refund') THEN finance_ledger_entry.amount ELSE 0 END),0)"

	// Totals
	type totalRow struct {
		Revenue float64
		Cost    float64
	}
	var tr totalRow
	if err := s.profitSummaryQuery(f).
		Select(revenueExpr + " AS revenue, " + costExpr + " AS cost").
		Scan(&tr).Error; err != nil {
		return nil, err
	}
	profit := tr.Revenue - tr.Cost
	margin := 0.0
	if tr.Revenue > 0 {
		margin = profit / tr.Revenue
	}

	// By platform
	type platformRow struct {
		PlatformID *int64
		Revenue    float64
		Cost       float64
	}
	var prs []platformRow
	if err := s.profitSummaryQuery(f).
		Select("sales_order.platform_id, " + revenueExpr + " AS revenue, " + costExpr + " AS cost").
		Group("sales_order.platform_id").
		Scan(&prs).Error; err != nil {
		return nil, err
	}
	byPlatform := make([]ProfitByPlatform, 0, len(prs))
	for _, pr := range prs {
		byPlatform = append(byPlatform, ProfitByPlatform{
			PlatformID: pr.PlatformID,
			Revenue:    pr.Revenue,
			Cost:       pr.Cost,
			Profit:     pr.Revenue - pr.Cost,
		})
	}

	// By month
	type monthRow struct {
		Month   string
		Revenue float64
		Cost    float64
	}
	var mrs []monthRow
	if err := s.profitSummaryQuery(f).
		Select("TO_CHAR(sales_order.created_at, 'YYYY-MM') AS month, " + revenueExpr + " AS revenue, " + costExpr + " AS cost").
		Group("TO_CHAR(sales_order.created_at, 'YYYY-MM')").
		Order("month ASC").
		Scan(&mrs).Error; err != nil {
		return nil, err
	}
	byMonth := make([]ProfitByMonth, 0, len(mrs))
	for _, mr := range mrs {
		byMonth = append(byMonth, ProfitByMonth{
			Month:   mr.Month,
			Revenue: mr.Revenue,
			Cost:    mr.Cost,
			Profit:  mr.Revenue - mr.Cost,
		})
	}

	return &ProfitSummary{
		TotalRevenue: tr.Revenue,
		TotalCost:    tr.Cost,
		TotalProfit:  profit,
		ProfitMargin: margin,
		ByPlatform:   byPlatform,
		ByMonth:      byMonth,
	}, nil
}

// ListOrderLedger returns all ledger entries for a given order.
func (s *Service) ListOrderLedger(orderID int64) ([]FinanceLedgerEntry, error) {
	var items []FinanceLedgerEntry
	if err := s.db.Where("order_id = ?", orderID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// OrderProfit returns the per-order profit summary aggregated from ledger entries.
func (s *Service) OrderProfit(orderID int64) (*OrderProfit, error) {
	type row struct {
		EntryType string
		Total     float64
	}
	var rows []row
	if err := s.db.Model(&FinanceLedgerEntry{}).
		Where("order_id = ?", orderID).
		Select("entry_type, COALESCE(SUM(amount),0) AS total").
		Group("entry_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	p := &OrderProfit{OrderID: orderID}
	for _, r := range rows {
		switch r.EntryType {
		case "revenue":
			p.Revenue = r.Total
		case "product_cost":
			p.ProductCost = r.Total
		case "shipping_cost":
			p.ShippingCost = r.Total
		case "platform_fee":
			p.PlatformFee = r.Total
		case "payment_fee":
			p.PaymentFee = r.Total
		case "other_fee":
			p.OtherFee = r.Total
		}
	}
	cost := p.ProductCost + p.ShippingCost + p.PlatformFee + p.PaymentFee + p.OtherFee
	p.Profit = p.Revenue - cost
	if p.Revenue > 0 {
		p.Margin = p.Profit / p.Revenue
	}
	return p, nil
}

// RebuildOrderLedger deletes and regenerates ledger entries for an order.
// It reads from sales_order (which already snapshots shipping fee, product cost
// and fees) and sales_order_item to rebuild revenue and cost lines.
func (s *Service) RebuildOrderLedger(ctx context.Context, orderID int64) ([]FinanceLedgerEntry, error) {
	o, err := s.orderReader.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.orderReader.GetItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Revenue from order items subtotal (fallback to pay_amount).
	itemSubtotal := 0.0
	for _, it := range items {
		itemSubtotal += it.Subtotal
	}
	revenue := itemSubtotal
	if revenue == 0 {
		revenue = o.PayAmount
	}

	entries := []FinanceLedgerEntry{
		{OrderID: &orderID, EntryType: "revenue", Amount: revenue, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "order revenue snapshot"},
		{OrderID: &orderID, EntryType: "product_cost", Amount: o.ProductCost, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "product cost snapshot"},
		{OrderID: &orderID, EntryType: "shipping_cost", Amount: o.ShippingFee, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "shipping snapshot"},
		{OrderID: &orderID, EntryType: "platform_fee", Amount: o.PlatformFee, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "platform fee snapshot"},
		{OrderID: &orderID, EntryType: "payment_fee", Amount: o.PaymentFee, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "payment fee snapshot"},
		{OrderID: &orderID, EntryType: "other_fee", Amount: o.OtherFee, Currency: "CNY", CostLayer: "snapshot", SourceType: "sales_order", SourceID: &orderID, Description: "other fee snapshot"},
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", orderID).Delete(&FinanceLedgerEntry{}).Error; err != nil {
			return err
		}
		return tx.Create(&entries).Error
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ---------- Profit Calculation ----------

// CalculateOrderProfit calculates profit for a single order and stores per-SKU records.
// Profit = Revenue - PlatformFee - LogisticsFee - PurchaseCost - AdvertisingCost - OtherCost.
func (s *Service) CalculateOrderProfit(ctx context.Context, orderID int64) (*ProfitCalculation, error) {
	o, err := s.orderReader.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.orderReader.GetItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Remove any existing profit records for this order
	if err := s.db.Where("order_id = ?", orderID).Delete(&ProfitCalculation{}).Error; err != nil {
		return nil, err
	}

	totalSubtotal := 0.0
	for _, it := range items {
		totalSubtotal += it.Subtotal
	}
	if totalSubtotal == 0 {
		totalSubtotal = o.PayAmount
	}

	// Derive time boundaries from the order
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	aggregate := &ProfitCalculation{
		OrderID:     orderID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}

	for _, it := range items {
		ratio := 1.0
		if totalSubtotal > 0 {
			ratio = it.Subtotal / totalSubtotal
		}

		revenue := it.Subtotal
		platformFee := o.PlatformFee * ratio
		logisticsFee := o.ShippingFee * ratio
		purchaseCost := o.ProductCost * ratio
		otherCost := o.OtherFee * ratio
		netProfit := revenue - platformFee - logisticsFee - purchaseCost - otherCost

		margin := 0.0
		if revenue > 0 {
			margin = netProfit / revenue
		}

		rec := ProfitCalculation{
			OrderID:      orderID,
			SkuID:        it.SkuID,
			Revenue:      revenue,
			PlatformFee:  platformFee,
			LogisticsFee: logisticsFee,
			PurchaseCost: purchaseCost,
			OtherCost:    otherCost,
			NetProfit:    netProfit,
			ProfitMargin: margin,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
		}
		if err := s.db.Create(&rec).Error; err != nil {
			return nil, err
		}

		// Accumulate into the aggregate
		aggregate.Revenue += revenue
		aggregate.PlatformFee += platformFee
		aggregate.LogisticsFee += logisticsFee
		aggregate.PurchaseCost += purchaseCost
		aggregate.OtherCost += otherCost
		aggregate.NetProfit += netProfit
	}

	if aggregate.Revenue > 0 {
		aggregate.ProfitMargin = aggregate.NetProfit / aggregate.Revenue
	}

	return aggregate, nil
}

// BatchCalculate calculates profit for all orders within the given time range.
// Returns the number of orders processed.
func (s *Service) BatchCalculate(ctx context.Context, since, until time.Time) (int, error) {
	orders, err := s.orderReader.ListByTimeRange(ctx, since, until)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, o := range orders {
		if _, err := s.CalculateOrderProfit(ctx, o.ID); err != nil {
			s.logger.Warn("batch calculate: order profit failed", zap.Int64("order_id", o.ID), zap.Error(err))
			continue
		}
		count++
	}
	return count, nil
}

// GetProfitSummary returns profit summary for the given time range.
func (s *Service) GetProfitSummary(ctx context.Context, since, until time.Time) (*ProfitSummaryResult, error) {
	q := s.db.Model(&ProfitCalculation{}).
		Where("period_start >= ? AND period_end <= ?", since, until)

	type row struct {
		TotalProfit  float64
		AvgMargin    float64
		LossSKUCount int64
		PeriodCount  int64
		TotalRevenue float64
		TotalCost    float64
	}
	var r row
	if err := q.Select(`
		COALESCE(SUM(net_profit),0) AS total_profit,
		COALESCE(AVG(profit_margin),0) AS avg_margin,
		COALESCE(SUM(CASE WHEN net_profit < 0 THEN 1 ELSE 0 END),0) AS loss_sku_count,
		COUNT(DISTINCT order_id) AS period_count,
		COALESCE(SUM(revenue),0) AS total_revenue,
		COALESCE(SUM(platform_fee+logistics_fee+purchase_cost+advertising_cost+other_cost),0) AS total_cost
	`).Scan(&r).Error; err != nil {
		return nil, err
	}

	return &ProfitSummaryResult{
		TotalProfit:  r.TotalProfit,
		AvgMargin:    r.AvgMargin,
		LossSKUCount: r.LossSKUCount,
		PeriodCount:  r.PeriodCount,
		TotalRevenue: r.TotalRevenue,
		TotalCost:    r.TotalCost,
	}, nil
}

// GetSKUProfitRanking returns SKU profit ranking ordered by net profit descending.
func (s *Service) GetSKUProfitRanking(ctx context.Context, since time.Time, limit int) ([]*ProfitCalculation, error) {
	if limit <= 0 {
		limit = 20
	}
	var results []*ProfitCalculation
	if err := s.db.Where("period_start >= ?", since).
		Order("net_profit DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// Mock generates N fake ledger entries for development/demo.
func (s *Service) Mock(count int) ([]FinanceLedgerEntry, error) {
	if count <= 0 {
		count = 10
	}
	entries := make([]FinanceLedgerEntry, 0, count)
	types := []string{"revenue", "product_cost", "shipping_cost", "platform_fee", "payment_fee"}
	for i := 0; i < count; i++ {
		amt := float64((i%10 + 1) * 100)
		if i%2 == 1 {
			amt = -amt
		}
		orderID := int64(i%5 + 1)
		entries = append(entries, FinanceLedgerEntry{
			OrderID:     &orderID,
			EntryType:   types[i%len(types)],
			Amount:      amt,
			Currency:    "CNY",
			CostLayer:   "estimated",
			SourceType:  "mock",
			Description: "mock entry " + time.Now().Format("2006-01-02"),
		})
	}
	if err := s.db.Create(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// ExecuteBankTransfer transfers money from one account to another, utilizing the BankAdapter.
// It is transaction-safe.
func (s *Service) ExecuteBankTransfer(ctx context.Context, fromAccountID, toAccountID int64, amountCents int64, currency string) (*TransferResponse, error) {
	if s.bankAdapter == nil {
		return nil, fmt.Errorf("bank adapter not configured")
	}

	var fromAcct, toAcct FinanceAccount
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&fromAcct, fromAccountID).Error; err != nil {
			return err
		}
		if err := tx.First(&toAcct, toAccountID).Error; err != nil {
			return err
		}
		if fromAcct.Status != "active" || toAcct.Status != "active" {
			return fmt.Errorf("both accounts must be active")
		}
		amountFloat := float64(amountCents) / 100.0
		if fromAcct.Balance < amountFloat {
			return fmt.Errorf("insufficient balance: account has %.2f, transfer requires %.2f", fromAcct.Balance, amountFloat)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	req := &TransferRequest{
		SourceAccount: fromAcct.Name,
		TargetAccount: toAcct.Name,
		AmountCents:   amountCents,
		Currency:      currency,
		Description:   fmt.Sprintf("Transfer from Account %d to %d", fromAccountID, toAccountID),
	}

	resp, err := s.bankAdapter.ExecuteTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return resp, nil
	}

	amountFloat := float64(amountCents) / 100.0
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&FinanceAccount{}).Where("id = ?", fromAccountID).UpdateColumn("balance", gorm.Expr("balance - ?", amountFloat)).Error; err != nil {
			return err
		}
		if err := tx.Model(&FinanceAccount{}).Where("id = ?", toAccountID).UpdateColumn("balance", gorm.Expr("balance + ?", amountFloat)).Error; err != nil {
			return err
		}

		tFrom := FinanceTransaction{
			AccountID:       fromAccountID,
			TransactionType: "transfer",
			Amount:          amountFloat,
			Currency:        currency,
			Description:     fmt.Sprintf("Bank Transfer to %s (Tx ID: %s)", toAcct.Name, resp.TransactionID),
		}
		if err := tx.Create(&tFrom).Error; err != nil {
			return err
		}

		tTo := FinanceTransaction{
			AccountID:       toAccountID,
			TransactionType: "revenue",
			Amount:          amountFloat,
			Currency:        currency,
			Description:     fmt.Sprintf("Bank Transfer from %s (Tx ID: %s)", fromAcct.Name, resp.TransactionID),
		}
		if err := tx.Create(&tTo).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateDoubleEntryLedger creates matching debit and credit entries.
func (s *Service) CreateDoubleEntryLedger(ctx context.Context, entries []LedgerEntry) error {
	var totalDebit int64
	var totalCredit int64
	for _, e := range entries {
		totalDebit += e.DebitCents
		totalCredit += e.CreditCents
	}
	if totalDebit != totalCredit {
		return fmt.Errorf("ledger entries unbalanced: total debits (%d) must equal total credits (%d)", totalDebit, totalCredit)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&entries).Error
	})
}
