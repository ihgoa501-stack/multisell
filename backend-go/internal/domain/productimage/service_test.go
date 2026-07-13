package productimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"gorm.io/gorm"
)

type fakeImageService struct {
	putCalls               int
	createCalls            int
	executeCalls           int
	authorizedExecuteCalls int
	lastExecutionToken     string
	jobs                   map[string]imageservice.Job
	attempts               map[string][]imageservice.Attempt
	blobs                  map[string][]byte
	available              map[string]bool
}

type countingAuthorizedImageService struct {
	*fakeImageService
	authorizedCalls atomic.Int64
}

func (f *countingAuthorizedImageService) EnqueueAuthorizedExecution(_ context.Context, id, key, token string) (*imageservice.Attempt, error) {
	f.authorizedCalls.Add(1)
	f.lastExecutionToken = token
	return &imageservice.Attempt{ID: "attempt-paid", JobID: id, IdempotencyKey: key, Status: "queued"}, nil
}

func newFakeImageService() *fakeImageService {
	return &fakeImageService{jobs: map[string]imageservice.Job{}, attempts: map[string][]imageservice.Attempt{}, blobs: map[string][]byte{}, available: map[string]bool{"deterministic": true}}
}
func (f *fakeImageService) ProcessorAvailable(code string) bool { return f.available[code] }
func (f *fakeImageService) PutBlob(_ context.Context, _ string, body io.Reader) (*imageservice.PutBlobResponse, error) {
	f.putCalls++
	b, _ := io.ReadAll(body)
	digest := sha256.Sum256(b)
	return &imageservice.PutBlobResponse{BlobID: hex.EncodeToString(digest[:])}, nil
}
func (f *fakeImageService) CreateJob(_ context.Context, in imageservice.CreateJobRequest) (*imageservice.Job, error) {
	f.createCalls++
	id := "job-1"
	processor := in.Processor
	if processor == "" && in.Operation == "DETERMINISTIC_RESIZE" {
		processor = "deterministic"
	}
	j := imageservice.Job{ID: id, OwnerID: in.OwnerID, LingMirrorTaskID: in.LingMirrorTaskID, LingMirrorTaskVersion: in.LingMirrorTaskVersion, Status: "QUEUED", ManifestHash: in.ManifestHash, Operation: in.Operation, Processor: processor, Prompt: in.Prompt, MaxCost: in.MaxCost, Currency: in.Currency, Region: in.Region, ProviderEnvironment: in.ProviderEnvironment, Sandbox: in.Sandbox, Watermarked: in.Watermarked, NonPublishable: in.NonPublishable, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.jobs[id] = j
	return &j, nil
}
func (f *fakeImageService) GetJob(_ context.Context, id string) (*imageservice.Job, error) {
	j, ok := f.jobs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &j, nil
}
func (f *fakeImageService) QuiesceJob(_ context.Context, id string, in imageservice.QuiesceJobRequest) (*imageservice.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if job.OwnerID != in.OwnerID || job.LingMirrorTaskID != in.LingMirrorTaskID || job.LingMirrorTaskVersion != in.LingMirrorTaskVersion || job.ManifestHash != in.ManifestHash {
		return nil, errors.New("job identity mismatch")
	}
	if job.Status == "RUNNING" || job.Status == "READY" {
		return nil, errors.New("job active")
	}
	job.Status, job.ErrorCode = "FAILED", "CANCELLED_NO_CHARGE_RECONCILIATION"
	f.jobs[id] = job
	return &job, nil
}
func (f *fakeImageService) EnqueueExecution(_ context.Context, id, key string) (*imageservice.Attempt, error) {
	f.executeCalls++
	for i := range f.attempts[id] {
		if f.attempts[id][i].IdempotencyKey == key {
			replay := f.attempts[id][i]
			return &replay, nil
		}
	}
	a := imageservice.Attempt{ID: "attempt-1", JobID: id, IdempotencyKey: key, Status: "queued"}
	f.attempts[id] = append(f.attempts[id], a)
	return &a, nil
}
func (f *fakeImageService) EnqueueAuthorizedExecution(_ context.Context, id, key, token string) (*imageservice.Attempt, error) {
	f.authorizedExecuteCalls++
	f.lastExecutionToken = token
	a := imageservice.Attempt{ID: "attempt-paid", JobID: id, IdempotencyKey: key, Status: "queued"}
	return &a, nil
}
func (f *fakeImageService) ListAttempts(_ context.Context, id string) ([]imageservice.Attempt, error) {
	return f.attempts[id], nil
}
func (f *fakeImageService) GetBlob(_ context.Context, id string) ([]byte, string, error) {
	return f.blobs[id], "image/png", nil
}

func newTestService(t *testing.T) (*Service, *fakeImageService) {
	t.Helper()
	db := dbtest.NewDB(t, &canonicalSKU{}, &Asset{}, &Task{}, &Review{}, &RightsGrant{}, &CostEntry{})
	if err := db.Create(&canonicalSKU{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	client := newFakeImageService()
	return NewService(db, dbtest.NewLogger(t), client), client
}

func validTaskInput(asset *Asset, key string) CreateTaskInput {
	return CreateTaskInput{AssetID: asset.ID, SKUID: 1, RecipeKey: "recipe-1", RecipeVersion: 1, CandidateRound: 1, Recipe: RecipeManifest{SceneStructure: "clean studio", Model: "deterministic", ModelVersion: "1", Parameters: json.RawMessage(`{}`), MustNotChange: []string{"product shape", "color"}}, IdempotencyKey: key, Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Width: 100, Height: 100, Format: "png"}
}

func seedTaskRights(t *testing.T, svc *Service, owner int64, asset *Asset, purpose, channel string) {
	t.Helper()
	assetID := asset.ID
	now := time.Now().UTC()
	grant := RightsGrant{OwnerID: owner, AssetID: &assetID, AssetSHA: asset.SHA256, CanCopy: true, CanModify: true, Purpose: purpose, Channel: channel, Jurisdiction: "ru-ru", Provider: "deterministic", Region: "local", Grantor: "owner", RightsChain: "test", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now.Add(-time.Minute), IdempotencyKey: "input-rights-" + strconv.FormatInt(owner, 10) + "-" + asset.SHA256[:8], RequestHash: strings.Repeat("f", 64), Version: 1}
	if err := svc.db.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAssetTaskExecutionFlowAndIdempotentReplay(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	ctx := context.Background()
	asset, err := svc.CreateAsset(ctx, 11, "product.png", "image/png", []byte("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if asset.Truth != TruthUnknown || remote.putCalls != 1 {
		t.Fatalf("asset=%+v calls=%d", asset, remote.putCalls)
	}

	seedTaskRights(t, svc, 11, asset, "listing_main", "ozon")
	in := validTaskInput(asset, "task-key")
	in.Width, in.Height = 1000, 1000
	task, err := svc.CreateTask(ctx, 11, in)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := svc.CreateTask(ctx, 11, in)
	if err != nil {
		t.Fatal(err)
	}
	if replay.ID != task.ID || remote.createCalls != 1 {
		t.Fatalf("replay=%+v calls=%d", replay, remote.createCalls)
	}

	attempt, err := svc.Execute(ctx, 11, task.ID, "execute-key")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.JobID != task.ImageServiceJobID {
		t.Fatalf("attempt=%+v", attempt)
	}
	items, err := svc.Attempts(ctx, 11, task.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestRecipeIsFrozenAndReworkMustFollowParent(t *testing.T) {
	svc, _ := newTestService(t)
	asset, err := svc.CreateAsset(t.Context(), 11, "product.png", "image/png", []byte("recipe-image"))
	if err != nil {
		t.Fatal(err)
	}
	seedTaskRights(t, svc, 11, asset, "listing_main", "ozon")
	first, err := svc.CreateTask(t.Context(), 11, validTaskInput(asset, "recipe-first"))
	if err != nil {
		t.Fatal(err)
	}
	if first.SKUID != 1 || first.RecipeHash == "" || len(first.RecipeManifest) == 0 || first.CandidateRound != 1 {
		t.Fatalf("recipe not frozen: %+v", first)
	}
	rework := validTaskInput(asset, "recipe-rework")
	rework.ParentTaskID, rework.RecipeVersion, rework.CandidateRound = &first.ID, 2, 2
	rework.Recipe.Prompt = "keep the product unchanged on a bright shelf"
	second, err := svc.CreateTask(t.Context(), 11, rework)
	if err != nil {
		t.Fatal(err)
	}
	if second.ParentTaskID == nil || *second.ParentTaskID != first.ID || second.RecipeVersion != 2 || second.RecipeHash == first.RecipeHash {
		t.Fatalf("rework lineage missing: first=%+v second=%+v", first, second)
	}
	bad := validTaskInput(asset, "recipe-bad-rework")
	bad.ParentTaskID, bad.RecipeVersion, bad.CandidateRound = &first.ID, 1, 2
	if _, err := svc.CreateTask(t.Context(), 11, bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid rework chain, got %v", err)
	}
}

func TestCandidateFeedbackAndRecipeSummary(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Now().UTC()
	tasks := []Task{
		{OwnerID: 11, AssetID: 1, SKUID: 1, RecipeKey: "summary-recipe", RecipeVersion: 1, RecipeManifest: json.RawMessage(`{}`), RecipeHash: strings.Repeat("a", 64), CandidateRound: 1, IdempotencyKey: "summary-1", ManifestHash: strings.Repeat("b", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "scene_gallery", Channel: "ozon", Region: "local", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("1", 64), CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)},
		{OwnerID: 11, AssetID: 1, SKUID: 1, RecipeKey: "summary-recipe", RecipeVersion: 1, RecipeManifest: json.RawMessage(`{}`), RecipeHash: strings.Repeat("a", 64), CandidateRound: 1, IdempotencyKey: "summary-2", ManifestHash: strings.Repeat("c", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "scene_gallery", Channel: "ozon", Region: "local", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("2", 64), CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-time.Minute)},
		{OwnerID: 11, AssetID: 1, SKUID: 1, RecipeKey: "summary-recipe", RecipeVersion: 2, RecipeManifest: json.RawMessage(`{}`), RecipeHash: strings.Repeat("d", 64), CandidateRound: 2, IdempotencyKey: "summary-3", ManifestHash: strings.Repeat("e", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "scene_gallery", Channel: "ozon", Region: "local", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("3", 64), CreatedAt: now.Add(-time.Minute), UpdatedAt: now},
	}
	for i := range tasks {
		if err := svc.db.Create(&tasks[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	passed := &Review{OwnerID: 11, TaskID: tasks[0].ID, Decision: "five_axis_review", AssetSHA: tasks[0].OutputBlobID, ProductAuthenticity: ReviewPassed, RightsStatus: ReviewPassed, ChannelRules: ReviewPassed, ClaimsScene: ReviewPassed, TechnicalVisual: ReviewPassed, IdempotencyKey: "summary-gate", RequestHash: strings.Repeat("f", 64)}
	if err := svc.db.Create(passed).Error; err != nil {
		t.Fatal(err)
	}
	selected, err := svc.CreateCandidateFeedback(t.Context(), 11, tasks[0].ID, CandidateFeedbackInput{Outcome: "selected", AssetSHA: tasks[0].OutputBlobID, ReviewSeconds: 30, ErrorRegions: json.RawMessage(`[]`), IdempotencyKey: "feedback-selected", ExpectedVersion: 1})
	if err != nil || selected.Outcome != "selected" {
		t.Fatalf("selected=%+v err=%v", selected, err)
	}
	if _, err := svc.CreateCandidateFeedback(t.Context(), 11, tasks[1].ID, CandidateFeedbackInput{Outcome: "rejected", ReasonCodes: []string{"color"}, AssetSHA: tasks[1].OutputBlobID, ReviewSeconds: 20, ErrorRegions: json.RawMessage(`[{"x":1,"y":2}]`), IdempotencyKey: "feedback-rejected", ExpectedVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCandidateFeedback(t.Context(), 11, tasks[2].ID, CandidateFeedbackInput{Outcome: "rework_requested", ReasonCodes: []string{"scene"}, ReworkInstruction: "use a brighter shelf", AssetSHA: tasks[2].OutputBlobID, ReviewSeconds: 10, ErrorRegions: json.RawMessage(`[]`), IdempotencyKey: "feedback-rework", ExpectedVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&CostEntry{OwnerID: 11, TaskID: tasks[0].ID, Kind: "actual", Category: "provider", Provider: "deterministic", Amount: "1.25", Currency: "USD", ExchangeRate: "1", ExchangeRateSource: "test", ObservedAt: now, BillingStatus: "paid", IdempotencyKey: "summary-cost", RequestHash: strings.Repeat("9", 64), ExpectedTaskVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&Task{OwnerID: 11, AssetID: 1, SKUID: 2, RecipeKey: "summary-recipe", RecipeVersion: 1, RecipeManifest: json.RawMessage(`{}`), RecipeHash: strings.Repeat("8", 64), CandidateRound: 1, IdempotencyKey: "summary-other-sku", ManifestHash: strings.Repeat("7", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "scene_gallery", Channel: "ozon", Region: "local", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("6", 64), CreatedAt: now.Add(-time.Minute), UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	summary, err := svc.RecipeSummary(t.Context(), 11, 1, "summary-recipe")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Candidates != 3 || summary.Selected != 1 || summary.Rejected != 1 || summary.ReworkRequested != 1 || summary.ReworkRounds != 1 || summary.ReviewSeconds != 60 || summary.ActualCost != "1.2500" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestOwnerIsolationReturnsRecordNotFoundWithoutRemoteCall(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	asset, err := svc.CreateAsset(context.Background(), 11, "product.png", "image/png", []byte("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateTask(context.Background(), 22, validTaskInput(asset, "foreign"))
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected hidden foreign asset, got %v", err)
	}
	if remote.createCalls != 0 {
		t.Fatalf("remote called %d times", remote.createCalls)
	}
}

func TestOrdinaryReviewCannotAssertActual(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	err := svc.CreateReview(context.Background(), &Review{OwnerID: 1, TaskID: 1, Decision: "selected", Truth: TruthActual})
	if err != ErrTruthRequiresOwner {
		t.Fatalf("got %v", err)
	}
	var count int64
	if err := svc.db.Model(&Review{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestOutputContentRejectsHashMismatch(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	asset, err := svc.CreateAsset(context.Background(), 11, "product.png", "image/png", []byte("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	seedTaskRights(t, svc, 11, asset, "listing_main", "ozon")
	task, err := svc.CreateTask(context.Background(), 11, validTaskInput(asset, "hash-task"))
	if err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte("expected output"))
	blobID := hex.EncodeToString(expected[:])
	job := remote.jobs[task.ImageServiceJobID]
	job.OwnerID = 11
	job.ManifestHash = task.ManifestHash
	job.Status = "READY"
	job.OutputBlobID = blobID
	remote.jobs[task.ImageServiceJobID] = job
	remote.blobs[blobID] = []byte("tampered output")

	_, _, err = svc.OutputContent(context.Background(), 11, task.ID)
	if !errors.Is(err, ErrOutputHashMismatch) {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
}

func TestReconciliationMergeRecoversOutputAndPreservesTerminalEvidence(t *testing.T) {
	task := &Task{Status: "RECONCILE_REQUIRED", ErrorCode: "INTERNAL_DISPATCH_OUTCOME_UNKNOWN"}
	mergeRemoteTaskState(task, &imageservice.Job{Status: "READY", OutputBlobID: strings.Repeat("a", 64)})
	if task.Status != "READY" || task.ErrorCode != "" || task.OutputBlobID != strings.Repeat("a", 64) {
		t.Fatalf("recovered task=%+v", task)
	}

	terminal := &Task{Status: "FAILED", ErrorCode: "CHARGED_OUTPUT_UNRECOVERABLE"}
	mergeRemoteTaskState(terminal, &imageservice.Job{Status: "RECONCILE_REQUIRED", ErrorCode: "REMOTE_UNKNOWN"})
	if terminal.Status != "FAILED" || terminal.ErrorCode != "CHARGED_OUTPUT_UNRECOVERABLE" {
		t.Fatalf("terminal evidence overwritten: %+v", terminal)
	}

	pendingEvidence := &Task{Status: "RECONCILE_REQUIRED", ErrorCode: "INTERNAL_DISPATCH_OUTCOME_UNKNOWN"}
	mergeRemoteTaskState(pendingEvidence, &imageservice.Job{Status: "FAILED", ErrorCode: "CANCELLED_NO_CHARGE_RECONCILIATION"})
	if pendingEvidence.Status != "RECONCILE_REQUIRED" || pendingEvidence.ErrorCode != "INTERNAL_DISPATCH_OUTCOME_UNKNOWN" {
		t.Fatalf("quiesce window closed reconciliation early: %+v", pendingEvidence)
	}
}

func TestPaidExecutionRequiresOwnerApprovalAndConsumesItOnce(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &Review{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	remote := newFakeImageService()
	remote.available["paid-test"] = true
	svc := NewService(db, dbtest.NewLogger(t), remote, "0123456789abcdef0123456789abcdef")
	nowPolicy := time.Now().UTC()
	if _, err := svc.CreateBudgetPolicy(context.Background(), 11, BudgetPolicyInput{Currency: "USD", PeriodStart: nowPolicy.Add(-time.Hour), PeriodEnd: nowPolicy.Add(time.Hour), TotalAmount: "10.00", IdempotencyKey: "policy"}); err != nil {
		t.Fatal(err)
	}
	task := &Task{OwnerID: 11, AssetID: 1, ImageServiceJobID: "paid-job", IdempotencyKey: "paid-task", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 4, Width: 1024, Height: 1024, Format: "png", Status: "QUEUED"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: task.OwnerID, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Status: "QUEUED"}
	if _, err := svc.Execute(context.Background(), 11, task.ID, "paid-attempt"); err == nil {
		t.Fatal("execution without approval succeeded")
	}
	if remote.authorizedExecuteCalls != 0 {
		t.Fatal("remote called before approval")
	}
	if _, err := svc.ApproveExecution(context.Background(), 22, task.ID, ApprovalInput{Processor: "paid-test", MaxCost: "1.25", Currency: "USD", ExpectedVersion: 4}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign owner approval: %v", err)
	}
	approval, err := svc.ApproveExecution(context.Background(), 11, task.ID, ApprovalInput{Processor: "paid-test", MaxCost: "1.25", Currency: "usd", ExpectedVersion: 4})
	if err != nil {
		t.Fatal(err)
	}
	if approval.Nonce == "" || approval.Currency != "USD" || approval.ConsumedAt != nil {
		t.Fatalf("bad approval: %+v", approval)
	}
	var budgetCost CostEntry
	if err := db.Where("owner_id = ? AND task_id = ? AND kind = ?", 11, task.ID, "estimated").First(&budgetCost).Error; err != nil {
		t.Fatalf("approval did not create cost record: %v", err)
	}
	if budgetCost.Amount != "1.25" || budgetCost.BillingStatus != "estimated" {
		t.Fatalf("bad approval cost: %+v", budgetCost)
	}
	if _, err := svc.Execute(context.Background(), 11, task.ID, "paid-attempt"); err != nil {
		t.Fatal(err)
	}
	if remote.authorizedExecuteCalls != 1 || remote.lastExecutionToken == "" {
		t.Fatal("authorized execution not sent exactly once")
	}
	var saved ExecutionApproval
	if err := db.First(&saved, approval.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.ConsumedAt == nil {
		t.Fatal("approval not marked consumed")
	}
	if _, err := svc.Execute(context.Background(), 11, task.ID, "paid-attempt-2"); err == nil {
		t.Fatal("consumed approval reused")
	}
	if remote.authorizedExecuteCalls != 1 {
		t.Fatal("remote called for replay")
	}
	if err := db.Model(task).Update("status", "READY").Error; err != nil {
		t.Fatal(err)
	}
	finished := remote.jobs[task.ImageServiceJobID]
	finished.Status = "READY"
	remote.jobs[task.ImageServiceJobID] = finished
	if _, err := svc.ApproveExecution(context.Background(), 11, task.ID, ApprovalInput{Processor: "paid-test", MaxCost: "1.25", Currency: "USD", ExpectedVersion: 4}); err == nil {
		t.Fatal("terminal task received a new execution approval")
	}
}

func TestPaidExecutionRejectsLegacyApprovalWithoutCostRecord(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{})
	remote := newFakeImageService()
	remote.available["paid-test"] = true
	svc := NewService(db, dbtest.NewLogger(t), remote, "0123456789abcdef0123456789abcdef")
	task := &Task{OwnerID: 11, AssetID: 1, ImageServiceJobID: "paid-no-cost", IdempotencyKey: "paid-no-cost", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: 11, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: 1, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Status: "QUEUED"}
	now := time.Now().UTC()
	approval := ExecutionApproval{ExecutionID: "legacy", OwnerID: 11, TaskID: task.ID, TaskVersion: 1, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: "1.00", Currency: "USD", Nonce: strings.Repeat("b", 64), ApprovedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := db.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.Execute(context.Background(), 11, task.ID, "attempt")
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Code != "BUDGET_COST_REQUIRED" {
		t.Fatalf("err=%v", err)
	}
	if remote.authorizedExecuteCalls != 0 {
		t.Fatal("remote called without cost record")
	}
}

func TestPaidExecutionConcurrentApprovalIsClaimedOnce(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	base := newFakeImageService()
	base.available["paid-test"] = true
	remote := &countingAuthorizedImageService{fakeImageService: base}
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	task := &Task{OwnerID: 11, AssetID: 1, ImageServiceJobID: "paid-concurrent", IdempotencyKey: "paid-concurrent", ManifestHash: strings.Repeat("a", 64), Operation: "PAID_TEST", Processor: "paid-test", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	base.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: 11, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: 1, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Status: "QUEUED"}
	now := time.Now().UTC()
	approval := ExecutionApproval{ExecutionID: "concurrent", OwnerID: 11, TaskID: task.ID, TaskVersion: 1, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: "1.00", Currency: "USD", Nonce: strings.Repeat("b", 64), ApprovedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := db.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	policy := BudgetPolicy{OwnerID: 11, Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "10.00", IdempotencyKey: "concurrent-policy", RequestHash: strings.Repeat("d", 64)}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	reservation := BudgetReservation{OwnerID: 11, PolicyID: policy.ID, ApprovalID: approval.ID, TaskID: task.ID, TaskVersion: 1, ManifestHash: task.ManifestHash, Provider: "paid-test", Currency: "USD", ReservedAmount: "1.00", State: "reserved"}
	if err := db.Create(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	cost := CostEntry{OwnerID: 11, TaskID: task.ID, Kind: "estimated", Category: "provider", Provider: "paid-test", Amount: "1.00", Currency: "USD", ExchangeRate: "1", ExchangeRateSource: "test", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "concurrent-cost", RequestHash: strings.Repeat("c", 64), ExpectedTaskVersion: 1}
	if err := db.Create(&cost).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = svc.Execute(context.Background(), 11, task.ID, "concurrent-"+strconv.Itoa(i))
		}(i)
	}
	wg.Wait()
	if got := remote.authorizedCalls.Load(); got != 1 {
		t.Fatalf("same approval reached external submit %d times", got)
	}
}
