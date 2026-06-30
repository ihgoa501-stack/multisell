package supplychain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// sqliteTrackingDDL is the SQLite-compatible table DDL for supply_chain_tracking.
const sqliteTrackingDDL = `
	CREATE TABLE IF NOT EXISTS supply_chain_tracking (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		flow_id TEXT,
		order_id TEXT DEFAULT '',
		carrier_code TEXT DEFAULT '',
		tracking_no TEXT DEFAULT '',
		status TEXT DEFAULT 'pending',
		estimated_delivery DATETIME,
		actual_delivery DATETIME,
		status_history TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
`

func TestTrackingCreate_Success(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	req := &CreateTrackingRequest{
		FlowID:      "flow-001",
		OrderID:     "ORD-12345",
		CarrierCode: "CNExpress",
		TrackingNo:  "CN1234567890",
		Status:      "pending",
	}

	got, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
	if got.FlowID != "flow-001" {
		t.Errorf("expected FlowID 'flow-001', got '%s'", got.FlowID)
	}
	if got.CarrierCode != "CNExpress" {
		t.Errorf("expected CarrierCode 'CNExpress', got '%s'", got.CarrierCode)
	}
	if got.TrackingNo != "CN1234567890" {
		t.Errorf("expected TrackingNo 'CN1234567890', got '%s'", got.TrackingNo)
	}
	if got.Status != "pending" {
		t.Errorf("expected Status 'pending', got '%s'", got.Status)
	}

	// Verify status_history has the initial event.
	if got.StatusHistory == nil || string(*got.StatusHistory) == "[]" {
		t.Error("expected non-empty status_history")
	}
}

func TestTrackingCreate_DefaultStatus(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	req := &CreateTrackingRequest{
		FlowID:      "flow-002",
		CarrierCode: "DHL",
		TrackingNo:  "DHL9876543210",
	}

	got, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("expected default status 'pending', got '%s'", got.Status)
	}
}

