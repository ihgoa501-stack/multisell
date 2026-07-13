package photoroom

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/jobs"
)

const testSecret = "photoroom-super-secret-never-log-this"

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 96, 64))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSandboxWatermarkIsPresentAtExactPixelsAfterReencode(t *testing.T) {
	encoded, err := applyAndVerifySandboxWatermark(pngBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	const unitsWide, unitsHigh = 45, 11
	scale := img.Bounds().Dx() / unitsWide
	if byHeight := img.Bounds().Dy() / (unitsHigh * 3); byHeight < scale {
		scale = byHeight
	}
	if scale < 1 {
		scale = 1
	}
	x0 := img.Bounds().Min.X + (img.Bounds().Dx()-unitsWide*scale)/2
	y0 := img.Bounds().Max.Y - unitsHigh*scale
	if !hasExactSandboxWatermark(img, x0, y0, scale) {
		t.Fatal("exact SANDBOX marker pixels are missing")
	}
	tampered := image.NewNRGBA(img.Bounds())
	draw.Draw(tampered, img.Bounds(), img, img.Bounds().Min, draw.Src)
	tampered.Set(x0, y0, color.NRGBA{A: 255})
	if hasExactSandboxWatermark(tampered, x0, y0, scale) {
		t.Fatal("tampered marker was accepted")
	}
}

func fixture(t *testing.T) (*blobstore.Store, string) {
	t.Helper()
	s, err := blobstore.New(filepath.Join(t.TempDir(), "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Put(pngBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	return s, id
}

func configured(t *testing.T, server *httptest.Server, blobs *blobstore.Store) *Provider {
	t.Helper()
	p, err := New(Config{APIKey: testSecret, BaseURL: server.URL, Blobs: blobs, Repository: testRepository(t), HTTPClient: server.Client(), TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func testRepository(t *testing.T) core.Repository {
	t.Helper()
	s, err := core.OpenStore(filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestUnavailableUnlessAllPrivacyAndSandboxGatesConfirmed(t *testing.T) {
	blobs, _ := fixture(t)
	for name, cfg := range map[string]Config{
		"no key":                        {Blobs: blobs, Repository: testRepository(t), TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true},
		"training unknown":              {APIKey: testSecret, Blobs: blobs, Repository: testRepository(t), SandboxAccountConfirmed: true},
		"account not confirmed sandbox": {APIKey: testSecret, Blobs: blobs, Repository: testRepository(t), TrainingOptOutConfirmed: true},
		"no blob store":                 {APIKey: testSecret, Repository: testRepository(t), TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true},
		"no durable repository":         {APIKey: testSecret, Blobs: blobs, TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true},
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := New(cfg)
			if p.Available() {
				t.Fatal("provider must be unavailable")
			}
			_, got := p.Execute(context.Background(), core.Job{})
			if got == nil || got.Code != "PROVIDER_UNAVAILABLE" {
				t.Fatalf("error=%+v", got)
			}
			if strings.Contains(got.Error(), testSecret) {
				t.Fatal("credential leaked")
			}
		})
	}
}

func TestBaseURLAllowlistRejectsCredentialQueryAndForeignHost(t *testing.T) {
	blobs, _ := fixture(t)
	for _, raw := range []string{"https://user:pass@image-api.photoroom.com", "https://image-api.photoroom.com?key=secret", "https://evil.example", "http://image-api.photoroom.com"} {
		if _, err := New(Config{APIKey: testSecret, BaseURL: raw, Blobs: blobs, Repository: testRepository(t), TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true}); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestRedirectIsNeverFollowedAndCredentialNeverCrossesHost(t *testing.T) {
	blobs, inputID := fixture(t)
	var redirectedCalls atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		redirectedCalls.Add(1)
		if r.Header.Get("X-Api-Key") != "" {
			t.Error("credential crossed host")
		}
	}))
	defer foreign.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.URL+"/stolen", http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	p := configured(t, origin, blobs)
	_, got := p.Execute(context.Background(), core.Job{ID: "redirect-job", Operation: WhiteBackground, InputBlobID: inputID})
	if got == nil || got.Code != "PROVIDER_REJECTED" {
		t.Fatalf("error=%+v", got)
	}
	if redirectedCalls.Load() != 0 {
		t.Fatalf("redirect target calls=%d", redirectedCalls.Load())
	}
}

func TestConcurrentProvidersShareDurableSingleSubmitClaim(t *testing.T) {
	blobs, inputID := fixture(t)
	repo := testRepository(t)
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(t))
	}))
	defer s.Close()
	makeProvider := func() *Provider {
		p, err := New(Config{APIKey: testSecret, BaseURL: s.URL, Blobs: blobs, Repository: repo, HTTPClient: s.Client(), TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	p1, p2 := makeProvider(), makeProvider()
	job := core.Job{ID: "one-job", Operation: WhiteBackground, InputBlobID: inputID}
	start := make(chan struct{})
	results := make(chan *jobs.ExecutionError, 2)
	for _, p := range []*Provider{p1, p2} {
		go func(p *Provider) { <-start; _, e := p.Execute(context.Background(), job); results <- e }(p)
	}
	close(start)
	e1, e2 := <-results, <-results
	if calls.Load() != 1 {
		t.Fatalf("external calls=%d", calls.Load())
	}
	if !((e1 == nil && e2 != nil && e2.Code == "RECONCILE_REQUIRED") || (e2 == nil && e1 != nil && e1.Code == "RECONCILE_REQUIRED")) {
		t.Fatalf("errors=%v / %v", e1, e2)
	}
}

func TestOfficialFileMultipartAndPermanentSandboxRestriction(t *testing.T) {
	blobs, inputID := fixture(t)
	input, _ := blobs.Get(inputID)
	for _, tc := range []struct {
		operation string
		fields    map[string]string
	}{
		{RemoveBackground, map[string]string{"removeBackground": "true", "background.color": "transparent"}},
		{WhiteBackground, map[string]string{"removeBackground": "true", "background.color": "FFFFFF"}},
		{AIShadow, map[string]string{"removeBackground": "true", "background.color": "FFFFFF", "shadow.mode": "ai.soft"}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/v2/edit" || r.URL.RawQuery != "" {
					t.Errorf("request=%s %s", r.Method, r.URL.String())
				}
				if r.Header.Get("X-Api-Key") != testSecret {
					t.Error("missing API key")
				}
				if err := r.ParseMultipartForm(12 << 20); err != nil {
					t.Errorf("multipart: %v", err)
					return
				}
				if r.FormValue("imageUrl") != "" || r.FormValue("url") != "" {
					t.Error("URL input must never be sent")
				}
				for k, want := range tc.fields {
					if got := r.FormValue(k); got != want {
						t.Errorf("%s=%q want=%q", k, got, want)
					}
				}
				if r.FormValue("format") != "png" {
					t.Errorf("format=%q", r.FormValue("format"))
				}
				file, header, err := r.FormFile("image_file")
				if err != nil {
					t.Errorf("file: %v", err)
					return
				}
				defer file.Close()
				got, _ := io.ReadAll(file)
				if header.Filename != "input.png" || !bytes.Equal(got, input) {
					t.Error("uploaded bytes differ")
				}
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("x-request-id", "req-fixture")
				_, _ = w.Write(pngBytes(t))
			}))
			defer s.Close()
			p := configured(t, s, blobs)
			result, execErr := p.Execute(context.Background(), core.Job{ID: "job-" + tc.operation, Operation: tc.operation, InputBlobID: inputID})
			if execErr != nil {
				t.Fatalf("execute=%+v", execErr)
			}
			meta, err := blobs.GetMetadata(result.OutputID)
			if err != nil {
				t.Fatal(err)
			}
			if !meta.Sandbox || !meta.Watermarked || !meta.NonPublishable || meta.RestrictionReason != "photoroom_sandbox_output" {
				t.Fatalf("metadata=%+v", meta)
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}

func TestUnsupportedOperationsAndPromptNeverReachNetwork(t *testing.T) {
	blobs, inputID := fixture(t)
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer s.Close()
	p := configured(t, s, blobs)
	for i, job := range []core.Job{
		{ID: "bad-op", Operation: "PHOTOROOM_RELIGHT", InputBlobID: inputID},
		{ID: "prompt", Operation: WhiteBackground, InputBlobID: inputID, Prompt: "change product label"},
		{Operation: WhiteBackground, InputBlobID: inputID},
	} {
		if _, got := p.Execute(context.Background(), job); got == nil || got.Code != "VALIDATION_ERROR" {
			t.Fatalf("case %d: %+v", i, got)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestAmbiguousFailuresAreReconcileRequiredAndNeverRetried(t *testing.T) {
	blobs, inputID := fixture(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
		client  func(*httptest.Server) *http.Client
	}{
		{"timeout", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(40 * time.Millisecond)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes(t))
		}, func(_ *httptest.Server) *http.Client { return &http.Client{Timeout: 5 * time.Millisecond} }},
		{"500", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream secret body", http.StatusInternalServerError)
		}, nil},
		{"truncated", func(w http.ResponseWriter, _ *http.Request) {
			h, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("no hijacker")
			}
			conn, rw, err := h.Hijack()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_, _ = rw.WriteString("HTTP/1.1 200 OK\r\nContent-Type: image/png\r\nContent-Length: 9999\r\n\r\nshort")
			_ = rw.Flush()
		}, nil},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); tc.handler(w, r) }))
			defer s.Close()
			client := s.Client()
			if tc.client != nil {
				client = tc.client(s)
			}
			p, _ := New(Config{APIKey: testSecret, BaseURL: s.URL, Blobs: blobs, Repository: testRepository(t), HTTPClient: client, TrainingOptOutConfirmed: true, SandboxAccountConfirmed: true})
			job := core.Job{ID: fmt.Sprintf("ambiguous-%d", i), Operation: WhiteBackground, InputBlobID: inputID}
			_, got := p.Execute(context.Background(), job)
			if got == nil || got.Code != "RECONCILE_REQUIRED" || got.Retryable {
				t.Fatalf("error=%+v", got)
			}
			_, second := p.Execute(context.Background(), job)
			if second == nil || second.Code != "RECONCILE_REQUIRED" {
				t.Fatalf("second=%+v", second)
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
			if strings.Contains(got.Error()+second.Error(), testSecret) || strings.Contains(got.Error(), "upstream secret body") {
				t.Fatal("secret leaked")
			}
		})
	}
}

