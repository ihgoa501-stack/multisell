package supplychain

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var flowDBCounter atomic.Int64

// newTestFlowDB creates an isolated in-memory SQLite database with a
// supply_chain_flow table that is SQLite-compatible (avoids the
// PostgreSQL-specific uuid/default gen_random_uuid() tag in the model).
func newTestFlowDB(t testing.TB) *gorm.DB {
	t.Helper()
	n := flowDBCounter.Add(1)
	dsn := fmt.Sprintf("file:flow_test_%d?mode=memory&cache=shared", n)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("newTestFlowDB: failed to open in-memory SQLite: %v", err)
	}

	// Create supply_chain_flow table with SQLite-compatible DDL.
	if err := db.Exec(`
		CREATE TABLE supply_chain_flow (
			id TEXT PRIMARY KEY,
			source_type VARCHAR(50),
			source_id VARCHAR(100),
			status VARCHAR(20) DEFAULT 'pending',
			context JSONB,
			carrier_summary JSONB,
			financial_summary JSONB,
			error_log JSONB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("newTestFlowDB: failed to create supply_chain_flow table: %v", err)
	}

	// Create sku_return_stats table for ReturnRateTracker.
	if err := db.Exec(`
		CREATE TABLE sku_return_stats (
			sku_id INTEGER PRIMARY KEY,
			total_orders INTEGER DEFAULT 0,
			total_returns INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("newTestFlowDB: failed to create sku_return_stats table: %v", err)
	}

	return db
}

func TestOrchestrator_HandleRecommendEvent(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	// Spy on the expected output topic.
	received := make(chan eventbus.Event, 1)
	bus.Subscribe("supplychain.quote_requested", func(ctx context.Context, evt eventbus.Event) error {
		received <- evt
		return nil
	})

	orch := NewOrchestrator(bus, nil, nil, nil)

	payload := map[string]interface{}{
		"id":         int64(42),
		"source_url": "https://detail.1688.com/offer/123.html",
		"title":      "Test Product",
		"score":      8,
	}

	evt := eventbus.Event{
		Topic:   "sourcing.recommend",
		Source:  "A8",
		Payload: payload,
	}

	if err := orch.HandleRecommendEvent(ctx, evt); err != nil {
		t.Fatalf("HandleRecommendEvent returned error: %v", err)
	}

	select {
	case published := <-received:
		if published.Topic != "supplychain.quote_requested" {
			t.Errorf("expected topic supplychain.quote_requested, got %s", published.Topic)
		}
		if published.Source != "supplychain" {
			t.Errorf("expected source supplychain, got %s", published.Source)
		}
		// JSON serialization converts int64 to float64 in map[string]interface{}.
		pid, ok := published.Payload["product_id"].(float64)
		if !ok {
			t.Errorf("expected product_id in payload, got %v (type %T)",
				published.Payload["product_id"], published.Payload["product_id"])
		} else if int64(pid) != 42 {
			t.Errorf("expected product_id 42, got %v", pid)
		}
		src, ok := published.Payload["source_url"].(string)
		if !ok || src != "https://detail.1688.com/offer/123.html" {
			t.Errorf("expected source_url in payload, got %v", published.Payload["source_url"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for supplychain.quote_requested event")
	}
}

func TestOrchestrator_HandleTick(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	orch := NewOrchestrator(bus, nil, nil, nil)

	ctx := context.Background()
	evt := eventbus.Event{Topic: "scheduler.tick.orch"}

	if err := orch.HandleTick(ctx, evt); err != nil {
		t.Errorf("HandleTick should be no-op, got error: %v", err)
	}
}

func TestOrchestrator_HandleAftersaleReturn(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	orch := NewOrchestrator(bus, db, logger, rt)

	ctx := context.Background()

	payload := map[string]interface{}{
		"aftersale_id": float64(101),
		"order_id":     float64(5001),
		"sku_id":       float64(42),
		"quantity":     float64(2),
		"reason":       "defective item",
	}

	evt := eventbus.Event{
		Topic:   "supplychain.aftersale.returned",
		Source:  "aftersales",
		Payload: payload,
	}

	if err := orch.HandleAftersaleReturn(ctx, evt); err != nil {
		t.Fatalf("HandleAftersaleReturn returned error: %v", err)
	}

	// Verify the return was tracked.
	rate, err := rt.GetReturnRate(42)
	if err != nil {
		t.Fatalf("GetReturnRate error: %v", err)
	}
	if rate != 200.0 {
		t.Errorf("expected return rate 200.0 for SKU 42 (qty=2), got %v", rate)
	}

	// Verify flow was created in the DB.
	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "aftersale_return").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 reverse flow, got %d", len(flows))
	}
	if flows[0].SourceID != "101" {
		t.Errorf("expected source_id 101, got %s", flows[0].SourceID)
	}
	if flows[0].Status != "inspecting" {
		t.Errorf("expected flow status inspecting, got %s", flows[0].Status)
	}
}

func TestOrchestrator_HandleAftersaleReturn_Int64Payload(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	orch := NewOrchestrator(bus, db, logger, rt)

	ctx := context.Background()

	// Simulate an in-process payload (int64 instead of float64).
	payload := map[string]interface{}{
		"aftersale_id": int64(202),
		"order_id":     int64(6002),
		"sku_id":       int64(99),
		"quantity":     int64(1),
		"reason":       "wrong size",
	}

	evt := eventbus.Event{
		Topic:   "supplychain.aftersale.returned",
		Source:  "aftersales",
		Payload: payload,
	}

	if err := orch.HandleAftersaleReturn(ctx, evt); err != nil {
		t.Fatalf("HandleAftersaleReturn returned error: %v", err)
	}

	rate, err := rt.GetReturnRate(99)
	if err != nil {
		t.Fatalf("GetReturnRate error: %v", err)
	}
	if rate != 100.0 {
		t.Errorf("expected return rate 100.0 for SKU 99 (qty=1), got %v", rate)
	}

	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "aftersale_return").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 reverse flow, got %d", len(flows))
	}
	if flows[0].SourceID != "202" {
		t.Errorf("expected source_id 202, got %s", flows[0].SourceID)
	}
	if flows[0].Status != "inspecting" {
		t.Errorf("expected flow status inspecting, got %s", flows[0].Status)
	}
}

