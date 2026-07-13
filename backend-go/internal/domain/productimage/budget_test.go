package productimage

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type failingAuthorizedImageService struct {
	*fakeImageService
	calls atomic.Int32
}

func (f *failingAuthorizedImageService) EnqueueAuthorizedExecution(context.Context, string, string, string) (*imageservice.Attempt, error) {
	f.calls.Add(1)
	return nil, errors.New("dispatch response lost")
}

func TestBudgetPolicyReservationReleaseAndImmutableCharges(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	svc := NewService(db, dbtest.NewLogger(t), nil, strings.Repeat("s", 32))
	now := time.Now().UTC()
	p, err := svc.CreateBudgetPolicy(t.Context(), 7, BudgetPolicyInput{Currency: "usd", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "2.00", IdempotencyKey: "monthly"})
	if err != nil || p.Currency != "USD" {
		t.Fatalf("policy=%+v err=%v", p, err)
	}
	replay, err := svc.CreateBudgetPolicy(t.Context(), 7, BudgetPolicyInput{Currency: "usd", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "2.00", IdempotencyKey: "monthly"})
	if err != nil || replay.ID != p.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	if _, err := svc.CreateBudgetPolicy(t.Context(), 7, BudgetPolicyInput{Currency: "USD", PeriodStart: now, PeriodEnd: now.Add(2 * time.Hour), TotalAmount: "5.00", IdempotencyKey: "overlap"}); err == nil {
		t.Fatal("overlapping policy accepted")
	}

	newTask := func(idem string) *Task {
		v := &Task{OwnerID: 7, AssetID: 1, ImageServiceJobID: idem, IdempotencyKey: idem, ManifestHash: strings.Repeat(idem[:1], 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
		return v
	}
	t1 := newTask("alpha")
	a1, err := svc.ApproveExecution(t.Context(), 7, t1.ID, ApprovalInput{Processor: "paid-test", MaxCost: "1.50", Currency: "USD", ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	t2 := newTask("bravo")
	if _, err := svc.ApproveExecution(t.Context(), 7, t2.ID, ApprovalInput{Processor: "paid-test", MaxCost: "0.51", Currency: "USD", ExpectedVersion: 1}); err == nil {
		t.Fatal("aggregate budget exceeded")
	}
	var r BudgetReservation
	if err := db.Where("approval_id=?", a1.ID).First(&r).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReleaseBudgetReservation(t.Context(), 8, r.ID, "wrong owner"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("owner isolation err=%v", err)
	}
	if _, err := svc.ReleaseBudgetReservation(t.Context(), 7, r.ID, "Owner cancelled before claim"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApproveExecution(t.Context(), 7, t2.ID, ApprovalInput{Processor: "paid-test", MaxCost: "0.51", Currency: "USD", ExpectedVersion: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestBudgetClaimCannotReleaseAndChargesAreAppendOnly(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	remote := newFakeImageService()
	remote.available["paid-test"] = true
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	now := time.Now().UTC()
	_, _ = svc.CreateBudgetPolicy(t.Context(), 7, BudgetPolicyInput{Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1.00", IdempotencyKey: "p"})
	task := &Task{OwnerID: 7, AssetID: 1, ImageServiceJobID: "paid", IdempotencyKey: "paid", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs["paid"] = imageserviceJob(task)
	a, err := svc.ApproveExecution(t.Context(), 7, task.ID, ApprovalInput{Processor: "paid-test", MaxCost: "0.80", Currency: "USD", ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Execute(t.Context(), 7, task.ID, "one"); err != nil {
		t.Fatal(err)
	}
	var r BudgetReservation
	if err := db.Where("approval_id=?", a.ID).First(&r).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReleaseBudgetReservation(t.Context(), 7, r.ID, "unsafe"); err == nil {
		t.Fatal("claimed reservation released")
	}
	first, err := svc.ReconcileBudgetCharge(t.Context(), 7, r.ID, BudgetChargeInput{Amount: "0.90", Currency: "USD", EvidenceSHA: strings.Repeat("e", 64), ObservedAt: now, IdempotencyKey: "invoice"})
	if err != nil || first.Kind != "actual" || first.OverBudget {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	late, err := svc.ReconcileBudgetCharge(t.Context(), 7, r.ID, BudgetChargeInput{Amount: "0.20", Currency: "USD", EvidenceSHA: strings.Repeat("f", 64), ObservedAt: now.Add(time.Minute), IdempotencyKey: "late"})
	if err != nil || late.Kind != "late_fee" || !late.OverBudget {
		t.Fatalf("late=%+v err=%v", late, err)
	}
	var count int64
	db.Model(&BudgetCharge{}).Where("reservation_id=?", r.ID).Count(&count)
	if count != 2 {
		t.Fatalf("charges=%d", count)
	}
}

func TestUnknownDispatchRequiresNoChargeEvidenceBeforeBudgetRelease(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	base := newFakeImageService()
	base.available["paid-test"] = true
	remote := &failingAuthorizedImageService{fakeImageService: base}
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	now := time.Now().UTC()
	if _, err := svc.CreateBudgetPolicy(t.Context(), 7, BudgetPolicyInput{Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1.00", IdempotencyKey: "unknown-policy"}); err != nil {
		t.Fatal(err)
	}
	task := &Task{OwnerID: 7, AssetID: 1, ImageServiceJobID: "unknown-job", IdempotencyKey: "unknown-job", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	base.jobs[task.ImageServiceJobID] = imageserviceJob(task)
	if err := db.Create(&CostEntry{OwnerID: 7, TaskID: task.ID, Kind: "estimated", Category: "provider", Provider: "paid-test", Amount: "0.80", Currency: "USD", ExchangeRate: "1", ExchangeRateSource: "test", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "unknown-cost", RequestHash: strings.Repeat("c", 64), ExpectedTaskVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	approval, err := svc.ApproveExecution(t.Context(), 7, task.ID, ApprovalInput{Processor: "paid-test", MaxCost: "0.80", Currency: "USD", ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Execute(t.Context(), 7, task.ID, "unknown-execute"); err == nil {
		t.Fatal("lost dispatch response was accepted")
	}
	if _, err := svc.Execute(t.Context(), 7, task.ID, "unknown-execute-retry"); err == nil || remote.calls.Load() != 1 {
		t.Fatalf("reconciliation task retried: err=%v calls=%d", err, remote.calls.Load())
	}
	if err := db.First(task, task.ID).Error; err != nil || task.Status != "RECONCILE_REQUIRED" || task.ErrorCode != "INTERNAL_DISPATCH_OUTCOME_UNKNOWN" {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	var reservation BudgetReservation
	if err := db.Where("approval_id=?", approval.ID).First(&reservation).Error; err != nil || reservation.State != "claimed" {
		t.Fatalf("reservation=%+v err=%v", reservation, err)
	}
	providerFailure := base.jobs[task.ImageServiceJobID]
	providerFailure.Status, providerFailure.ErrorCode = "FAILED", "PROVIDER_REJECTED"
	base.jobs[task.ImageServiceJobID] = providerFailure
	refreshed, err := svc.GetTask(t.Context(), 7, task.ID)
	if err != nil || refreshed.Status != "FAILED" || refreshed.ErrorCode != "PROVIDER_REJECTED" {
		t.Fatalf("known provider failure=%+v err=%v", refreshed, err)
	}
	*task = *refreshed
	if err := db.Model(&Task{}).Where("id=?", task.ID).Update("error_code", nil).Error; err != nil {
		t.Fatal(err)
	}
	input := BudgetNoChargeInput{EvidenceSHA: strings.Repeat("e", 64), ObservedAt: task.UpdatedAt.Add(time.Second), Reason: "Provider dashboard confirms no request and no charge", IdempotencyKey: "unknown-no-charge"}
	noCharge, err := svc.ReconcileBudgetNoCharge(t.Context(), 7, reservation.ID, input)
	if err != nil || noCharge.Kind != "no_charge" || noCharge.Amount != "0" {
		t.Fatalf("noCharge=%+v err=%v", noCharge, err)
	}
	replay, err := svc.ReconcileBudgetNoCharge(t.Context(), 7, reservation.ID, input)
	if err != nil || replay.ID != noCharge.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	if err := db.First(&reservation, reservation.ID).Error; err != nil || reservation.State != "no_charge" || reservation.ReleasedAt == nil {
		t.Fatalf("reservation=%+v err=%v", reservation, err)
	}
	if err := db.First(task, task.ID).Error; err != nil || task.Status != "FAILED" || task.ErrorCode != "NO_CHARGE_CONFIRMED" {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	late, err := svc.ReconcileBudgetCharge(t.Context(), 7, reservation.ID, BudgetChargeInput{Amount: "0.25", Currency: "USD", EvidenceSHA: strings.Repeat("f", 64), ObservedAt: input.ObservedAt.Add(time.Minute), IdempotencyKey: "unknown-late-charge"})
	if err != nil || late.Kind != "late_fee" || late.DeltaAmount != "0.2500" {
		t.Fatalf("late=%+v err=%v", late, err)
	}
	var afterLate BudgetReservation
	if err := db.First(&afterLate, reservation.ID).Error; err != nil || afterLate.State != "spent" || afterLate.ReleasedAt != nil {
		t.Fatalf("late charge did not reclaim reservation: %+v err=%v", afterLate, err)
	}
	if err := db.First(task, task.ID).Error; err != nil || task.Status != "FAILED" || task.ErrorCode != "CHARGED_OUTPUT_UNRECOVERABLE" {
		t.Fatalf("late charge task=%+v err=%v", task, err)
	}
	if _, err := svc.ReconcileBudgetNoCharge(t.Context(), 8, reservation.ID, input); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-owner err=%v", err)
	}
}

func TestChargedWithoutRecoverableOutputClosesReconciliation(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	now := time.Now().UTC()
	policy := &BudgetPolicy{OwnerID: 7, Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1.00", IdempotencyKey: "charged-policy", RequestHash: strings.Repeat("p", 64)}
	if err := db.Create(policy).Error; err != nil {
		t.Fatal(err)
	}
	task := &Task{OwnerID: 7, AssetID: 1, ImageServiceJobID: "charged-job", IdempotencyKey: "charged-job", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "RECONCILE_REQUIRED", ErrorCode: "INTERNAL_DISPATCH_OUTCOME_UNKNOWN"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	approval := &ExecutionApproval{OwnerID: 7, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: "0.80", Currency: "USD", ExecutionID: "charged-approval", ExpiresAt: now.Add(time.Hour)}
	if err := db.Create(approval).Error; err != nil {
		t.Fatal(err)
	}
	reservation := &BudgetReservation{OwnerID: 7, PolicyID: policy.ID, ApprovalID: approval.ID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Provider: task.Processor, Currency: "USD", ReservedAmount: "0.80", State: "claimed", ClaimedAt: &now}
	if err := db.Create(reservation).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: 7, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: task.Version, ManifestHash: task.ManifestHash, Status: "RECONCILE_REQUIRED"}
	if err := db.Model(&Task{}).Where("id=?", task.ID).Update("error_code", nil).Error; err != nil {
		t.Fatal(err)
	}
	input := BudgetChargeInput{Amount: "0.75", Currency: "USD", EvidenceSHA: strings.Repeat("e", 64), ObservedAt: task.UpdatedAt.Add(time.Second), IdempotencyKey: "charged-no-output", Resolution: "charged_no_output"}
	charge, err := svc.ReconcileBudgetCharge(t.Context(), 7, reservation.ID, input)
	if err != nil || charge.Kind != "charged_no_output" || charge.Amount != "0.75" {
		t.Fatalf("charge=%+v err=%v", charge, err)
	}
	replay, err := svc.ReconcileBudgetCharge(t.Context(), 7, reservation.ID, input)
	if err != nil || replay.ID != charge.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	var after Task
	if err := db.First(&after, task.ID).Error; err != nil || after.Status != "FAILED" || after.ErrorCode != "CHARGED_OUTPUT_UNRECOVERABLE" {
		t.Fatalf("task=%+v err=%v", after, err)
	}
	var afterReservation BudgetReservation
	if err := db.First(&afterReservation, reservation.ID).Error; err != nil || afterReservation.State != "spent" {
		t.Fatalf("reservation=%+v err=%v", afterReservation, err)
	}
}

func imageserviceJob(t *Task) imageservice.Job {
	return imageservice.Job{ID: t.ImageServiceJobID, OwnerID: t.OwnerID, LingMirrorTaskID: strconv.FormatInt(t.ID, 10), LingMirrorTaskVersion: t.Version, ManifestHash: t.ManifestHash, Operation: t.Operation, Processor: t.Processor, Status: "QUEUED"}
}

// TestPostgresBudgetReservationConcurrency requires a disposable database with
// all migrations applied. It proves the advisory-lock invariant SQLite cannot.
func TestPostgresBudgetReservationConcurrency(t *testing.T) {
	dsn := os.Getenv("PRODUCTIMAGE_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("PRODUCTIMAGE_POSTGRES_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, zap.NewNop(), nil, strings.Repeat("s", 32))
	owner := time.Now().UnixNano()%1000000000 + 700000000
	now := time.Now().UTC()
	if _, err := svc.CreateBudgetPolicy(context.Background(), owner, BudgetPolicyInput{Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1.00", IdempotencyKey: "concurrency"}); err != nil {
		t.Fatal(err)
	}
	asset := Asset{OwnerID: owner, BlobID: strings.Repeat("b", 64), Filename: "budget.png", ContentType: "image/png", SizeBytes: 1, SHA256: strings.Repeat("a", 64), Truth: TruthUnknown, SourceKind: "upload", ChannelRestriction: "*"}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	tasks := make([]Task, 100)
	for i := range tasks {
		tasks[i] = Task{OwnerID: owner, AssetID: asset.ID, ImageServiceJobID: "budget-" + strconv.Itoa(i), IdempotencyKey: "budget-" + strconv.Itoa(i), ManifestHash: strings.Repeat(strconv.Itoa(i%10), 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", Purpose: "listing_main", Channel: "ozon", Region: "test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
		if err := db.Create(&tasks[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	var approved atomic.Int64
	errs := make(chan error, 100)
	var wg sync.WaitGroup
	for i := range tasks {
		wg.Add(1)
		go func(task Task) {
			defer wg.Done()
			_, err := svc.ApproveExecution(context.Background(), owner, task.ID, ApprovalInput{Processor: "openai", MaxCost: "0.02", Currency: "USD", ExpectedVersion: 1})
			if err == nil {
				approved.Add(1)
				return
			}
			var c *ConflictError
			if !errors.As(err, &c) || c.Code != "BUDGET_EXCEEDED" {
				errs <- err
			}
		}(tasks[i])
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("unexpected: %v", err)
	}
	if approved.Load() != 50 {
		t.Fatalf("approved=%d want 50", approved.Load())
	}
	var total string
	if err := db.Raw("SELECT COALESCE(SUM(reserved_amount),0)::text FROM product_image_budget_reservations WHERE owner_id=?", owner).Scan(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != "1.0000" && total != "1.00" && total != "1" {
		t.Fatalf("reserved=%s", total)
	}
}
