package command

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// Core dispatcher tests
// ---------------------------------------------------------------------------

func TestRegisterAndDispatch(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var called bool
	d.Register("test.action", func(_ context.Context, input map[string]interface{}) (*Result, error) {
		called = true
		if input["key"] != "value" {
			t.Errorf("expected input[key]=value, got %v", input["key"])
		}
		return &Result{Success: true, BusinessID: "123"}, nil
	})

	result, err := d.Dispatch(context.Background(), "test.action", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.BusinessID != "123" {
		t.Errorf("expected BusinessID=123, got %s", result.BusinessID)
	}
}

func TestUnregisteredCommandReturnsHandlerNotFound(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	_, err := d.Dispatch(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unregistered command")
	}
	if !IsHandlerNotFound(err) {
		t.Errorf("expected HandlerNotFoundError, got %T: %v", err, err)
	}
}

func TestHandlerErrorReturnsFailedResult(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("fail", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return nil, errors.New("something went wrong")
	})

	result, err := d.Dispatch(context.Background(), "fail", nil)
	if err != nil {
		t.Fatalf("Dispatch should not propagate handler errors, got: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for failed handler")
	}
	if result.ErrorMessage != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", result.ErrorMessage)
	}
}

func TestMultipleHandlersIndependentDispatch(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var mu sync.Mutex
	var calls []string

	d.Register("action1", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		mu.Lock()
		calls = append(calls, "action1")
		mu.Unlock()
		return &Result{Success: true}, nil
	})
	d.Register("action2", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		mu.Lock()
		calls = append(calls, "action2")
		mu.Unlock()
		return &Result{Success: true}, nil
	})

	mustDispatch(t, d, "action1", nil)
	mustDispatch(t, d, "action2", nil)
	mustDispatch(t, d, "action1", nil)

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 3 {
		t.Fatalf("expected 3 total calls, got %d", len(calls))
	}
	if calls[0] != "action1" || calls[1] != "action2" || calls[2] != "action1" {
		t.Errorf("unexpected call order: %v", calls)
	}
}

func TestHandlerOverwrite(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("dup", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return &Result{Success: false}, nil
	})
	d.Register("dup", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return &Result{Success: true}, nil
	})

	result, err := d.Dispatch(context.Background(), "dup", nil)
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}
	if !result.Success {
		t.Error("expected second handler (overwrite) to be called")
	}
}

func TestUnregister(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("temp", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return &Result{Success: true}, nil
	})

	mustDispatch(t, d, "temp", nil)

	d.Unregister("temp")

	_, err := d.Dispatch(context.Background(), "temp", nil)
	if !IsHandlerNotFound(err) {
		t.Errorf("expected HandlerNotFoundError after unregister, got: %v", err)
	}
}

func TestResultWithSnapshot(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("with_snapshot", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return &Result{
			Success:    true,
			BusinessID: "sku-001",
			AfterSnapshot: map[string]interface{}{
				"status": "active",
				"count":  float64(42),
			},
		}, nil
	})

	result, err := d.Dispatch(context.Background(), "with_snapshot", nil)
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}
	if result.BusinessID != "sku-001" {
		t.Errorf("expected BusinessID=sku-001, got %s", result.BusinessID)
	}
	if result.AfterSnapshot["status"] != "active" {
		t.Errorf("expected snapshot status=active, got %v", result.AfterSnapshot["status"])
	}
}

// ---------------------------------------------------------------------------
// RegisteredTypes / HandlerCount
// ---------------------------------------------------------------------------

func TestRegisteredTypesAndHandlerCount(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	if len(d.RegisteredTypes()) != 0 {
		t.Error("expected empty RegisteredTypes initially")
	}
	if d.HandlerCount() != 0 {
		t.Errorf("expected HandlerCount=0, got %d", d.HandlerCount())
	}

	d.Register("a", okHandler)
	d.Register("b", okHandler)
	d.Register("c", okHandler)

	if d.HandlerCount() != 3 {
		t.Errorf("expected HandlerCount=3, got %d", d.HandlerCount())
	}

	types := d.RegisteredTypes()
	if len(types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(types))
	}
	m := make(map[string]bool)
	for _, tt := range types {
		m[tt] = true
	}
	if !m["a"] || !m["b"] || !m["c"] {
		t.Errorf("registered types missing expected entries: %v", types)
	}

	d.Unregister("b")
	if d.HandlerCount() != 2 {
		t.Errorf("expected HandlerCount=2 after unregister, got %d", d.HandlerCount())
	}
}