func TestOrchestrator_HandleAftersaleReturn_MultipleReturns(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	orch := NewOrchestrator(bus, db, logger, rt)

	ctx := context.Background()

	// First return for SKU 50.
	evt1 := eventbus.Event{
		Topic: "supplychain.aftersale.returned",
		Payload: map[string]interface{}{
			"aftersale_id": float64(1),
			"order_id":     float64(100),
			"sku_id":       float64(50),
			"quantity":     float64(1),
			"reason":       "defect",
		},
	}
	if err := orch.HandleAftersaleReturn(ctx, evt1); err != nil {
		t.Fatalf("first HandleAftersaleReturn: %v", err)
	}

	// Second return for the same SKU.
	evt2 := eventbus.Event{
		Topic: "supplychain.aftersale.returned",
		Payload: map[string]interface{}{
			"aftersale_id": float64(2),
			"order_id":     float64(101),
			"sku_id":       float64(50),
			"quantity":     float64(3),
			"reason":       "damaged",
		},
	}
	if err := orch.HandleAftersaleReturn(ctx, evt2); err != nil {
		t.Fatalf("second HandleAftersaleReturn: %v", err)
	}

	// Verify the return count: 2 returns for SKU 50.
	rate, err := rt.GetReturnRate(50)
	if err != nil {
		t.Fatalf("GetReturnRate error: %v", err)
	}
	if rate != 200.0 {
		t.Errorf("expected return rate 200.0 for SKU 50 (1->4 returns), got %v", rate)
	}

	// Verify two flows were created.
	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "aftersale_return").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 reverse flows, got %d", len(flows))
	}
}

