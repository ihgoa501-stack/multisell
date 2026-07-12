package jobs

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/core"
)

type fakeExecutor struct {
	output string
	err    *ExecutionError
	calls  int
}

func (f *fakeExecutor) Execute(context.Context, core.Job) (string, *ExecutionError) {
	f.calls++
	return f.output, f.err
}

func workerStore(t *testing.T) (*core.Store, *core.Job, *core.Attempt) {
	t.Helper()
	store, err := core.OpenStore(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(core.CreateJob{OwnerID: 1, IdempotencyKey: "job", ManifestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Operation: "DETERMINISTIC_RESIZE", InputBlobID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Width: 100, Height: 100, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	attempt, _, err := store.EnqueueAttempt(job.ID, "execution")
	if err != nil {
		t.Fatal(err)
	}
	return store, job, attempt
}

func TestWorkerCompletesJobAndAttempt(t *testing.T) {
	store, job, _ := workerStore(t)
	executor := &fakeExecutor{output: "output-blob"}
	worker, err := NewWorker(store, executor, "worker-1", time.Second, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	handled, err := worker.RunOne(context.Background())
	if err != nil || !handled {
		t.Fatalf("run: handled=%v err=%v", handled, err)
	}
	done, _ := store.Get(job.ID)
	if done.Status != core.JobReady || done.OutputBlobID != "output-blob" || executor.calls != 1 {
		t.Fatalf("unexpected job/executor: %+v calls=%d", done, executor.calls)
	}
	attempts := store.ListAttempts(job.ID)
	if len(attempts) != 1 || attempts[0].Status != core.AttemptSucceeded {
		t.Fatalf("unexpected attempts: %+v", attempts)
	}
}

func TestWorkerPersistsExecutionFailure(t *testing.T) {
	store, job, _ := workerStore(t)
	executor := &fakeExecutor{err: &ExecutionError{Code: "PROCESSING_FAILED", Err: errors.New("bad image")}}
	worker, _ := NewWorker(store, executor, "worker-1", time.Second, time.Millisecond)
	if handled, err := worker.RunOne(context.Background()); err != nil || !handled {
		t.Fatalf("run: handled=%v err=%v", handled, err)
	}
	failed, _ := store.Get(job.ID)
	if failed.Status != core.JobFailed || failed.ErrorCode != "PROCESSING_FAILED" {
		t.Fatalf("unexpected failed job: %+v", failed)
	}
	attempts := store.ListAttempts(job.ID)
	if len(attempts) != 1 || attempts[0].Status != core.AttemptFailed || attempts[0].ErrorCode != "PROCESSING_FAILED" {
		t.Fatalf("unexpected attempts: %+v", attempts)
	}
}

func TestWorkerReconcilesTerminalJobWithoutExecutingAgain(t *testing.T) {
	store, job, _ := workerStore(t)
	if _, err := store.Transition(job.ID, core.JobQueued, core.JobRunning, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(job.ID, core.JobRunning, core.JobReady, "already-written", ""); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{}
	worker, _ := NewWorker(store, executor, "worker-1", time.Second, time.Millisecond)
	if handled, err := worker.RunOne(context.Background()); err != nil || !handled {
		t.Fatalf("run: handled=%v err=%v", handled, err)
	}
	if executor.calls != 0 {
		t.Fatalf("executor called %d times", executor.calls)
	}
	attempts := store.ListAttempts(job.ID)
	if attempts[0].Status != core.AttemptSucceeded {
		t.Fatalf("unexpected attempt: %+v", attempts[0])
	}
}

func TestWorkerProcessesQueuedAttemptAfterStoreReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	store, err := core.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(core.CreateJob{OwnerID: 1, IdempotencyKey: "job", ManifestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Operation: "DETERMINISTIC_RESIZE", InputBlobID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Width: 100, Height: 100, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueAttempt(job.ID, "execution"); err != nil {
		t.Fatal(err)
	}

	reopened, err := core.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{output: "output-after-reopen"}
	worker, _ := NewWorker(reopened, executor, "worker-1", time.Second, time.Millisecond)
	if handled, err := worker.RunOne(context.Background()); err != nil || !handled {
		t.Fatalf("run: handled=%v err=%v", handled, err)
	}
	done, _ := reopened.Get(job.ID)
	if done.Status != core.JobReady || done.OutputBlobID != "output-after-reopen" {
		t.Fatalf("unexpected reopened job: %+v", done)
	}
}
