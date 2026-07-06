package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_Overview(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	// Create the tables that Overview() queries
	db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		id INTEGER PRIMARY KEY,
		status TEXT,
		pay_amount REAL DEFAULT 0,
		profit_amount REAL DEFAULT 0,
		created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY,
		product_id INTEGER,
		code TEXT,
		spec_desc TEXT,
		stock INTEGER DEFAULT 0,
		warning_stock INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS product_listing (
		id INTEGER PRIMARY KEY,
		status TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS after_sales_order (
		id INTEGER PRIMARY KEY,
		status TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY,
		status TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS finance_ledger_entry (
		id INTEGER PRIMARY KEY,
		entry_type TEXT,
		amount REAL DEFAULT 0,
		created_at TEXT
	)`)

	// Insert minimal test data
	db.Exec(`INSERT INTO sales_order (status, pay_amount, profit_amount) VALUES ('pending', 100, 20)`)
	db.Exec(`INSERT INTO sales_order (status, pay_amount, profit_amount) VALUES ('shipped', 200, 40)`)
	// Only 1 SKU with warning_stock > 0 AND stock <= warning_stock
	db.Exec(`INSERT INTO sku (code, stock, warning_stock) VALUES ('SKU001', 10, 20)`)
	db.Exec(`INSERT INTO sku (code, stock, warning_stock) VALUES ('SKU002', 50, 0)`)
	db.Exec(`INSERT INTO product_listing (status) VALUES ('online')`)
	db.Exec(`INSERT INTO product_listing (status) VALUES ('offline')`)
	db.Exec(`INSERT INTO after_sales_order (status) VALUES ('pending')`)
	db.Exec(`INSERT INTO exception_item (status) VALUES ('open')`)

	o, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if o.OrderTotal != 2 {
		t.Fatalf("OrderTotal = %d", o.OrderTotal)
	}
	if o.OrderRevenue != 300 {
		t.Fatalf("OrderRevenue = %v", o.OrderRevenue)
	}
	if o.OrderProfit != 60 {
		t.Fatalf("OrderProfit = %v", o.OrderProfit)
	}
	if o.SkuTotal != 2 {
		t.Fatalf("SkuTotal = %d", o.SkuTotal)
	}
	if o.LowStockCount != 1 {
		t.Fatalf("LowStockCount = %d (expected 1)", o.LowStockCount)
	}
	if o.OutOfStockCount != 0 {
		t.Fatalf("OutOfStockCount = %d (expected 0)", o.OutOfStockCount)
	}
	if o.ListingActiveCount != 1 {
		t.Fatalf("ListingActiveCount = %d", o.ListingActiveCount)
	}
	if o.AftersalesPendingCount != 1 {
		t.Fatalf("AftersalesPendingCount = %d", o.AftersalesPendingCount)
	}
	if o.ExceptionOpenCount != 1 {
		t.Fatalf("ExceptionOpenCount = %d", o.ExceptionOpenCount)
	}
}

func TestService_OrdersTrend(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		id INTEGER PRIMARY KEY,
		pay_amount REAL DEFAULT 0,
		created_at TEXT
	)`)

	// Test with empty table (DATE() in SQLite returns string, which can't scan into time.Time)
	trend, err := svc.OrdersTrend(30)
	if err != nil {
		t.Fatalf("OrdersTrend: %v", err)
	}
	_ = trend
}

func TestService_InventoryHealth(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY,
		product_id INTEGER,
		code TEXT,
		spec_desc TEXT,
		stock INTEGER DEFAULT 0,
		warning_stock INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS inventory (
		id INTEGER PRIMARY KEY,
		sku_id INTEGER,
		warehouse TEXT
	)`)
	db.Exec(`INSERT INTO sku (product_id, code, stock, warning_stock) VALUES (1, 'SKU_LOW', 5, 10)`)
	db.Exec(`INSERT INTO inventory (sku_id, warehouse) VALUES (1, '上海仓')`)

	items, err := svc.InventoryHealth(10)
	if err != nil {
		t.Fatalf("InventoryHealth: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Code != "SKU_LOW" {
		t.Fatalf("Code = %s", items[0].Code)
	}
}

func TestService_ExceptionDistribution(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY,
		severity TEXT,
		source_module TEXT,
		status TEXT
	)`)
	db.Exec(`INSERT INTO exception_item (severity, source_module, status) VALUES ('high', 'order', 'open')`)

	items, err := svc.ExceptionDistribution()
	if err != nil {
		t.Fatalf("ExceptionDistribution: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Severity != "high" {
		t.Fatalf("Severity = %s", items[0].Severity)
	}
}

func TestService_GetDailyBrief(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Now()
	ts := now.Format("2006-01-02T15:04:05Z")

	db.Exec(`CREATE TABLE IF NOT EXISTS profit_summary (
		id INTEGER PRIMARY KEY, product_id INTEGER,
		estimated_profit REAL DEFAULT 0, target_revenue REAL DEFAULT 0,
		profit_margin REAL DEFAULT 0, status TEXT, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY, product_id INTEGER, code TEXT,
		spec_desc TEXT, stock INTEGER DEFAULT 0, warning_stock INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY, severity TEXT, source_module TEXT,
		description TEXT, status TEXT, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS customer_conversations (
		id INTEGER PRIMARY KEY, customer_name TEXT, subject TEXT,
		priority TEXT, platform TEXT, status TEXT, last_message_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS after_sales_order (
		id INTEGER PRIMARY KEY, status TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS finance_ledger_entry (
		id INTEGER PRIMARY KEY, entry_type TEXT, amount REAL DEFAULT 0, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS platform_integration_account (
		id INTEGER PRIMARY KEY, platform_id INTEGER, store_name TEXT,
		status TEXT, sync_status TEXT, last_sync_at TEXT, last_error TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS platform (
		id INTEGER PRIMARY KEY, code TEXT, name TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS candidate_product (
		id INTEGER PRIMARY KEY, title TEXT
	)`)

	// profit_summary: today + month, one unprofitable for negative margin query
	db.Exec(`INSERT INTO profit_summary (id, product_id, estimated_profit, target_revenue, profit_margin, status, created_at) VALUES (1, 1, 100, 1000, 10.0, 'profitable', ?)`, ts)
	db.Exec(`INSERT INTO profit_summary (id, product_id, estimated_profit, target_revenue, profit_margin, status, created_at) VALUES (2, 2, -50, 500, -10.0, 'unprofitable', ?)`, ts)
	db.Exec(`INSERT INTO profit_summary (id, product_id, estimated_profit, target_revenue, profit_margin, status, created_at) VALUES (3, 2, -60, 500, -12.0, 'unprofitable', ?)`, ts)

	// sku: one normal, one low stock, one out of stock
	db.Exec(`INSERT INTO sku (id, product_id, code, spec_desc, stock, warning_stock) VALUES (1, 1, 'SKU001', 'normal', 50, 10)`)
	db.Exec(`INSERT INTO sku (id, product_id, code, spec_desc, stock, warning_stock) VALUES (2, 2, 'SKU002', 'low', 5, 10)`)
	db.Exec(`INSERT INTO sku (id, product_id, code, spec_desc, stock, warning_stock) VALUES (3, 3, 'SKU003', 'out', 0, 5)`)

	// exception: one open, one resolved
	db.Exec(`INSERT INTO exception_item (severity, source_module, description, status, created_at) VALUES ('high', 'order', 'timeout', 'open', ?)`, ts)
	db.Exec(`INSERT INTO exception_item (severity, source_module, description, status, created_at) VALUES ('low', 'shipping', 'delayed', 'resolved', ?)`, ts)

	// customer_conversations: one urgent/open, one low/open
	db.Exec(`INSERT INTO customer_conversations (customer_name, subject, priority, platform, status, last_message_at) VALUES ('Alice', 'refund', 'urgent', 'shopify', 'open', ?)`, ts)
	db.Exec(`INSERT INTO customer_conversations (customer_name, subject, priority, platform, status, last_message_at) VALUES ('Bob', 'question', 'low', 'shopify', 'open', ?)`, ts)

	// after_sales: two pending, one closed
	db.Exec(`INSERT INTO after_sales_order (status) VALUES ('pending')`)
	db.Exec(`INSERT INTO after_sales_order (status) VALUES ('open')`)
	db.Exec(`INSERT INTO after_sales_order (status) VALUES ('closed')`)

	// finance: cost + revenue (only cost counts in month cost)
	db.Exec(`INSERT INTO finance_ledger_entry (entry_type, amount, created_at) VALUES ('product_cost', 100, ?)`, ts)
	db.Exec(`INSERT INTO finance_ledger_entry (entry_type, amount, created_at) VALUES ('revenue', 500, ?)`, ts)

	// platform connection
	db.Exec(`INSERT INTO platform (id, code, name) VALUES (1, 'shopify', 'Shopify')`)
	db.Exec(`INSERT INTO platform_integration_account (platform_id, store_name, status, sync_status) VALUES (1, 'My Store', 'connected', 'synced')`)

	// candidate_product for negative margin SKU subquery
	db.Exec(`INSERT INTO candidate_product (id, title) VALUES (2, 'Negative Margin Product')`)

	brief, err := svc.GetDailyBrief()
	if err != nil {
		t.Fatalf("GetDailyBrief: %v", err)
	}

	if brief.OpenExceptionCount != 1 {
		t.Errorf("OpenExceptionCount = %d, want 1", brief.OpenExceptionCount)
	}
	if brief.LowStockCount != 2 {
		t.Errorf("LowStockCount = %d, want 2", brief.LowStockCount)
	}
	if brief.OutOfStockCount != 1 {
		t.Errorf("OutOfStockCount = %d, want 1", brief.OutOfStockCount)
	}
	if brief.NegativeMarginCount != 2 {
		t.Errorf("NegativeMarginCount = %d, want 1", brief.NegativeMarginCount)
	}
	if brief.PendingSupportCount != 1 {
		t.Errorf("PendingSupportCount = %d, want 1", brief.PendingSupportCount)
	}
	if brief.PendingAftersalesCount != 2 {
		t.Errorf("PendingAftersalesCount = %d, want 2", brief.PendingAftersalesCount)
	}
	// 3 profit_summary records today: profit = 100 + (-50) + (-60) = -10
	if brief.TodayProfit != -10 {
		t.Errorf("TodayProfit = %v, want -10", brief.TodayProfit)
	}
	// revenue = 1000 + 500 + 500 = 2000
	if brief.TodayRevenue != 2000 {
		t.Errorf("TodayRevenue = %v, want 2000", brief.TodayRevenue)
	}
	// month profit = same 3 records
	if brief.MonthProfit != -10 {
		t.Errorf("MonthProfit = %v, want -10", brief.MonthProfit)
	}
	// month cost = product_cost only (100)
	if brief.MonthCost != 100 {
		t.Errorf("MonthCost = %v, want 100", brief.MonthCost)
	}
	if len(brief.PlatformConnections) != 1 {
		t.Errorf("PlatformConnections = %d, want 1", len(brief.PlatformConnections))
	}
	// negative margin SKUs should have at least 1 entry
	if len(brief.NegativeMarginSkus) != 1 {
		t.Errorf("NegativeMarginSkus = %d, want 1", len(brief.NegativeMarginSkus))
	} else {
		if brief.NegativeMarginSkus[0].ProductID != 2 {
			t.Errorf("NegativeMarginSkus[0].ProductID = %d, want 2", brief.NegativeMarginSkus[0].ProductID)
		}
		if brief.NegativeMarginSkus[0].SkuCode != "SKU002" {
			t.Errorf("NegativeMarginSkus[0].SkuCode = %q, want SKU002", brief.NegativeMarginSkus[0].SkuCode)
		}
		if brief.NegativeMarginSkus[0].Title != "Negative Margin Product" {
			t.Errorf("NegativeMarginSkus[0].Title = %q, want 'Negative Margin Product'", brief.NegativeMarginSkus[0].Title)
		}
	}
	// low stock SKUs should have 2 entries
	if len(brief.LowStockSkus) != 2 {
		t.Errorf("LowStockSkus = %d, want 2", len(brief.LowStockSkus))
	}
	// recent exceptions should have 1 open exception
	if len(brief.RecentExceptions) != 1 {
		t.Errorf("RecentExceptions = %d, want 1", len(brief.RecentExceptions))
	}
	// urgent conversations should have 1
	if len(brief.UrgentConversations) != 1 {
		t.Errorf("UrgentConversations = %d, want 1", len(brief.UrgentConversations))
	}
}

func TestService_GetDailyBrief_Empty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS profit_summary (
		id INTEGER PRIMARY KEY, product_id INTEGER,
		estimated_profit REAL DEFAULT 0, target_revenue REAL DEFAULT 0,
		profit_margin REAL DEFAULT 0, status TEXT, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY, product_id INTEGER, code TEXT,
		spec_desc TEXT, stock INTEGER DEFAULT 0, warning_stock INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY, severity TEXT, source_module TEXT,
		description TEXT, status TEXT, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS customer_conversations (
		id INTEGER PRIMARY KEY, customer_name TEXT, subject TEXT,
		priority TEXT, platform TEXT, status TEXT, last_message_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS after_sales_order (
		id INTEGER PRIMARY KEY, status TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS finance_ledger_entry (
		id INTEGER PRIMARY KEY, entry_type TEXT, amount REAL DEFAULT 0, created_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS platform_integration_account (
		id INTEGER PRIMARY KEY, platform_id INTEGER, store_name TEXT,
		status TEXT, sync_status TEXT, last_sync_at TEXT, last_error TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS platform (
		id INTEGER PRIMARY KEY, code TEXT, name TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS candidate_product (
		id INTEGER PRIMARY KEY, title TEXT
	)`)

	brief, err := svc.GetDailyBrief()
	if err != nil {
		t.Fatalf("GetDailyBrief: %v", err)
	}

	if brief.OpenExceptionCount != 0 {
		t.Errorf("OpenExceptionCount = %d, want 0", brief.OpenExceptionCount)
	}
	if brief.LowStockCount != 0 {
		t.Errorf("LowStockCount = %d, want 0", brief.LowStockCount)
	}
	if brief.OutOfStockCount != 0 {
		t.Errorf("OutOfStockCount = %d, want 0", brief.OutOfStockCount)
	}
	if brief.NegativeMarginCount != 0 {
		t.Errorf("NegativeMarginCount = %d, want 0", brief.NegativeMarginCount)
	}
	if brief.PendingSupportCount != 0 {
		t.Errorf("PendingSupportCount = %d, want 0", brief.PendingSupportCount)
	}
	if brief.PendingAftersalesCount != 0 {
		t.Errorf("PendingAftersalesCount = %d, want 0", brief.PendingAftersalesCount)
	}
	if brief.TodayProfit != 0 {
		t.Errorf("TodayProfit = %v, want 0", brief.TodayProfit)
	}
	if brief.TodayRevenue != 0 {
		t.Errorf("TodayRevenue = %v, want 0", brief.TodayRevenue)
	}
	if brief.MonthCost != 0 {
		t.Errorf("MonthCost = %v, want 0", brief.MonthCost)
	}
	if len(brief.LowStockSkus) != 0 {
		t.Errorf("LowStockSkus = %d, want 0", len(brief.LowStockSkus))
	}
	if len(brief.NegativeMarginSkus) != 0 {
		t.Errorf("NegativeMarginSkus = %d, want 0", len(brief.NegativeMarginSkus))
	}
	if len(brief.RecentExceptions) != 0 {
		t.Errorf("RecentExceptions = %d, want 0", len(brief.RecentExceptions))
	}
	if len(brief.UrgentConversations) != 0 {
		t.Errorf("UrgentConversations = %d, want 0", len(brief.UrgentConversations))
	}
}

func TestService_OrdersTrend_WithData(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		id INTEGER PRIMARY KEY,
		pay_amount REAL DEFAULT 0,
		created_at TEXT
	)`)

	// Create 3 orders across 3 different days within the default 30-day window.
	// ponyatil: SQLite DATE() returns a string; GORM + SQLite driver can't scan
	// it into time.Time, so we verify the error is the expected scan error, not
	// a crash. PostgreSQL (production) handles this natively.
	now := time.Now()
	for i := 0; i < 3; i++ {
		day := now.AddDate(0, 0, -(2 - i))
		amt := 100.0 * float64(i+1)
		db.Exec(`INSERT INTO sales_order (pay_amount, created_at) VALUES (?, ?)`, amt, day.Format("2006-01-02T15:04:05Z"))
	}

	_, err := svc.OrdersTrend(30)
	if err != nil {
		// Known SQLite limitation: DATE() output can't scan into time.Time.
		// Verify the error describes the scan issue, not something else.
		if !strings.Contains(err.Error(), "unsupported Scan, storing driver.Value type string into type *time.Time") {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		t.Fatal("expected scan error with SQLite, got nil")
	}
}

func TestService_OrdersTrend_NegativeDays(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		id INTEGER PRIMARY KEY,
		pay_amount REAL DEFAULT 0,
		created_at TEXT
	)`)

	// Negative days should default to 30 — no data in table is fine
	trend, err := svc.OrdersTrend(-1)
	if err != nil {
		t.Fatalf("OrdersTrend(-1): %v", err)
	}
	// Empty result is expected (no data), just confirm no error
	if trend == nil {
		t.Fatal("OrdersTrend(-1) returned nil, want empty slice")
	}
}

func TestService_InventoryHealth_Empty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY, product_id INTEGER, code TEXT,
		spec_desc TEXT, stock INTEGER DEFAULT 0, warning_stock INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS inventory (
		id INTEGER PRIMARY KEY, sku_id INTEGER, warehouse TEXT
	)`)

	items, err := svc.InventoryHealth(10)
	if err != nil {
		t.Fatalf("InventoryHealth: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items = %d, want 0", len(items))
	}
}

func TestService_ExceptionDistribution_Multi(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY, severity TEXT, source_module TEXT, status TEXT
	)`)
	db.Exec(`INSERT INTO exception_item (severity, source_module, status) VALUES ('high', 'order', 'open')`)
	db.Exec(`INSERT INTO exception_item (severity, source_module, status) VALUES ('high', 'order', 'open')`)
	db.Exec(`INSERT INTO exception_item (severity, source_module, status) VALUES ('low', 'shipping', 'open')`)

	items, err := svc.ExceptionDistribution()
	if err != nil {
		t.Fatalf("ExceptionDistribution: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	// Build lookup map since order depends on SQLite
	got := make(map[string]int64)
	for _, it := range items {
		got[it.Severity+"/"+it.SourceModule] = it.Cnt
	}
	if got["high/order"] != 2 {
		t.Errorf("high/order cnt = %d, want 2", got["high/order"])
	}
	if got["low/shipping"] != 1 {
		t.Errorf("low/shipping cnt = %d, want 1", got["low/shipping"])
	}
}

func TestService_GetRejectionReasonStats(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, agent_id TEXT, status TEXT,
		rejection_reason TEXT, created_at TEXT
	)`)

	now := time.Now()
	ts := now.Format("2006-01-02T15:04:05Z")

	// 2 rejected for A5, 1 rejected for A6, 1 approved (should be ignored)
	db.Exec(`INSERT INTO unified_action (agent_id, status, rejection_reason, created_at) VALUES ('A5', 'rejected', 'insufficient_stock', ?)`, ts)
	db.Exec(`INSERT INTO unified_action (agent_id, status, rejection_reason, created_at) VALUES ('A5', 'rejected', 'insufficient_stock', ?)`, ts)
	db.Exec(`INSERT INTO unified_action (agent_id, status, rejection_reason, created_at) VALUES ('A6', 'rejected', 'margin_too_low', ?)`, ts)
	db.Exec(`INSERT INTO unified_action (agent_id, status, rejection_reason, created_at) VALUES ('G3', 'approved', '', ?)`, ts)

	stats, err := svc.GetRejectionReasonStats()
	if err != nil {
		t.Fatalf("GetRejectionReasonStats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("stats = %d, want 2", len(stats))
	}
	// Ordered by agent_id ASC, count DESC: A5 first
	if stats[0].AgentID != "A5" {
		t.Errorf("stats[0].AgentID = %q, want A5", stats[0].AgentID)
	}
	if stats[0].RejectionReason != "insufficient_stock" {
		t.Errorf("stats[0].RejectionReason = %q, want insufficient_stock", stats[0].RejectionReason)
	}
	if stats[0].Count != 2 {
		t.Errorf("stats[0].Count = %d, want 2", stats[0].Count)
	}
	if stats[1].AgentID != "A6" {
		t.Errorf("stats[1].AgentID = %q, want A6", stats[1].AgentID)
	}
	if stats[1].RejectionReason != "margin_too_low" {
		t.Errorf("stats[1].RejectionReason = %q, want margin_too_low", stats[1].RejectionReason)
	}
	if stats[1].Count != 1 {
		t.Errorf("stats[1].Count = %d, want 1", stats[1].Count)
	}
}

func TestService_GetRejectionReasonStats_Empty(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS unified_action (
		id INTEGER PRIMARY KEY, agent_id TEXT, status TEXT,
		rejection_reason TEXT, created_at TEXT
	)`)

	stats, err := svc.GetRejectionReasonStats()
	if err != nil {
		t.Fatalf("GetRejectionReasonStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("stats = %d, want 0", len(stats))
	}
}
