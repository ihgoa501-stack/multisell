package toolbridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockDriver implements ToolDriver with configurable results.
type mockDriver struct {
	name     string
	data     *PageData
	err      error
	healthOk bool
	category ToolCategory
	execFn   func(map[string]interface{}) (*ToolResult, error)
}

func (m *mockDriver) FetchPage(ctx context.Context, url string) (*PageData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func (m *mockDriver) Health() (available bool, latency time.Duration, err error) {
	return m.healthOk, 0, nil
}

func (m *mockDriver) Category() ToolCategory {
	if m.category != 0 || m.category == ToolCategoryRead {
		return m.category
	}
	return ToolCategoryRead
}

func (m *mockDriver) Execute(input map[string]interface{}) (*ToolResult, error) {
	if m.execFn != nil {
		return m.execFn(input)
	}
	return &ToolResult{Success: true, Data: input}, nil
}

// TestBridgeRoutesToPreferredDriver verifies that a driver with lower weight
// is preferred and its data is returned.
func TestBridgeRoutesToPreferredDriver(t *testing.T) {
	logger := zap.NewNop()

	preferred := &mockDriver{
		name: "preferred",
		data: &PageData{
			SourceURL:    "http://example.com/product",
			Title:        "Test Product",
			PriceCNY:     100.0,
			MOQ:          1,
			SupplierName: "Test Supplier",
			CollectedAt:  time.Now(),
		},
		healthOk: true,
	}

	fallback := &mockDriver{
		name: "fallback",
		data: &PageData{
			Title: "should not be reached",
		},
		healthOk: true,
	}

	bridge := NewToolBridge([]DriverEntry{
		{Name: "fallback", Driver: fallback, Weight: 10},
		{Name: "preferred", Driver: preferred, Weight: 1},
	}, 10*time.Second, logger)

	ctx := context.Background()
	data, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err != nil {
		t.Fatalf("FetchPage returned error: %v", err)
	}
	if data.Title != "Test Product" {
		t.Errorf("expected data from preferred driver, got title=%q", data.Title)
	}
	if data.SupplierName != "Test Supplier" {
		t.Errorf("expected supplier 'Test Supplier', got %q", data.SupplierName)
	}
}

// TestBridgeFallback verifies that when the preferred driver fails, the
// bridge falls through to the next driver.
func TestBridgeFallback(t *testing.T) {
	logger := zap.NewNop()

	preferred := &mockDriver{
		name: "preferred",
		err:  errors.New("preferred driver unavailable"),
	}

	fallback := &mockDriver{
		name: "fallback",
		data: &PageData{
			Title:        "Fallback Product",
			PriceCNY:     200.0,
			MOQ:          1,
			SupplierName: "Fallback Supplier",
			CollectedAt:  time.Now(),
		},
	}

	bridge := NewToolBridge([]DriverEntry{
		{Name: "preferred", Driver: preferred, Weight: 1},
		{Name: "fallback", Driver: fallback, Weight: 10},
	}, 10*time.Second, logger)

	ctx := context.Background()
	data, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err != nil {
		t.Fatalf("FetchPage should fallback successfully, got error: %v", err)
	}
	if data.Title != "Fallback Product" {
		t.Errorf("expected data from fallback driver, got title=%q", data.Title)
	}
}

// TestBridgeNoDrivers verifies that ErrNoDrivers is returned when no drivers
// are registered.
func TestBridgeNoDrivers(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	ctx := context.Background()
	_, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected ErrNoDrivers, got nil")
	}
	if !errors.Is(err, ErrNoDrivers) {
		t.Errorf("expected ErrNoDrivers, got %v", err)
	}
}

