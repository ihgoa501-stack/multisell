package pipeline

import (
	"sync"
	"sync/atomic"
	"time"
)

// --- Circuit Breaker for Agent Pipeline ---

// BreakerState represents the state of a circuit breaker.
type BreakerState int

const (
	BreakerClosed   BreakerState = iota // normal operation
	BreakerOpen                         // failing, fast-fail
	BreakerHalfOpen                     // probing recovery
)

func (s BreakerState) String() string {
	switch s {
	case BreakerClosed:
		return "CLOSED"
	case BreakerOpen:
		return "OPEN"
	case BreakerHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// BreakerConfig configures a single circuit breaker.
type BreakerConfig struct {
	// Threshold is the failure ratio that triggers open (0.0–1.0). Default 0.5.
	Threshold float64
	// WindowSize is the number of recent calls to evaluate. Default 10.
	WindowSize int
	// Cooldown is how long to stay OPEN before HALF_OPEN. Default 30s.
	Cooldown time.Duration
	// HalfOpenMax is how many probe calls to allow in HALF_OPEN. Default 1.
	HalfOpenMax int
}

// record is a single invocation outcome within the sliding window.
type record struct {
	failed bool
	at     time.Time
}

// CircuitBreaker tracks recent call outcomes for one agent.
type CircuitBreaker struct {
	mu     sync.RWMutex
	config BreakerConfig
	state  BreakerState
	window []record
	// halfOpenCount tracks probe attempts in HALF_OPEN.
	halfOpenCount int
	// openedAt is when the breaker transitioned to OPEN.
	openedAt time.Time
	// Total counters for observability.
	totalSuccess atomic.Int64
	totalFailure atomic.Int64
}

// DefaultBreakerConfig returns a sensible default configuration.
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		Threshold:   0.5,
		WindowSize:  10,
		Cooldown:    30 * time.Second,
		HalfOpenMax: 1,
	}
}

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(cfg BreakerConfig) *CircuitBreaker {
	if cfg.Threshold <= 0 || cfg.Threshold > 1 {
		cfg.Threshold = 0.5
	}
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 10
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 1
	}
	return &CircuitBreaker{config: cfg, state: BreakerClosed}
}

// Allow returns true if the call should proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	state := cb.state
	cooldown := cb.config.Cooldown
	openedAt := cb.openedAt
	cb.mu.RUnlock()

	switch state {
	case BreakerClosed:
		return true
	case BreakerOpen:
		if time.Since(openedAt) >= cooldown {
			cb.mu.Lock()
			// Double-check after acquiring write lock.
			if cb.state == BreakerOpen && time.Since(cb.openedAt) >= cb.config.Cooldown {
				cb.state = BreakerHalfOpen
				cb.halfOpenCount = 1
			}
			allowed := cb.state == BreakerHalfOpen
			cb.mu.Unlock()
			return allowed
		}
		return false
	case BreakerHalfOpen:
		cb.mu.Lock()
		allowed := cb.halfOpenCount < cb.config.HalfOpenMax
		if allowed {
			cb.halfOpenCount++
		}
		cb.mu.Unlock()
		return allowed
	}
	return true
}

// Record records a call outcome and transitions state if needed.
func (cb *CircuitBreaker) Record(success bool) {
	now := time.Now()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.totalSuccess.Add(1)
	} else {
		cb.totalFailure.Add(1)
	}

	// Append to sliding window.
	cb.window = append(cb.window, record{failed: !success, at: now})

	// Prune records outside the window.
	windowStart := now.Add(-cb.config.Cooldown * 3) // keep ~3 cooldowns
	cut := 0
	for i, r := range cb.window {
		if r.at.After(windowStart) {
			cut = i
			break
		}
	}
	if cut > 0 && cut <= len(cb.window) {
		cb.window = cb.window[cut:]
	}

	// Only evaluate when window is full enough.
	if len(cb.window) < cb.config.WindowSize {
		return
	}

	// Trim to window size.
	if len(cb.window) > cb.config.WindowSize {
		cb.window = cb.window[len(cb.window)-cb.config.WindowSize:]
	}

	// Count failures in window.
	var failures int
	for _, r := range cb.window {
		if r.failed {
			failures++
		}
	}
	ratio := float64(failures) / float64(len(cb.window))

	switch cb.state {
	case BreakerClosed:
		if ratio >= cb.config.Threshold {
			cb.state = BreakerOpen
			cb.openedAt = now
		}
	case BreakerHalfOpen:
		if !success {
			// Probe failed — back to open.
			cb.state = BreakerOpen
			cb.openedAt = now
		} else {
			// Probe succeeded — close.
			cb.state = BreakerClosed
			cb.window = nil // reset window
		}
	}
}

// Reset forces the breaker back to CLOSED state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = BreakerClosed
	cb.window = nil
	cb.halfOpenCount = 0
}

// State returns the current state (thread-safe).
func (cb *CircuitBreaker) State() BreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current breaker statistics.
func (cb *CircuitBreaker) Stats() (state BreakerState, success, failure int64, windowSize int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, cb.totalSuccess.Load(), cb.totalFailure.Load(), len(cb.window)
}

// --- Registry ---

// CircuitBreakerRegistry manages per-agent circuit breakers.
type CircuitBreakerRegistry struct {
	mu      sync.RWMutex
	config  BreakerConfig
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerRegistry creates a registry with the given default config.
func NewCircuitBreakerRegistry(cfg BreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		config:   cfg,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate returns the breaker for the given agent, creating one if needed.
func (r *CircuitBreakerRegistry) GetOrCreate(agentID string) *CircuitBreaker {
	r.mu.RLock()
	b, ok := r.breakers[agentID]
	r.mu.RUnlock()
	if ok {
		return b
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check.
	if b, ok := r.breakers[agentID]; ok {
		return b
	}
	b = NewCircuitBreaker(r.config)
	r.breakers[agentID] = b
	return b
}

// Allow delegates to the named agent's breaker.
func (r *CircuitBreakerRegistry) Allow(agentID string) bool {
	return r.GetOrCreate(agentID).Allow()
}

// Record delegates to the named agent's breaker.
func (r *CircuitBreakerRegistry) Record(agentID string, success bool) {
	r.GetOrCreate(agentID).Record(success)
}

// ResetAll resets all registered breakers.
func (r *CircuitBreakerRegistry) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.breakers {
		b.Reset()
	}
}

// BreakerNames returns the list of known agent IDs.
func (r *CircuitBreakerRegistry) BreakerNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.breakers))
	for name := range r.breakers {
		names = append(names, name)
	}
	return names
}
