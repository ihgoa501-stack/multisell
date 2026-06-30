package supplychain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// MockCarrierClient.FetchTrackingEvents
// ---------------------------------------------------------------------------

func TestMockCarrier_HappyLifecycle(t *testing.T) {
	client := NewMockCarrierClient()

	events, err := client.FetchTrackingEvents("TRK-HAPPY-001")
	if err != nil {
		t.Fatalf("FetchTrackingEvents failed: %v", err)
	}

	if len(events) != 6 {
		t.Fatalf("expected 6 lifecycle events, got %d", len(events))
	}

	wantOrder := []string{"picked_up", "outbound", "transit", "customs", "last_mile", "delivered"}
	for i, e := range events {
		if e.Status != wantOrder[i] {
			t.Errorf("event[%d]: expected status %s, got %s", i, wantOrder[i], e.Status)
		}
		if e.Location == "" {
			t.Errorf("event[%d] (%s): expected non-empty location", i, e.Status)
		}
		if e.Message == "" {
			t.Errorf("event[%d] (%s): expected non-empty message", i, e.Status)
		}
	}

	// Timestamps must be monotonically increasing.
	for i := 1; i < len(events); i++ {
		if !events[i].Timestamp.After(events[i-1].Timestamp) {
			t.Errorf("event[%d] timestamp %v not after event[%d] timestamp %v",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
		}
	}
}

func TestMockCarrier_ExceptionLifecycle(t *testing.T) {
	client := NewMockCarrierClient()

	// Tracking numbers prefixed with "EXC-" return an exception path.
	events, err := client.FetchTrackingEvents("EXC-PROBLEM-001")
	if err != nil {
		t.Fatalf("FetchTrackingEvents failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 exception-path events, got %d", len(events))
	}

	if events[2].Status != "exception" {
		t.Errorf("expected final status 'exception', got '%s'", events[2].Status)
	}
}

func TestMockCarrier_EmptyTrackingNo(t *testing.T) {
	client := NewMockCarrierClient()

	_, err := client.FetchTrackingEvents("")
	if !errors.Is(err, ErrMockCarrierTrackingNotFound) {
		t.Errorf("expected ErrMockCarrierTrackingNotFound, got %v", err)
	}
}

func TestMockCarrier_Deterministic(t *testing.T) {
	client := NewMockCarrierClient()

	first, err := client.FetchTrackingEvents("TRK-DET-001")
	if err != nil {
		t.Fatalf("first FetchTrackingEvents failed: %v", err)
	}

	second, err := client.FetchTrackingEvents("TRK-DET-001")
	if err != nil {
		t.Fatalf("second FetchTrackingEvents failed: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("deterministic check: length mismatch %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].Status != second[i].Status {
			t.Errorf("event[%d] status mismatch: %s vs %s", i, first[i].Status, second[i].Status)
		}
		if !first[i].Timestamp.Equal(second[i].Timestamp) {
			t.Errorf("event[%d] timestamp mismatch: %v vs %v", i, first[i].Timestamp, second[i].Timestamp)
		}
	}
}

// ---------------------------------------------------------------------------
// TrackingService.SyncFromCarrier
// ---------------------------------------------------------------------------

// setupTrackingDB creates an isolated SQLite DB with the supply_chain_tracking
// table for SyncFromCarrier tests.
func setupTrackingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := dbtest.NewDB(t)
	db.Exec(sqliteTrackingDDL)
	return db
}

func TestSyncFromCarrier_AppendsFullLifecycle(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	client := NewMockCarrierClient()
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-sync-1",
		CarrierCode: "CNExpress",
		TrackingNo:  "TRK-SYNC-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.SyncFromCarrier(ctx, created.ID, client)
	if err != nil {
		t.Fatalf("SyncFromCarrier failed: %v", err)
	}

	// History should contain: 1 creation event + 6 mock lifecycle events.
	var history []TrackingEvent
	if err := json.Unmarshal(*updated.StatusHistory, &history); err != nil {
		t.Fatalf("unmarshal status_history: %v", err)
	}
	if len(history) != 7 {
		t.Errorf("expected 7 history entries (1 create + 6 lifecycle), got %d", len(history))
	}

	if updated.Status != "delivered" {
		t.Errorf("expected final status 'delivered', got '%s'", updated.Status)
	}

	if updated.ActualDelivery == nil {
		t.Error("expected ActualDelivery to be set for delivered status")
	}
	if updated.EstimatedDelivery == nil {
		t.Error("expected EstimatedDelivery to be set for delivered status")
	}
}

