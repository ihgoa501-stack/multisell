package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/core"
)

func integrationStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("IMAGE_SERVICE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("IMAGE_SERVICE_TEST_DATABASE_URL is unset; PostgreSQL integration was not verified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatalf("migrate PostgreSQL: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func unique(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func createInput(key, operation string) core.CreateJob {
	input := core.CreateJob{OwnerID: time.Now().UnixNano(), IdempotencyKey: key,
		ManifestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Operation:    operation, Processor: "deterministic", InputBlobID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Width: 100, Height: 100, Format: "png"}
	if operation != "DETERMINISTIC_RESIZE" {
		input.LingMirrorTaskID = unique("task-")
		input.LingMirrorTaskVersion = 1
	}
	return input
}

func cleanupOwner(t *testing.T, s *Store, ownerID int64) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = s.db.Exec(`DELETE FROM image_consumed_nonces WHERE attempt_id IN
			(SELECT a.id FROM image_attempts a JOIN image_jobs j ON j.id=a.job_id WHERE j.owner_id=$1)`, ownerID)
		_, _ = s.db.Exec(`DELETE FROM image_attempts WHERE job_id IN (SELECT id FROM image_jobs WHERE owner_id=$1)`, ownerID)
		_, _ = s.db.Exec(`DELETE FROM image_jobs WHERE owner_id=$1`, ownerID)
	})
}

func TestPostgresConcurrentJobAndAttemptIdempotency(t *testing.T) {
	s := integrationStore(t)
	in := createInput(unique("job-"), "DETERMINISTIC_RESIZE")
	cleanupOwner(t, s, in.OwnerID)
	const callers = 100
	ids := make(chan string, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, _, err := s.Create(in)
			if err != nil {
				errs <- err
				return
			}
			ids <- j.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent create: %v", err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Errorf("idempotent create returned %s and %s", first, id)
		}
	}

	attemptKey := unique("attempt-")
	attemptIDs := make(chan string, callers)
	attemptErrs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, _, err := s.EnqueueAttempt(first, attemptKey)
			if err != nil {
				attemptErrs <- err
				return
			}
			attemptIDs <- a.ID
		}()
	}
	wg.Wait()
	close(attemptIDs)
	close(attemptErrs)
	for err := range attemptErrs {
		t.Errorf("concurrent attempt: %v", err)
	}
	var firstAttempt string
	for id := range attemptIDs {
		if firstAttempt == "" {
			firstAttempt = id
		}
		if id != firstAttempt {
			t.Errorf("attempt replay returned different IDs")
		}
	}
}

func TestPostgresNonceConsumptionAndLeaseTakeover(t *testing.T) {
	s := integrationStore(t)
	in := createInput(unique("paid-job-"), "CREATIVE_GENERATE")
	cleanupOwner(t, s, in.OwnerID)
	j, _, err := s.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	nonce := unique("nonce-")
	a, err := s.EnqueueAuthorizedAttempt(j.ID, unique("paid-attempt-"), nonce)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.EnqueueAuthorizedAttempt(j.ID, unique("replay-"), nonce); !errors.Is(err, core.ErrNonceConsumed) {
		t.Fatalf("want nonce consumed, got %v", err)
	}

	claimed, ok, err := s.ClaimAttempt("worker-a", 25*time.Millisecond)
	if err != nil || !ok || claimed.ID != a.ID {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	time.Sleep(35 * time.Millisecond)
	taken, ok, err := s.ClaimAttempt("worker-b", time.Second)
	if err != nil || !ok || taken.ID != a.ID {
		t.Fatalf("takeover: ok=%v err=%v", ok, err)
	}
	if _, err := s.CompleteAttempt(a.ID, "worker-a", core.AttemptSucceeded, ""); !errors.Is(err, core.ErrAttemptLeaseLost) {
		t.Fatalf("stale completion got %v", err)
	}
	if _, err := s.CompleteAttempt(a.ID, "worker-b", core.AttemptSucceeded, ""); err != nil {
		t.Fatal(err)
	}
}