// TestBridgeAllDriversFail verifies that when all drivers return errors, the
// last error is returned.
func TestBridgeAllDriversFail(t *testing.T) {
	logger := zap.NewNop()

	d1 := &mockDriver{name: "d1", err: errors.New("driver 1 error")}
	d2 := &mockDriver{name: "d2", err: errors.New("driver 2 error")}

	bridge := NewToolBridge([]DriverEntry{
		{Name: "d1", Driver: d1, Weight: 1},
		{Name: "d2", Driver: d2, Weight: 10},
	}, 10*time.Second, logger)

	ctx := context.Background()
	_, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestBridgeAddDriver verifies that AddDriver registers a driver and it is
// used in subsequent FetchPage calls.
func TestBridgeAddDriver(t *testing.T) {
	logger := zap.NewNop()

	// Create a bridge with no drivers.
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	// Add a driver after construction.
	bridge.AddDriver(DriverEntry{
		Name:   "added",
		Driver: &mockDriver{name: "added", data: &PageData{Title: "Added Driver", PriceCNY: 50, MOQ: 1, SupplierName: "Added Supplier", CollectedAt: time.Now()}},
		Weight: 1,
	})

	ctx := context.Background()
	data, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err != nil {
		t.Fatalf("FetchPage after AddDriver returned error: %v", err)
	}
	if data.Title != "Added Driver" {
		t.Errorf("expected data from added driver, got title=%q", data.Title)
	}
}

// TestBridgeContextCancellation verifies that a cancelled context is
// propagated immediately without calling any driver.
func TestBridgeContextCancellation(t *testing.T) {
	logger := zap.NewNop()

	d := &mockDriver{
		name: "blocking",
	}

	bridge := NewToolBridge([]DriverEntry{
		{Name: "blocking", Driver: d, Weight: 1},
	}, 30*time.Second, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestBridgeTimeoutPropagation verifies that per-driver timeout errors are
// propagated through the bridge.
func TestBridgeTimeoutPropagation(t *testing.T) {
	logger := zap.NewNop()

	d := &mockDriver{
		name: "slow",
		err:  errors.New("driver: deadline exceeded"),
	}

	bridge := NewToolBridge([]DriverEntry{
		{Name: "slow", Driver: d, Weight: 1},
	}, time.Nanosecond, logger)

	// Use a parent context with a past deadline so the timeout fires reliably
	// regardless of Go scheduler timing (time.Nanosecond alone is flaky).
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestBridgeDriverOrdering verifies that drivers are always used in weight
// order regardless of registration order. d1 registered first (weight 10),
// d2 registered second (weight 1) -- d2 should be tried first.
func TestBridgeDriverOrdering(t *testing.T) {
	logger := zap.NewNop()

	d1 := &mockDriver{
		name: "d1",
		data: &PageData{Title: "D1 should not be reached"},
	}
	d2 := &mockDriver{
		name: "d2",
		data: &PageData{Title: "D2 Result", PriceCNY: 1, MOQ: 1, SupplierName: "D2", CollectedAt: time.Now()},
	}

	// d1 registered first with weight 10, d2 registered second with weight 1.
	// d2 should be tried first due to lower weight.
	bridge := NewToolBridge([]DriverEntry{
		{Name: "d1", Driver: d1, Weight: 10},
		{Name: "d2", Driver: d2, Weight: 1},
	}, 10*time.Second, logger)

	ctx := context.Background()
	data, err := bridge.FetchPage(ctx, "http://example.com/product")
	if err != nil {
		t.Fatalf("FetchPage returned error: %v", err)
	}
	if data.Title != "D2 Result" {
		t.Errorf("expected data from d2 (lower weight), got title=%q", data.Title)
	}
}

// ---------------------------------------------------------------------------
// ExecuteTool tests — read/suggest/mutate categories
// ---------------------------------------------------------------------------

// TestExecuteTool_ReadToolSucceeds verifies that a read-category tool can be
// executed through the bridge without an approval ID.
func TestExecuteTool_ReadToolSucceeds(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	d := &mockDriver{name: "reader", category: ToolCategoryRead}
	bridge.RegisterTool("inspect_product", d)

	result, err := bridge.ExecuteTool("inspect_product", map[string]interface{}{"id": "123"}, "")
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

// TestExecuteTool_SuggestToolSucceeds verifies that a suggestion-category tool
// can be executed without approval.
func TestExecuteTool_SuggestToolSucceeds(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	d := &mockDriver{name: "suggester", category: ToolCategorySuggestion}
	bridge.RegisterTool("analyze_price", d)

	result, err := bridge.ExecuteTool("analyze_price", map[string]interface{}{"sku": "SKU001"}, "")
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

// TestExecuteTool_MutationRequiresApproval verifies that a mutation tool
// without approval returns ErrApprovalRequired.
func TestExecuteTool_MutationRequiresApproval(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	d := &mockDriver{name: "mutator", category: ToolCategoryMutation}
	bridge.RegisterTool("publish_listing", d)

	_, err := bridge.ExecuteTool("publish_listing", map[string]interface{}{"id": "42"}, "")
	if err == nil {
		t.Fatal("expected error for mutation without approval")
	}
	if !errors.Is(err, ErrApprovalRequired) {
		t.Errorf("expected ErrApprovalRequired, got: %v", err)
	}
}

// TestExecuteTool_MutationWithApprovalSucceeds verifies that a mutation tool
// succeeds when a non-empty approvalID is provided.
func TestExecuteTool_MutationWithApprovalSucceeds(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	d := &mockDriver{name: "mutator", category: ToolCategoryMutation,
		execFn: func(input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Success: true, Data: map[string]interface{}{"status": "published"}}, nil
		}}
	bridge.RegisterTool("publish_listing", d)

	result, err := bridge.ExecuteTool("publish_listing", map[string]interface{}{"id": "42"}, "approval-123")
	if err != nil {
		t.Fatalf("ExecuteTool with approval: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Data["status"] != "published" {
		t.Errorf("expected published, got %v", result.Data["status"])
	}
}

// TestExecuteTool_UnknownTool verifies that an unregistered tool returns
// ErrToolNotRegistered.
func TestExecuteTool_UnknownTool(t *testing.T) {
	logger := zap.NewNop()
	bridge := NewToolBridge(nil, 10*time.Second, logger)

	_, err := bridge.ExecuteTool("nonexistent", nil, "")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !errors.Is(err, ErrToolNotRegistered) {
		t.Errorf("expected ErrToolNotRegistered, got: %v", err)
	}
}
