// Package photoroom contains a deliberately non-registered sandbox adapter.
// Its synchronous API has no public query/idempotency contract, so every job
// may cross the network at most once and every ambiguous result is terminal.
package photoroom

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/jobs"
)

const (
	RemoveBackground = "PHOTOROOM_REMOVE_BACKGROUND_SANDBOX"
	WhiteBackground  = "PHOTOROOM_WHITE_BACKGROUND_SANDBOX"
	AIShadow         = "PHOTOROOM_AI_SHADOW_SANDBOX"
	DefaultBaseURL   = "https://image-api.photoroom.com"
	MaxResponseBytes = blobstore.MaxBlobBytes
)

type Config struct {
	APIKey                  string
	BaseURL                 string
	Blobs                   *blobstore.Store
	HTTPClient              *http.Client
	TrainingOptOutConfirmed bool
	SandboxAccountConfirmed bool
	Repository              core.Repository
}

type Provider struct {
	apiKey                  string
	baseURL                 string
	blobs                   *blobstore.Store
	client                  *http.Client
	trainingOptOutConfirmed bool
	sandboxAccountConfirmed bool
	repository              core.Repository
}

func New(cfg Config) (*Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 90 * time.Second}
	}
	// Never follow redirects. The request carries both a credential and the
	// Owner's image bytes; neither may be replayed to a redirected host.
	client := *cfg.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return nil, err
	}
	return &Provider{
		apiKey: strings.TrimSpace(cfg.APIKey), baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		blobs: cfg.Blobs, client: &client,
		trainingOptOutConfirmed: cfg.TrainingOptOutConfirmed,
		sandboxAccountConfirmed: cfg.SandboxAccountConfirmed,
		repository:              cfg.Repository,
	}, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("invalid Photoroom base URL")
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme == "https" && host == "image-api.photoroom.com" && u.Port() == "" {
		return nil
	}
	// Loopback HTTP is reserved for hermetic fixtures; the server never accepts a BaseURL environment override.
	if u.Scheme == "http" && (host == "127.0.0.1" || host == "localhost") {
		return nil
	}
	return errors.New("Photoroom base URL is not allowlisted")
}

// Available is intentionally stricter than possession of a credential. A
// self-serve account may train on inputs until its team setting is opted out.
func (p *Provider) Available() bool {
	return p != nil && p.apiKey != "" && p.blobs != nil && p.repository != nil && p.trainingOptOutConfirmed && p.sandboxAccountConfirmed
}

func (p *Provider) Execute(ctx context.Context, job core.Job) (jobs.ExecutionResult, *jobs.ExecutionError) {
	if !p.Available() {
		return jobs.ExecutionResult{}, executionError("PROVIDER_UNAVAILABLE", "Photoroom sandbox is unavailable", "")
	}
	fields, ok := operationFields(job.Operation)
	if !ok || job.ID == "" || job.InputBlobID == "" || strings.TrimSpace(job.Prompt) != "" {
		return jobs.ExecutionResult{}, executionError("VALIDATION_ERROR", "unsupported Photoroom sandbox request", "")
	}
	input, err := p.blobs.Get(job.InputBlobID)
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "INPUT_BLOB_INVALID", Err: errors.New("input blob is unavailable")}
	}
	meta, err := p.blobs.GetMetadata(job.InputBlobID)
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "INPUT_BLOB_INVALID", Err: errors.New("input blob metadata is unavailable")}
	}
	filename := "input.png"
	if meta.Format == "jpeg" {
		filename = "input.jpg"
	}

	claimed, err := p.repository.ClaimProviderSubmit(job.ID, "photoroom")
	if err != nil {
		return jobs.ExecutionResult{}, executionError("SUBMIT_GUARD_ERROR", "sandbox submit gate unavailable", "")
	}
	if !claimed {
		return jobs.ExecutionResult{}, executionError("RECONCILE_REQUIRED", "sandbox submit was already attempted; do not retry", "")
	}

	output, requestID, execErr := p.editOnce(ctx, input, filename, fields)
	if execErr != nil {
		return jobs.ExecutionResult{}, execErr
	}
	output, err = applyAndVerifySandboxWatermark(output)
	if err != nil {
		return jobs.ExecutionResult{}, executionError("OUTPUT_INVALID", "sandbox output could not be visibly watermarked", requestID)
	}
	id, err := p.blobs.Put(output)
	if err != nil {
		return jobs.ExecutionResult{}, executionError("OUTPUT_INVALID", "Photoroom returned an unsafe image", requestID)
	}
	if err := p.blobs.MarkRestricted(id, blobstore.Restriction{Sandbox: true, Watermarked: true, NonPublishable: true, Reason: "photoroom_sandbox_output"}); err != nil {
		return jobs.ExecutionResult{}, executionError("OUTPUT_RESTRICTION_FAILED", "sandbox output could not be restricted", requestID)
	}
	return jobs.ExecutionResult{OutputID: id, ProviderRequestID: requestID}, nil
}

