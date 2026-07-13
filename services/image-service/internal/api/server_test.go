package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	jobrunner "github.com/lingmirror/image-service/internal/jobs"
)

func TestDeterministicJobAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	srv := httptest.NewServer(New("secret", jobs, blobs).Handler())
	defer srv.Close()
	var img bytes.Buffer
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			src.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	_ = png.Encode(&img, src)
	blob := request(t, srv, "POST", "/internal/v1/blobs", img.Bytes(), 201)["blob_id"].(string)
	h := sha256.Sum256([]byte("manifest"))
	payload := map[string]any{"owner_id": 1, "idempotency_key": "key-1", "manifest_hash": hex.EncodeToString(h[:]), "operation": "DETERMINISTIC_RESIZE", "input_blob_id": blob, "width": 100, "height": 100, "format": "png"}
	b, _ := json.Marshal(payload)
	first := request(t, srv, "POST", "/internal/v1/jobs", b, 201)
	second := request(t, srv, "POST", "/internal/v1/jobs", b, 201)
	if first["id"] != second["id"] {
		t.Fatal("idempotent replay created another job")
	}
	attemptBody, _ := json.Marshal(map[string]string{"idempotency_key": "execute-key-1"})
	attempt := request(t, srv, "POST", "/internal/v1/jobs/"+first["id"].(string)+"/executions", attemptBody, 202)
	if attempt["status"] != "QUEUED" {
		t.Fatalf("unexpected attempt: %#v", attempt)
	}
	worker, _ := jobrunner.NewWorker(jobs, jobrunner.NewDeterministicExecutor(blobs), "test-worker", time.Second, time.Millisecond)
	handled, err := worker.RunOne(context.Background())
	if err != nil || !handled {
		t.Fatalf("worker failed: handled=%v err=%v", handled, err)
	}
	done := request(t, srv, "GET", "/internal/v1/jobs/"+first["id"].(string), nil, 200)
	if done["status"] != "READY" || done["output_blob_id"] == "" {
		t.Fatalf("unexpected result: %#v", done)
	}
}
func TestUnauthorized(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	r := httptest.NewRequest("POST", "/internal/v1/jobs", nil)
	w := httptest.NewRecorder()
	New("secret", jobs, blobs).Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}

