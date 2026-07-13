package productimage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeControlledPublisher struct {
	mu         sync.Mutex
	calls      int
	last       ControlledPublishRequest
	publishErr error
	receipt    ControlledPublishReceipt
	reconcile  ReconcileResult
}

func TestControlledPublishHTTPPersistsUnsupportedAttempt(t *testing.T) {
	svc, _, _, a := publishFixture(t, nil)
	h := NewPublishHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(42)) })
	r.POST("/product-images/release-attestations/:attestation_id/publish-attempts", h.Execute)
	r.GET("/product-images/publish-attempts/:attempt_id", h.Get)
	w := httptest.NewRecorder()
	path := fmt.Sprintf("/product-images/release-attestations/%d/publish-attempts", a.ID)
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"idempotency_key":"http-unsupported"}`)))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "CONTROLLED_PUBLISHER_UNSUPPORTED") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var attempt ImagePublishAttempt
	if err := svc.db.Where("owner_id=? AND idempotency_key=?", 42, "http-unsupported").First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/product-images/publish-attempts/%d", attempt.ID), nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"unsupported"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func (f *fakeControlledPublisher) PublishControlled(_ context.Context, req ControlledPublishRequest) (ControlledPublishReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.last = req
	return f.receipt, f.publishErr
}

func (f *fakeControlledPublisher) ReconcileControlled(_ context.Context, req ControlledPublishRequest) (ReconcileResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.last = req
	return f.reconcile, nil
}

func (f *fakeControlledPublisher) count() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

func publishFixture(t *testing.T, p ControlledPublisher) (*PublishService, *ReleaseService, *fakeImageService, *ImageReleaseAttestation) {
	t.Helper()
	release, _, remote, set, _, rule := releaseFixture(t)
	a := issueRelease(t, release, set, rule, "publish-attestation", time.Hour)
	registry := NewPublisherRegistry()
	if p != nil {
		if err := registry.Register("ozon", p); err != nil {
			t.Fatal(err)
		}
	}
	return NewPublishService(release.db, remote, release, registry), release, remote, a
}

func TestControlledPublishUnsupportedLegacyAdapterFailsClosed(t *testing.T) {
	svc, release, _, a := publishFixture(t, nil)
	attempt, err := svc.Execute(context.Background(), 42, a.ID, "unsupported")
	if !errors.Is(err, ErrUnsupportedPublisher) || attempt == nil || attempt.Status != PublishAttemptUnsupported {
		t.Fatalf("attempt=%+v err=%v", attempt, err)
	}
	current, err := release.Get(context.Background(), 42, a.ID)
	if err != nil || current.Status != ReleaseStatusIssued {
		t.Fatalf("unsupported publisher consumed attestation: %+v err=%v", current, err)
	}
}

func TestControlledPublishRejectsHashTamperAndCrossOwnerBeforeCall(t *testing.T) {
	p := &fakeControlledPublisher{receipt: ControlledPublishReceipt{RemoteReference: "remote/1"}}
	svc, _, remote, a := publishFixture(t, p)
	if _, err := svc.Execute(context.Background(), 99, a.ID, "other-owner"); err == nil {
		t.Fatal("cross-owner publication succeeded")
	}
	remote.blobs[a.Items[0].BlobID] = []byte("tampered")
	if _, err := svc.Execute(context.Background(), 42, a.ID, "tampered"); !errors.Is(err, ErrReleaseGateBlocked) {
		t.Fatalf("tamper err=%v", err)
	}
	if p.count() != 0 {
		t.Fatalf("publisher calls=%d", p.count())
	}
}

func TestControlledPublishPermanentlyRejectsSandboxOutputBeforeCall(t *testing.T) {
	p := &fakeControlledPublisher{receipt: ControlledPublishReceipt{RemoteReference: "must-not-exist"}}
	svc, _, remote, a := publishFixture(t, p)
	item := a.Items[0]
	var task Task
	if err := svc.db.First(&task, item.TaskID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&task).Updates(map[string]any{"sandbox": true, "watermarked": true, "non_publishable": true}).Error; err != nil {
		t.Fatal(err)
	}
	job := remote.jobs[task.ImageServiceJobID]
	job.Sandbox, job.Watermarked, job.NonPublishable = true, true, true
	remote.jobs[task.ImageServiceJobID] = job
	if _, err := svc.Execute(t.Context(), 42, a.ID, "sandbox-publish"); !errors.Is(err, ErrReleaseGateBlocked) {
		t.Fatalf("sandbox publish err=%v", err)
	}
	if p.count() != 0 {
		t.Fatalf("sandbox bytes reached publisher: %d", p.count())
	}
}

func TestControlledPublishUnknownOutcomeRequiresReconcileWithoutRetry(t *testing.T) {
	p := &fakeControlledPublisher{publishErr: errors.New("response stream lost")}
	svc, release, _, a := publishFixture(t, p)
	attempt, err := svc.Execute(context.Background(), 42, a.ID, "lost-response")
	if !errors.Is(err, ErrReconcileRequired) || attempt.Status != PublishAttemptReconcileRequired || p.count() != 1 {
		t.Fatalf("attempt=%+v calls=%d err=%v", attempt, p.count(), err)
	}
	attestation, _ := release.Get(context.Background(), 42, a.ID)
	if attestation.Status != ReleaseStatusReconcile {
		t.Fatalf("status=%s", attestation.Status)
	}
	if _, err := svc.Execute(context.Background(), 42, a.ID, "lost-response"); !errors.Is(err, ErrReconcileRequired) {
		t.Fatalf("replay err=%v", err)
	}
	if p.count() != 1 {
		t.Fatalf("automatic retry calls=%d", p.count())
	}
	p.reconcile = ReconcileResult{Resolved: true, Success: true, Receipt: ControlledPublishReceipt{RemoteReference: "ozon/media/123", ReceiptEvidence: json.RawMessage(`{"remote_status":"accepted"}`)}, Evidence: json.RawMessage(`{"lookup":"idempotency-key"}`)}
	completed, err := svc.Reconcile(context.Background(), 42, attempt.ID)
	if err != nil || completed.Status != PublishAttemptSucceeded || completed.RemoteReference != "ozon/media/123" || completed.MediaManifestSHA != a.MediaManifestSHA {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}
	attestation, _ = release.Get(context.Background(), 42, a.ID)
	if attestation.Status != ReleaseStatusConsumed || attestation.ConsumedByID != attempt.ID {
		t.Fatalf("attestation=%+v", attestation)
	}
}

func TestControlledPublishConcurrentHundredMakesOneExternalCall(t *testing.T) {
	p := &fakeControlledPublisher{receipt: ControlledPublishReceipt{RemoteReference: "ozon/media/one", ReceiptEvidence: json.RawMessage(`{"ok":true}`)}}
	svc, _, _, a := publishFixture(t, p)
	const workers = 100
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Execute(context.Background(), 42, a.ID, "same-intent")
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, ErrPublishInProgress) {
			t.Fatalf("unexpected err=%v", err)
		}
	}
	if p.count() != 1 {
		t.Fatalf("external calls=%d want=1", p.count())
	}
	var attempt ImagePublishAttempt
	if err := svc.db.WithContext(context.Background()).Where("owner_id=? AND idempotency_key=?", 42, "same-intent").First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.Status != PublishAttemptSucceeded {
		t.Fatalf("status=%s", attempt.Status)
	}
}
