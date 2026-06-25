package toolregistry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// -- helpers --

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestTool(name, version, squad string, risk RiskLevel) *Tool {
	return &Tool{
		Name:        name,
		Version:     version,
		Description: "test tool " + name,
		Squad:       squad,
		RiskLevel:   risk,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	}
}

// -- Tool.Key --

func TestToolKey(t *testing.T) {
	tool := newTestTool("purchase_order.create", "1.0.0", "fulfillment", RiskMedium)
	if key := tool.Key(); key != "purchase_order.create@1.0.0" {
		t.Errorf("expected key %q, got %q", "purchase_order.create@1.0.0", key)
	}
}

// -- NewToolRegistry --

func TestNewToolRegistry(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.ToolCount() != 0 {
		t.Errorf("expected empty registry, got %d tools", r.ToolCount())
	}
}

// -- Register --

func TestRegister_Success(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := newTestTool("test.read", "1.0.0", "default", RiskLow)
	r.Register(tool)
	if r.ToolCount() != 1 {
		t.Errorf("expected 1 tool, got %d", r.ToolCount())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := newTestTool("test.dup", "1.0.0", "default", RiskLow)
	r.Register(tool)

	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	r.Register(tool)
}

func TestRegister_NilTool(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on nil tool")
		}
	}()
	r.Register(nil)
}

func TestRegister_EmptyName(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := &Tool{
		Name:    "",
		Version: "1.0.0",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) { return nil, nil },
	}
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on empty name")
		}
	}()
	r.Register(tool)
}

func TestRegister_EmptyVersion(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := &Tool{
		Name:    "test.no-version",
		Version: "",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) { return nil, nil },
	}
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on empty version")
		}
	}()
	r.Register(tool)
}

// -- Lookup --

func TestLookup_ExactKey(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := newTestTool("test.read", "1.0.0", "default", RiskLow)
	r.Register(tool)

	found, ok := r.Lookup("test.read@1.0.0")
	if !ok {
		t.Fatal("expected to find tool by exact key")
	}
	if found.Name != "test.read" || found.Version != "1.0.0" {
		t.Errorf("unexpected tool: %+v", found)
	}
}

func TestLookup_ByName(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	tool := newTestTool("test.read", "1.0.0", "default", RiskLow)
	r.Register(tool)

	found, ok := r.Lookup("test.read")
	if !ok {
		t.Fatal("expected to find tool by name")
	}
	if found.Name != "test.read" {
		t.Errorf("expected tool name %q, got %q", "test.read", found.Name)
	}
}

func TestLookup_LatestVersion(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("test.read", "1.0.0", "default", RiskLow))
	r.Register(newTestTool("test.read", "1.1.0", "default", RiskLow))
	r.Register(newTestTool("test.read", "2.0.0", "default", RiskLow))

	found, ok := r.Lookup("test.read")
	if !ok {
		t.Fatal("expected to find tool by name")
	}
	if found.Version != "2.0.0" {
		t.Errorf("expected latest version 2.0.0, got %s", found.Version)
	}
}

func TestLookup_NotFound(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	_, ok := r.Lookup("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent tool")
	}
}

func TestLookup_EmptyRegistry(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	_, ok := r.Lookup("anything")
	if ok {
		t.Error("expected not found on empty registry")
	}
}

// -- List --

func TestList_All(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("a.read", "1.0.0", "alpha", RiskLow))
	r.Register(newTestTool("b.write", "1.0.0", "beta", RiskMedium))
	r.Register(newTestTool("c.delete", "1.0.0", "gamma", RiskHigh))

	tools := r.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestList_BySquad(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("a.read", "1.0.0", "alpha", RiskLow))
	r.Register(newTestTool("b.write", "1.0.0", "beta", RiskMedium))
	r.Register(newTestTool("c.delete", "1.0.0", "alpha", RiskHigh))

	tools := r.List("alpha")
	if len(tools) != 2 {
		t.Errorf("expected 2 alpha tools, got %d", len(tools))
	}
	for _, tool := range tools {
		if tool.Squad != "alpha" {
			t.Errorf("expected all tools to be from squad alpha, got %s", tool.Squad)
		}
	}
}

func TestList_ByMultipleSquads(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("a.read", "1.0.0", "alpha", RiskLow))
	r.Register(newTestTool("b.write", "1.0.0", "beta", RiskMedium))
	r.Register(newTestTool("c.delete", "1.0.0", "gamma", RiskHigh))

	tools := r.List("alpha", "gamma")
	if len(tools) != 2 {
		t.Errorf("expected 2 tools for alpha+gamma, got %d", len(tools))
	}
}

