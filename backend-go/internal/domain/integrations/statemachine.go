package integrations

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// SyncTaskStatusTransitions defines allowed sync task status transitions.
// pending → syncing → completed/failed
// Failed tasks can be retried (back to pending).
var SyncTaskStatusTransitions = map[string]map[string]bool{
	"pending":   {"syncing": true},
	"syncing":   {"completed": true, "failed": true},
	"completed": {},
	"failed":    {"pending": true}, // retry
}

// NewSyncTaskStateMachine creates a state machine for platform sync task status transitions.
func NewSyncTaskStateMachine() *statemachine.StateMachine {
	return statemachine.New(SyncTaskStatusTransitions)
}
