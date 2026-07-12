package core

import (
	"context"
	"time"
)

// Repository is the durable authority for image execution jobs, attempts and
// consumed execution-token nonces. Implementations must preserve the atomicity
// and compare-and-swap semantics documented by each method.
type Repository interface {
	Create(CreateJob) (*Job, bool, error)
	GetJob(string) (*Job, bool, error)
	Transition(string, JobStatus, JobStatus, string, string) (*Job, error)
	EnqueueAttempt(string, string) (*Attempt, bool, error)
	EnqueueAuthorizedAttempt(string, string, string) (*Attempt, error)
	ClaimAttempt(string, time.Duration) (*Attempt, bool, error)
	CompleteAttempt(string, string, AttemptStatus, string) (*Attempt, error)
	RenewAttemptLease(string, string, time.Duration) error
	ListJobAttempts(string) ([]Attempt, error)
	Ping(context.Context) error
	Close() error
}
