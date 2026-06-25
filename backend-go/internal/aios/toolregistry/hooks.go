package toolregistry

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ToolHook defines the interface for middleware hooks around tool execution.
// Hooks are executed in registration order before the handler (Before), and
// in reverse order after the handler (After). If any Before hook returns an
// error, execution is aborted and After hooks still run for cleanup.
type ToolHook interface {
	// Before is called before the tool handler executes. Return an error to
	// abort execution. The context may be enriched with additional values.
	Before(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error)

	// After is called after the tool handler completes (or after a Before hook
	// aborted execution). The output and err parameters carry the handler's
	// result, or nil and the abort error respectively. Errors from After hooks
	// are not propagated, but each hook is expected to handle its own errors.
	After(ctx context.Context, tool *Tool, input map[string]interface{}, output interface{}, err error) error
}

// ToolHookFunc adapts a plain function to the ToolHook interface.
// The function serves as the Before hook; After is a no-op.
// Use this for simple hooks that only need pre-execution logic.
type ToolHookFunc func(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error)

// Before implements ToolHook by delegating to the wrapped function.
func (f ToolHookFunc) Before(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
	return f(ctx, tool, input)
}

// After is a no-op implementation for ToolHookFunc.
func (f ToolHookFunc) After(_ context.Context, _ *Tool, _ map[string]interface{}, _ interface{}, _ error) error {
	return nil
}

// --- AuditHook ---

// AuditHook logs every tool call with its input and output or error.
// Use this for observability, compliance, and debugging.
type AuditHook struct {
	logger *zap.Logger
}

// NewAuditHook creates an AuditHook that logs tool invocations via the
// provided zap logger. Log level is Info on success and Error on failure.
func NewAuditHook(logger *zap.Logger) *AuditHook {
	return &AuditHook{logger: logger}
}

// Before logs the tool call input before execution.
func (h *AuditHook) Before(ctx context.Context, tool *Tool, input map[string]interface{}) (context.Context, error) {
	h.logger.Info("tool call started",
		zap.String("tool", tool.Name),
		zap.String("version", tool.Version),
		zap.Any("input", input),
	)
	return ctx, nil
}

// After logs the tool call result after execution.
func (h *AuditHook) After(_ context.Context, tool *Tool, _ map[string]interface{}, output interface{}, err error) error {
	if err != nil {
		h.logger.Error("tool call failed",
			zap.String("tool", tool.Name),
			zap.String("version", tool.Version),
			zap.Error(err),
		)
		return nil
	}
	h.logger.Info("tool call completed",
		zap.String("tool", tool.Name),
		zap.String("version", tool.Version),
		zap.Any("output", output),
	)
	return nil
}

// --- PermissionHook ---

// PermissionHook checks that the caller has all required permissions for the
// tool being invoked. It uses a caller-provided function to resolve the
// caller's permissions from the context.
type PermissionHook struct {
	getUserPermissions func(ctx context.Context) []string
}

// NewPermissionHook creates a PermissionHook using the provided function to
// resolve caller permissions from context. The function should extract the
// caller identity (e.g., agent ID, user ID) from context and return their
// permission list.
func NewPermissionHook(getPerms func(ctx context.Context) []string) *PermissionHook {
	return &PermissionHook{getUserPermissions: getPerms}
}

// Before checks that the caller has all required permissions for the tool.
// If the tool has no RequiredPermissions, access is granted.
func (h *PermissionHook) Before(ctx context.Context, tool *Tool, _ map[string]interface{}) (context.Context, error) {
	if len(tool.RequiredPermissions) == 0 {
		return ctx, nil
	}
	callerPerms := h.getUserPermissions(ctx)
	permSet := make(map[string]struct{}, len(callerPerms))
	for _, p := range callerPerms {
		permSet[p] = struct{}{}
	}
	for _, required := range tool.RequiredPermissions {
		if _, ok := permSet[required]; !ok {
			return ctx, ErrPermissionDenied{Name: tool.Name, Permission: required}
		}
	}
	return ctx, nil
}

// After is a no-op for PermissionHook.
func (h *PermissionHook) After(_ context.Context, _ *Tool, _ map[string]interface{}, _ interface{}, _ error) error {
	return nil
}

// --- RateLimitHook ---

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// RateLimitHook limits the number of tool calls per time window.
// It tracks call counts per tool name within sliding windows and rejects
// calls that exceed the configured maximum.
type RateLimitHook struct {
	mu       sync.Mutex
	limits   map[string]*rateLimitEntry
	maxCalls int
	window   time.Duration
	logger   *zap.Logger
}

// NewRateLimitHook creates a RateLimitHook that allows at most maxCalls
// invocations per tool within the given time window.
func NewRateLimitHook(maxCalls int, window time.Duration, logger *zap.Logger) *RateLimitHook {
	return &RateLimitHook{
		limits:   make(map[string]*rateLimitEntry),
		maxCalls: maxCalls,
		window:   window,
		logger:   logger,
	}
}

