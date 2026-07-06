package producthub

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newFreshnessService(t *testing.T) (*gorm.DB, FreshnessService) {
	t.Helper()
	db := dbtest.NewDB(t, &DataFreshness{})
	return db, NewFreshnessService(db, zap.NewNop())
}

func TestComputeFreshnessLabel_Fresh(t *testing.T) {
	f := DataFreshness{Status: "fresh", DriftDetected: false}
	label := computeFreshnessLabel(f, time.Now())
	if label != "fresh" {
		t.Fatalf("expected 'fresh', got '%s'", label)
	}
}

func TestComputeFreshnessLabel_Stale(t *testing.T) {
	f := DataFreshness{Status: "stale", DriftDetected: false}
	label := computeFreshnessLabel(f, time.Now())
	if label != "stale" {
		t.Fatalf("expected 'stale', got '%s'", label)
	}
}

func TestComputeFreshnessLabel_Expired(t *testing.T) {
	f := DataFreshness{Status: "expired", DriftDetected: false}
	label := computeFreshnessLabel(f, time.Now())
	if label != "expired" {
		t.Fatalf("expected 'expired', got '%s'", label)
	}
}

func TestComputeFreshnessLabel_Drift(t *testing.T) {
	f := DataFreshness{DriftDetected: true, Status: "fresh"}
	label := computeFreshnessLabel(f, time.Now())
	if label != "drift" {
		t.Fatalf("expected 'drift' (overrides status), got '%s'", label)
	}
}

func TestDetectDrift_NoChange(t *testing.T) {
	_, svc := newFreshnessService(t)
	ctx := t.Context()

	if err := svc.RecordVerification(ctx, 1, "pricing", "100"); err != nil {
		t.Fatal(err)
	}

	drifted, err := svc.DetectDrift(ctx, 1, "pricing", "100")
	if err != nil {
		t.Fatal(err)
	}
	if drifted {
		t.Fatal("expected no drift when value matches")
	}
}

func TestDetectDrift_Changed(t *testing.T) {
	_, svc := newFreshnessService(t)
	ctx := t.Context()

	if err := svc.RecordVerification(ctx, 1, "pricing", "100"); err != nil {
		t.Fatal(err)
	}

	drifted, err := svc.DetectDrift(ctx, 1, "pricing", "200")
	if err != nil {
		t.Fatal(err)
	}
	if !drifted {
		t.Fatal("expected drift when value differs")
	}
}

func TestDetectDrift_NoRecord(t *testing.T) {
	_, svc := newFreshnessService(t)
	ctx := t.Context()

	drifted, err := svc.DetectDrift(ctx, 999, "pricing", "100")
	if err != nil {
		t.Fatal(err)
	}
	if drifted {
		t.Fatal("expected no drift when no record exists")
	}
}

func TestRecordVerification(t *testing.T) {
	db, svc := newFreshnessService(t)
	ctx := t.Context()

	// First verification creates a record with "fresh" status
	if err := svc.RecordVerification(ctx, 1, "pricing", "100"); err != nil {
		t.Fatal(err)
	}

	var record DataFreshness
	if err := db.Where("product_id = ? AND dimension = ?", 1, "pricing").
		First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.Status != "fresh" {
		t.Fatalf("expected status 'fresh', got '%s'", record.Status)
	}
	if record.CurrentValue != "100" {
		t.Fatalf("expected current_value '100', got '%s'", record.CurrentValue)
	}

	// Re-verify with a different value; drift should be detected but the
	// original last_value preserved as the reference.
	if err := svc.RecordVerification(ctx, 1, "pricing", "200"); err != nil {
		t.Fatal(err)
	}

	var updated DataFreshness
	if err := db.Where("product_id = ? AND dimension = ?", 1, "pricing").
		First(&updated).Error; err != nil {
		t.Fatal(err)
	}
	if !updated.DriftDetected {
		t.Fatal("expected drift_detected after value change")
	}
	if updated.CurrentValue != "200" {
		t.Fatalf("expected current_value '200', got '%s'", updated.CurrentValue)
	}
	if updated.LastValue != "100" {
		t.Fatalf("expected last_value preserved as '100', got '%s'", updated.LastValue)
	}
}

func TestCheckFreshness(t *testing.T) {
	db, svc := newFreshnessService(t)
	ctx := t.Context()
	now := time.Now()

	// Record A: overdue by 5 days — should become "stale"
	a := DataFreshness{
		ProductID:     1,
		Dimension:     "pricing",
		LastVerifiedAt: now.AddDate(0, 0, -35),
		NextCheckAt:   now.AddDate(0, 0, -5),
		FreshnessDays: 30,
		Status:        "fresh",
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}

	// Record B: overdue by 70 days (> 2x freshness_days safety net) — should become "expired"
	b := DataFreshness{
		ProductID:     2,
		Dimension:     "content",
		LastVerifiedAt: now.AddDate(0, 0, -100),
		NextCheckAt:   now.AddDate(0, 0, -70),
		FreshnessDays: 30,
		Status:        "fresh",
	}
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}

	// Record C: still fresh (future next_check_at) — should NOT be returned
	c := DataFreshness{
		ProductID:     3,
		Dimension:     "inventory",
		LastVerifiedAt: now,
		NextCheckAt:   now.AddDate(0, 0, 30),
		FreshnessDays: 30,
		Status:        "fresh",
	}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}

	results, err := svc.CheckFreshness(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 stale/expired records, got %d", len(results))
	}

	// Verify A is now "stale" and B is "expired"
	var updatedA, updatedB DataFreshness
	db.First(&updatedA, a.ID)
	db.First(&updatedB, b.ID)

	if updatedA.Status != "stale" {
		t.Fatalf("record A: expected status 'stale', got '%s'", updatedA.Status)
	}
	if updatedB.Status != "expired" {
		t.Fatalf("record B: expected status 'expired', got '%s'", updatedB.Status)
	}

	// Record C should still be "fresh"
	var updatedC DataFreshness
	db.First(&updatedC, c.ID)
	if updatedC.Status != "fresh" {
		t.Fatalf("record C: expected status 'fresh', got '%s'", updatedC.Status)
	}
}
