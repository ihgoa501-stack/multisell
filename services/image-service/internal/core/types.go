package core

import "time"

type JobStatus string

const (
	JobQueued            JobStatus = "QUEUED"
	JobRunning           JobStatus = "RUNNING"
	JobReady             JobStatus = "READY"
	JobFailed            JobStatus = "FAILED"
	JobReconcileRequired JobStatus = "RECONCILE_REQUIRED"
)

type Job struct {
	ID                    string    `json:"id"`
	OwnerID               int64     `json:"owner_id"`
	LingMirrorTaskID      string    `json:"lingmirror_task_id,omitempty"`
	LingMirrorTaskVersion int64     `json:"lingmirror_task_version,omitempty"`
	IdempotencyKey        string    `json:"idempotency_key"`
	ManifestHash          string    `json:"manifest_hash"`
	Operation             string    `json:"operation"`
	Processor             string    `json:"processor"`
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
	Status                JobStatus `json:"status"`
	ErrorCode             string    `json:"error_code,omitempty"`
	Version               int64     `json:"version"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type AttemptStatus string

const (
	AttemptQueued            AttemptStatus = "QUEUED"
	AttemptRunning           AttemptStatus = "RUNNING"
	AttemptSucceeded         AttemptStatus = "SUCCEEDED"
	AttemptFailed            AttemptStatus = "FAILED"
	AttemptReconcileRequired AttemptStatus = "RECONCILE_REQUIRED"
)

type Attempt struct {
	ID                string        `json:"id"`
	JobID             string        `json:"job_id"`
	IdempotencyKey    string        `json:"idempotency_key"`
	Number            int           `json:"number"`
	Status            AttemptStatus `json:"status"`
	LeaseOwner        string        `json:"lease_owner,omitempty"`
	LeaseUntil        *time.Time    `json:"lease_until,omitempty"`
	ErrorCode         string        `json:"error_code,omitempty"`
	ProviderRequestID string        `json:"provider_request_id,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	StartedAt         *time.Time    `json:"started_at,omitempty"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty"`
}

type AttemptFinalization struct {
	JobID             string
	FromJobStatus     JobStatus
	ToJobStatus       JobStatus
	OutputBlobID      string
	ErrorCode         string
	AttemptID         string
	LeaseOwner        string
	AttemptStatus     AttemptStatus
	ProviderRequestID string
}

type CreateJob struct {
	OwnerID               int64  `json:"owner_id"`
	LingMirrorTaskID      string `json:"lingmirror_task_id,omitempty"`
	LingMirrorTaskVersion int64  `json:"lingmirror_task_version,omitempty"`
	IdempotencyKey        string `json:"idempotency_key"`
	ManifestHash          string `json:"manifest_hash"`
	Operation             string `json:"operation"`
	Processor             string `json:"processor"`
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
