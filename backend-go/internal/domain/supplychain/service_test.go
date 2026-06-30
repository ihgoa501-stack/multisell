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

// createEventOutboxTable creates the event_outbox table manually in SQLite
// for tests that need to verify event_outbox queries (Service.GetEvents).
func createEventOutboxTable(t testing.TB, db *gorm.DB) {
	t.Helper()
	db.Exec(`
		CREATE TABLE IF NOT EXISTS event_outbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic TEXT,
			source TEXT,
			payload TEXT,
			priority INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME
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

// TestGetEvents_ReturnsEventsFromOutbox verifies that GetEvents queries the
// event_outbox table and returns events whose payload.flow_id matches the
// requested flow ID, ordered by created_at ascending.
func TestGetEvents_ReturnsEventsFromOutbox(t *testing.T) {
	db := dbtest.NewDB(t)
	createTestTable(t, db)
	createEventOutboxTable(t, db)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)
	ctx := context.Background()

	// Create a flow record.
	flow := &SupplyChainFlow{
		SourceType: "sourcing_recommend",
		SourceID:   "42",
		Status:     "pending",
	}
	if err := svc.Create(ctx, flow); err != nil {
		t.Fatalf("Create flow: %v", err)
	}

	// Insert three event_outbox rows: two for this flow, one for another flow.
	// SQLite's json_extract is used to filter by payload.flow_id.
	insertOutbox := func(topic, source, payloadJSON string) {
		if err := db.Exec(
			`INSERT INTO event_outbox (topic, source, payload, priority, status, created_at) VALUES (?, ?, ?, ?, 'delivered', ?)`,
			topic, source, payloadJSON, 0, time.Now().UTC().Format(time.RFC3339Nano),
		).Error; err != nil {
			t.Fatalf("insert event_outbox: %v", err)
		}
	}

	insertOutbox("supplychain.quote_requested", "supplychain",
		`{"flow_id":"`+flow.ID+`","product_id":42}`)
	insertOutbox("supplychain.quote_ready", "A10",
		`{"flow_id":"`+flow.ID+`","channel_name":"CNAIR","total_shipping_fee":236.25}`)
	insertOutbox("supplychain.quote_requested", "supplychain",
		`{"flow_id":"another-flow-id","product_id":99}`)

	resp, err := svc.GetEvents(ctx, flow.ID)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if resp.Flow == nil || resp.Flow.ID != flow.ID {
		t.Fatalf("expected flow with ID %s, got %+v", flow.ID, resp.Flow)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("expected 2 events for flow %s, got %d", flow.ID, len(resp.Events))
	}
	// Events should be ordered ascending by created_at.
	if resp.Events[0].Topic != "supplychain.quote_requested" {
		t.Errorf("expected first event topic supplychain.quote_requested, got %s", resp.Events[0].Topic)
	}
	if resp.Events[1].Topic != "supplychain.quote_ready" {
		t.Errorf("expected second event topic supplychain.quote_ready, got %s", resp.Events[1].Topic)
	}
	// Verify payload was deserialized.
	pid, ok := resp.Events[0].Payload["product_id"].(float64)
	if !ok || int64(pid) != 42 {
		t.Errorf("expected product_id 42 in first event payload, got %v", resp.Events[0].Payload["product_id"])
	}
}

// TestGetEvents_NotFound verifies that GetEvents returns ErrRecordNotFound when
// the flow ID does not exist.
func TestGetEvents_NotFound(t *testing.T) {
	db := dbtest.NewDB(t)
	createTestTable(t, db)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)
	ctx := context.Background()

	_, err := svc.GetEvents(ctx, "nonexistent-id")
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// TestGetEvents_NoOutboxTable verifies graceful degradation when event_outbox
// does not exist — the flow is still returned with an empty events list.
func TestGetEvents_NoOutboxTable(t *testing.T) {
	db := dbtest.NewDB(t)
	createTestTable(t, db)
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)
	ctx := context.Background()

	flow := &SupplyChainFlow{
		SourceType: "sourcing_recommend",
		SourceID:   "1",
		Status:     "pending",
	}
	if err := svc.Create(ctx, flow); err != nil {
		t.Fatalf("Create flow: %v", err)
	}

	resp, err := svc.GetEvents(ctx, flow.ID)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if resp.Flow == nil || resp.Flow.ID != flow.ID {
		t.Errorf("expected flow returned, got %+v", resp.Flow)
	}
	if len(resp.Events) != 0 {
		t.Errorf("expected 0 events when outbox missing, got %d", len(resp.Events))
	}
}