func TestQuiesceRequiresExactIdentityAndStopsQueuedJob(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	manifest := strings.Repeat("a", 64)
	job, _, err := jobs.Create(core.CreateJob{OwnerID: 7, LingMirrorTaskID: "42", LingMirrorTaskVersion: 3, IdempotencyKey: "paid-job", ManifestHash: manifest, Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", InputBlobID: strings.Repeat("b", 64), Width: 1024, Height: 1024, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	handler := New("secret", jobs, blobs).Handler()
	call := func(owner int64) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"owner_id": owner, "lingmirror_task_id": "42", "lingmirror_task_version": 3, "manifest_hash": manifest})
		r := httptest.NewRequest(http.MethodPost, "/internal/v1/jobs/"+job.ID+"/quiesce", bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer secret")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call(8); w.Code != http.StatusConflict {
		t.Fatalf("mismatched identity status=%d body=%s", w.Code, w.Body.String())
	}
	if w := call(7); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "CANCELLED_NO_CHARGE_RECONCILIATION") {
		t.Fatalf("quiesce status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRestrictedBlobResponseCarriesNonPublishableHeaders(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	var encoded bytes.Buffer
	_ = png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	id, err := blobs.Put(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := blobs.MarkRestricted(id, blobstore.Restriction{Sandbox: true, Watermarked: true, NonPublishable: true, Reason: "photoroom_sandbox_output"}); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/internal/v1/blobs/"+id+"/content", nil)
	r.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	New("secret", jobs, blobs).Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	for key, want := range map[string]string{"X-Image-Sandbox": "true", "X-Image-Watermarked": "true", "X-Image-Publishable": "false", "X-Image-Restriction": "photoroom_sandbox_output"} {
		if got := w.Header().Get(key); got != want {
			t.Errorf("%s=%q want=%q", key, got, want)
		}
	}
}

func TestReadyChecksRepositoryAndBlobPersistence(t *testing.T) {
	t.Run("ready and probe leaves no garbage", func(t *testing.T) {
		dir := t.TempDir()
		jobs, err := core.OpenStore(filepath.Join(dir, "jobs.json"))
		if err != nil {
			t.Fatal(err)
		}
		blobRoot := filepath.Join(dir, "blobs")
		blobs, err := blobstore.New(blobRoot)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		New("secret", jobs, blobs).Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		entries, err := os.ReadDir(blobRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("readiness probe left files behind: %v", entries)
		}
	})

	t.Run("unavailable blob returns sanitized 503", func(t *testing.T) {
		dir := t.TempDir()
		jobs, err := core.OpenStore(filepath.Join(dir, "jobs.json"))
		if err != nil {
			t.Fatal(err)
		}
		blobRoot := filepath.Join(dir, "blobs")
		blobs, err := blobstore.New(blobRoot)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(blobRoot); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(blobRoot, []byte("sensitive-path-detail"), 0o600); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		New("secret", jobs, blobs).Handler().ServeHTTP(w, r)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if body := w.Body.String(); !strings.Contains(body, `"code":"NOT_READY"`) || !strings.Contains(body, "blob persistence unavailable") || strings.Contains(body, blobRoot) || strings.Contains(body, "sensitive-path-detail") {
			t.Fatalf("unexpected or leaking response: %s", body)
		}
	})
}

func TestOpenAIProductionJobRequiresConfiguredStrictPaidContract(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	var encoded bytes.Buffer
	_ = png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	inputBlobID, _ := blobs.Put(encoded.Bytes())
	operation := "OPENAI_IMAGE_EDIT"
	valid := core.CreateJob{
		OwnerID: 1, LingMirrorTaskID: "task-1", LingMirrorTaskVersion: 1,
		IdempotencyKey: "paid-key", ManifestHash: strings.Repeat("a", 64),
		Operation: operation, Processor: "openai", Prompt: "put the product in a clean studio scene",
		InputBlobID: inputBlobID, Width: 1024, Height: 1024, Format: "png",
		MaxCost: "1.25", Currency: "USD", Region: "us", ProviderEnvironment: "production",
	}
	closed := httptest.NewServer(New("secret", jobs, blobs).Handler())
	postJobBody(t, closed.URL, valid, http.StatusUnprocessableEntity)
	closed.Close()
	misconfigured := httptest.NewServer(NewConfigured("secret", jobs, blobs, testExecutionKey, Config{PaidOperations: map[string]string{operation: "photoroom"}}).Handler())
	misconfiguredRequest := valid
	misconfiguredRequest.IdempotencyKey = "misconfigured-provider"
	postJobBody(t, misconfigured.URL, misconfiguredRequest, http.StatusUnprocessableEntity)
	misconfigured.Close()

	srv := httptest.NewServer(NewConfigured("secret", jobs, blobs, testExecutionKey, Config{PaidOperations: map[string]string{operation: "openai"}}).Handler())
	defer srv.Close()
	postJobBody(t, srv.URL, valid, http.StatusCreated)

	for name, mutate := range map[string]func(*core.CreateJob){
		"empty prompt":        func(in *core.CreateJob) { in.Prompt = " " },
		"zero cost":           func(in *core.CreateJob) { in.MaxCost = "0" },
		"wrong currency":      func(in *core.CreateJob) { in.Currency = "EUR" },
		"sandbox environment": func(in *core.CreateJob) { in.ProviderEnvironment = "sandbox" },
		"sandbox flag":        func(in *core.CreateJob) { in.Sandbox = true },
		"watermarked flag":    func(in *core.CreateJob) { in.Watermarked = true },
		"nonpublishable flag": func(in *core.CreateJob) { in.NonPublishable = true },
		"unsupported size":    func(in *core.CreateJob) { in.Width = 1200; in.Height = 1200 },
	} {
		t.Run(name, func(t *testing.T) {
			in := valid
			in.IdempotencyKey = "invalid-" + name
			mutate(&in)
			postJobBody(t, srv.URL, in, http.StatusUnprocessableEntity)
		})
	}

	capabilities := request(t, srv, "GET", "/internal/v1/processors", nil, http.StatusOK)
	encodedCapabilities, _ := json.Marshal(capabilities)
	for _, expected := range []string{`"code":"openai"`, `"safety_level":"production_paid"`, `"provider_environment":"production"`, `"available":true`} {
		if !bytes.Contains(encodedCapabilities, []byte(expected)) {
			t.Fatalf("OpenAI capability missing %s: %s", expected, encodedCapabilities)
		}
	}
}

func TestCreateJobRejectsUnknownFieldsAndInvalidDimensions(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	srv := httptest.NewServer(New("secret", jobs, blobs).Handler())
	defer srv.Close()

	base := `{"owner_id":1,"idempotency_key":"key","manifest_hash":"` + strings.Repeat("a", 64) + `","operation":"DETERMINISTIC_RESIZE","input_blob_id":"` + strings.Repeat("b", 64) + `","width":99,"height":100,"format":"png"}`
	request(t, srv, "POST", "/internal/v1/jobs", []byte(base), http.StatusUnprocessableEntity)
	withUnknown := strings.TrimSuffix(base, "}") + `,"provider_key":"secret"}`
	request(t, srv, "POST", "/internal/v1/jobs", []byte(withUnknown), http.StatusUnprocessableEntity)
}

func TestDeterministicJobAcceptsExplicitLocalRegion(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	srv := httptest.NewServer(New("secret", jobs, blobs).Handler())
	defer srv.Close()

	var encoded bytes.Buffer
	_ = png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	inputBlobID, _ := blobs.Put(encoded.Bytes())
	payload := `{"owner_id":1,"idempotency_key":"local-region","manifest_hash":"` + strings.Repeat("a", 64) + `","operation":"DETERMINISTIC_RESIZE","processor":"deterministic","input_blob_id":"` + inputBlobID + `","width":100,"height":100,"format":"png","region":"local"}`
	request(t, srv, "POST", "/internal/v1/jobs", []byte(payload), http.StatusCreated)
}
func request(t *testing.T, s *httptest.Server, method, path string, body []byte, want int) map[string]any {
	t.Helper()
	r, _ := http.NewRequest(method, s.URL+path, bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer secret")
	if len(body) > 0 {
		r.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("%s %s status=%d", method, path, resp.StatusCode)
	}
	var out map[string]any
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		t.Fatal("bad json")
	}
	return out
}
