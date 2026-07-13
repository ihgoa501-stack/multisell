package sourcing1688

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type watchDraftHashGuard struct {
	ID                    int64 `gorm:"primaryKey"`
	SourcingProductID     int64
	ApprovalContentSHA256 string
}

func (watchDraftHashGuard) TableName() string { return "sourcing_listing_draft" }

func newWatchService(t *testing.T) (*Service, *gorm.DB, Sourcing1688Product, Sourcing1688Snapshot, Sourcing1688Snapshot) {
	t.Helper()
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &SourcingWatchSubscription{}, &SourcingWatchRefreshRun{}, &SourcingWatchAlert{}, &watchDraftHashGuard{}, &demandCaseRow{})
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/9001.html", SourceOfferID: "9001", MOQ: 1, SupplierName: "工厂A"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	t0 := time.Now().UTC().Add(-time.Hour)
	price1, price2 := 10.0, 12.0
	before := Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: t0, CollectedBy: 42, Driver: "extension", ParserVersion: "v1", RawPayload: json.RawMessage(`{"spec_variants":[{"spec":"颜色:红;尺码:S","stock":9}],"offer_status":"online"}`), RawSHA256: strings.Repeat("a", 64), ObservedPrice: &price1, ObservedMOQ: 1, ObservedSupplier: "工厂A"}
	after := Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: t0.Add(time.Minute), CollectedBy: 42, Driver: "extension", ParserVersion: "v1", RawPayload: json.RawMessage(`{"spec_variants":[{"spec":"颜色:红;尺码:S","stock":3},{"spec":"颜色:蓝;尺码:M","stock":0}],"is_delisted":true}`), RawSHA256: strings.Repeat("b", 64), ObservedPrice: &price2, ObservedMOQ: 2, ObservedSupplier: "工厂B"}
	if err := db.Create(&before).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&after).Error; err != nil {
		t.Fatal(err)
	}
	return NewService(db, zap.NewNop()), db, product, before, after
}

func TestSourcingWatchRequiresExplicitOwnerSubscriptionAndIsolatesOwner(t *testing.T) {
	svc, _, product, _, _ := newWatchService(t)
	if _, err := svc.CreateSourcingWatchRun(42, product.ID, "watch-1"); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("run without enabled watch = %v", err)
	}
	watch, err := svc.SetSourcingWatch(42, product.ID, true)
	if err != nil || !watch.Enabled {
		t.Fatalf("enable watch = %#v, %v", watch, err)
	}
	if _, err := svc.GetSourcingWatch(99, product.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross-owner read = %v", err)
	}
	if _, err := svc.CreateSourcingWatchRun(99, product.ID, "watch-x"); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross-owner create = %v", err)
	}
	if _, err := svc.SetSourcingWatch(42, product.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourcingWatchRun(42, product.ID, "watch-2"); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("disabled watch run = %v", err)
	}
}

