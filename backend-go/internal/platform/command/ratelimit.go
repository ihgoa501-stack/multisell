package command

import (
	"fmt"
	"sync"
	"time"
)

// slidingWindow tracks request counts within a time window.
type slidingWindow struct {
	entries []time.Time
	limit   int
	window  time.Duration
}

func newSlidingWindow(limit int, window time.Duration) *slidingWindow {
	return &slidingWindow{
		entries: make([]time.Time, 0, limit),
		limit:   limit,
		window:  window,
	}
}

func (sw *slidingWindow) allow() bool {
	now := time.Now()
	cutoff := now.Add(-sw.window)
	// Trim expired entries
	start := 0
	for i, t := range sw.entries {
		if t.After(cutoff) {
			start = i
			break
		}
	}
	sw.entries = sw.entries[start:]
	if len(sw.entries) >= sw.limit {
		return false
	}
	sw.entries = append(sw.entries, now)
	return true
}

// ErrRateLimited is returned when an action is rate-limited.
var ErrRateLimited = fmt.Errorf("command: action rate limited")

// RateLimiter enforces per-(agent, action_type) rate limits using sliding windows.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]*slidingWindow
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with the given limit per window duration.
// Default: 20 actions/hour per (agent, action_type).
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Hour
	}
	return &RateLimiter{
		windows: make(map[string]*slidingWindow),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if an action from agentID of type actionType is allowed.
// Returns true if within limit, false if rate-limited.
func (rl *RateLimiter) Allow(agentID, actionType string) bool {
	key := agentID + ":" + actionType
	rl.mu.Lock()
	defer rl.mu.Unlock()
	sw, ok := rl.windows[key]
	if !ok {
		sw = newSlidingWindow(rl.limit, rl.window)
		rl.windows[key] = sw
	}
	return sw.allow()
}

// Reset clears all rate limit counters for the given agent.
func (rl *RateLimiter) Reset(agentID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	prefix := agentID + ":"
	for k := range rl.windows {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(rl.windows, k)
		}
	}
}
