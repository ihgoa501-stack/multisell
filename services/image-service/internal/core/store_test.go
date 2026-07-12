package core

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func createTestJob(t *testing.T, store *Store, key string) *Job {
	t.Helper()
	job, _, err := store.Create(CreateJob{OwnerID: 1, IdempotencyKey: key, ManifestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Operation: "DETERMINISTIC_RESIZE", InputBlobID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Width: 100, Height: 100, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func TestAttemptLifecycleIsIdempotentAndDurable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	job := createTestJob(t, store, "job-1")

	attempt, replay, err := store.EnqueueAttempt(job.ID, "execution-1")
	if err != nil || replay {
		t.Fatalf("enqueue: replay=%v err=%v", replay, err)
	}
	replayed, replay, err := store.EnqueueAttempt(job.ID, "execution-1")
	if err != nil || !replay || replayed.ID != attempt.ID {
		t.Fatalf("replay: got=%+v replay=%v err=%v", replayed, replay, err)
	}

	claimed, ok, err := store.ClaimAttempt("worker-a", time.Minute)
	if err != nil || !ok || claimed.ID != attempt.ID || claimed.Status != AttemptRunning {
		t.Fatalf("claim: got=%+v ok=%v err=%v", claimed, ok, err)
	}
	if _, ok, err := store.ClaimAttempt("worker-b", time.Minute); err != nil || ok {
		t.Fatalf("active lease was claimed: ok=%v err=%v", ok, err)
	}
	completed, err := store.CompleteAttempt(attempt.ID, "worker-a", AttemptSucceeded, "")
	if err != nil || completed.Status != AttemptSucceeded || completed.LeaseUntil != nil {
		t.Fatalf("complete: got=%+v err=%v", completed, err)
	}

	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	attempts := reopened.ListAttempts(job.ID)
	if len(attempts) != 1 || attempts[0].ID != attempt.ID || attempts[0].Status != AttemptSucceeded {
		t.Fatalf("unexpected reopened attempts: %+v", attempts)
	}
}

func TestAttemptIdempotencyConflictAndSingleActiveAttempt(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	job1 := createTestJob(t, store, "job-1")
	job2 := createTestJob(t, store, "job-2")
	first, _, err := store.EnqueueAttempt(job1.ID, "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueAttempt(job2.ID, "same-key"); !errors.Is(err, ErrAttemptIdempotencyConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if first.Number != 1 {
		t.Fatalf("unexpected first number: %d", first.Number)
	}
	if _, _, err := store.EnqueueAttempt(job1.ID, "retry-key"); !errors.Is(err, ErrJobAlreadyActive) {
		t.Fatalf("expected one active attempt per job, got %v", err)
	}
}

func TestExpiredAttemptLeaseCanBeTakenOver(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	job := createTestJob(t, store, "job-1")
	attempt, _, err := store.EnqueueAttempt(job.ID, "execution-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.ClaimAttempt("worker-a", time.Millisecond); err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}
	time.Sleep(10 * time.Millisecond)
	reclaimed, ok, err := store.ClaimAttempt("worker-b", time.Minute)
	if err != nil || !ok || reclaimed.ID != attempt.ID || reclaimed.LeaseOwner != "worker-b" {
		t.Fatalf("reclaim: got=%+v ok=%v err=%v", reclaimed, ok, err)
	}
	if _, err := store.CompleteAttempt(attempt.ID, "worker-a", AttemptSucceeded, ""); !errors.Is(err, ErrAttemptLeaseLost) {
		t.Fatalf("stale completion should fail, got %v", err)
	}
}
