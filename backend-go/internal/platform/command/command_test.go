package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
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
		"sku_code":        "SKU003",
		"suggested_price": float64(29.99),
		"reason":          "Competitor price drop",
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

	// Create notification table via GORM auto-migrate.
	if err := db.AutoMigrate(&notification.Notification{}); err != nil {
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

func TestDispatchSafe_Production_PriceReviewRejectedByCatalog(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := newDurableTestDispatcher(t, logger, WithCatalog(cat))
	d.Register("price_review", okHandler)

	action := AgentAction{
		ActionType: "price_review",
		Mode:       ModeProduction,
		RiskLevel:  RiskHigh,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err == nil {
		t.Fatal("expected error: price_review requires approval per catalog")
	}
}

func TestDispatchSafe_Production_PriceReviewWithApproval(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := newDurableTestDispatcher(t, logger, WithCatalog(cat))
	d.Register("price_review", okHandler)

	approvalID := int64(42)
	action := AgentAction{
		ActionType:       "price_review",
		AgentID:          "A5",
		Actor:            "system",
		Mode:             ModeProduction,
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		IdempotencyKey:   "price-review:approved",
	}
	mockPolicy := &mockPolicyChecker{approved: true}
	result, err := d.DispatchSafe(context.Background(), action, mockPolicy)
	if err != nil {
		t.Fatalf("price_review with approval should pass, got: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

func TestDispatchSafe_Production_UnknownActionRejected(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := NewDispatcher(logger, WithCatalog(cat))

	action := AgentAction{
		ActionType: "nonexistent_action",
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeProduction,
		RiskLevel:  RiskLow,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err == nil {
		t.Fatal("expected error for unknown action type in production mode")
	}
}

func TestDispatchSafe_Sandbox_UnknownActionAllowed(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := NewDispatcher(logger, WithCatalog(cat))

	var executed bool
	d.Register("custom_action", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		executed = true
		return &Result{Success: true}, nil
	})

	action := AgentAction{
		ActionType: "custom_action",
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeSandbox,
		RiskLevel:  RiskLow,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("sandbox should allow unknown actions, got: %v", err)
	}
	if !executed {
		t.Error("handler was not executed in sandbox mode")
	}
}

func TestDispatchSafe_DryRun_ChecksCatalog(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := NewDispatcher(logger, WithCatalog(cat))

	d.Register("stock_alert", okHandler)
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeDryRun,
		RiskLevel:  RiskLow,
	}
	result, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("dry-run of catalog action should pass, got: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for dry run")
	}

	d.Register("ad_hoc_action", okHandler)
	action2 := AgentAction{
		ActionType: "ad_hoc_action",
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeDryRun,
	}
	_, err2 := d.DispatchSafe(context.Background(), action2, nil)
	if err2 == nil {
		t.Fatal("dry-run should reject actions not in catalog")
	}
}

func TestDispatchSafe_Production_StockAlertNoApprovalNeeded(t *testing.T) {
	logger := dbtest.NewLogger(t)
	cat := actioncatalog.Default()
	d := NewDispatcher(logger, WithCatalog(cat))

	var executed bool
	d.Register("stock_alert", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		executed = true
		return &Result{Success: true}, nil
	})

	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeProduction,
		RiskLevel:  RiskLow,
	}
	result, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("L1 stock_alert should pass in production: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !executed {
		t.Error("handler was not executed")
	}
}

// ---------------------------------------------------------------------------
// P2: Audit recorder — high-risk production actions trigger audit.
// ---------------------------------------------------------------------------

func TestDispatchSafe_AuditRecorder_CalledForHighRiskProduction(t *testing.T) {
	logger := dbtest.NewLogger(t)
	var recorded bool
	var recordedAction AgentAction
	var recordedResult *Result

	auditRecorder := func(_ context.Context, action AgentAction, result *Result) {
		recorded = true
		recordedAction = action
		recordedResult = result
	}

	d := newDurableTestDispatcher(t, logger, WithAuditRecorder(auditRecorder))
	approvalID := int64(42)
	d.Register("price_update", okHandler)

	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		Mode:             ModeProduction,
		Input:            map[string]interface{}{"price": 29.99},
		IdempotencyKey:   "price-update:audit-success",
	}
	result, err := d.DispatchSafe(context.Background(), action, &mockPolicyChecker{approved: true})
	if err != nil {
		t.Fatalf("DispatchSafe with valid approval should succeed, got: %v", err)
	}
	if !recorded {
		t.Error("audit recorder should have been called for high-risk production action")
	}
	if recordedAction.ActionType != "price_update" {
		t.Errorf("expected action_type price_update in audit, got %s", recordedAction.ActionType)
	}
	if recordedResult != result {
		t.Error("recorded result should match the dispatch result")
	}
}

