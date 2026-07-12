// Package imageservice provides the private HTTP client used by LingMirror to
// invoke the internal Image Service. It deliberately contains no business
// approval or publication logic.
package imageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout          = 15 * time.Second
	defaultMaxResponseBytes = int64(1 << 20)
)

// Config defines the private service connection. HTTPClient is primarily
// useful when the caller needs a custom transport; its Timeout is overridden
// by Timeout on a shallow copy so the caller's client is never mutated.
type Config struct {
	BaseURL          string
	SharedSecret     string
	Timeout          time.Duration
	MaxResponseBytes int64
	HTTPClient       *http.Client
}

// Client is safe for concurrent use.
type Client struct {
	baseURL          *url.URL
	sharedSecret     string
	httpClient       *http.Client
	maxResponseBytes int64
}

// New constructs a private Image Service client.
func New(cfg Config) (*Client, error) {
	base, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("imageservice: base URL must be an absolute HTTP(S) URL")
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, errors.New("imageservice: base URL must use http or https")
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("imageservice: base URL must not contain credentials, query, or fragment")
	}
	if strings.TrimSpace(cfg.SharedSecret) == "" {
		return nil, errors.New("imageservice: shared secret is required")
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if timeout < 0 {
		return nil, errors.New("imageservice: timeout must not be negative")
	}
	limit := cfg.MaxResponseBytes
	if limit == 0 {
		limit = defaultMaxResponseBytes
	}
	if limit < 1 {
		return nil, errors.New("imageservice: response limit must be positive")
	}
	httpClient := &http.Client{}
	if cfg.HTTPClient != nil {
		copy := *cfg.HTTPClient
		httpClient = &copy
	}
	httpClient.Timeout = timeout
	base.Path = strings.TrimRight(base.Path, "/")
	return &Client{baseURL: base, sharedSecret: cfg.SharedSecret, httpClient: httpClient, maxResponseBytes: limit}, nil
}

// PutBlob uploads exact image bytes and returns their content-addressed ID.
func (c *Client) PutBlob(ctx context.Context, contentType string, body io.Reader) (*PutBlobResponse, error) {
	if body == nil {
		return nil, errors.New("imageservice: blob body is required")
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	var out PutBlobResponse
	if err := c.do(ctx, http.MethodPost, "/internal/v1/blobs", contentType, body, &out); err != nil {
		return nil, err
	}
	if out.BlobID == "" {
		return nil, &ProtocolError{Message: "blob response is missing blob_id"}
	}
	return &out, nil
}

// CreateJob creates or idempotently replays a deterministic image job.
func (c *Client) CreateJob(ctx context.Context, in CreateJobRequest) (*Job, error) {
	var out Job
	if err := c.doJSON(ctx, http.MethodPost, "/internal/v1/jobs", in, &out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, &ProtocolError{Message: "job response is missing id"}
	}
	return &out, nil
}

// GetJob retrieves the technical execution state for one job.
func (c *Client) GetJob(ctx context.Context, id string) (*Job, error) {
	path, err := resourcePath(id, "")
	if err != nil {
		return nil, err
	}
	var out Job
	if err := c.do(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, &ProtocolError{Message: "job response is missing id"}
	}
	return &out, nil
}

// EnqueueExecution creates or replays the durable attempt for a queued job.
// A 2xx response only means the attempt was persisted, not that processing
// succeeded.
func (c *Client) EnqueueExecution(ctx context.Context, id, idempotencyKey string) (*Attempt, error) {
	return c.enqueueExecution(ctx, id, idempotencyKey, "")
}

func (c *Client) EnqueueAuthorizedExecution(ctx context.Context, id, idempotencyKey, token string) (*Attempt, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("imageservice: execution token is required")
	}
	return c.enqueueExecution(ctx, id, idempotencyKey, token)
}

func (c *Client) enqueueExecution(ctx context.Context, id, idempotencyKey, token string) (*Attempt, error) {
	path, err := resourcePath(id, "/executions")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("imageservice: execution idempotency key is required")
	}
	var out Attempt
	if err := c.doJSON(ctx, http.MethodPost, path, EnqueueExecutionRequest{IdempotencyKey: idempotencyKey, ExecutionToken: token}, &out); err != nil {
		return nil, err
	}
	if out.ID == "" || out.JobID == "" {
		return nil, &ProtocolError{Message: "attempt response is missing id or job_id"}
	}
	return &out, nil
}

// ListAttempts returns all currently persisted attempts for a job.
func (c *Client) ListAttempts(ctx context.Context, id string) ([]Attempt, error) {
	path, err := resourcePath(id, "/attempts")
	if err != nil {
		return nil, err
	}
	var out AttemptList
	if err := c.do(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []Attempt{}
	}
	return out.Items, nil
}

func (c *Client) GetBlob(ctx context.Context, id string) ([]byte, string, error) {
	if len(id) != 64 || strings.ContainsAny(id, "/?#") {
		return nil, "", errors.New("imageservice: invalid blob id")
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/internal/v1/blobs/" + url.PathEscape(id) + "/content"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.sharedSecret)
	req.Header.Set("Accept", "image/png, image/jpeg")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("imageservice: request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := readLimited(resp.Body, 10<<20)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{StatusCode: resp.StatusCode, Code: "BLOB_READ_FAILED", Message: "blob unavailable"}
	}
	media := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if media != "image/png" && media != "image/jpeg" {
		return nil, "", &ProtocolError{StatusCode: resp.StatusCode, Message: "invalid blob content type"}
	}
	return data, media, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("imageservice: encode request: %w", err)
	}
	return c.do(ctx, method, path, "application/json", bytes.NewReader(payload), out)
}

func (c *Client) do(ctx context.Context, method, path, contentType string, body io.Reader, out any) error {
	if ctx == nil {
		return errors.New("imageservice: context is required")
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("imageservice: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.sharedSecret)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("imageservice: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := readLimited(resp.Body, c.maxResponseBytes)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope errorEnvelope
		if err := decodeStrict(data, &envelope); err != nil || envelope.Error.Code == "" || envelope.Error.Message == "" {
			return &ProtocolError{StatusCode: resp.StatusCode, Message: "invalid error response"}
		}
		return &APIError{StatusCode: resp.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message, Details: envelope.Error.Details}
	}
	if out == nil {
		return nil
	}
	if err := decodeStrict(data, out); err != nil {
		return &ProtocolError{StatusCode: resp.StatusCode, Message: "invalid success response", Err: err}
	}
	return nil
}

func resourcePath(id, suffix string) (string, error) {
	if id == "" {
		return "", errors.New("imageservice: job id is required")
	}
	if strings.ContainsAny(id, "/?#") {
		return "", errors.New("imageservice: invalid job id")
	}
	return "/internal/v1/jobs/" + url.PathEscape(id) + suffix, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("imageservice: read response: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, &ResponseTooLargeError{Limit: limit}
	}
	return data, nil
}

func decodeStrict(data []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
