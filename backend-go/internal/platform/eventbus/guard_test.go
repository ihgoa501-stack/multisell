package eventbus

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.uber.org/zap"
)

type mockAuditLogger struct {
	mu     sync.Mutex
	inputs []*MutationAuditInput
}

func (m *mockAuditLogger) LogStructured(input *MutationAuditInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs = append(m.inputs, input)
	return nil
}

func (m *mockAuditLogger) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.inputs)
}

func (m *mockAuditLogger) last() *MutationAuditInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.inputs) == 0 {
		return nil
	}
	return m.inputs[len(m.inputs)-1]
}

func TestMutationGuard_SuccessLogsAudit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	handlerCalled := false
	wrapped := guard.Guard(MutationInfo{SystemAction: "test.action", Domain: "inventory"},
		func(ctx context.Context, evt Event) error {
			handlerCalled = true
			return nil
		})
	err := wrapped(context.Background(), Event{ID: "evt-1", Topic: "test.topic", Actor: "system:test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler was not called")
	}
	if audit.count() != 2 {
		t.Fatalf("expected 2 audit entries, got %d", audit.count())
	}
	if audit.inputs[0].Result != "pending" {
		t.Errorf("first entry: expected 'pending', got %q", audit.inputs[0].Result)
	}
	if audit.inputs[1].Result != "executed" {
		t.Errorf("second entry: expected 'executed', got %q", audit.inputs[1].Result)
	}
}

func TestMutationGuard_FailureLogsFailed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	expectedErr := errors.New("handler failure")
	wrapped := guard.Guard(MutationInfo{SystemAction: "test.action.fail", Domain: "order"},
		func(ctx context.Context, evt Event) error { return expectedErr })
	err := wrapped(context.Background(), Event{ID: "evt-2", Topic: "test.fail"})
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if audit.count() != 2 {
		t.Fatalf("expected 2 audit entries, got %d", audit.count())
	}
	if audit.last().Result != "failed" {
		t.Errorf("expected 'failed', got %q", audit.last().Result)
	}
}

func TestMutationGuard_UsesEntityIDAsResourceID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	wrapped := guard.Guard(MutationInfo{SystemAction: "test.action.entity", Domain: "sku"},
		func(ctx context.Context, evt Event) error { return nil })
	_ = wrapped(context.Background(), Event{ID: "evt-3", Topic: "test.entity", EntityID: "sku-42"})
	if audit.inputs[0].ResourceID != "sku-42" {
		t.Errorf("expected ResourceID 'sku-42', got %q", audit.inputs[0].ResourceID)
	}
}

func TestMutationGuard_NilGuardPassesThrough(t *testing.T) {
	handlerCalled := false
	wrapped := (*MutationGuard)(nil).Guard(MutationInfo{SystemAction: "test", Domain: "test"},
		func(ctx context.Context, evt Event) error {
			handlerCalled = true
			return nil
		})
	_ = wrapped(context.Background(), Event{ID: "evt-4", Topic: "test.nil"})
	if !handlerCalled {
		t.Fatal("handler was not called")
	}
}

func TestMutationGuard_NilAuditPassesThrough(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	guard := NewMutationGuard(logger, nil)
	handlerCalled := false
	wrapped := guard.Guard(MutationInfo{SystemAction: "test", Domain: "test"},
		func(ctx context.Context, evt Event) error { handlerCalled = true; return nil })
	_ = wrapped(context.Background(), Event{ID: "evt-5", Topic: "test.noaudit"})
	if !handlerCalled {
		t.Fatal("handler was not called")
	}
}

func TestMutationGuard_OperatorFallsBackToSystemDomain(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	wrapped := guard.Guard(MutationInfo{SystemAction: "test.action.noactor", Domain: "inventory"},
		func(ctx context.Context, evt Event) error { return nil })
	_ = wrapped(context.Background(), Event{ID: "evt-6", Topic: "test.noactor"})
	if audit.inputs[0].Operator != "system:inventory" {
		t.Errorf("expected Operator 'system:inventory', got %q", audit.inputs[0].Operator)
	}
}

func TestMutationGuard_ReportsTriggerType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	audit := &mockAuditLogger{}
	guard := NewMutationGuard(logger, audit)
	wrapped := guard.Guard(MutationInfo{SystemAction: "test.action.triggertype", Domain: "inventory"},
		func(ctx context.Context, evt Event) error { return nil })
	_ = wrapped(context.Background(), Event{ID: "evt-7", Topic: "test.triggertype"})
	if audit.inputs[0].TriggerType != "eventbus" {
		t.Errorf("first entry: expected trigger_type 'eventbus', got %q", audit.inputs[0].TriggerType)
	}
}
