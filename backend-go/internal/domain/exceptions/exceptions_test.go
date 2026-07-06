package exceptions

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	e := &ExceptionItem{
		SourceModule: "order",
		Severity:     "high",
		Title:        "订单支付异常",
		Description:  "订单 #12345 支付超时",
		Status:       "open",
	}
	err := svc.Create(e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("ID should be set")
	}

	// GetByID
	got, err := svc.GetByID(e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "订单支付异常" {
		t.Fatalf("Title = %s", got.Title)
	}

	// List
	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter by severity
	items, total, err = svc.List(ListFilter{Severity: "high"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("filtered total = %d", total)
	}

	// Assign
	assigned, err := svc.Assign(e.ID, "user_zhang")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assigned.AssignedTo != "user_zhang" {
		t.Fatalf("AssignedTo = %s", assigned.AssignedTo)
	}
	if assigned.Status != "assigned" {
		t.Fatalf("Status = %s", assigned.Status)
	}

	// Resolve
	resolved, err := svc.Resolve(e.ID, "user_li", "已处理")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %s", resolved.Status)
	}
	if resolved.ResolvedBy != "user_li" {
		t.Fatalf("ResolvedBy = %s", resolved.ResolvedBy)
	}

	// Delete
	if err := svc.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(e.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_AutoDetect(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create minimal related tables for auto-detection queries.
	db.Exec(`CREATE TABLE order_profit_record (id INTEGER PRIMARY KEY, order_id INTEGER UNIQUE, profit REAL)`)
	db.Exec(`CREATE TABLE inventory (id INTEGER PRIMARY KEY, sku_id INTEGER, quantity INTEGER, locked_quantity INTEGER DEFAULT 0, safety_stock INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE fulfillment_tracking (id INTEGER PRIMARY KEY, order_id INTEGER, is_lost INTEGER DEFAULT 0, is_returned INTEGER DEFAULT 0, is_damaged INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE sales_order (id INTEGER PRIMARY KEY, platform_id INTEGER, shipping_fee REAL DEFAULT 0, platform_fee REAL DEFAULT 0, pay_amount REAL DEFAULT 0, shipped_at TIMESTAMP, delivered_at TIMESTAMP)`)
	db.Exec(`CREATE TABLE platform_fee_rule (id INTEGER PRIMARY KEY, platform_id INTEGER, fee_type TEXT, fee_rate_pct REAL, status TEXT DEFAULT 'active', priority INTEGER DEFAULT 0)`)

	ctx := context.Background()

	// Empty data -- should not error and return no items.
	items, err := svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect on empty data: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items on empty data, got %d", len(items))
	}

	// Seed a loss order.
	db.Exec(`INSERT INTO order_profit_record (id, order_id, profit) VALUES (1, 101, -50.0)`)

	items, err = svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect: %v", err)
	}

	found := false
	for _, item := range items {
		if item.SourceType == TypeLossOrder && item.SourceID != nil && *item.SourceID == 101 {
			found = true
			if item.Severity != "high" {
				t.Errorf("loss order severity = %s, want high", item.Severity)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected loss order exception for order 101")
	}

	// Duplicate avoidance: second detection should not create another exception for order 101.
	items, err = svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect (2nd call): %v", err)
	}
	dupCount := 0
	for _, item := range items {
		if item.SourceType == TypeLossOrder && item.SourceID != nil && *item.SourceID == 101 {
			dupCount++
		}
	}
	if dupCount > 0 {
		t.Fatal("AutoDetect created duplicate exception for order 101")
	}
}
