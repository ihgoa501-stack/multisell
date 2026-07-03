package toolbridge

import (
	"fmt"
	"sync"
	"time"
)

// DegradedErr is returned when a platform is marked degraded.
var DegradedErr = fmt.Errorf("toolbridge: platform is degraded, skipping execution")

// platformStats holds recent call stats for one external platform.
type platformStats struct {
	TotalCalls          int
	FailedCalls         int
	LastFailureAt       time.Time
	LastError           string
	ConsecutiveFailures int
	Degraded            bool
	DegradedAt          time.Time
}

// ExternalCallTracker monitors external platform call health.
type ExternalCallTracker struct {
	mu        sync.Mutex
	platforms map[string]*platformStats
	threshold int // consecutive failures to mark degraded (default 3)
}

// NewExternalCallTracker creates a tracker with custom degradation threshold.
func NewExternalCallTracker(threshold int) *ExternalCallTracker {
	if threshold <= 0 {
		threshold = 3
	}
	return &ExternalCallTracker{
		platforms: make(map[string]*platformStats),
		threshold: threshold,
	}
}

// RecordCall records the result of an external platform call.
func (t *ExternalCallTracker) RecordCall(platform string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ps, ok := t.platforms[platform]
	if !ok {
		ps = &platformStats{}
		t.platforms[platform] = ps
	}
	ps.TotalCalls++
	if err != nil {
		ps.FailedCalls++
		ps.LastFailureAt = time.Now()
		ps.LastError = err.Error()
		ps.ConsecutiveFailures++
		if ps.ConsecutiveFailures >= t.threshold {
			ps.Degraded = true
			ps.DegradedAt = time.Now()
		}
	} else {
		ps.ConsecutiveFailures = 0
		ps.Degraded = false
	}
}

// IsDegraded returns true if the platform is currently degraded.
func (t *ExternalCallTracker) IsDegraded(platform string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	ps, ok := t.platforms[platform]
	if !ok {
		return false
	}
	return ps.Degraded
}

// PlatformStatsSnapshot is a snapshot of platform stats for dashboard use.
type PlatformStatsSnapshot struct {
	Platform            string `json:"platform"`
	TotalCalls          int    `json:"total_calls"`
	FailedCalls         int    `json:"failed_calls"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Degraded            bool   `json:"degraded"`
	LastFailureAt       string `json:"last_failure_at,omitempty"`
	LastError           string `json:"last_error,omitempty"`
}

// Stats returns all platform stats for the dashboard.
func (t *ExternalCallTracker) Stats() []PlatformStatsSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]PlatformStatsSnapshot, 0, len(t.platforms))
	for name, ps := range t.platforms {
		s := PlatformStatsSnapshot{
			Platform:            name,
			TotalCalls:          ps.TotalCalls,
			FailedCalls:         ps.FailedCalls,
			ConsecutiveFailures: ps.ConsecutiveFailures,
			Degraded:            ps.Degraded,
			LastError:           ps.LastError,
		}
		if !ps.LastFailureAt.IsZero() {
			s.LastFailureAt = ps.LastFailureAt.Format(time.RFC3339)
		}
		result = append(result, s)
	}
	return result
}
