package supplychain

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"gorm.io/gorm"
)

// createTestTable creates the supply_chain_flow table manually in SQLite
// (the model uses PostgreSQL-specific gen_random_uuid() which SQLite cannot parse).
func createTestTable(t testing.TB, db *gorm.DB) {
	t.Helper()
	db.Exec(`
		CREATE TABLE IF NOT EXISTS supply_chain_flow (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			source_type TEXT,
			source_id TEXT,
			status TEXT DEFAULT 'pending',
			context TEXT,
			carrier_summary TEXT,
			financial_summary TEXT,
			error_log TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
}

// flywheelSpy subscribes to supplychain.flywheel and records the first event.
type flywheelSpy struct {
	event    *eventbus.Event
	received chan struct{}
}

func newFlywheelSpy(bus *eventbus.Bus) *flywheelSpy {
	spy := &flywheelSpy{received: make(chan struct{}, 1)}
	bus.Subscribe("supplychain.flywheel", func(ctx context.Context, evt eventbus.Event) error {
		spy.event = &evt
		select {
		case spy.received <- struct{}{}:
		default:
		}
		return nil
	})
	return spy
}

func (s *flywheelSpy) wait(t *testing.T) *eventbus.Event {
	t.Helper()
	select {
	case <-s.received:
		return s.event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for supplychain.flywheel event")
		return nil
	}
}

func TestUpdate_ToCompleted_PublishesFlywheelEvent(t *testing.T) {
	db := dbtest.NewDB(t) // no models — SQLite can't parse gen_random_uuid()
	createTestTable(t, db)
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	spy := newFlywheelSpy(bus)
	svc := NewService(db, logger, bus)

	// Create a flow in "shipped" status.
	flow := &SupplyChainFlow{
		SourceType: "purchase_order",
		SourceID:   "PO-001",
		Status:     "shipped",
	}
	if err := svc.Create(ctx, flow); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update to "completed" with carrier and financial summaries.
	carrierSummary := json.RawMessage(`{
		"channel": "CNAIR",
		"provider": "CNExpress",
		"category": "Electronics",
		"delivery_days": 9,
		"is_lost": false
	}`)
	financialSummary := json.RawMessage(`{
		"actual_cost": 236.25,
		"currency": "CNY"
	}`)

	req := UpdateFlowRequest{
		Status:           "completed",
		CarrierSummary:   &carrierSummary,
		FinancialSummary: &financialSummary,
	}
	if err := svc.Update(ctx, flow.ID, req); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	evt := spy.wait(t)
	if evt == nil {
		t.Fatal("expected flywheel event")
	}
	if evt.Topic != "supplychain.flywheel" {
		t.Errorf("expected topic 'supplychain.flywheel', got '%s'", evt.Topic)
	}

	// Verify payload fields.
	channel, _ := evt.Payload["channel_name"].(string)
	if channel != "CNAIR" {
		t.Errorf("expected channel_name 'CNAIR', got '%s'", channel)
	}
	provider, _ := evt.Payload["provider_name"].(string)
	if provider != "CNExpress" {
		t.Errorf("expected provider_name 'CNExpress', got '%s'", provider)
	}
	cost, _ := evt.Payload["actual_cost"].(float64)
	if cost != 236.25 {
		t.Errorf("expected actual_cost 236.25, got %f", cost)
	}
}

func TestUpdate_ToNonCompleted_DoesNotPublish(t *testing.T) {
	db := dbtest.NewDB(t)
	createTestTable(t, db)
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	published := make(chan struct{}, 1)
	bus.Subscribe("supplychain.flywheel", func(ctx context.Context, evt eventbus.Event) error {
		published <- struct{}{}
		return nil
	})

	svc := NewService(db, logger, bus)

	flow := &SupplyChainFlow{
		SourceType: "purchase_order",
		SourceID:   "PO-002",
		Status:     "pending",
	}
	if err := svc.Create(ctx, flow); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update to "shipped" — not "completed", so no flywheel event.
	req := UpdateFlowRequest{Status: "shipped"}
	if err := svc.Update(ctx, flow.ID, req); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	select {
	case <-published:
		t.Fatal("flywheel event should NOT have been published for non-completed status")
	case <-time.After(200 * time.Millisecond):
		// Expected — no event.
	}
}

func TestUpdate_WithoutBus_DoesNotPanic(t *testing.T) {
	db := dbtest.NewDB(t)
	createTestTable(t, db)
	logger := dbtest.NewLogger(t)
	// Create service without bus (nil).
	svc := NewService(db, logger, nil)

	ctx := context.Background()

	flow := &SupplyChainFlow{
		SourceType: "purchase_order",
		SourceID:   "PO-003",
		Status:     "pending",
	}
	if err := svc.Create(ctx, flow); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update to completed without a bus should not panic.
	req := UpdateFlowRequest{Status: "completed"}
	if err := svc.Update(ctx, flow.ID, req); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestBuildFlywheelEvent_FullData(t *testing.T) {
	carrierSummary := json.RawMessage(`{
		"channel": "CNAIR",
		"provider": "CNExpress",
		"category": "Electronics",
		"delivery_days": 9,
		"is_lost": false
	}`)
	financialSummary := json.RawMessage(`{
		"actual_cost": 236.25,
		"currency": "CNY"
	}`)

	req := UpdateFlowRequest{
		Status:           "completed",
		CarrierSummary:   &carrierSummary,
		FinancialSummary: &financialSummary,
	}

	evt := buildFlywheelEvent("flow-123", req)

	if evt.FlowID != "flow-123" {
		t.Errorf("expected FlowID 'flow-123', got '%s'", evt.FlowID)
	}
	if evt.ChannelName != "CNAIR" {
		t.Errorf("expected ChannelName 'CNAIR', got '%s'", evt.ChannelName)
	}
	if evt.ProviderName != "CNExpress" {
		t.Errorf("expected ProviderName 'CNExpress', got '%s'", evt.ProviderName)
	}
	if evt.CategoryName != "Electronics" {
		t.Errorf("expected CategoryName 'Electronics', got '%s'", evt.CategoryName)
	}
	if evt.DeliveryDays != 9 {
		t.Errorf("expected DeliveryDays 9, got %d", evt.DeliveryDays)
	}
	if evt.IsLost {
		t.Errorf("expected IsLost false")
	}
	if evt.ActualCost != 236.25 {
		t.Errorf("expected ActualCost 236.25, got %f", evt.ActualCost)
	}
	if evt.Currency != "CNY" {
		t.Errorf("expected Currency 'CNY', got '%s'", evt.Currency)
	}
}

func TestBuildFlywheelEvent_LostPackage(t *testing.T) {
	carrierSummary := json.RawMessage(`{
		"channel": "CNSEA",
		"provider": "CNExpress",
		"delivery_days": 0,
		"is_lost": true
	}`)

	req := UpdateFlowRequest{
		Status:         "completed",
		CarrierSummary: &carrierSummary,
	}

	evt := buildFlywheelEvent("flow-lost-1", req)

	if !evt.IsLost {
		t.Errorf("expected IsLost true for lost package")
	}
	if evt.ChannelName != "CNSEA" {
		t.Errorf("expected ChannelName 'CNSEA', got '%s'", evt.ChannelName)
	}
}

func TestBuildFlywheelEvent_NoSummaries(t *testing.T) {
	req := UpdateFlowRequest{
		Status: "completed",
	}

	evt := buildFlywheelEvent("flow-empty", req)

	if evt.FlowID != "flow-empty" {
		t.Errorf("expected FlowID 'flow-empty', got '%s'", evt.FlowID)
	}
	if evt.ChannelName != "" {
		t.Errorf("expected empty ChannelName, got '%s'", evt.ChannelName)
	}
	if evt.ActualCost != 0 {
		t.Errorf("expected ActualCost 0, got %f", evt.ActualCost)
	}
}
