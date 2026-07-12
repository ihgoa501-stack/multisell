package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/executionauth"
)

const testExecutionKey = "0123456789abcdef0123456789abcdef"

func TestPaidJobCannotExistWithoutLingMirrorTaskBinding(t *testing.T) {
	store, err := core.OpenStore(filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Create(core.CreateJob{OwnerID: 7, IdempotencyKey: "paid", ManifestHash: strings.Repeat("a", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai"})
	if err == nil {
		t.Fatal("paid job without task/version binding was created")
	}
}

func signTestClaims(t *testing.T, claims executionauth.Claims) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	b, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	input := header + "." + payload
	mac := hmac.New(sha256.New, []byte(testExecutionKey))
	_, _ = mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestPaidExecutionAuthorizationRejectsEveryTargetMismatchBeforeEnqueue(t *testing.T) {
	dir := t.TempDir()
	store, _ := core.OpenStore(filepath.Join(dir, "jobs.json"))
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	job, _, err := store.Create(core.CreateJob{OwnerID: 7, LingMirrorTaskID: "42", LingMirrorTaskVersion: 3, IdempotencyKey: "paid-job", ManifestHash: strings.Repeat("a", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", InputBlobID: strings.Repeat("b", 64), Width: 1024, Height: 1024, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New("secret", store, blobs, testExecutionKey).Handler())
	defer srv.Close()
	now := time.Now().UTC()
	base := executionauth.Claims{ApprovalExecutionID: "approval-1", TaskID: "42", TaskVersion: 3, OwnerID: 7, JobID: job.ID, ManifestHash: job.ManifestHash, Operation: job.Operation, Processor: job.Processor, MaxCost: "1.25", Currency: "USD", Nonce: "nonce-valid", IssuedAt: now.Unix(), NotBefore: now.Add(-time.Second).Unix(), ExpiresAt: now.Add(time.Minute).Unix(), Audience: executionauth.Audience}
	cases := map[string]func(*executionauth.Claims){
		"wrong audience": func(c *executionauth.Claims) { c.Audience = "other" },
		"expired": func(c *executionauth.Claims) {
			c.IssuedAt = now.Add(-2 * time.Minute).Unix()
			c.NotBefore = c.IssuedAt
			c.ExpiresAt = now.Add(-time.Second).Unix()
		},
		"wrong job":       func(c *executionauth.Claims) { c.JobID = "wrong" },
		"wrong task":      func(c *executionauth.Claims) { c.TaskID = "43" },
		"wrong version":   func(c *executionauth.Claims) { c.TaskVersion = 4 },
		"wrong owner":     func(c *executionauth.Claims) { c.OwnerID = 8 },
		"wrong manifest":  func(c *executionauth.Claims) { c.ManifestHash = strings.Repeat("c", 64) },
		"wrong operation": func(c *executionauth.Claims) { c.Operation = "OTHER" },
		"wrong processor": func(c *executionauth.Claims) { c.Processor = "adobe" },
		"zero max cost":   func(c *executionauth.Claims) { c.MaxCost = "0" },
		"wrong currency":  func(c *executionauth.Claims) { c.Currency = "usd" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := base
			c.Nonce = "nonce-" + name
			mutate(&c)
			postExecution(t, srv.URL, job.ID, signTestClaims(t, c), http.StatusForbidden)
			if got := len(store.ListAttempts(job.ID)); got != 0 {
				t.Fatalf("invalid token enqueued %d attempts", got)
			}
		})
	}
	t.Run("tampered signature", func(t *testing.T) {
		token := signTestClaims(t, base)
		parts := strings.Split(token, ".")
		if parts[2][0] == 'A' {
			parts[2] = "B" + parts[2][1:]
		} else {
			parts[2] = "A" + parts[2][1:]
		}
		token = strings.Join(parts, ".")
		postExecution(t, srv.URL, job.ID, token, http.StatusForbidden)
		if got := len(store.ListAttempts(job.ID)); got != 0 {
			t.Fatalf("tampered token enqueued %d attempts", got)
		}
	})
}

func TestPaidExecutionAuthorizationIsPersistedAndConsumedExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")
	store, _ := core.OpenStore(path)
	blobs, _ := blobstore.New(filepath.Join(dir, "blobs"))
	job, _, _ := store.Create(core.CreateJob{OwnerID: 7, LingMirrorTaskID: "42", LingMirrorTaskVersion: 3, IdempotencyKey: "paid-job", ManifestHash: strings.Repeat("a", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", InputBlobID: strings.Repeat("b", 64), Width: 1024, Height: 1024, Format: "png"})
	now := time.Now().UTC()
	claims := executionauth.Claims{ApprovalExecutionID: "approval-1", TaskID: "42", TaskVersion: 3, OwnerID: 7, JobID: job.ID, ManifestHash: job.ManifestHash, Operation: job.Operation, Processor: job.Processor, MaxCost: "1.25", Currency: "USD", Nonce: "single-use-nonce", IssuedAt: now.Unix(), NotBefore: now.Add(-time.Second).Unix(), ExpiresAt: now.Add(time.Minute).Unix(), Audience: executionauth.Audience}
	token := signTestClaims(t, claims)
	srv := httptest.NewServer(New("secret", store, blobs, testExecutionKey).Handler())
	postExecution(t, srv.URL, job.ID, token, http.StatusAccepted)
	postExecution(t, srv.URL, job.ID, token, http.StatusConflict)
	srv.Close()
	if got := len(store.ListAttempts(job.ID)); got != 1 {
		t.Fatalf("got %d attempts", got)
	}
	reopened, err := core.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	srv = httptest.NewServer(New("secret", reopened, blobs, testExecutionKey).Handler())
	defer srv.Close()
	postExecution(t, srv.URL, job.ID, token, http.StatusConflict)
	if got := len(reopened.ListAttempts(job.ID)); got != 1 {
		t.Fatalf("replay after restart created attempt: %d", got)
	}
}

func postExecution(t *testing.T, base, jobID, token string, want int) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"idempotency_key": "attempt-1", "execution_token": token})
	req, _ := http.NewRequest(http.MethodPost, base+"/internal/v1/jobs/"+jobID+"/executions", bytes.NewReader(body))
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
