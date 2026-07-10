package integrations

import (
	"context"
	"testing"
)

func TestNewSandboxAdapter(t *testing.T) {
	a := NewSandboxAdapter()
	if a == nil {
		t.Fatal("NewSandboxAdapter returned nil")
	}
}

func TestSandboxPublish_Success(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &PublishInput{
		ProductID:   1,
		PlatformID:  10,
		AccountID:   100,
		ProductName: "Test Product",
		Description: "A valid test product",
		CategoryID:  42,
		BrandID:     7,
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "TEST-SKU-001"},
		},
		Prices:      map[int64]string{1: "29.99"},
		Inventories: map[int64]int{1: 100},
	}

	result, err := a.Publish(ctx, input)
	if err != nil {
		t.Fatalf("Publish returned unexpected error: %v", err)
	}
	if result.SyncMessage != SandboxStatusCompleted {
		t.Errorf("expected SyncMessage %q, got %q", SandboxStatusCompleted, result.SyncMessage)
	}
	if result.PlatformProductID == "" {
		t.Error("expected non-empty PlatformProductID")
	}
	if result.PlatformSKU != "TEST-SKU-001" {
		t.Errorf("expected PlatformSKU %q, got %q", "TEST-SKU-001", result.PlatformSKU)
	}
	if result.PlatformURL == "" {
		t.Error("expected non-empty PlatformURL")
	}
	if result.PublishedData == nil {
		t.Error("expected non-nil PublishedData")
	}
}

func TestSandboxPublish_BlockedMissingData(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	// Empty ProductName should trigger missing data
	input := &PublishInput{
		ProductID:   1,
		PlatformID:  10,
		AccountID:   100,
		ProductName: "",
		Description: "A product with no name",
		CategoryID:  42,
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "SKU-001"},
		},
		Prices:      map[int64]string{1: "29.99"},
		Inventories: map[int64]int{1: 100},
	}

	result, err := a.Publish(ctx, input)
	if err != nil {
		t.Fatalf("Publish returned unexpected error: %v", err)
	}
	if result.SyncMessage != SandboxStatusBlockedMissingData {
		t.Errorf("expected SyncMessage %q, got %q", SandboxStatusBlockedMissingData, result.SyncMessage)
	}
}

func TestSandboxPublish_BlockedLossMaking(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	// Price of $1.00 yields profit <= 0 after cost/fee/shipping
	// profit = 1.00 - 0.50 - 0.15 - 5.00 = -4.65
	input := &PublishInput{
		ProductID:   1,
		PlatformID:  10,
		AccountID:   100,
		ProductName: "Cheap Product",
		Description: "A product priced too low",
		CategoryID:  42,
		BrandID:     7,
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "CHEAP-SKU"},
		},
		Prices:      map[int64]string{1: "1.00"},
		Inventories: map[int64]int{1: 50},
	}

	result, err := a.Publish(ctx, input)
	if err != nil {
		t.Fatalf("Publish returned unexpected error: %v", err)
	}
	if result.SyncMessage != SandboxStatusBlockedLossMaking {
		t.Errorf("expected SyncMessage %q, got %q", SandboxStatusBlockedLossMaking, result.SyncMessage)
	}
	if result.PublishedData == nil {
		t.Error("expected non-nil PublishedData")
	}
	if explanation, ok := result.PublishedData["explanation"]; ok {
		if exp, ok := explanation.(string); ok && exp == "" {
			t.Error("expected non-empty explanation")
		}
	}
}

func TestSandboxPublish_FailedPlatformValidation_InvalidChar(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	// SKU code with space should trigger platform validation failure
	input := &PublishInput{
		ProductID:   1,
		PlatformID:  10,
		AccountID:   100,
		ProductName: "Invalid SKU Product",
		Description: "SKU has invalid characters",
		CategoryID:  42,
		BrandID:     7,
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "INVALID SKU CODE"},
		},
		Prices:      map[int64]string{1: "49.99"},
		Inventories: map[int64]int{1: 10},
	}

	result, err := a.Publish(ctx, input)
	if err != nil {
		t.Fatalf("Publish returned unexpected error: %v", err)
	}
	if result.SyncMessage != SandboxStatusFailedPlatformValidation {
		t.Errorf("expected SyncMessage %q, got %q", SandboxStatusFailedPlatformValidation, result.SyncMessage)
	}
}

func TestSandboxPublish_FailedPlatformValidation_NoCategory(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	// CategoryID == 0 should trigger platform validation failure
	input := &PublishInput{
		ProductID:   1,
		PlatformID:  10,
		AccountID:   100,
		ProductName: "No Category Product",
		Description: "A product without a category",
		CategoryID:  0,
		BrandID:     7,
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "NO-CAT-SKU"},
		},
		Prices:      map[int64]string{1: "99.99"},
		Inventories: map[int64]int{1: 5},
	}

	result, err := a.Publish(ctx, input)
	if err != nil {
		t.Fatalf("Publish returned unexpected error: %v", err)
	}
	if result.SyncMessage != SandboxStatusFailedPlatformValidation {
		t.Errorf("expected SyncMessage %q, got %q", SandboxStatusFailedPlatformValidation, result.SyncMessage)
	}
}