func TestTrackingCreate_InvalidStatus(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	req := &CreateTrackingRequest{
		FlowID:      "flow-003",
		CarrierCode: "UPS",
		TrackingNo:  "UPS123",
		Status:      "invalid_status",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if err.Error() != "invalid tracking status: invalid_status" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTrackingGetByID(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-get-1",
		CarrierCode: "FedEx",
		TrackingNo:  "FX100200",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID '%s', got '%s'", created.ID, got.ID)
	}
	if got.CarrierCode != "FedEx" {
		t.Errorf("expected CarrierCode 'FedEx', got '%s'", got.CarrierCode)
	}
}

func TestTrackingGetByID_NotFound(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestTrackingUpdateStatus(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-upd-1",
		CarrierCode: "CNExpress",
		TrackingNo:  "CN555666",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.UpdateStatus(ctx, created.ID, &UpdateTrackingRequest{
		Status:   "picked_up",
		Location: "Shenzhen Warehouse",
		Message:  "Package picked up at origin",
	})
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if updated.Status != "picked_up" {
		t.Errorf("expected Status 'picked_up', got '%s'", updated.Status)
	}

	// Verify status_history has 2 entries: creation + status update.
	var history []TrackingEvent
	if err := json.Unmarshal(*updated.StatusHistory, &history); err != nil {
		t.Fatalf("failed to unmarshal status_history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 status_history entries, got %d", len(history))
	}
	if history[1].Status != "picked_up" {
		t.Errorf("expected second entry status 'picked_up', got '%s'", history[1].Status)
	}
	if history[1].Location != "Shenzhen Warehouse" {
		t.Errorf("expected location 'Shenzhen Warehouse', got '%s'", history[1].Location)
	}
}

func TestTrackingUpdateStatus_InvalidStatus(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-upd-inv",
		CarrierCode: "DHL",
		TrackingNo:  "DHL-INVALID",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = svc.UpdateStatus(ctx, created.ID, &UpdateTrackingRequest{
		Status: "bogus_status",
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if err.Error() != "invalid tracking status: bogus_status" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTrackingGetByFlowID(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, &CreateTrackingRequest{
			FlowID:      "flow-group-1",
			CarrierCode: "Carrier",
			TrackingNo:  "TRK",
		})
		if err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}

	items, err := svc.GetByFlowID(ctx, "flow-group-1")
	if err != nil {
		t.Fatalf("GetByFlowID failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	items, err = svc.GetByFlowID(ctx, "nonexistent-flow")
	if err != nil {
		t.Fatalf("GetByFlowID failed for nonexistent flow: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items for nonexistent flow, got %d", len(items))
	}
}

func TestTrackingList_WithFilters(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	svc.Create(ctx, &CreateTrackingRequest{
		FlowID: "flow-l-1", CarrierCode: "DHL", TrackingNo: "DHL-1", Status: "transit",
	})
	svc.Create(ctx, &CreateTrackingRequest{
		FlowID: "flow-l-2", CarrierCode: "DHL", TrackingNo: "DHL-2", Status: "delivered",
	})
	svc.Create(ctx, &CreateTrackingRequest{
		FlowID: "flow-l-3", CarrierCode: "FedEx", TrackingNo: "FX-1", Status: "pending",
	})

	items, total, err := svc.List(ctx, &ListTrackingRequest{
		CarrierCode: "DHL",
		Size:        20,
		Page:        1,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2 for DHL, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items for DHL, got %d", len(items))
	}

	items, total, err = svc.List(ctx, &ListTrackingRequest{
		Status: "delivered",
		Size:   20,
		Page:   1,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for delivered, got %d", total)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for delivered, got %d", len(items))
	}
}

func TestTrackingList_Empty(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	items, total, err := svc.List(ctx, &ListTrackingRequest{Size: 20, Page: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestTrackingStatusHistory_Progression(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID: "flow-prog", CarrierCode: "UPS", TrackingNo: "UPS-PROG",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	statusUpdates := []struct {
		status   string
		location string
		message  string
	}{
		{"picked_up", "Shenzhen", "Picked up from supplier"},
		{"outbound", "Guangzhou", "Departed from sorting center"},
		{"transit", "International Hub", "In transit to destination"},
		{"customs", "LAX", "Customs clearance in progress"},
		{"last_mile", "Los Angeles", "With local delivery carrier"},
		{"delivered", "Customer Address", "Package delivered"},
	}

	for _, upd := range statusUpdates {
		updated, err := svc.UpdateStatus(ctx, created.ID, &UpdateTrackingRequest{
			Status:   upd.status,
			Location: upd.location,
			Message:  upd.message,
		})
		if err != nil {
			t.Fatalf("UpdateStatus to %s failed: %v", upd.status, err)
		}
		created = updated
	}

	if created.Status != "delivered" {
		t.Errorf("expected final status 'delivered', got '%s'", created.Status)
	}

	var history []TrackingEvent
	if err := json.Unmarshal(*created.StatusHistory, &history); err != nil {
		t.Fatalf("failed to unmarshal status_history: %v", err)
	}

	expectedCount := 1 + len(statusUpdates)
	if len(history) != expectedCount {
		t.Errorf("expected %d status_history entries, got %d", expectedCount, len(history))
	}

	last := history[len(history)-1]
	if last.Status != "delivered" {
		t.Errorf("expected last status 'delivered', got '%s'", last.Status)
	}
	if last.Location != "Customer Address" {
		t.Errorf("expected last location 'Customer Address', got '%s'", last.Location)
	}
}

func TestTrackingAddHistoryEntry(t *testing.T) {
	db := dbtest.NewDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	db.Exec(sqliteTrackingDDL)

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID: "flow-addhist", CarrierCode: "DHL", TrackingNo: "DHL-HIST",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add informational entry without changing the tracking status.
	updated, err := svc.AddHistoryEntry(ctx, created.ID, TrackingEvent{
		Status:   "pending",
		Location: "Regional Hub",
		Message:  "Package arrived at regional sorting center",
	})
	if err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	// Status should remain unchanged.
	if updated.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", updated.Status)
	}

	var history []TrackingEvent
	if err := json.Unmarshal(*updated.StatusHistory, &history); err != nil {
		t.Fatalf("failed to unmarshal status_history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 entries, got %d", len(history))
	}
	if history[1].Message != "Package arrived at regional sorting center" {
		t.Errorf("expected informational message, got '%s'", history[1].Message)
	}
}