// ---------------------------------------------------------------------------
// Nil / empty inputs
// ---------------------------------------------------------------------------

func TestDispatchWithNilPayload(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var gotInput map[string]interface{}
	d.Register("nil_payload", func(_ context.Context, input map[string]interface{}) (*Result, error) {
		gotInput = input
		return &Result{Success: true}, nil
	})

	_, err := d.Dispatch(context.Background(), "nil_payload", nil)
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}
	if gotInput != nil {
		t.Errorf("expected nil input, got %v", gotInput)
	}
}

func TestDispatchWithEmptyPayload(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var gotInput map[string]interface{}
	d.Register("empty", func(_ context.Context, input map[string]interface{}) (*Result, error) {
		gotInput = input
		return &Result{Success: true}, nil
	})

	_, err := d.Dispatch(context.Background(), "empty", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}
	if len(gotInput) != 0 {
		t.Errorf("expected empty input, got %v", gotInput)
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestConcurrentRegisterAndDispatch(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var wg sync.WaitGroup
	n := 50

	// Concurrent registrations.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			actionType := string(rune('a' + i%26))
			d.Register(actionType, okHandler)
		}(i)
	}
	wg.Wait()

	// Concurrent dispatches.
	var dispatchWg sync.WaitGroup
	for i := 0; i < n; i++ {
		dispatchWg.Add(1)
		go func(i int) {
			defer dispatchWg.Done()
			d.Register("concurrent", okHandler)
			_, err := d.Dispatch(context.Background(), "concurrent", nil)
			if err != nil {
				t.Errorf("concurrent dispatch error: %v", err)
			}
		}(i)
	}
	dispatchWg.Wait()
}

func TestConcurrentReadsDoNotRace(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("r1", okHandler)
	d.Register("r2", okHandler)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.RegisteredTypes()
			d.HandlerCount()
			mustDispatch(t, d, "r1", nil)
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Built-in handler tests
// ---------------------------------------------------------------------------

func TestBuiltInStockAlertHandler(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)

	handler := StockAlertHandler(db, logger)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":      "SKU001",
		"stock_status":  "low",
		"sellable_days": float64(3),
		"risk_reason":   "Below safety stock",
	})
	if err != nil {
		t.Fatalf("StockAlertHandler returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.BusinessID != "SKU001" {
		t.Errorf("expected BusinessID=SKU001, got %s", result.BusinessID)
	}
	if result.AfterSnapshot["alert_created"] != true {
		t.Error("expected alert_created=true in snapshot")
	}
}

func TestBuiltInInventoryReplenishHandler(t *testing.T) {
	logger := dbtest.NewLogger(t)

	handler := InventoryReplenishHandler(nil, logger)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":                "SKU002",
		"suggested_replenish_qty": float64(100),
		"urgency":                 "high",
		"moq":                     float64(50),
	})
	if err != nil {
		t.Fatalf("InventoryReplenishHandler returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.AfterSnapshot["replenish_qty"] != float64(100) {
		t.Errorf("expected replenish_qty=100, got %v", result.AfterSnapshot["replenish_qty"])
	}
	if result.AfterSnapshot["status"] != "pending_approval" {
		t.Errorf("expected status=pending_approval, got %v", result.AfterSnapshot["status"])
	}
}

func TestBuiltInPriceAdjustHandler(t *testing.T) {
	logger := dbtest.NewLogger(t)

	handler := PriceAdjustHandler(nil, logger, nil)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":       "SKU003",
		"suggested_price": float64(29.99),
		"reason":         "Competitor price drop",
	})
	if err != nil {
		t.Fatalf("PriceAdjustHandler returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.AfterSnapshot["adjusted_price"] != float64(29.99) {
		t.Errorf("expected adjusted_price=29.99, got %v", result.AfterSnapshot["adjusted_price"])
	}
}

