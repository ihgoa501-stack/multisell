package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/jobs"
)

type stubExecutor struct{}

func (stubExecutor) Execute(context.Context, core.Job) (jobs.ExecutionResult, *jobs.ExecutionError) {
	return jobs.ExecutionResult{OutputID: "ok"}, nil
}

func TestRegistryRejectsDuplicateAndUnknownOperations(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("OPENAI_IMAGE_EDIT", stubExecutor{}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("OPENAI_IMAGE_EDIT", stubExecutor{}); !errors.Is(err, ErrDuplicateOperation) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if _, executionErr := r.Execute(context.Background(), core.Job{Operation: "UNKNOWN"}); executionErr == nil || executionErr.Code != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("unexpected unknown-operation result: %+v", executionErr)
	}
}

func TestRegistryDispatchesRegisteredOperation(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("OPENAI_IMAGE_EDIT", stubExecutor{}); err != nil {
		t.Fatal(err)
	}
	got, executionErr := r.Execute(context.Background(), core.Job{Operation: "OPENAI_IMAGE_EDIT"})
	if executionErr != nil || got.OutputID != "ok" {
		t.Fatalf("got=%+v err=%v", got, executionErr)
	}
}