// Before checks whether the tool call would exceed the rate limit for the
// tool name. If so, it returns an ErrRateLimited to abort execution.
func (h *RateLimitHook) Before(ctx context.Context, tool *Tool, _ map[string]interface{}) (context.Context, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.limits[tool.Name]
	now := time.Now()

	// No existing entry or window expired: start a new window
	if !ok || now.Sub(entry.windowStart) >= h.window {
		h.limits[tool.Name] = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return ctx, nil
	}

	// Within window: check limit
	if entry.count >= h.maxCalls {
		h.logger.Warn("rate limit exceeded",
			zap.String("tool", tool.Name),
			zap.Int("max_calls", h.maxCalls),
			zap.Duration("window", h.window),
		)
		return ctx, ErrRateLimited{Name: tool.Name}
	}

	entry.count++
	return ctx, nil
}

// After is a no-op for RateLimitHook.
func (h *RateLimitHook) After(_ context.Context, _ *Tool, _ map[string]interface{}, _ interface{}, _ error) error {
	return nil
}

// --- CircuitBreakerHook ---

// CircuitState represents the current state of a per-tool circuit breaker.
type CircuitState int

const (
	// CircuitClosed is the normal operating state — requests are allowed.
	CircuitClosed CircuitState = iota
	// CircuitOpen rejects all requests until the recovery timeout expires.
	CircuitOpen
	// CircuitHalfOpen allows a limited number of requests to test recovery.
	CircuitHalfOpen
)

// circuitStatus tracks the runtime state of a single tool's circuit breaker.
type circuitStatus struct {
	state           CircuitState
	failureCount    int
	lastFailureTime time.Time
	halfOpenSuccess int
	mu              sync.Mutex
}

// CircuitBreakerHook implements per-tool circuit breaker pattern.
// It tracks consecutive failures for each tool and opens the circuit when
// the configured failure threshold is reached. After the recovery timeout,
// the circuit transitions to half-open to test recovery.
type CircuitBreakerHook struct {
	states map[string]*circuitStatus
	mu     sync.Mutex
	logger *zap.Logger
}

// NewCircuitBreakerHook creates a CircuitBreakerHook.
func NewCircuitBreakerHook(logger *zap.Logger) *CircuitBreakerHook {
	return &CircuitBreakerHook{
		states: make(map[string]*circuitStatus),
		logger: logger,
	}
}

// getOrCreateStatus returns the circuit breaker status for a tool, creating
// it if necessary. Returns nil if the tool has no circuit breaker config.
func (h *CircuitBreakerHook) getOrCreateStatus(tool *Tool) *circuitStatus {
	if tool.CircuitBreaker == nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	key := tool.Key()
	status, ok := h.states[key]
	if !ok {
		status = &circuitStatus{state: CircuitClosed}
		h.states[key] = status
	}
	return status
}

// Before checks if the circuit breaker is open and rejects the request if so.
// If the recovery timeout has elapsed, transitions from open to half-open.
// If in half-open and the max trial count is reached, rejects.
func (h *CircuitBreakerHook) Before(ctx context.Context, tool *Tool, _ map[string]interface{}) (context.Context, error) {
	if tool.CircuitBreaker == nil {
		return ctx, nil
	}

	status := h.getOrCreateStatus(tool)
	if status == nil {
		return ctx, nil
	}

	status.mu.Lock()
	defer status.mu.Unlock()

	switch status.state {
	case CircuitOpen:
		if time.Since(status.lastFailureTime) > tool.CircuitBreaker.RecoveryTimeout {
			status.state = CircuitHalfOpen
			status.halfOpenSuccess = 0
			h.logger.Info("circuit breaker half-open",
				zap.String("tool", tool.Name),
				zap.String("version", tool.Version),
			)
			return ctx, nil
		}
		return ctx, ErrCircuitOpen{Name: tool.Name}

	case CircuitHalfOpen:
		if status.halfOpenSuccess >= tool.CircuitBreaker.HalfOpenMax {
			return ctx, ErrCircuitOpen{Name: tool.Name}
		}
		return ctx, nil
	}

	return ctx, nil
}

// After records the result of a tool call and updates circuit state.
// On failure: increments the failure counter, opens the circuit if threshold met.
// On success in half-open: increments success counter, closes the circuit if threshold met.
// On success in closed: resets the failure counter.
func (h *CircuitBreakerHook) After(_ context.Context, tool *Tool, _ map[string]interface{}, _ interface{}, cbErr error) error {
	if tool.CircuitBreaker == nil {
		return nil
	}

	status := h.getOrCreateStatus(tool)
	if status == nil {
		return nil
	}

	status.mu.Lock()
	defer status.mu.Unlock()

	switch {
	case cbErr != nil:
		// Don't count our own circuit-open errors as tool failures
		if _, ok := cbErr.(ErrCircuitOpen); ok {
			return nil
		}
		status.failureCount++
		status.lastFailureTime = time.Now()

		if status.state == CircuitHalfOpen || status.failureCount >= tool.CircuitBreaker.FailureThreshold {
			status.state = CircuitOpen
			status.halfOpenSuccess = 0
			h.logger.Warn("circuit breaker opened",
				zap.String("tool", tool.Name),
				zap.String("version", tool.Version),
				zap.Int("failure_count", status.failureCount),
			)
		}

	default:
		// Handler succeeded
		if status.state == CircuitHalfOpen {
			status.halfOpenSuccess++
			if status.halfOpenSuccess >= tool.CircuitBreaker.HalfOpenMax {
				status.state = CircuitClosed
				status.failureCount = 0
				h.logger.Info("circuit breaker closed",
					zap.String("tool", tool.Name),
					zap.String("version", tool.Version),
				)
			}
		} else {
			status.failureCount = 0
		}
	}

	return nil
}
