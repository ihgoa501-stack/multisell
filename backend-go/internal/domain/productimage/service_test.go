package productimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
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
}

func newFakeImageService() *fakeImageService {
	return &fakeImageService{jobs: map[string]imageservice.Job{}, attempts: map[string][]imageservice.Attempt{}, blobs: map[string][]byte{}}
}
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
	j := imageservice.Job{ID: id, OwnerID: in.OwnerID, LingMirrorTaskID: in.LingMirrorTaskID, LingMirrorTaskVersion: in.LingMirrorTaskVersion, Status: "queued", ManifestHash: in.ManifestHash, Operation: in.Operation, Processor: processor, CreatedAt: time.Now(), UpdatedAt: time.Now()}
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
func (f *fakeImageService) EnqueueExecution(_ context.Context, id, key string) (*imageservice.Attempt, error) {
	f.executeCalls++
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
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &Review{})
	client := newFakeImageService()
	return NewService(db, dbtest.NewLogger(t), client), client
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

	in := CreateTaskInput{AssetID: asset.ID, IdempotencyKey: "task-key", Operation: "DETERMINISTIC_RESIZE", Width: 1000, Height: 1000, Format: "png"}
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

func TestOwnerIsolationReturnsRecordNotFoundWithoutRemoteCall(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	asset, err := svc.CreateAsset(context.Background(), 11, "product.png", "image/png", []byte("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateTask(context.Background(), 22, CreateTaskInput{AssetID: asset.ID, IdempotencyKey: "foreign", Operation: "DETERMINISTIC_RESIZE", Width: 100, Height: 100, Format: "png"})
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
	task, err := svc.CreateTask(context.Background(), 11, CreateTaskInput{AssetID: asset.ID, IdempotencyKey: "hash-task", Operation: "DETERMINISTIC_RESIZE", Width: 100, Height: 100, Format: "png"})
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

func TestPaidExecutionRequiresOwnerApprovalAndConsumesItOnce(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &Review{}, &ExecutionApproval{}, &CostEntry{})
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote, "0123456789abcdef0123456789abcdef")
	task := &Task{OwnerID: 11, AssetID: 1, ImageServiceJobID: "paid-job", IdempotencyKey: "paid-task", ManifestHash: strings.Repeat("a", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", Version: 4, Width: 1024, Height: 1024, Format: "png", Status: "QUEUED"}
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
	if _, err := svc.ApproveExecution(context.Background(), 22, task.ID, ApprovalInput{Processor: "openai", MaxCost: "1.25", Currency: "USD", ExpectedVersion: 4}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign owner approval: %v", err)
	}
	approval, err := svc.ApproveExecution(context.Background(), 11, task.ID, ApprovalInput{Processor: "openai", MaxCost: "1.25", Currency: "usd", ExpectedVersion: 4})
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
	if _, err := svc.ApproveExecution(context.Background(), 11, task.ID, ApprovalInput{Processor: "openai", MaxCost: "1.25", Currency: "USD", ExpectedVersion: 4}); err == nil {
		t.Fatal("terminal task received a new execution approval")
	}
}

func TestPaidExecutionRejectsLegacyApprovalWithoutCostRecord(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{})
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote, "0123456789abcdef0123456789abcdef")
	task := &Task{OwnerID: 11, AssetID: 1, ImageServiceJobID: "paid-no-cost", IdempotencyKey: "paid-no-cost", ManifestHash: strings.Repeat("a", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", Version: 1, Width: 100, Height: 100, Format: "png", Status: "QUEUED"}
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
