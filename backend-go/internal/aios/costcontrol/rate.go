package costcontrol

import (
	"sync"
	"time"
)

// spendEvent is one cost observation with a timestamp.
type spendEvent struct {
	at   time.Time
	cost float64
}

// BurstDetector tracks cost in a sliding time window.
// Thread-safe. Uses a slice of events with periodic cleanup.
// ponytail: in-memory only, restarts reset. Upgrade to Redis if multi-instance.
type BurstDetector struct {
	mu     sync.Mutex
	window time.Duration
	events []spendEvent
}

// NewBurstDetector creates a detector with the given window size.
func NewBurstDetector(window time.Duration) *BurstDetector {
	return &BurstDetector{window: window}
}

// Record adds a cost observation at the current time.
func (b *BurstDetector) Record(cost float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.events = append(b.events, spendEvent{at: now, cost: cost})
	b.evict(now)
}

// SpendLast returns the total cost in the sliding window.
func (b *BurstDetector) SpendLast() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.evict(now)
	var total float64
	for _, e := range b.events {
		total += e.cost
	}
	return total
}

// IsBurst returns true when the cost in the sliding window exceeds
// the allowed amount. This detects a sudden spend spike.
func (b *BurstDetector) IsBurst(allowed float64) bool {
	return b.SpendLast() > allowed
}

// evict drops events outside the window. Caller must hold mu.
func (b *BurstDetector) evict(now time.Time) {
	cutoff := now.Add(-b.window)
	i := 0
	for ; i < len(b.events); i++ {
		if b.events[i].at.After(cutoff) {
			break
		}
	}
	b.events = b.events[i:]
}