func TestList_NoMatch(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("a.read", "1.0.0", "alpha", RiskLow))

	tools := r.List("nonexistent")
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

// -- AddHook & Call --

func TestCall_ToolNotFound(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	_, err := r.Call(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestCall_Success(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("test.echo", "1.0.0", "default", RiskLow))

	out, err := r.Call(context.Background(), "test.echo", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected output %q, got %v", "ok", out)
	}
}

func TestCall_HandlerError(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(&Tool{
		Name:    "test.fail",
		Version: "1.0.0",
		Squad:   "default",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return nil, errors.New("handler error")
		},
	})

	_, err := r.Call(context.Background(), "test.fail", nil)
	if err == nil || err.Error() != "handler error" {
		t.Errorf("expected handler error, got %v", err)
	}
}

func TestCall_NilContext(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(newTestTool("test.nilctx", "1.0.0", "default", RiskLow))

	out, err := r.Call(nil, "test.nilctx", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected output %q, got %v", "ok", out)
	}
}

func TestCall_NilInput(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(&Tool{
		Name:    "test.nil-input",
		Version: "1.0.0",
		Squad:   "default",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			if input == nil {
				return nil, errors.New("handler received nil input")
			}
			return input, nil
		},
	})

	out, err := r.Call(context.Background(), "test.nil-input", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := out.(map[string]interface{})
	if !ok {
		t.Errorf("expected map output (normalized input), got %T", out)
	}
}

func TestCall_HookChainOrder(t *testing.T) {
	r := NewToolRegistry(newTestLogger())

	var order []string
	hook1 := ToolHookFunc(func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
		order = append(order, "hook1:before")
		return ctx, nil
	})
	hook2 := ToolHookFunc(func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
		order = append(order, "hook2:before")
		return ctx, nil
	})
	r.AddHook(hook1)
	r.AddHook(hook2)

	r.Register(&Tool{
		Name:    "test.order",
		Version: "1.0.0",
		Squad:   "default",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			order = append(order, "handler")
			return "ok", nil
		},
	})

	_, err := r.Call(context.Background(), "test.order", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"hook1:before", "hook2:before", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected order %v, got %v", expected, order)
	}
	for i, step := range expected {
		if order[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, order[i])
		}
	}
}

func TestCall_HookAbortsExecution(t *testing.T) {
	r := NewToolRegistry(newTestLogger())

	hook := ToolHookFunc(func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
		return ctx, errors.New("hook abort")
	})
	r.AddHook(hook)

	var handlerCalled bool
	r.Register(&Tool{
		Name:    "test.abort",
		Version: "1.0.0",
		Squad:   "default",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			handlerCalled = true
			return "ok", nil
		},
	})

	_, err := r.Call(context.Background(), "test.abort", nil)
	if err == nil || err.Error() != "hook abort" {
		t.Errorf("expected hook abort error, got %v", err)
	}
	if handlerCalled {
		t.Error("handler should not have been called after hook abort")
	}
}

func TestCall_WithCircuitAndMaxDuration(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	r.Register(&Tool{
		Name:    "test.combo",
		Version: "1.0.0",
		Squad:   "default",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  time.Second,
			HalfOpenMax:      1,
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})

	out, err := r.Call(context.Background(), "test.combo", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected output %q, got %v", "ok", out)
	}
}

// -- AuditHook --