func TestSyncFromCarrier_Idempotent(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	client := NewMockCarrierClient()
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-sync-2",
		CarrierCode: "DHL",
		TrackingNo:  "TRK-IDEMP-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	first, err := svc.SyncFromCarrier(ctx, created.ID, client)
	if err != nil {
		t.Fatalf("first SyncFromCarrier failed: %v", err)
	}

	second, err := svc.SyncFromCarrier(ctx, created.ID, client)
	if err != nil {
		t.Fatalf("second SyncFromCarrier failed: %v", err)
	}

	var firstHistory, secondHistory []TrackingEvent
	if err := json.Unmarshal(*first.StatusHistory, &firstHistory); err != nil {
		t.Fatalf("unmarshal first history: %v", err)
	}
	if err := json.Unmarshal(*second.StatusHistory, &secondHistory); err != nil {
		t.Fatalf("unmarshal second history: %v", err)
	}

	if len(firstHistory) != len(secondHistory) {
		t.Errorf("idempotent check: history length changed %d → %d",
			len(firstHistory), len(secondHistory))
	}

	if second.Status != first.Status {
		t.Errorf("idempotent check: status changed %s → %s", first.Status, second.Status)
	}
}

func TestSyncFromCarrier_ExceptionPath(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	client := NewMockCarrierClient()
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-sync-3",
		CarrierCode: "YunExpress",
		TrackingNo:  "EXC-TRK-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.SyncFromCarrier(ctx, created.ID, client)
	if err != nil {
		t.Fatalf("SyncFromCarrier failed: %v", err)
	}

	if updated.Status != "exception" {
		t.Errorf("expected status 'exception', got '%s'", updated.Status)
	}

	// ActualDelivery should NOT be set for exception status.
	if updated.ActualDelivery != nil {
		t.Error("expected ActualDelivery to be nil for exception status")
	}
}

func TestSyncFromCarrier_NilClient(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-sync-4",
		CarrierCode: "UPS",
		TrackingNo:  "TRK-NIL-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = svc.SyncFromCarrier(ctx, created.ID, nil)
	if err == nil {
		t.Fatal("expected error for nil carrier client")
	}
}

func TestSyncFromCarrier_NotFound(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	client := NewMockCarrierClient()
	ctx := context.Background()

	_, err := svc.SyncFromCarrier(ctx, "nonexistent-id", client)
	if err == nil {
		t.Fatal("expected error for nonexistent tracking record")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestSyncFromCarrier_PreservesExistingManualEntries(t *testing.T) {
	db := setupTrackingDB(t)
	logger := dbtest.NewLogger(t)
	svc := NewTrackingService(db, logger)
	client := NewMockCarrierClient()
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTrackingRequest{
		FlowID:      "flow-sync-5",
		CarrierCode: "FedEx",
		TrackingNo:  "TRK-MANUAL-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add a manual informational entry before syncing.
	manualEvt := TrackingEvent{
		Status:    "pending",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Location:  "Warehouse Pre-Check",
		Message:   "Manual pre-shipment inspection completed",
	}
	if _, err := svc.AddHistoryEntry(ctx, created.ID, manualEvt); err != nil {
		t.Fatalf("AddHistoryEntry failed: %v", err)
	}

	// Now sync — the manual entry must be preserved alongside mock events.
	updated, err := svc.SyncFromCarrier(ctx, created.ID, client)
	if err != nil {
		t.Fatalf("SyncFromCarrier failed: %v", err)
	}

	var history []TrackingEvent
	if err := json.Unmarshal(*updated.StatusHistory, &history); err != nil {
		t.Fatalf("unmarshal history: %v", err)
	}

	// Expected: 1 creation + 1 manual + 6 lifecycle = 8.
	if len(history) != 8 {
		t.Errorf("expected 8 history entries, got %d", len(history))
	}

	found := false
	for _, h := range history {
		if h.Message == "Manual pre-shipment inspection completed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("manual history entry was lost after SyncFromCarrier")
	}
}
