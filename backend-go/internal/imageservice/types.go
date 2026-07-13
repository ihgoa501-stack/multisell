package imageservice

import (
	"encoding/json"
	"fmt"
	"time"
)

type CreateJobRequest struct {
	OwnerID               int64  `json:"owner_id"`
	LingMirrorTaskID      string `json:"lingmirror_task_id,omitempty"`
	LingMirrorTaskVersion int64  `json:"lingmirror_task_version,omitempty"`
	IdempotencyKey        string `json:"idempotency_key"`
	ManifestHash          string `json:"manifest_hash"`
	Operation             string `json:"operation"`
	Processor             string `json:"processor,omitempty"`
	Prompt                string `json:"prompt,omitempty"`
	InputBlobID           string `json:"input_blob_id"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	Format                string `json:"format"`
	MaxCost               string `json:"max_cost,omitempty"`
	Currency              string `json:"currency,omitempty"`
	Region                string `json:"region,omitempty"`
	ProviderEnvironment   string `json:"provider_environment,omitempty"`
	Sandbox               bool   `json:"sandbox"`
	Watermarked           bool   `json:"watermarked"`
	NonPublishable        bool   `json:"non_publishable"`
}

type Job struct {
	ID                    string    `json:"id"`
	OwnerID               int64     `json:"owner_id"`
	LingMirrorTaskID      string    `json:"lingmirror_task_id,omitempty"`
	LingMirrorTaskVersion int64     `json:"lingmirror_task_version,omitempty"`
	IdempotencyKey        string    `json:"idempotency_key"`
	ManifestHash          string    `json:"manifest_hash"`
	Operation             string    `json:"operation"`
	Processor             string    `json:"processor,omitempty"`
	Prompt                string    `json:"prompt,omitempty"`
	InputBlobID           string    `json:"input_blob_id"`
	OutputBlobID          string    `json:"output_blob_id,omitempty"`
	Width                 int       `json:"width"`
	Height                int       `json:"height"`
	Format                string    `json:"format"`
	MaxCost               string    `json:"max_cost,omitempty"`
	Currency              string    `json:"currency,omitempty"`
	Region                string    `json:"region,omitempty"`
	ProviderEnvironment   string    `json:"provider_environment,omitempty"`
	Sandbox               bool      `json:"sandbox"`
	Watermarked           bool      `json:"watermarked"`
	NonPublishable        bool      `json:"non_publishable"`
	Status                string    `json:"status"`
	ErrorCode             string    `json:"error_code,omitempty"`
	Version               int64     `json:"version"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Attempt struct {
	ID                string     `json:"id"`
	JobID             string     `json:"job_id"`
	IdempotencyKey    string     `json:"idempotency_key"`
	Number            int        `json:"number"`
	Status            string     `json:"status"`
	LeaseOwner        string     `json:"lease_owner,omitempty"`
	LeaseUntil        *time.Time `json:"lease_until,omitempty"`
	ErrorCode         string     `json:"error_code,omitempty"`
	ProviderRequestID string     `json:"provider_request_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type EnqueueExecutionRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	ExecutionToken string `json:"execution_token,omitempty"`
}

type QuiesceJobRequest struct {
	OwnerID               int64  `json:"owner_id"`
	LingMirrorTaskID      string `json:"lingmirror_task_id"`
	LingMirrorTaskVersion int64  `json:"lingmirror_task_version"`
	ManifestHash          string `json:"manifest_hash"`
}

type AttemptList struct {
	Items []Attempt `json:"items"`
}

type PutBlobResponse struct {
	BlobID string `json:"blob_id"`
}

type ProcessorCapability struct {
	Code                string   `json:"code"`
	Available           bool     `json:"available"`
	Operations          []string `json:"operations"`
	SafetyLevel         string   `json:"safety_level"`
	ProviderEnvironment string   `json:"provider_environment,omitempty"`
	Region              string   `json:"region,omitempty"`
	Watermarked         bool     `json:"watermarked"`
	NonPublishable      bool     `json:"non_publishable"`
	QuotaAvailable      bool     `json:"quota_available,omitempty"`
	QuotaRemaining      int64    `json:"quota_remaining,omitempty"`
}

type ProcessorCapabilityList struct {
	Items []ProcessorCapability `json:"items"`
}

type errorEnvelope struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details,omitempty"`
	} `json:"error"`
}

// APIError is a valid structured error returned by Image Service.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("imageservice: %s (%d): %s", e.Code, e.StatusCode, e.Message)
}

// ProtocolError means Image Service returned a response that violated the
// private API contract. It is distinct from a provider/business error.
type ProtocolError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *ProtocolError) Error() string {
	if e.Err == nil {
		return "imageservice: protocol error: " + e.Message
	}
	return "imageservice: protocol error: " + e.Message + ": " + e.Err.Error()
}

func (e *ProtocolError) Unwrap() error { return e.Err }

// ResponseTooLargeError prevents an untrusted service response from consuming
// unbounded backend memory.
type ResponseTooLargeError struct{ Limit int64 }

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("imageservice: response exceeds %d byte limit", e.Limit)
}
