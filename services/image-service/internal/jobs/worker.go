package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/processor"
)

type ExecutionError struct {
	Code      string
	Err       error
	RequestID string
	Retryable bool
}

type ExecutionResult struct {
	OutputID          string
	ProviderRequestID string
}

func (e *ExecutionError) Error() string {
	if e == nil || e.Err == nil {
		return "image execution failed"
	}
	return e.Err.Error()
}

type Executor interface {
	Execute(context.Context, core.Job) (ExecutionResult, *ExecutionError)
}

type DeterministicExecutor struct{ blobs *blobstore.Store }

func NewDeterministicExecutor(blobs *blobstore.Store) *DeterministicExecutor {
	return &DeterministicExecutor{blobs: blobs}
}

func (e *DeterministicExecutor) Execute(ctx context.Context, job core.Job) (ExecutionResult, *ExecutionError) {
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, &ExecutionError{Code: "EXECUTION_CANCELLED", Err: err}
	}
	input, err := e.blobs.Get(job.InputBlobID)
	if err != nil {
		return ExecutionResult{}, &ExecutionError{Code: "INPUT_BLOB_INVALID", Err: err}
	}
	output, err := processor.Process(input, job.Width, job.Height, job.Format)
	if err != nil {
		return ExecutionResult{}, &ExecutionError{Code: "PROCESSING_FAILED", Err: err}
	}
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, &ExecutionError{Code: "EXECUTION_CANCELLED", Err: err}
	}
	id, err := e.blobs.Put(output)
	if err != nil {
		return ExecutionResult{}, &ExecutionError{Code: "STORE_ERROR", Err: err}
	}
	return ExecutionResult{OutputID: id}, nil
}

type Worker struct {
	store        core.Repository
	executor     Executor
	id           string
	lease        time.Duration
	pollInterval time.Duration
}

func NewWorker(store core.Repository, executor Executor, id string, lease, pollInterval time.Duration) (*Worker, error) {
	if store == nil || executor == nil || id == "" {
		return nil, errors.New("worker store, executor and id are required")
	}
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	if pollInterval <= 0 {
		pollInterval = 250 * time.Millisecond
	}
	return &Worker{store: store, executor: executor, id: id, lease: lease, pollInterval: pollInterval}, nil
}

// RunOne claims and executes at most one durable attempt. handled is false
// only when no work was available.
func (w *Worker) RunOne(ctx context.Context) (handled bool, err error) {
	attempt, claimed, err := w.store.ClaimAttempt(w.id, w.lease)
	if err != nil || !claimed {
		return false, err
	}
	job, ok, getErr := w.store.GetJob(attempt.JobID)
	if getErr != nil {
		return true, getErr
	}
	if !ok {
		_, completeErr := w.store.CompleteAttempt(attempt.ID, w.id, core.AttemptFailed, "JOB_NOT_FOUND")
		return true, completeErr
	}

	// A crash can leave the job terminal while its attempt is still leased.
	// Reconciliation completes the attempt without rerunning the processor.
	if job.Status == core.JobReady {
		_, err = w.store.CompleteAttempt(attempt.ID, w.id, core.AttemptSucceeded, "")
		return true, err
	}
	if job.Status == core.JobFailed {
		_, err = w.store.CompleteAttempt(attempt.ID, w.id, core.AttemptFailed, job.ErrorCode)
		return true, err
	}
	if job.Status == core.JobReconcileRequired {
		_, err = w.store.CompleteAttempt(attempt.ID, w.id, core.AttemptReconcileRequired, job.ErrorCode)
		return true, err
	}
	if job.Status == core.JobQueued {
		job, err = w.store.Transition(job.ID, core.JobQueued, core.JobRunning, "", "")
		if err != nil {
			return true, fmt.Errorf("start job: %w", err)
		}
	} else if job.Status != core.JobRunning {
		return true, fmt.Errorf("unsupported job status %s", job.Status)
	}

	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatDone := make(chan struct{})
	heartbeatErr := make(chan error, 1)
	go w.heartbeat(execCtx, attempt.ID, heartbeatDone, heartbeatErr, cancel)
	result, executionErr := w.executor.Execute(execCtx, *job)
	cancel()
	<-heartbeatDone
	select {
	case leaseErr := <-heartbeatErr:
		return true, leaseErr
	default:
	}

	if executionErr != nil {
		jobStatus, attemptStatus := core.JobFailed, core.AttemptFailed
		if executionErr.Code == "RECONCILE_REQUIRED" {
			jobStatus, attemptStatus = core.JobReconcileRequired, core.AttemptReconcileRequired
		}
		_, _, err = w.store.FinalizeAttempt(core.AttemptFinalization{JobID: job.ID, FromJobStatus: core.JobRunning, ToJobStatus: jobStatus, ErrorCode: executionErr.Code, AttemptID: attempt.ID, LeaseOwner: w.id, AttemptStatus: attemptStatus, ProviderRequestID: executionErr.RequestID})
		return true, err
	}
	_, _, err = w.store.FinalizeAttempt(core.AttemptFinalization{JobID: job.ID, FromJobStatus: core.JobRunning, ToJobStatus: core.JobReady, OutputBlobID: result.OutputID, AttemptID: attempt.ID, LeaseOwner: w.id, AttemptStatus: core.AttemptSucceeded, ProviderRequestID: result.ProviderRequestID})
	return true, err
}

func (w *Worker) heartbeat(ctx context.Context, attemptID string, done chan<- struct{}, errOut chan<- error, cancel context.CancelFunc) {
	defer close(done)
	interval := w.lease / 3
	if interval <= 0 {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.store.RenewAttemptLease(attemptID, w.id, w.lease); err != nil {
				select {
				case errOut <- err:
				default:
				}
				cancel()
				return
			}
		}
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		handled, err := w.RunOne(ctx)
		if err != nil {
			return err
		}
		if handled {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
