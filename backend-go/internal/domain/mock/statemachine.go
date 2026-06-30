package mock

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// SyncStatusTransitions defines allowed sync task status transitions.
//   pending → {in_progress, failed}
//   in_progress → {success, failed}
//   success → {pending}  (re-sync)
//   failed  → {pending}  (retry)
var SyncStatusTransitions = map[string]map[string]bool{
	"pending":     {"in_progress": true, "failed": true},
	"in_progress": {"success": true, "failed": true},
	"success":     {"pending": true},
	"failed":      {"pending": true},
}

// NewSyncStateMachine creates a state machine for sync task status transitions.
func NewSyncStateMachine() *statemachine.StateMachine {
	return statemachine.New(SyncStatusTransitions)
}
