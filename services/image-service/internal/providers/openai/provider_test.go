package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func stores(t *testing.T) (*blobstore.Store, string) {
	t.Helper()
	s, err := blobstore.New(filepath.Join(t.TempDir(), "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Put(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	return s, id
}

func TestUnavailableWithoutAPIKey(t *testing.T) {
	p, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Available() {
		t.Fatal("provider should fail closed without API key")
	}
	_, executionErr := p.Execute(context.Background(), core.Job{Operation: Operation})
	if executionErr == nil || executionErr.Code != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("unexpected error: %+v", executionErr)
	}
}

func TestExecuteSendsOfficialMultipartContractAndStoresSanitizedOutput(t *testing.T) {
	blobs, inputID := stores(t)
	output := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/edits" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing auth")
		}
		if r.Header.Get("Idempotency-Key") != "job-idempotency" {
			t.Errorf("missing stable idempotency key")
		}
		if err := r.ParseMultipartForm(12 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		for k, want := range map[string]string{"model": "gpt-image-2", "prompt": "put the unchanged product on a clean studio background", "size": "1024x1024", "output_format": "png"} {
			if got := r.FormValue(k); got != want {
				t.Errorf("%s=%q want %q", k, got, want)
			}
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			t.Errorf("image: %v", err)
			return
		}
		defer file.Close()
		if header.Filename != "input.png" {
			t.Errorf("filename=%q", header.Filename)
		}
		body, _ := io.ReadAll(file)
		if !bytes.Equal(body, output) {
			t.Error("input bytes differ")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-request-id", "req_success")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]string{"b64_json": base64.StdEncoding.EncodeToString(output)}}})
	}))
	defer server.Close()
	p, err := New(Config{APIKey: "secret", BaseURL: server.URL, Blobs: blobs, HTTPClient: server.Client(), MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	id, executionErr := p.Execute(context.Background(), core.Job{Operation: Operation, IdempotencyKey: "job-idempotency", InputBlobID: inputID, Prompt: "put the unchanged product on a clean studio background", Width: 1024, Height: 1024})
	if executionErr != nil {
		t.Fatalf("execute: %+v", executionErr)
	}
	if _, err := blobs.GetMetadata(id); err != nil {
		t.Fatalf("stored output: %v", err)
	}
}

func TestRetriesOnly429And5xxAndPreservesRequestID(t *testing.T) {
	blobs, inputID := stores(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		w.Header().Set("x-request-id", "req_retry")
		if n == 1 {
			http.Error(w, `{"error":{"message":"busy","type":"server_error","code":"busy"}}`, http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]string{"b64_json": base64.StdEncoding.EncodeToString(testPNG(t))}}})
	}))
	defer server.Close()
	p, _ := New(Config{APIKey: "secret", BaseURL: server.URL, Blobs: blobs, HTTPClient: server.Client(), MaxAttempts: 2, RetryDelay: time.Millisecond})
	if _, err := p.Execute(context.Background(), core.Job{Operation: Operation, InputBlobID: inputID, Prompt: "edit", Width: 1024, Height: 1024}); err != nil {
		t.Fatalf("execute: %+v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}

	calls.Store(0)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("x-request-id", "req_bad")
		http.Error(w, `{"error":{"message":"bad prompt","type":"invalid_request_error","code":"invalid_prompt"}}`, http.StatusBadRequest)
	}))
	defer bad.Close()
	p, _ = New(Config{APIKey: "secret", BaseURL: bad.URL, Blobs: blobs, HTTPClient: bad.Client(), MaxAttempts: 3})
	_, got := p.Execute(context.Background(), core.Job{Operation: Operation, InputBlobID: inputID, Prompt: "edit", Width: 1024, Height: 1024})
	if got == nil || got.Code != "PROVIDER_REJECTED" || got.RequestID != "req_bad" || got.Retryable {
		t.Fatalf("unexpected error: %+v", got)
	}
	if calls.Load() != 1 {
		t.Fatalf("400 was retried: calls=%d", calls.Load())
	}
}

func TestRejectsOversizedMalformedAndEmptyResponses(t *testing.T) {
	blobs, inputID := stores(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"oversized", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), MaxResponseBytes+1))
		}},
		{"malformed", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":[{"b64_json":"%%%"}]}`)) }},
		{"empty", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(tc.handler)
			defer s.Close()
			p, _ := New(Config{APIKey: "secret", BaseURL: s.URL, Blobs: blobs, HTTPClient: s.Client(), MaxAttempts: 1})
			_, got := p.Execute(context.Background(), core.Job{Operation: Operation, InputBlobID: inputID, Prompt: "edit", Width: 1024, Height: 1024})
			if got == nil || got.Code != "OUTPUT_INVALID" {
				t.Fatalf("unexpected error: %+v", got)
			}
		})
	}
}

func TestTimeoutIsBoundedAndNotRetried(t *testing.T) {
	blobs, inputID := stores(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	p, _ := New(Config{APIKey: "secret", BaseURL: server.URL, Blobs: blobs, HTTPClient: &http.Client{Timeout: 5 * time.Millisecond}, MaxAttempts: 3})
	_, got := p.Execute(context.Background(), core.Job{Operation: Operation, InputBlobID: inputID, Prompt: "edit", Width: 1024, Height: 1024})
	if got == nil || got.Code != "PROVIDER_NETWORK_ERROR" || got.Retryable {
		t.Fatalf("unexpected error: %+v", got)
	}
	if calls.Load() != 1 {
		t.Fatalf("network timeout was retried: calls=%d", calls.Load())
	}
}

func TestMultipartWriterUsesAFilePart(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := writeMultipart(w, []byte("png"), "input.png", "gpt-image-2", "prompt", "1024x1024"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	if !strings.Contains(b.String(), `name="image"; filename="input.png"`) {
		t.Fatal("missing image file part")
	}
}
