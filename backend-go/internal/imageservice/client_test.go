package imageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientWorkflowAndAuthentication(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 12, 9, 0, 0, 0, time.UTC)
	job := Job{ID: "job-1", OwnerID: 7, IdempotencyKey: "idem", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", InputBlobID: "blob-in", Width: 10, Height: 20, Format: "png", Status: "QUEUED", Version: 1, CreatedAt: now, UpdatedAt: now}
	attempt := Attempt{ID: "attempt-1", JobID: job.ID, IdempotencyKey: "exec-idem", Number: 1, Status: "QUEUED", CreatedAt: now}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer shared-secret" {
			t.Fatalf("authorization = %q", got)
		}
		isBlobContent := strings.Contains(r.URL.Path, "/blobs/") && strings.HasSuffix(r.URL.Path, "/content")
		if !isBlobContent && r.Header.Get("Accept") != "application/json" {
			t.Fatal("missing JSON accept header")
		}
		if isBlobContent && r.Header.Get("Accept") != "image/png, image/jpeg" {
			t.Fatal("missing image accept header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /internal/v1/blobs":
			if r.Header.Get("Content-Type") != "image/png" {
				t.Fatalf("blob content type = %q", r.Header.Get("Content-Type"))
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "PNG" {
				t.Fatalf("blob = %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"blob_id":"blob-in"}`)
		case "POST /internal/v1/jobs":
			var got CreateJobRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got.OwnerID != 7 {
				t.Fatalf("request = %#v err=%v", got, err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(job)
		case "GET /internal/v1/jobs/job-1":
			_ = json.NewEncoder(w).Encode(job)
		case "POST /internal/v1/jobs/job-1/executions":
			var got EnqueueExecutionRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got.IdempotencyKey != "exec-idem" {
				t.Fatalf("execution request = %#v err=%v", got, err)
			}
			if got.ExecutionToken != "" && got.ExecutionToken != "server-only-token" {
				t.Fatalf("unexpected execution token")
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(attempt)
		case "GET /internal/v1/jobs/job-1/attempts":
			_ = json.NewEncoder(w).Encode(AttemptList{Items: []Attempt{attempt}})
		case "GET /internal/v1/blobs/" + strings.Repeat("a", 64) + "/content":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNG-BYTES"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 1<<20, nil)
	ctx := context.Background()
	blob, err := client.PutBlob(ctx, "image/png", strings.NewReader("PNG"))
	if err != nil || blob.BlobID != "blob-in" {
		t.Fatalf("PutBlob = %#v, %v", blob, err)
	}
	created, err := client.CreateJob(ctx, CreateJobRequest{OwnerID: 7})
	if err != nil || created.ID != job.ID {
		t.Fatalf("CreateJob = %#v, %v", created, err)
	}
	got, err := client.GetJob(ctx, job.ID)
	if err != nil || got.Status != "QUEUED" {
		t.Fatalf("GetJob = %#v, %v", got, err)
	}
	queued, err := client.EnqueueExecution(ctx, job.ID, "exec-idem")
	if err != nil || queued.ID != attempt.ID {
		t.Fatalf("EnqueueExecution = %#v, %v", queued, err)
	}
	if _, err := client.EnqueueAuthorizedExecution(ctx, job.ID, "exec-idem", "server-only-token"); err != nil {
		t.Fatalf("authorized execution: %v", err)
	}
	attempts, err := client.ListAttempts(ctx, job.ID)
	if err != nil || len(attempts) != 1 || attempts[0].ID != attempt.ID {
		t.Fatalf("ListAttempts = %#v, %v", attempts, err)
	}
	content, media, err := client.GetBlob(ctx, strings.Repeat("a", 64))
	if err != nil || string(content) != "PNG-BYTES" || media != "image/png" {
		t.Fatalf("GetBlob = %q %q %v", content, media, err)
	}
}

func TestStructuredAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":{"code":"IDEMPOTENCY_CONFLICT","message":"manifest differs","details":{"job_id":"one"}}}`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, 1024, nil)
	_, err := client.GetJob(context.Background(), "job-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 || apiErr.Code != "IDEMPOTENCY_CONFLICT" || !bytes.Contains(apiErr.Details, []byte("job_id")) {
		t.Fatalf("error = %#v", err)
	}
}

func TestStrictResponseParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{"unknown field", `{"blob_id":"one","unexpected":true}`},
		{"multiple values", `{"blob_id":"one"}{"blob_id":"two"}`},
		{"missing identity", `{"blob_id":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, tt.body) }))
			defer server.Close()
			client := newTestClient(t, server.URL, 1024, nil)
			_, err := client.PutBlob(context.Background(), "image/png", strings.NewReader("x"))
			var protocolErr *ProtocolError
			if !errors.As(err, &protocolErr) {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestResponseLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 33))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, 32, nil)
	_, err := client.GetJob(context.Background(), "job-1")
	var tooLarge *ResponseTooLargeError
	if !errors.As(err, &tooLarge) || tooLarge.Limit != 32 {
		t.Fatalf("error = %#v", err)
	}
}

func TestContextCancellationAndTimeout(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, SharedSecret: "secret", Timeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetJob(context.Background(), "job-1")
	<-started
	if err == nil || !strings.Contains(err.Error(), "Client.Timeout") {
		t.Fatalf("timeout error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.GetJob(ctx, "job-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestConfigAndInputValidation(t *testing.T) {
	t.Parallel()
	bad := []Config{
		{},
		{BaseURL: "ftp://example.test", SharedSecret: "x"},
		{BaseURL: "http://user@example.test", SharedSecret: "x"},
		{BaseURL: "http://example.test?x=1", SharedSecret: "x"},
		{BaseURL: "http://example.test", SharedSecret: " "},
		{BaseURL: "http://example.test", SharedSecret: "x", Timeout: -1},
		{BaseURL: "http://example.test", SharedSecret: "x", MaxResponseBytes: -1},
	}
	for _, cfg := range bad {
		if _, err := New(cfg); err == nil {
			t.Fatalf("accepted config %#v", cfg)
		}
	}
	client := newTestClient(t, "http://example.test", 1024, &http.Client{Timeout: time.Hour})
	if client.httpClient.Timeout != defaultTimeout {
		t.Fatalf("timeout = %v", client.httpClient.Timeout)
	}
	if _, err := client.GetJob(context.Background(), ""); err == nil {
		t.Fatal("accepted empty job id")
	}
	if _, err := client.GetJob(context.Background(), "a/b"); err == nil {
		t.Fatal("accepted path job id")
	}
	if _, err := client.EnqueueExecution(context.Background(), "job-1", ""); err == nil {
		t.Fatal("accepted empty execution idempotency key")
	}
	if _, err := client.EnqueueAuthorizedExecution(context.Background(), "job-1", "key", ""); err == nil {
		t.Fatal("accepted empty execution token")
	}
	if _, err := client.PutBlob(context.Background(), "", nil); err == nil {
		t.Fatal("accepted nil blob body")
	}
}

func TestMalformedErrorResponseIsProtocolError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `<html>proxy error</html>`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, 1024, nil)
	_, err := client.GetJob(context.Background(), "job-1")
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) || protocolErr.StatusCode != 502 {
		t.Fatalf("error = %#v", err)
	}
}

func newTestClient(t *testing.T, baseURL string, limit int64, httpClient *http.Client) *Client {
	t.Helper()
	client, err := New(Config{BaseURL: baseURL, SharedSecret: "shared-secret", MaxResponseBytes: limit, HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