func TestDispatchSafe_AuditRecorder_NotCalledForLowRiskProduction(t *testing.T) {
	logger := dbtest.NewLogger(t)
	var recorded bool
	auditRecorder := func(_ context.Context, _ AgentAction, _ *Result) {
		recorded = true
	}

	d := newDurableTestDispatcher(t, logger, WithAuditRecorder(auditRecorder))
	d.Register("stock_alert", okHandler)

	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeProduction,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("low-risk dispatch should succeed, got: %v", err)
	}
	if recorded {
		t.Error("audit recorder should NOT be called for low-risk action")
	}
}

func TestDispatchSafe_AuditRecorder_NotCalledForDryRun(t *testing.T) {
	logger := dbtest.NewLogger(t)
	var recorded bool
	auditRecorder := func(_ context.Context, _ AgentAction, _ *Result) {
		recorded = true
	}

	d := NewDispatcher(logger, WithAuditRecorder(auditRecorder))
	d.Register("price_update", okHandler)

	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		Mode:             ModeDryRun,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("dry-run should succeed, got: %v", err)
	}
	if recorded {
		t.Error("audit recorder should NOT be called for dry-run")
	}
}

func TestDispatchSafe_AuditRecorder_NotCalledForFailedExecution(t *testing.T) {
	logger := dbtest.NewLogger(t)
	var recorded bool
	auditRecorder := func(_ context.Context, _ AgentAction, _ *Result) {
		recorded = true
	}

	d := newDurableTestDispatcher(t, logger, WithAuditRecorder(auditRecorder))
	approvalID := int64(42)
	d.Register("price_update", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		return &Result{Success: false, ErrorMessage: "handler failed"}, nil
	})

	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		Mode:             ModeProduction,
		IdempotencyKey:   "price-update:audit-failure",
	}
	_, err := d.DispatchSafe(context.Background(), action, &mockPolicyChecker{approved: true})
	if err != nil {
		t.Fatalf("DispatchSafe should return result even on handler failure, got: %v", err)
	}
	if recorded {
		t.Error("audit recorder should NOT be called for failed high-risk execution")
	}
}

// ---------------------------------------------------------------------------
// P2: Production PolicyChecker with approval — end-to-end validation.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// P4: RateLimiter — sliding window per (agent, action_type)
// ---------------------------------------------------------------------------

func TestRateLimiter_Allow_Basic(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	if !rl.Allow("A5", "stock_alert") {
		t.Error("expected first call to be allowed")
	}
	if !rl.Allow("A5", "stock_alert") {
		t.Error("expected second call to be allowed")
	}
	if rl.Allow("A5", "stock_alert") {
		t.Error("expected third call to be rate limited")
	}
}

func TestRateLimiter_AllowsDifferentTypes(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	rl.Allow("A5", "stock_alert") // 1
	rl.Allow("A5", "stock_alert") // 2 (max)
	// Different action type should still be allowed
	if !rl.Allow("A5", "price_update") {
		t.Error("different action type should not share rate limit")
	}
	// Different agent should be allowed
	if !rl.Allow("A6", "stock_alert") {
		t.Error("different agent should not share rate limit")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.Allow("A5", "stock_alert")
	if rl.Allow("A5", "stock_alert") {
		t.Error("expected rate limited after hitting limit")
	}
	rl.Reset("A5")
	if !rl.Allow("A5", "stock_alert") {
		t.Error("expected allowed after reset")
	}
}

func TestDispatchSafe_Production_HighRisk_RequiresValidApproval(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := newDurableTestDispatcher(t, logger)
	d.Register("price_update", okHandler)

	// An approval exists but was rejected by policy — action must be blocked.
	// The mock simulates what ApprovalPolicyChecker does.
	approvalID := int64(42)
	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		Mode:             ModeProduction,
		IdempotencyKey:   "price-update:policy-rejected",
	}
	// Policy says rejected — should fail.
	mockPolicy := &mockPolicyChecker{approved: false}
	_, err := d.DispatchSafe(context.Background(), action, mockPolicy)
	if err != ErrApprovalRequired {
		t.Errorf("expected ErrApprovalRequired when policy rejects, got %v", err)
	}
}