func TestSourcingWatchEvaluationIsIdempotentAndDoesNotChangeDraftHash(t *testing.T) {
	svc, db, product, before, after := newWatchService(t)
	if _, err := svc.SetSourcingWatch(42, product.ID, true); err != nil {
		t.Fatal(err)
	}
	run, err := svc.CreateSourcingWatchRun(42, product.ID, "watch-evaluate-1")
	if err != nil || run.Status != WatchRunPendingBrowser {
		t.Fatalf("create run = %#v, %v", run, err)
	}
	replay, err := svc.CreateSourcingWatchRun(42, product.ID, "watch-evaluate-1")
	if err != nil || replay.ID != run.ID {
		t.Fatalf("run replay = %#v, %v", replay, err)
	}
	runs, err := svc.ListSourcingWatchRuns(42, product.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("recover runs = %#v, %v", runs, err)
	}
	after.CollectedAt = run.CreatedAt.Add(time.Second)
	if err := db.Model(&Sourcing1688Snapshot{}).Where("id = ?", after.ID).Update("collected_at", after.CollectedAt).Error; err != nil {
		t.Fatal(err)
	}

	const frozenHash = "draft-content-must-not-change"
	if err := db.Create(&watchDraftHashGuard{SourcingProductID: product.ID, ApprovalContentSHA256: frozenHash}).Error; err != nil {
		t.Fatal(err)
	}
	in := EvaluateSourcingWatchRunInput{PreviousSnapshotID: before.ID, CurrentSnapshotID: after.ID}
	evaluated, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, in)
	if err != nil || evaluated.Status != WatchRunEvaluated || evaluated.AlertCount != 6 {
		t.Fatalf("evaluate = %#v, %v", evaluated, err)
	}
	second, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, in)
	if err != nil || second.AlertCount != 6 {
		t.Fatalf("evaluate replay = %#v, %v", second, err)
	}
	alerts, err := svc.ListSourcingWatchAlerts(42, product.ID)
	if err != nil || len(alerts) != 6 {
		t.Fatalf("alerts = %#v, %v", alerts, err)
	}
	types := map[string]bool{}
	for _, alert := range alerts {
		types[alert.ChangeType] = true
		if alert.PreviousSnapshotID != before.ID || alert.CurrentSnapshotID != after.ID {
			t.Fatalf("bad alert refs: %#v", alert)
		}
	}
	for _, typ := range []string{"price", "moq", "supplier", "sku_set", "quoted_inventory", "offer_state"} {
		if !types[typ] {
			t.Errorf("missing alert %s", typ)
		}
	}
	var draft watchDraftHashGuard
	if err := db.First(&draft).Error; err != nil {
		t.Fatal(err)
	}
	if draft.ApprovalContentSHA256 != frozenHash {
		t.Fatalf("watch changed draft hash to %q", draft.ApprovalContentSHA256)
	}
	var count int64
	db.Model(&SourcingWatchAlert{}).Count(&count)
	if count != 6 {
		t.Fatalf("idempotent alert count = %d", count)
	}
}

func TestSourcingWatchRejectsCrossSourceSnapshotsAndChangedReplay(t *testing.T) {
	svc, db, product, before, after := newWatchService(t)
	if _, err := svc.SetSourcingWatch(42, product.ID, true); err != nil {
		t.Fatal(err)
	}
	run, _ := svc.CreateSourcingWatchRun(42, product.ID, "watch-guard")
	after.CollectedAt = run.CreatedAt.Add(time.Second)
	if err := db.Model(&Sourcing1688Snapshot{}).Where("id = ?", after.ID).Update("collected_at", after.CollectedAt).Error; err != nil {
		t.Fatal(err)
	}
	other := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/2.html", SourceOfferID: "2", MOQ: 1}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	bad := after
	bad.ID = 0
	bad.SourcingProductID = other.ID
	bad.RawSHA256 = strings.Repeat("c", 64)
	if err := db.Create(&bad).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, EvaluateSourcingWatchRunInput{PreviousSnapshotID: before.ID, CurrentSnapshotID: bad.ID}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-source snapshot = %v", err)
	}
	if _, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, EvaluateSourcingWatchRunInput{PreviousSnapshotID: before.ID, CurrentSnapshotID: after.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, EvaluateSourcingWatchRunInput{PreviousSnapshotID: after.ID, CurrentSnapshotID: before.ID}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("changed replay = %v", err)
	}
}

func TestSourcingWatchRequiresNewObservationForRun(t *testing.T) {
	svc, _, product, before, after := newWatchService(t)
	if _, err := svc.SetSourcingWatch(42, product.ID, true); err != nil {
		t.Fatal(err)
	}
	run, err := svc.CreateSourcingWatchRun(42, product.ID, "watch-new-only")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, EvaluateSourcingWatchRunInput{PreviousSnapshotID: before.ID, CurrentSnapshotID: after.ID}); !errors.Is(err, ErrInvalidWorkflow) {
		t.Fatalf("historical current snapshot err=%v", err)
	}
	if _, err := svc.EvaluateSourcingWatchRun(42, product.ID, run.ID, EvaluateSourcingWatchRunInput{PreviousSnapshotID: before.ID, CurrentSnapshotID: before.ID}); !errors.Is(err, ErrInvalidWorkflow) {
		t.Fatalf("same snapshot err=%v", err)
	}
}

func TestSourcingWatchMigrationMakesAlertsAppendOnly(t *testing.T) {
	body, err := os.ReadFile("../../../migrations/000147_sourcing_1688_watch_monitor.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{"sourcing_1688_watch_subscription", "pending_browser", "sourcing_1688_watch_alert", "BEFORE UPDATE ON sourcing_1688_watch_alert", "BEFORE DELETE ON sourcing_1688_watch_alert", "ux_sourcing_watch_alert_run_type"} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration missing %q", required)
		}
	}
}