func TestOrchestrator_HandleStockCritical(t *testing.T) {
	log := dbtest.NewLogger(t)
	bus := eventbus.New(log, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	// Spy on sourcing.rescan events.
	rescanReceived := make(chan eventbus.Event, 1)
	bus.Subscribe("sourcing.rescan", func(ctx context.Context, evt eventbus.Event) error {
		rescanReceived <- evt
		return nil
	})

	db := newTestFlowDB(t)
	orch := NewOrchestrator(bus, db, log, nil)

	// Publish a Float64-format payload simulating JSON-deserialized event.
	payload := map[string]interface{}{
		"sku_id":        float64(101),
		"current_stock": float64(50),
		"safety_stock":  float64(100),
		"sellable_days": float64(3.5),
	}

	evt := eventbus.Event{
		Topic:   "supplychain.stock.critical",
		Source:  "A5",
		Payload: payload,
	}

	if err := orch.HandleStockCritical(ctx, evt); err != nil {
		t.Fatalf("HandleStockCritical returned error: %v", err)
	}

	// Verify flow was created and status is "sourcing_requested".
	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "stock_critical").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].SourceID != "101" {
		t.Errorf("expected source_id 101, got %s", flows[0].SourceID)
	}
	if flows[0].Status != "sourcing_requested" {
		t.Errorf("expected flow status sourcing_requested, got %s", flows[0].Status)
	}

	// Verify sourcing.rescan event was published.
	select {
	case published := <-rescanReceived:
		if published.Topic != "sourcing.rescan" {
			t.Errorf("expected topic sourcing.rescan, got %s", published.Topic)
		}
		if published.Source != "supplychain" {
			t.Errorf("expected source supplychain, got %s", published.Source)
		}
		pid := getInt64(published.Payload, "sku_id")
		if pid != 101 {
			t.Errorf("expected sku_id 101 in rescan payload, got %v", pid)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sourcing.rescan event")
	}
}

func TestOrchestrator_HandleStockCritical_Int64Payload(t *testing.T) {
	log := dbtest.NewLogger(t)
	bus := eventbus.New(log, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	rescanReceived := make(chan eventbus.Event, 1)
	bus.Subscribe("sourcing.rescan", func(ctx context.Context, evt eventbus.Event) error {
		rescanReceived <- evt
		return nil
	})

	db := newTestFlowDB(t)
	orch := NewOrchestrator(bus, db, log, nil)

	// Simulate an in-process payload (int64 instead of float64).
	payload := map[string]interface{}{
		"sku_id":        int64(202),
		"current_stock": int64(30),
		"safety_stock":  int64(80),
		"sellable_days": int64(2),
	}

	evt := eventbus.Event{
		Topic:   "supplychain.stock.critical",
		Source:  "A5",
		Payload: payload,
	}

	if err := orch.HandleStockCritical(ctx, evt); err != nil {
		t.Fatalf("HandleStockCritical returned error: %v", err)
	}

	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "stock_critical").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].SourceID != "202" {
		t.Errorf("expected source_id 202, got %s", flows[0].SourceID)
	}
	if flows[0].Status != "sourcing_requested" {
		t.Errorf("expected flow status sourcing_requested, got %s", flows[0].Status)
	}

	select {
	case published := <-rescanReceived:
		pid := getInt64(published.Payload, "sku_id")
		if pid != 202 {
			t.Errorf("expected sku_id 202 in rescan payload, got %v", pid)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sourcing.rescan event")
	}
}

func TestOrchestrator_HandleStockCritical_MissingSkuID(t *testing.T) {
	log := dbtest.NewLogger(t)
	bus := eventbus.New(log)
	db := newTestFlowDB(t)
	orch := NewOrchestrator(bus, db, log, nil)

	ctx := context.Background()

	// Payload with zero/invalid sku_id.
	payload := map[string]interface{}{
		"sku_id":        float64(0),
		"current_stock": float64(50),
		"safety_stock":  float64(100),
	}

	evt := eventbus.Event{
		Topic:   "supplychain.stock.critical",
		Source:  "A5",
		Payload: payload,
	}

	if err := orch.HandleStockCritical(ctx, evt); err != nil {
		t.Fatalf("HandleStockCritical with missing sku_id returned error: %v", err)
	}

	// Verify no flow was created.
	var count int64
	db.Model(&SupplyChainFlow{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 flows for missing sku_id, got %d", count)
	}
}

// fakeActionCreator is a test double for ActionCreator that records every
// CreateAction call so tests can assert the orchestrator emits approval-gated
// actions for high-risk operations (refund/resend, replenishment).
type fakeActionCreator struct {
	mu     sync.Mutex
	calls  []*ActionInput
	ids    int64
	failOn int // 1-based call index that should return an error; 0 = none
}

func (f *fakeActionCreator) CreateAction(in *ActionInput) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ids++
	idx := len(f.calls) + 1
	f.calls = append(f.calls, in)
	if f.failOn == idx {
		return 0, fmt.Errorf("simulated action create failure")
	}
	return f.ids, nil
}