func TestBuiltInListingDraftHandler(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS listing_draft (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sku_code TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			bullets TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("failed to create listing_draft table: %v", err)
	}

	handler := ListingDraftHandler(db, logger)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":          "SKU004",
		"optimized_title":   "Premium Widget",
		"optimized_bullets": []string{"Feature A", "Feature B"},
		"keywords":          "widget, premium",
	})
	if err != nil {
		t.Fatalf("ListingDraftHandler returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.AfterSnapshot["draft_created"] != true {
		t.Error("expected draft_created=true")
	}
	if result.AfterSnapshot["title"] != "Premium Widget" {
		t.Errorf("expected title='Premium Widget', got %v", result.AfterSnapshot["title"])
	}
}

func TestBuiltInFlagNonCompliantHandler(t *testing.T) {
	logger := dbtest.NewLogger(t)

	handler := FlagNonCompliantHandler(nil, logger)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":          "SKU005",
		"compliance_issues": []string{"missing_certification", "restricted_material"},
		"risk_level":        "critical",
	})
	if err != nil {
		t.Fatalf("FlagNonCompliantHandler returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.AfterSnapshot["risk"] != "critical" {
		t.Errorf("expected risk=critical, got %v", result.AfterSnapshot["risk"])
	}
	if result.AfterSnapshot["flagged"] != true {
		t.Error("expected flagged=true in snapshot")
	}
}

func TestBuiltInFlagNonCompliantHandlerDefaultRisk(t *testing.T) {
	logger := dbtest.NewLogger(t)

	// Without risk_level — should default to "high".
	handler := FlagNonCompliantHandler(nil, logger)
	result, err := handler(context.Background(), map[string]interface{}{
		"sku_code":          "SKU006",
		"compliance_issues": []string{"issue"},
	})
	if err != nil {
		t.Fatalf("FlagNonCompliantHandler returned error: %v", err)
	}
	if result.AfterSnapshot["risk"] != "high" {
		t.Errorf("expected default risk=high, got %v", result.AfterSnapshot["risk"])
	}
}

func TestBuiltInHandlersWithEmptyPayloadNoPanic(t *testing.T) {
	logger := dbtest.NewLogger(t)

	handlers := []Handler{
		StockAlertHandler(nil, logger),
		InventoryReplenishHandler(nil, logger),
		PriceAdjustHandler(nil, logger, nil),
		ListingDraftHandler(dbtest.NewDB(t), logger),
		FlagNonCompliantHandler(nil, logger),
	}

	for i, h := range handlers {
		result, err := h(context.Background(), map[string]interface{}{})
		if err != nil {
			t.Errorf("handler[%d] returned unexpected error: %v", i, err)
		}
		if result == nil {
			t.Errorf("handler[%d] returned nil result", i)
		}
	}
}

func TestBuiltInStockAlertWithDB(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)

	// Create notification table.
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notification (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("failed to create notification table: %v", err)
	}

	handler := StockAlertHandler(db, logger)
	_, err := handler(context.Background(), map[string]interface{}{
		"sku_code":      "SKU010",
		"stock_status":  "out_of_stock",
		"risk_reason":   "No stock",
		"sellable_days": float64(0),
	})
	if err != nil {
		t.Fatalf("StockAlertHandler returned error: %v", err)
	}

	var count int64
	db.Table("notification").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 notification record, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func okHandler(_ context.Context, _ map[string]interface{}) (*Result, error) {
	return &Result{Success: true}, nil
}

func mustDispatch(t *testing.T, d *Dispatcher, actionType string, payload map[string]interface{}) *Result {
	t.Helper()
	result, err := d.Dispatch(context.Background(), actionType, payload)
	if err != nil {
		t.Fatalf("Dispatch(%q) returned error: %v", actionType, err)
	}
	return result
}
