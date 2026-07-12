package providers

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/jobs"
)

var ErrDuplicateOperation = errors.New("provider operation already registered")

// Registry is the single operation-to-executor dispatch point. An operation
// not explicitly registered fails closed instead of falling back to another
// provider or to a synthetic result.
type Registry struct {
	mu        sync.RWMutex
	executors map[string]jobs.Executor
}

func NewRegistry() *Registry { return &Registry{executors: make(map[string]jobs.Executor)} }

func (r *Registry) Register(operation string, executor jobs.Executor) error {
	operation = strings.TrimSpace(operation)
	if operation == "" || executor == nil {
		return errors.New("provider operation and executor are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.executors[operation]; exists {
		return ErrDuplicateOperation
	}
	r.executors[operation] = executor
	return nil
}

func (r *Registry) Execute(ctx context.Context, job core.Job) (string, *jobs.ExecutionError) {
	r.mu.RLock()
	executor := r.executors[job.Operation]
	r.mu.RUnlock()
	if executor == nil {
		return "", &jobs.ExecutionError{Code: "PROVIDER_UNAVAILABLE", Err: errors.New("requested image provider is unavailable")}
	}
	return executor.Execute(ctx, job)
}