func TestRateLimitAndMaliciousResponsesAreNotRetriedOrPersisted(t *testing.T) {
	blobs, inputID := fixture(t)
	cases := []struct {
		name        string
		status      int
		contentType string
		body        func() []byte
		code        string
	}{
		{"429", 429, "application/json", func() []byte { return []byte(`{"error":"key ` + testSecret + `"}`) }, "RATE_LIMITED_NOT_RETRIED"},
		{"mime spoof", 200, "text/html", func() []byte { return pngBytes(t) }, "OUTPUT_INVALID"},
		{"invalid png", 200, "image/png", func() []byte { return []byte("not-an-image") }, "OUTPUT_INVALID"},
		{"oversized", 200, "image/png", func() []byte { return bytes.Repeat([]byte{'x'}, MaxResponseBytes+1) }, "OUTPUT_INVALID"},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", tc.contentType)
				w.Header().Set("x-request-id", "bad\r\n"+testSecret)
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.body())
			}))
			defer s.Close()
			p := configured(t, s, blobs)
			_, got := p.Execute(context.Background(), core.Job{ID: fmt.Sprintf("malicious-%d", i), Operation: RemoveBackground, InputBlobID: inputID})
			if got == nil || got.Code != tc.code || got.Retryable {
				t.Fatalf("error=%+v", got)
			}
			if strings.Contains(got.Error()+got.RequestID, testSecret) {
				t.Fatal("credential leaked")
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}
