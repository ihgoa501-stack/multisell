package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
)

func sandboxTinyPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestPhotoroomCapabilityAndCreateJobAreGated(t *testing.T) {
	dir := t.TempDir()
	repo, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	input, _ := blobs.Put(sandboxTinyPNG(t))
	operation := "PHOTOROOM_WHITE_BACKGROUND_SANDBOX"
	request := core.CreateJob{OwnerID: 1, LingMirrorTaskID: "task-1", LingMirrorTaskVersion: 1, IdempotencyKey: "photo-1", ManifestHash: strings.Repeat("a", 64), Operation: operation, Processor: "photoroom", InputBlobID: input, Width: 1000, Height: 1000, Format: "png", MaxCost: "0", Currency: "USD", Region: "us", ProviderEnvironment: "sandbox", Sandbox: true, Watermarked: true, NonPublishable: true}

	closed := httptest.NewServer(New("secret", repo, blobs, testExecutionKey).Handler())
	defer closed.Close()
	postJobBody(t, closed.URL, request, http.StatusUnprocessableEntity)

	open := httptest.NewServer(NewConfigured("secret", repo, blobs, testExecutionKey, Config{PaidOperations: map[string]string{operation: "photoroom"}}).Handler())
	defer open.Close()
	postJobBody(t, open.URL, request, http.StatusCreated)
	req, _ := http.NewRequest(http.MethodGet, open.URL+"/internal/v1/processors", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(body)
	for _, expected := range []string{`"safety_level":"sandbox_only"`, `"quota_remaining":1`, `"non_publishable":true`, `"region":"us"`} {
		if !bytes.Contains(encoded, []byte(expected)) {
			t.Fatalf("capability missing %s: %s", expected, encoded)
		}
	}
}

func postJobBody(t *testing.T, base string, in core.CreateJob, want int) {
	t.Helper()
	b, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, base+"/internal/v1/jobs", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("status=%d want=%d", resp.StatusCode, want)
	}
}