func TestAuditHook(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hook := NewAuditHook(logger)

	tool := newTestTool("audit.test", "1.0.0", "default", RiskLow)
	ctx, err := hook.Before(context.Background(), tool, map[string]interface{}{"foo": "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Error("expected non-nil context")
	}

	err = hook.After(context.Background(), tool, map[string]interface{}{"foo": "bar"}, "success", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = hook.After(context.Background(), tool, nil, nil, errors.New("handler error"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = logger.Sync()
}

// -- PermissionHook --

func TestPermissionHook_Allows(t *testing.T) {
	hook := NewPermissionHook(func(ctx context.Context) []string {
		return []string{"perm1", "perm2"}
	})

	tool := &Tool{
		Name:                "test.perm",
		Version:             "1.0.0",
		RequiredPermissions: []string{"perm1"},
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected allowed, got %v", err)
	}
}

func TestPermissionHook_Denies(t *testing.T) {
	hook := NewPermissionHook(func(ctx context.Context) []string {
		return []string{"perm1"}
	})

	tool := &Tool{
		Name:                "test.perm",
		Version:             "1.0.0",
		RequiredPermissions: []string{"perm1", "perm2"},
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected permission denied")
	}
	_, ok := err.(ErrPermissionDenied)
	if !ok {
		t.Errorf("expected ErrPermissionDenied, got %T: %v", err, err)
	}
}

func TestPermissionHook_NoPermissionsRequired(t *testing.T) {
	hook := NewPermissionHook(func(ctx context.Context) []string {
		return []string{"perm1"}
	})

	tool := &Tool{
		Name:    "test.noperm",
		Version: "1.0.0",
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected allowed (no permissions required), got %v", err)
	}
}

func TestPermissionHook_EmptyCallerPerms(t *testing.T) {
	hook := NewPermissionHook(func(ctx context.Context) []string {
		return nil
	})

	tool := &Tool{
		Name:                "test.need-perm",
		Version:             "1.0.0",
		RequiredPermissions: []string{"something"},
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected permission denied when caller has no permissions")
	}
	_, ok := err.(ErrPermissionDenied)
	if !ok {
		t.Errorf("expected ErrPermissionDenied, got %T: %v", err, err)
	}
}

func TestPermissionHook_ComprehensiveMatch(t *testing.T) {
	hook := NewPermissionHook(func(ctx context.Context) []string {
		return []string{"a", "b", "c"}
	})

	tool := &Tool{
		Name:                "test.match",
		Version:             "1.0.0",
		RequiredPermissions: []string{"c", "a"},
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected allowed, got %v", err)
	}
}

// -- RateLimitHook --

func TestRateLimitHook_Allows(t *testing.T) {
	hook := NewRateLimitHook(3, time.Minute, newTestLogger())

	tool := newTestTool("rate.test", "1.0.0", "default", RiskLow)
	for i := 0; i < 3; i++ {
		_, err := hook.Before(context.Background(), tool, nil)
		if err != nil {
			t.Errorf("call %d: unexpected error: %v", i, err)
		}
	}
}

func TestRateLimitHook_Denies(t *testing.T) {
	hook := NewRateLimitHook(2, time.Minute, newTestLogger())

	tool := newTestTool("rate.test", "1.0.0", "default", RiskLow)
	_, _ = hook.Before(context.Background(), tool, nil)
	_, _ = hook.Before(context.Background(), tool, nil)

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected rate limit exceeded")
	}
	_, ok := err.(ErrRateLimited)
	if !ok {
		t.Errorf("expected ErrRateLimited, got %T: %v", err, err)
	}
}

func TestRateLimitHook_DifferentToolsIndependent(t *testing.T) {
	hook := NewRateLimitHook(1, time.Minute, newTestLogger())

	tool1 := newTestTool("rate.a", "1.0.0", "default", RiskLow)
	tool2 := newTestTool("rate.b", "1.0.0", "default", RiskLow)

	_, err := hook.Before(context.Background(), tool1, nil)
	if err != nil {
		t.Fatalf("tool1: unexpected error: %v", err)
	}
	_, err = hook.Before(context.Background(), tool2, nil)
	if err != nil {
		t.Fatalf("tool2 should be independent of tool1: %v", err)
	}

	_, err = hook.Before(context.Background(), tool1, nil)
	if err == nil {
		t.Fatal("tool1 should be rate limited")
	}
}

func TestRateLimitHook_WindowReset(t *testing.T) {
	hook := NewRateLimitHook(1, 50*time.Millisecond, newTestLogger())

	tool := newTestTool("rate.test", "1.0.0", "default", RiskLow)

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	_, err = hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected rate limit after max calls")
	}

	time.Sleep(60 * time.Millisecond)

	_, err = hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("should pass after window reset, got: %v", err)
	}
}

func TestRateLimitHook_AfterNoop(t *testing.T) {
	hook := NewRateLimitHook(1, time.Minute, newTestLogger())
	err := hook.After(context.Background(), nil, nil, nil, nil)
	if err != nil {
		t.Errorf("After should not error: %v", err)
	}
}

func TestRateLimitHook_ZeroMaxCalls(t *testing.T) {
	hook := NewRateLimitHook(0, time.Minute, newTestLogger())
	tool := newTestTool("rate.zero", "1.0.0", "default", RiskLow)
	// First call: creates a new window entry, always passes
	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Fatalf("first call should pass (new window created), got: %v", err)
	}
	// Second call: 1 >= 0 in same window -> rate limited
	_, err = hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected rate limit exceeded with 0 max calls on second request")
	}
	_, ok := err.(ErrRateLimited)
	if !ok {
		t.Errorf("expected ErrRateLimited, got %T: %v", err, err)
	}
}

