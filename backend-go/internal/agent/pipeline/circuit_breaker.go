package pipeline

import (
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed is normal operation — calls are allowed.
	StateClosed State = iota
	// StateOpen means the circuit is broken — calls are blocked.
	StateOpen
	// StateHalfOpen means a probe call is allowed to test recovery.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig configures a single circuit breaker.
type CircuitBreakerConfig struct {
	// Threshold is the failure rate (0.0–1.0) that triggers the OPEN state.
	// Default: 0.5 (50%). Thresold is only checked when the sliding window
	// reaches WindowSize entries.
	Threshold float64

	// WindowSize is the number of recent calls to track.
	// Default: 10.
	WindowSize int

	// Cooldown is how long the breaker stays OPEN before transitioning to
	// HALF_OPEN and allowing a probe call.
	// Default: 30s.
	Cooldown time.Duration

	// HalfOpenMax is the maximum number of probe calls allowed in HALF_OPEN
	// state before the breaker decides.
	// Default: 1.
	HalfOpenMax int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Threshold:   0.5,
		WindowSize:  10,
		Cooldown:    30 * time.Second,
		HalfOpenMax: 1,
	}
}

// count records a single call result in the sliding window.
type count struct {
	success bool
	time    time.Time
}

// CircuitBreaker tracks success/failure for a single target and implements
// the circuit breaker pattern with a sliding window.
type CircuitBreaker struct {
	mu         sync.RWMutex
	state      State
	config     CircuitBreakerConfig
	window     []count
	lastOpened time.Time    // when the circuit was last opened
	halfOpenN  int          // calls allowed in current half-open window
	createdAt  time.Time
}

// NewCircuitBreaker creates a circuit breaker with the given config.
// If config has zero fields, defaults are applied per-field.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cfg := applyConfigDefaults(config)
	return &CircuitBreaker{
		state:     StateClosed,
		config:    cfg,
		window:    make([]count, 0, cfg.WindowSize),
		createdAt: time.Now(),
	}
}

func applyConfigDefaults(cfg CircuitBreakerConfig) CircuitBreakerConfig {
	def := DefaultCircuitBreakerConfig()
	if cfg.Threshold <= 0 || cfg.Threshold > 1 {
		cfg.Threshold = def.Threshold
	}
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = def.WindowSize
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = def.Cooldown
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = def.HalfOpenMax
	}
	return cfg
}

// Allow checks whether a call to the protected target is permitted based on
// the current circuit breaker state.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if cooldown has elapsed -> transition to HALF_OPEN.
		if time.Since(cb.lastOpened) >= cb.config.Cooldown {
			cb.state = StateHalfOpen
			cb.halfOpenN = 0
			return true
		}
		return false
	case StateHalfOpen:
		if cb.halfOpenN < cb.config.HalfOpenMax {
			cb.halfOpenN++
			return true
		}
		return false
	default:
		return true
	}
}

// Record records the outcome of a call and transitions state accordingly.
func (cb *CircuitBreaker) Record(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.recordClosed(success)
	case StateHalfOpen:
		cb.recordHalfOpen(success)
	default:
		// OPEN state: ignore (shouldn't normally arrive here).
	}
}

func (cb *CircuitBreaker) recordClosed(success bool) {
	// Add to window.
	cb.window = append(cb.window, count{success: success, time: time.Now()})

	// Trim window to WindowSize.
	if len(cb.window) > cb.config.WindowSize {
		cb.window = cb.window[len(cb.window)-cb.config.WindowSize:]
	}

	// Check failure rate only when the window is at capacity.
	if len(cb.window) == cb.config.WindowSize && cb.failureRateLocked() >= cb.config.Threshold {
		cb.state = StateOpen
		cb.lastOpened = time.Now()
	}
}

func (cb *CircuitBreaker) recordHalfOpen(success bool) {
	if success {
		// Probe succeeded — reset to CLOSED.
		cb.state = StateClosed
		cb.window = cb.window[:0] // reset window
	} else {
		// Probe failed — back to OPEN.
		cb.state = StateOpen
		cb.lastOpened = time.Now()
	}
}

// failureRateLocked returns the failure rate (0.0–1.0) in the current window.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) failureRateLocked() float64 {
	if len(cb.window) == 0 {
		return 0
	}
	var failures int
	for _, c := range cb.window {
		if !c.success {
			failures++
		}
	}
	return float64(failures) / float64(len(cb.window))
}

// FailureRate returns the failure rate (0.0–1.0) in the current window.
func (cb *CircuitBreaker) FailureRate() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureRateLocked()
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// WindowSize returns the number of calls in the current window.
func (cb *CircuitBreaker) WindowSize() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return len(cb.window)
}

// Reset forces the breaker back to CLOSED and clears the window.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.window = cb.window[:0]
	cb.lastOpened = time.Time{}
	cb.halfOpenN = 0
}

// =============================================================================
// CircuitBreakerRegistry — manages a map of named breakers
// =============================================================================

// CircuitBreakerRegistry holds all circuit breakers, keyed by agent ID.
// It is safe for concurrent use.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerRegistry creates an empty registry.
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate returns the breaker for the given name, creating it with the
// default config if it doesn't exist.
func (r *CircuitBreakerRegistry) GetOrCreate(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[name]
	r.mu.RUnlock()
	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	cb = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	r.breakers[name] = cb
	return cb
}

// GetOrCreateWithConfig returns the breaker for the given name, creating it
// with the specified config if it doesn't exist.
func (r *CircuitBreakerRegistry) GetOrCreateWithConfig(name string, config CircuitBreakerConfig) *CircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[name]
	r.mu.RUnlock()
	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	cb = NewCircuitBreaker(config)
	r.breakers[name] = cb
	return cb
}

// Get returns the breaker for the given name, or nil if not found.
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.breakers[name]
}

// Allow checks whether a call to the named target is permitted.
func (r *CircuitBreakerRegistry) Allow(name string) bool {
	return r.GetOrCreate(name).Allow()
}

// Record records the outcome of a call to the named target.
func (r *CircuitBreakerRegistry) Record(name string, success bool) {
	r.GetOrCreate(name).Record(success)
}

// ResetAll resets all breakers to CLOSED.
func (r *CircuitBreakerRegistry) ResetAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cb := range r.breakers {
		cb.Reset()
	}
}

// BreakerNames returns a copy of all registered breaker names.
func (r *CircuitBreakerRegistry) BreakerNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.breakers))
	for name := range r.breakers {
		names = append(names, name)
	}
	return names
}
