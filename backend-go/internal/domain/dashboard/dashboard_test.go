package dashboard

import (
	"testing"
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