func (f *fakeActionCreator) Calls() []*ActionInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*ActionInput, len(f.calls))
	copy(out, f.calls)
	return out
}

// TestOrchestrator_HandleAftersaleReturn_CreatesRefundAction verifies that
// the orchestrator creates an approval-gated refund/resend UnifiedAction when
// processing a supplychain.aftersale.returned event. The action must require
// approval (no auto-execution of refunds) and be high-risk.
func TestOrchestrator_HandleAftersaleReturn_CreatesRefundAction(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	ac := &fakeActionCreator{}
	orch := NewOrchestrator(bus, db, logger, rt).WithActionCreator(ac)

	ctx := context.Background()

	payload := map[string]interface{}{
		"aftersale_id": float64(301),
		"order_id":     float64(8001),
		"sku_id":       float64(55),
		"quantity":     float64(2),
		"reason":       "质量问题",
	}
	evt := eventbus.Event{Topic: "supplychain.aftersale.returned", Source: "aftersales", Payload: payload}

	if err := orch.HandleAftersaleReturn(ctx, evt); err != nil {
		t.Fatalf("HandleAftersaleReturn returned error: %v", err)
	}

	calls := ac.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateAction call, got %d", len(calls))
	}
	in := calls[0]
	if in.ActionType != "refund_or_resend" {
		t.Errorf("expected action_type refund_or_resend, got %s", in.ActionType)
	}
	if in.RiskLevel != "high" {
		t.Errorf("expected risk_level high, got %s", in.RiskLevel)
	}
	if !in.RequiresApproval {
		t.Error("expected RequiresApproval=true (no auto-refund)")
	}
	if in.SourceTable != "supply_chain_flow" {
		t.Errorf("expected source_table supply_chain_flow, got %s", in.SourceTable)
	}
	if in.AgentID != "A6" {
		t.Errorf("expected agent_id A6 (aftersales mgmt), got %s", in.AgentID)
	}
	// The flow ID is generated; verify it's non-empty and matches the persisted flow.
	if in.SourceID == "" {
		t.Error("expected non-empty source_id (flow ID)")
	}
	var flows []SupplyChainFlow
	if err := db.Where("source_type = ?", "aftersale_return").Find(&flows).Error; err != nil {
		t.Fatalf("query flows: %v", err)
	}
	if len(flows) != 1 || flows[0].ID != in.SourceID {
		t.Errorf("action source_id %s should match persisted flow id %s", in.SourceID, flows[0].ID)
	}
}

// TestOrchestrator_HandleAftersaleReturn_NoActionCreator verifies the
// orchestrator still creates the flow when no ActionCreator is wired —
// action creation is best-effort, not blocking.
func TestOrchestrator_HandleAftersaleReturn_NoActionCreator(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	// No WithActionCreator — orchestrator should still work.
	orch := NewOrchestrator(bus, db, logger, rt)

	ctx := context.Background()
	payload := map[string]interface{}{
		"aftersale_id": float64(302),
		"order_id":     float64(8002),
		"sku_id":       float64(56),
		"quantity":     float64(1),
		"reason":       "test",
	}
	evt := eventbus.Event{Topic: "supplychain.aftersale.returned", Payload: payload}

	if err := orch.HandleAftersaleReturn(ctx, evt); err != nil {
		t.Fatalf("HandleAftersaleReturn returned error: %v", err)
	}

	// Flow should still be created.
	var count int64
	db.Model(&SupplyChainFlow{}).Where("source_type = ?", "aftersale_return").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 flow even without ActionCreator, got %d", count)
	}
}

