package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
	Operation        = "OPENAI_IMAGE_EDIT"
	DefaultModel     = "gpt-image-2"
	MaxResponseBytes = 16 << 20
)

type Config struct {
	APIKey     string
	BaseURL    string
	Blobs      *blobstore.Store
	Repository core.Repository
	HTTPClient *http.Client
}

type Provider struct {
	apiKey     string
	baseURL    string
	blobs      *blobstore.Store
	repository core.Repository
	client     *http.Client
}

func New(cfg Config) (*Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 90 * time.Second}
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return nil, err
	}
	client := *cfg.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Provider{apiKey: strings.TrimSpace(cfg.APIKey), baseURL: strings.TrimRight(cfg.BaseURL, "/"), blobs: cfg.Blobs, repository: cfg.Repository, client: &client}, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("invalid OpenAI base URL")
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme == "https" && host == "api.openai.com" && u.Port() == "" {
		return nil
	}
	if u.Scheme == "http" && (host == "127.0.0.1" || host == "localhost") {
		return nil
	}
	return errors.New("OpenAI base URL is not allowlisted")
}

func (p *Provider) Available() bool {
	return p != nil && p.apiKey != "" && p.blobs != nil && p.repository != nil
}

func (p *Provider) Execute(ctx context.Context, job core.Job) (jobs.ExecutionResult, *jobs.ExecutionError) {
	if !p.Available() {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "PROVIDER_UNAVAILABLE", Err: errors.New("OpenAI image provider is not configured")}
	}
	if job.Operation != Operation || strings.TrimSpace(job.Prompt) == "" {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "VALIDATION_ERROR", Err: errors.New("OpenAI image edit requires a prompt")}
	}
	input, err := p.blobs.Get(job.InputBlobID)
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "INPUT_BLOB_INVALID", Err: err}
	}
	meta, err := p.blobs.GetMetadata(job.InputBlobID)
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "INPUT_BLOB_INVALID", Err: err}
	}
	filename := "input.png"
	if meta.Format == "jpeg" {
		filename = "input.jpg"
	}
	size, err := imageSize(job.Width, job.Height)
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "VALIDATION_ERROR", Err: err}
	}
	claimed, err := p.repository.ClaimProviderSubmit(job.ID, "openai")
	if err != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "SUBMIT_GUARD_ERROR", Err: errors.New("OpenAI submit gate unavailable")}
	}
	if !claimed {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "RECONCILE_REQUIRED", Err: errors.New("OpenAI submit was already attempted; do not retry")}
	}

	output, requestID, executionErr := p.editOnce(ctx, input, filename, strings.TrimSpace(job.Prompt), size, job.IdempotencyKey)
	if executionErr != nil {
		if executionErr.Code == "PROVIDER_NETWORK_ERROR" || executionErr.Code == "PROVIDER_RESPONSE_ERROR" || executionErr.Code == "PROVIDER_RETRYABLE" || executionErr.Code == "EXECUTION_CANCELLED" {
			executionErr.Code = "RECONCILE_REQUIRED"
			executionErr.Retryable = false
		}
		return jobs.ExecutionResult{}, executionErr
	}
	id, putErr := p.blobs.Put(output)
	if putErr != nil {
		return jobs.ExecutionResult{}, &jobs.ExecutionError{Code: "RECONCILE_REQUIRED", Err: putErr, RequestID: requestID}
	}
	return jobs.ExecutionResult{OutputID: id, ProviderRequestID: requestID}, nil
}

func (p *Provider) editOnce(ctx context.Context, input []byte, filename, prompt, size, idempotencyKey string) ([]byte, string, *jobs.ExecutionError) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := writeMultipart(w, input, filename, DefaultModel, prompt, size); err != nil {
		return nil, "", &jobs.ExecutionError{Code: "REQUEST_BUILD_FAILED", Err: err}
	}
	if err := w.Close(); err != nil {
		return nil, "", &jobs.ExecutionError{Code: "REQUEST_BUILD_FAILED", Err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/images/edits", &body)
	if err != nil {
		return nil, "", &jobs.ExecutionError{Code: "REQUEST_BUILD_FAILED", Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", &jobs.ExecutionError{Code: "EXECUTION_CANCELLED", Err: ctx.Err()}
		}
		return nil, "", &jobs.ExecutionError{Code: "PROVIDER_NETWORK_ERROR", Err: err}
	}
	defer resp.Body.Close()
	requestID := cleanToken(strings.ReplaceAll(resp.Header.Get("x-request-id"), p.apiKey, ""))
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, requestID, &jobs.ExecutionError{Code: "PROVIDER_RESPONSE_ERROR", Err: err, RequestID: requestID}
	}
	if len(b) > MaxResponseBytes {
		return nil, requestID, &jobs.ExecutionError{Code: "OUTPUT_INVALID", Err: errors.New("OpenAI response exceeds size limit"), RequestID: requestID}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := "PROVIDER_REJECTED"
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if retryable {
			code = "PROVIDER_RETRYABLE"
		}
		return nil, requestID, &jobs.ExecutionError{Code: code, Err: providerError(resp.StatusCode, b), RequestID: requestID, Retryable: retryable}
	}
	var decoded struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil || len(decoded.Data) != 1 || decoded.Data[0].B64JSON == "" {
		return nil, requestID, &jobs.ExecutionError{Code: "OUTPUT_INVALID", Err: errors.New("OpenAI returned an invalid image response"), RequestID: requestID}
	}
	imageBytes, err := base64.StdEncoding.DecodeString(decoded.Data[0].B64JSON)
	if err != nil || len(imageBytes) == 0 || len(imageBytes) > blobstore.MaxBlobBytes {
		return nil, requestID, &jobs.ExecutionError{Code: "OUTPUT_INVALID", Err: errors.New("OpenAI returned invalid image bytes"), RequestID: requestID}
	}
	return imageBytes, requestID, nil
}

func writeMultipart(w *multipart.Writer, input []byte, filename, model, prompt, size string) error {
	part, err := w.CreateFormFile("image", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(input); err != nil {
		return err
	}
	for key, value := range map[string]string{"model": model, "prompt": prompt, "size": size, "output_format": "png"} {
		if err := w.WriteField(key, value); err != nil {
			return err
		}
	}
	return nil
}

func imageSize(width, height int) (string, error) {
	switch {
	case width == 1024 && height == 1024:
		return "1024x1024", nil
	case width == 1024 && height == 1536:
		return "1024x1536", nil
	case width == 1536 && height == 1024:
		return "1536x1024", nil
	default:
		return "", fmt.Errorf("unsupported OpenAI image size %dx%d", width, height)
	}
}

func providerError(status int, body []byte) error {
	var decoded struct {
		Error struct {
			Type string `json:"type"`
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &decoded)
	return fmt.Errorf("OpenAI request failed: status=%d type=%s code=%s", status, cleanToken(decoded.Error.Type), cleanToken(decoded.Error.Code))
}

func cleanToken(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 80 {
		v = v[:80]
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_-.", r) {
			return r
		}
		return -1
	}, v)
}
