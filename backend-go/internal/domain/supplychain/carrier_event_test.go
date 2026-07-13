package supplychain

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

const sqliteCarrierEventDDL = `
CREATE TABLE supply_chain_carrier_event (
 id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER NOT NULL, tracking_id TEXT NOT NULL,
 source_system TEXT NOT NULL, external_event_id TEXT NOT NULL, status TEXT NOT NULL,
 occurred_at DATETIME NOT NULL, observed_at DATETIME NOT NULL, location TEXT, message TEXT,
 raw_payload TEXT NOT NULL, payload_sha256 TEXT NOT NULL, truth_status TEXT NOT NULL, created_at DATETIME,
 UNIQUE(owner_id, source_system, external_event_id));`

func newCarrierEventService(t *testing.T) *TrackingService {
	t.Helper()
	db := dbtest.NewDB(t)
	if err := db.Exec(sqliteTrackingDDL).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(sqliteCarrierEventDDL).Error; err != nil {
		t.Fatal(err)
	}
	return NewTrackingService(db, dbtest.NewLogger(t))
}

func TestCarrierEvent_ExternalDeliveredProjectsActualDeliveryIdempotently(t *testing.T) {
	svc := newCarrierEventService(t)
	ctx := context.Background()
	tracking, err := svc.CreateForOwner(ctx, 41, &CreateTrackingRequest{FlowID: "f1", CarrierCode: "dhl", TrackingNo: "T1"})
	if err != nil {
		t.Fatal(err)
	}
	occurred := time.Date(2026, 7, 12, 8, 0, 0, 0, time.UTC)
	req := &IngestCarrierEventRequest{SourceSystem: "dhl_api", ExternalEventID: "evt-1", Status: "delivered", OccurredAt: occurred, ObservedAt: occurred.Add(time.Minute), RawPayload: []byte(`{"event":"delivered"}`)}
	event, replay, err := svc.IngestCarrierEvent(ctx, 41, tracking.ID, req)
	if err != nil || replay {
		t.Fatalf("first ingest event=%+v replay=%v err=%v", event, replay, err)
	}
	if event.TruthStatus != "external_observed" {
		t.Fatalf("truth=%s", event.TruthStatus)
	}
	got, err := svc.GetByIDForOwner(ctx, 41, tracking.ID)
	if err != nil || got.ActualDelivery == nil || !got.ActualDelivery.Equal(occurred) {
		t.Fatalf("actual delivery not projected: %+v err=%v", got, err)
	}
	_, replay, err = svc.IngestCarrierEvent(ctx, 41, tracking.ID, req)
	if err != nil || !replay {
		t.Fatalf("expected idempotent replay, replay=%v err=%v", replay, err)
	}
	items, _ := svc.ListCarrierEvents(ctx, 41, tracking.ID)
	if len(items) != 1 {
		t.Fatalf("events=%d", len(items))
	}
}

func TestCarrierEvent_FailsClosedAcrossOwnersAndOnConflict(t *testing.T) {
	svc := newCarrierEventService(t)
	ctx := context.Background()
	tracking, err := svc.CreateForOwner(ctx, 41, &CreateTrackingRequest{FlowID: "f1", CarrierCode: "dhl", TrackingNo: "T1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	req := &IngestCarrierEventRequest{SourceSystem: "dhl_api", ExternalEventID: "evt-1", Status: "transit", OccurredAt: now, ObservedAt: now, RawPayload: []byte(`{"event":"transit"}`)}
	if _, _, err := svc.IngestCarrierEvent(ctx, 42, tracking.ID, req); !errors.Is(err, ErrTrackingNotOwned) {
		t.Fatalf("cross-owner err=%v", err)
	}
	if _, err := svc.GetByIDForOwner(ctx, 42, tracking.ID); err == nil {
		t.Fatal("cross-owner read must fail")
	}
	if _, err := svc.UpdateStatusForOwner(ctx, 42, tracking.ID, &UpdateTrackingRequest{Status: "transit"}); err == nil {
		t.Fatal("cross-owner update must fail")
	}
	if _, _, err := svc.IngestCarrierEvent(ctx, 41, tracking.ID, req); err != nil {
		t.Fatal(err)
	}
	conflict := *req
	conflict.Status = "delivered"
	if _, _, err := svc.IngestCarrierEvent(ctx, 41, tracking.ID, &conflict); !errors.Is(err, ErrCarrierEventConflict) {
		t.Fatalf("conflict err=%v", err)
	}
}

func TestManualTrackingUpdateCannotClaimActualDelivery(t *testing.T) {
	svc := newCarrierEventService(t)
	ctx := context.Background()
	tracking, err := svc.CreateForOwner(ctx, 41, &CreateTrackingRequest{FlowID: "f1", CarrierCode: "dhl", TrackingNo: "T1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_, err = svc.UpdateStatus(ctx, tracking.ID, &UpdateTrackingRequest{Status: "delivered", ActualDelivery: &now})
	if err == nil || !strings.Contains(err.Error(), "external carrier evidence") {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateTrackingWithOrderRequiresOwnerScopedExternalOrderFact(t *testing.T) {
	svc := newCarrierEventService(t)
	ctx := context.Background()
	if err := svc.db.Exec(`CREATE TABLE platform_order_ingest (owner_id INTEGER, external_order_id TEXT, truth_status TEXT, processing_status TEXT, normalized_order_id INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	req := &CreateTrackingRequest{FlowID: "f-order", OrderID: "EXT-1", CarrierCode: "dhl", TrackingNo: "T1"}
	if _, err := svc.CreateForOwner(ctx, 41, req); err == nil {
		t.Fatal("expected missing order fact to fail closed")
	}
	if err := svc.db.Exec(`INSERT INTO platform_order_ingest VALUES (42,'EXT-1','external_observed','applied',7)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateForOwner(ctx, 41, req); err == nil {
		t.Fatal("expected cross-owner fact to fail closed")
	}
	if err := svc.db.Exec(`INSERT INTO platform_order_ingest VALUES (41,'EXT-1','external_observed','applied',7)`).Error; err != nil {
		t.Fatal(err)
	}
	got, err := svc.CreateForOwner(ctx, 41, req)
	if err != nil || got.OwnerID != 41 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
