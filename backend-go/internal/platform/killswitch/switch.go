// Package killswitch provides a global production write kill switch.
//
// When enabled, all high-risk production mutations are blocked:
//   - Price changes
//   - Inventory changes
//   - Order cancellations
//   - Platform publishing
//   - Refunds
//   - Permission / RBAC changes
//   - Agent autonomous execution
//
// The switch is in-memory by default (survives config reload but not restart).
// For persistent state across restarts, pair with the DB-backed config.
package killswitch

import (
	"sync"
	"sync/atomic"
)

var (
	mu     sync.RWMutex
	active atomic.Bool
	reason string
)

// IsActive returns true if the production write kill switch is engaged.
func IsActive() bool {
	return active.Load()
}

// Reason returns why the kill switch was activated (empty string if inactive).
func Reason() string {
	mu.RLock()
	defer mu.RUnlock()
	return reason
}

// Activate engages the kill switch, blocking all production writes.
// Provide a reason for the audit log.
func Activate(r string) {
	mu.Lock()
	defer mu.Unlock()
	active.Store(true)
	reason = r
}

// Deactivate disengages the kill switch, restoring normal operation.
func Deactivate() {
	mu.Lock()
	defer mu.Unlock()
	active.Store(false)
	reason = ""
}
