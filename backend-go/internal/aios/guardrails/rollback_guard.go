package guardrails

import (
	"go.uber.org/zap"
	"context"
	"fmt"
	"sync"
	"time"
)

// RollbackEntry records a single action that can be rolled back.
type RollbackEntry struct {
	// ActionID uniquely identifies the action to roll back.
	ActionID string

	// ActionType describes the kind of action (e.g. "purchase", "replenish").
	ActionType string

	// OriginalState captures the state before the action was applied.
	// Typically contains fields like "before_quantity", "before_price", etc.
	OriginalState map[string]interface{}

	// RollbackFunc is the compensating function to undo the action.
	RollbackFunc func(ctx context.Context) error

	// Status tracks the lifecycle: "pending" | "rolled_back" | "failed".
	Status string

	// CreatedAt when the entry was recorded.
	CreatedAt time.Time

	// RolledBackAt is set when the rollback completes successfully.
	RolledBackAt *time.Time
}

// RollbackGuard implements L5 rollback guardrail for recording and
// undoing executed actions. Unlike L1-L4 guards, RollbackGuard does
// not implement the Guardrail Check interface in the traditional sense
// (it does not block actions). Instead, its Check method is a
// pass-through that wraps the action recording at the end of the chain.
//
// The primary API is:
//   - Record: register a compensatable action after successful execution.
//   - Rollback: trigger the compensating function for a recorded action.
//   - ListPending: enumerate actions eligible for rollback.
type RollbackGuard struct {
	entries []RollbackEntry
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewRollbackGuard creates a new RollbackGuard.
func NewRollbackGuard(logger *zap.Logger) *RollbackGuard {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RollbackGuard{
		entries: make([]RollbackEntry, 0),
		logger:  logger,
	}
}

// Name returns "rollback_guard".
func (g *RollbackGuard) Name() string {
	return "rollback_guard"
}

// Check implements the Guardrail interface. For the rollback guard, this
// is a pass-through that always passes — rollbacks are recorded separately
// via Record() rather than checked during the action approval flow.
func (g *RollbackGuard) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "rollback guard — always passes during check",
		Risk:    "low",
	}, nil
}

// Record registers an action that can be rolled back later.
func (g *RollbackGuard) Record(entry RollbackEntry) {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry.Status = "pending"
	entry.CreatedAt = time.Now()
	g.entries = append(g.entries, entry)

	g.logger.Info("rollback entry recorded",
		zap.String("action_id", entry.ActionID),
		zap.String("action_type", entry.ActionType),
	)
}

// Rollback attempts to undo a previously recorded action by ActionID.
// Returns an error if the action is not found, already rolled back,
// or the compensating function fails.
func (g *RollbackGuard) Rollback(actionID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, entry := range g.entries {
		if entry.ActionID != actionID {
			continue
		}

		if entry.Status == "rolled_back" {
			return fmt.Errorf("action %q has already been rolled back", actionID)
		}

		if entry.RollbackFunc == nil {
			g.entries[i].Status = "failed"
			return fmt.Errorf("action %q has no rollback function registered", actionID)
		}

		// Execute the compensating function.
		if err := entry.RollbackFunc(context.Background()); err != nil {
			g.entries[i].Status = "failed"
			now := time.Now()
			g.entries[i].RolledBackAt = &now
			g.logger.Error("rollback failed",
				zap.String("action_id", actionID),
				zap.Error(err),
			)
			return fmt.Errorf("rollback of %q failed: %w", actionID, err)
		}

		// Rollback succeeded.
		now := time.Now()
		g.entries[i].Status = "rolled_back"
		g.entries[i].RolledBackAt = &now

		g.logger.Info("rollback succeeded",
			zap.String("action_id", actionID),
			zap.String("action_type", entry.ActionType),
		)
		return nil
	}

	return fmt.Errorf("rollback entry %q not found", actionID)
}

// ListPending returns all entries that are still pending (not yet rolled back)
// and were created since the given time. Pass a zero-value time.Time to
// return all pending entries regardless of creation time.
func (g *RollbackGuard) ListPending(since time.Time) []RollbackEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []RollbackEntry
	for _, entry := range g.entries {
		if entry.Status != "pending" {
			continue
		}
		if !since.IsZero() && entry.CreatedAt.Before(since) {
			continue
		}
		result = append(result, entry)
	}

	if result == nil {
		return []RollbackEntry{}
	}
	return result
}