// -- CircuitBreakerHook --

func TestCircuitBreaker_NoConfig(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:           "test.nocircuit",
		Version:        "1.0.0",
		CircuitBreaker: nil,
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("no circuit breaker should not block: %v", err)
	}
}

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.circuit",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  10 * time.Second,
			HalfOpenMax:      2,
		},
	}

	for i := 0; i < 3; i++ {
		_, err := hook.Before(context.Background(), tool, nil)
		if err != nil {
			t.Errorf("call %d (before failures): unexpected error: %v", i, err)
		}
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("handler error"))
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected circuit open error")
	}
	_, ok := err.(ErrCircuitOpen)
	if !ok {
		t.Errorf("expected ErrCircuitOpen, got %T: %v", err, err)
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.halfopen",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  50 * time.Millisecond,
			HalfOpenMax:      2,
		},
	}

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected circuit open")
	}

	time.Sleep(60 * time.Millisecond)

	_, err = hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected half-open to allow one request, got: %v", err)
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.recover",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  50 * time.Millisecond,
			HalfOpenMax:      2,
		},
	}

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	time.Sleep(60 * time.Millisecond)

	for i := 0; i < 2; i++ {
		_, err := hook.Before(context.Background(), tool, nil)
		if err != nil {
			t.Fatalf("half-open call %d: unexpected error: %v", i, err)
		}
		_ = hook.After(context.Background(), tool, nil, "success", nil)
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected circuit closed, got: %v", err)
	}
}

func TestCircuitBreaker_HalfOpenFailsBackToOpen(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.half-fail",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  50 * time.Millisecond,
			HalfOpenMax:      3,
		},
	}

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	time.Sleep(60 * time.Millisecond)

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Fatalf("half-open request should be allowed, got: %v", err)
	}
	_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))

	_, err = hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected circuit open after half-open failure")
	}
}

func TestCircuitBreaker_CircuitOpenErrorNotCounted(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.no-count",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  50 * time.Millisecond,
			HalfOpenMax:      1,
		},
	}

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected circuit open")
	}

	_ = hook.After(context.Background(), tool, nil, nil, err)

	time.Sleep(60 * time.Millisecond)

	_, err = hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected half-open to allow, got: %v", err)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	hook := NewCircuitBreakerHook(newTestLogger())

	tool := &Tool{
		Name:    "test.reset",
		Version: "1.0.0",
		CircuitBreaker: &CircuitConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  10 * time.Second,
			HalfOpenMax:      1,
		},
	}

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	_, _ = hook.Before(context.Background(), tool, nil)
	_ = hook.After(context.Background(), tool, nil, "ok", nil)

	for i := 0; i < 2; i++ {
		_, _ = hook.Before(context.Background(), tool, nil)
		_ = hook.After(context.Background(), tool, nil, nil, errors.New("fail"))
	}

	_, err := hook.Before(context.Background(), tool, nil)
	if err != nil {
		t.Errorf("expected circuit closed after success-reset, got: %v", err)
	}
}

// -- ToolHookFunc.After --

func TestToolHookFunc_AfterNoop(t *testing.T) {
	fn := ToolHookFunc(func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
		return ctx, nil
	})
	err := fn.After(context.Background(), nil, nil, nil, nil)
	if err != nil {
		t.Errorf("ToolHookFunc.After should be no-op, got: %v", err)
	}
}

// -- Concurrency --

func TestConcurrentRegisterAndLookup(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tool := newTestTool("conc.read", "1.0.0", "default", RiskLow)
			defer func() {
				_ = recover()
			}()
			r.Register(tool)
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = r.Lookup("conc.read")
			_ = r.List()
			_ = r.ToolCount()
		}(i)
	}

	wg.Wait()
}

func TestConcurrentRegisterAndCall(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	var wg sync.WaitGroup

	r.Register(&Tool{
		Name:    "conc.call",
		Version: "1.0.0",
		Squad:   "default",
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := r.Call(context.Background(), "conc.call", nil)
			if err != nil {
				t.Errorf("concurrent call %d failed: %v", i, err)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentAddHook(t *testing.T) {
	r := NewToolRegistry(newTestLogger())
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hook := ToolHookFunc(func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
				return ctx, nil
			})
			r.AddHook(hook)
		}()
	}

	wg.Wait()
}