func operationFields(operation string) (map[string]string, bool) {
	base := map[string]string{"removeBackground": "true"}
	switch operation {
	case RemoveBackground:
		base["background.color"] = "transparent"
	case WhiteBackground:
		base["background.color"] = "FFFFFF"
	case AIShadow:
		base["background.color"] = "FFFFFF"
		base["shadow.mode"] = "ai.soft"
	default:
		return nil, false
	}
	return base, true
}

func (p *Provider) editOnce(ctx context.Context, input []byte, filename string, fields map[string]string) ([]byte, string, *jobs.ExecutionError) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image_file", filename)
	if err != nil {
		return nil, "", executionError("REQUEST_BUILD_FAILED", "could not build image upload", "")
	}
	if _, err = part.Write(input); err != nil {
		return nil, "", executionError("REQUEST_BUILD_FAILED", "could not build image upload", "")
	}
	for key, value := range fields {
		if err = w.WriteField(key, value); err != nil {
			return nil, "", executionError("REQUEST_BUILD_FAILED", "could not build edit fields", "")
		}
	}
	if err = w.WriteField("format", "png"); err != nil {
		return nil, "", executionError("REQUEST_BUILD_FAILED", "could not build output format", "")
	}
	if err = w.Close(); err != nil {
		return nil, "", executionError("REQUEST_BUILD_FAILED", "could not finalize image upload", "")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v2/edit", &body)
	if err != nil {
		return nil, "", executionError("REQUEST_BUILD_FAILED", "could not build sandbox request", "")
	}
	req.Header.Set("X-Api-Key", p.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "image/png")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", executionError("RECONCILE_REQUIRED", "sandbox response is unknown; do not retry", "")
	}
	defer resp.Body.Close()
	// Treat provider headers as untrusted. In addition to an allowlist, remove
	// the credential defensively in case a broken upstream reflects it.
	requestID := cleanToken(strings.ReplaceAll(resp.Header.Get("x-request-id"), p.apiKey, ""))
	b, readErr := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if readErr != nil {
		return nil, requestID, executionError("RECONCILE_REQUIRED", "sandbox response was interrupted; do not retry", requestID)
	}
	if len(b) > MaxResponseBytes {
		return nil, requestID, executionError("OUTPUT_INVALID", "sandbox response exceeds size limit", requestID)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, requestID, executionError("RATE_LIMITED_NOT_RETRIED", "sandbox request was rate limited; do not retry automatically", requestID)
	}
	if resp.StatusCode >= 500 {
		return nil, requestID, executionError("RECONCILE_REQUIRED", "sandbox processing state is unknown; do not retry", requestID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, requestID, executionError("PROVIDER_REJECTED", fmt.Sprintf("Photoroom rejected sandbox request with status %d", resp.StatusCode), requestID)
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if mediaType != "image/png" {
		return nil, requestID, executionError("OUTPUT_INVALID", "sandbox response MIME must be image/png", requestID)
	}
	if len(b) == 0 {
		return nil, requestID, executionError("OUTPUT_INVALID", "sandbox returned an empty image", requestID)
	}
	return b, requestID, nil
}

func executionError(code, message, requestID string) *jobs.ExecutionError {
	return &jobs.ExecutionError{Code: code, Err: errors.New(message), RequestID: cleanToken(requestID), Retryable: false}
}

func cleanToken(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 80 {
		v = v[:80]
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:-", r) {
			return r
		}
		return -1
	}, v)
}
