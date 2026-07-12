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

func TestPaidOpenAIJobIsNotReachableThroughCurrentExecutionAPI(t *testing.T) {
	dir := t.TempDir()
	jobs, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	srv := httptest.NewServer(New("secret", jobs, blobs).Handler())
	defer srv.Close()
	payload, _ := json.Marshal(map[string]any{
		"owner_id": 1, "idempotency_key": "paid-key",
		"manifest_hash": strings.Repeat("a", 64), "operation": "OPENAI_IMAGE_EDIT",
		"input_blob_id": strings.Repeat("b", 64), "prompt": "edit", "width": 1024, "height": 1024,
	})
	result := request(t, srv, "POST", "/internal/v1/jobs", payload, http.StatusUnprocessableEntity)
	errorBody, ok := result["error"].(map[string]any)
	if !ok || errorBody["code"] != "VALIDATION_ERROR" {
		t.Fatalf("unexpected failure: %#v", result)
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