// TestOrchestrator_HandleAftersaleReturn_ActionCreateFails verifies that
// an ActionCreator failure does NOT fail the handler — the flow is still
// created and operators can drive it manually from the cockpit.
func TestOrchestrator_HandleAftersaleReturn_ActionCreateFails(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	db := newTestFlowDB(t)
	rt := aftersales.NewReturnRateTracker(db, logger)
	ac := &fakeActionCreator{failOn: 1}
	orch := NewOrchestrator(bus, db, logger, rt).WithActionCreator(ac)

	ctx := context.Background()
	payload := map[string]interface{}{
		"aftersale_id": float64(303),
		"order_id":     float64(8003),
		"sku_id":       float64(57),
		"quantity":     float64(1),
		"reason":       "fail test",
	}
	evt := eventbus.Event{Topic: "supplychain.aftersale.returned", Payload: payload}

	if err := orch.HandleAftersaleReturn(ctx, evt); err != nil {
		t.Fatalf("HandleAftersaleReturn should not fail when action create fails, got: %v", err)
	}

	// Flow should still be created.
	var count int64
	db.Model(&SupplyChainFlow{}).Where("source_type = ?", "aftersale_return").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 flow even when action create fails, got %d", count)
	}
}

// TestOrchestrator_HandleStockCritical_CreatesReplenishAction verifies that
// the orchestrator creates an approval-gated replenishment UnifiedAction when
// processing a supplychain.stock.critical event. Replenishment places a
// purchase order (money + inventory), so it must require human approval.
func TestOrchestrator_HandleStockCritical_CreatesReplenishAction(t *testing.T) {
	log := dbtest.NewLogger(t)
	bus := eventbus.New(log)
	db := newTestFlowDB(t)
	ac := &fakeActionCreator{}
	orch := NewOrchestrator(bus, db, log, nil).WithActionCreator(ac)

	ctx := context.Background()
	payload := map[string]interface{}{
		"sku_id":        float64(404),
		"current_stock": float64(20),
		"safety_stock":  float64(100),
		"sellable_days": float64(2.5),
	}
	evt := eventbus.Event{Topic: "supplychain.stock.critical", Source: "A5", Payload: payload}

	if err := orch.HandleStockCritical(ctx, evt); err != nil {
		t.Fatalf("HandleStockCritical returned error: %v", err)
	}

	calls := ac.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateAction call, got %d", len(calls))
	}
	in := calls[0]
	if in.ActionType != "replenish_order" {
		t.Errorf("expected action_type replenish_order, got %s", in.ActionType)
	}
	if in.RiskLevel != "high" {
		t.Errorf("expected risk_level high, got %s", in.RiskLevel)
	}
	if !in.RequiresApproval {
		t.Error("expected RequiresApproval=true (no auto-replenish)")
	}
	if in.AgentID != "A5" {
		t.Errorf("expected agent_id A5 (inventory alert), got %s", in.AgentID)
	}
	if in.BusinessObjectID != "404" {
		t.Errorf("expected business_object_id 404, got %s", in.BusinessObjectID)
	}
}

// TestOrchestrator_HandleStockCritical_NoActionCreator verifies the
// orchestrator still creates the flow + publishes sourcing.rescan when no
// ActionCreator is wired — action creation is best-effort.
func TestOrchestrator_HandleStockCritical_NoActionCreator(t *testing.T) {
	log := dbtest.NewLogger(t)
	bus := eventbus.New(log, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	rescanReceived := make(chan eventbus.Event, 1)
	bus.Subscribe("sourcing.rescan", func(ctx context.Context, evt eventbus.Event) error {
		rescanReceived <- evt
		return nil
	})

	db := newTestFlowDB(t)
	// No WithActionCreator.
	orch := NewOrchestrator(bus, db, log, nil)

	payload := map[string]interface{}{
		"sku_id":        float64(505),
		"current_stock": float64(10),
		"safety_stock":  float64(80),
		"sellable_days": float64(1.5),
	}
	evt := eventbus.Event{Topic: "supplychain.stock.critical", Source: "A5", Payload: payload}

	if err := orch.HandleStockCritical(ctx, evt); err != nil {
		t.Fatalf("HandleStockCritical returned error: %v", err)
	}

	// Flow should still be created.
	var count int64
	db.Model(&SupplyChainFlow{}).Where("source_type = ?", "stock_critical").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 flow even without ActionCreator, got %d", count)
	}

	// sourcing.rescan should still be published.
	select {
	case <-rescanReceived:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sourcing.rescan event")
	}
}