func TestSandboxPublish_UnchangedDryRun(t *testing.T) {
	// Scenario 5: Dry-run mode is handled by the service layer's checkWriteMode,
	// not by the sandbox adapter. This test documents that the sandbox adapter
	// itself does not inspect or enforce execution mode — the service layer
	// routes to sandbox vs dry-run at a higher level.
	//
	// The sandbox adapter processes all inputs identically regardless of mode.
	// When the service layer receives ExecutionModeDryRun, it returns a mock
	// result without calling any PlatformAdapter implementation.
	t.Log("Dry-run mode is enforced by the service layer checkWriteMode — the sandbox adapter has no dry-run logic")
}

func TestSandboxAdapter_SyncStatus(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &SyncStatusInput{
		ListingID:         1,
		PlatformID:        10,
		PlatformProductID: "sandbox-TEST",
	}
	status, err := a.SyncStatus(ctx, input)
	if err != nil {
		t.Fatalf("SyncStatus returned unexpected error: %v", err)
	}
	if status != "synced" {
		t.Errorf("expected status %q, got %q", "synced", status)
	}
}

func TestSandboxAdapter_ValidateCredentials(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	ok, err := a.ValidateCredentials(ctx, 100)
	if err != nil {
		t.Fatalf("ValidateCredentials returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ValidateCredentials to return true")
	}
}

func TestSandboxAdapter_SyncInventory(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &SyncInventoryInput{
		PlatformID:  10,
		SkuCode:     "SKU-001",
		PlatformSKU: "sandbox-SKU-001",
		Quantity:    100,
	}
	ok, err := a.SyncInventory(ctx, input)
	if err != nil {
		t.Fatalf("SyncInventory returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected SyncInventory to return true")
	}
}

func TestSandboxAdapter_PushTracking(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &PushTrackingInput{
		PlatformID:     10,
		OrderSN:        "ORD-001",
		TrackingNumber: "TRACK-001",
		CarrierCode:    "UPS",
	}
	ok, err := a.PushTracking(ctx, input)
	if err != nil {
		t.Fatalf("PushTracking returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected PushTracking to return true")
	}
}

func TestSandboxAdapter_FetchOrders(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &FetchOrdersInput{}
	orders, err := a.FetchOrders(ctx, input)
	if err != nil {
		t.Fatalf("FetchOrders returned unexpected error: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected empty orders, got %d items", len(orders))
	}
}

func TestSandboxAdapter_FetchSettlements(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &FetchSettlementsInput{}
	settlements, err := a.FetchSettlements(ctx, input)
	if err != nil {
		t.Fatalf("FetchSettlements returned unexpected error: %v", err)
	}
	if len(settlements) != 0 {
		t.Errorf("expected empty settlements, got %d items", len(settlements))
	}
}

func TestSandboxAdapter_FetchReturns(t *testing.T) {
	a := NewSandboxAdapter()
	ctx := context.Background()

	input := &FetchReturnsInput{}
	returns, err := a.FetchReturns(ctx, input)
	if err != nil {
		t.Fatalf("FetchReturns returned unexpected error: %v", err)
	}
	if len(returns) != 0 {
		t.Errorf("expected empty returns, got %d items", len(returns))
	}
}

func TestNewSandboxStateMachine(t *testing.T) {
	sm := NewSandboxStateMachine()
	if sm == nil {
		t.Fatal("NewSandboxStateMachine returned nil")
	}
}

func TestSandboxStateMachine_AllowedTransitions(t *testing.T) {
	sm := NewSandboxStateMachine()
	ctx := context.Background()

	allowed := []struct {
		from, to string
	}{
		{"pending", "simulating"},
		{"simulating", SandboxStatusCompleted},
		{"simulating", SandboxStatusBlockedMissingData},
		{"simulating", SandboxStatusBlockedLossMaking},
		{"simulating", SandboxStatusFailedPlatformValidation},
		{SandboxStatusBlockedMissingData, "pending"},
		{SandboxStatusBlockedLossMaking, "pending"},
		{SandboxStatusFailedPlatformValidation, "pending"},
	}

	for _, tc := range allowed {
		if !sm.CanTransition(tc.from, tc.to) {
			t.Errorf("expected allowed transition %s -> %s", tc.from, tc.to)
		}
		// MustTransition should also pass (no guards registered)
		if err := sm.MustTransition(ctx, tc.from, tc.to, nil); err != nil {
			t.Errorf("MustTransition failed for %s -> %s: %v", tc.from, tc.to, err)
		}
	}
}

func TestSandboxStateMachine_BlockedTransitions(t *testing.T) {
	sm := NewSandboxStateMachine()

	blocked := []struct {
		from, to string
	}{
		{"pending", SandboxStatusCompleted},
		{"pending", SandboxStatusBlockedMissingData},
		{SandboxStatusCompleted, "pending"},
		{SandboxStatusCompleted, SandboxStatusBlockedMissingData},
		{SandboxStatusBlockedMissingData, SandboxStatusCompleted},
		{SandboxStatusBlockedMissingData, "simulating"},
	}

	for _, tc := range blocked {
		if sm.CanTransition(tc.from, tc.to) {
			t.Errorf("expected blocked transition %s -> %s", tc.from, tc.to)
		}
	}
}

func TestSandboxStateMachine_IsTerminal(t *testing.T) {
	sm := NewSandboxStateMachine()

	if !sm.IsTerminal(SandboxStatusCompleted) {
		t.Error("expected completed to be terminal")
	}
	if sm.IsTerminal("pending") {
		t.Error("expected pending to not be terminal")
	}
	if sm.IsTerminal("simulating") {
		t.Error("expected simulating to not be terminal")
	}
	if sm.IsTerminal(SandboxStatusBlockedMissingData) {
		t.Error("expected blocked_missing_data to not be terminal (retryable)")
	}
}
