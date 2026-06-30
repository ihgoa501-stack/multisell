package listingtask

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// ListingTaskStatusTransitions defines allowed listing task status transitions.
//   blocked   → {pending, cancelled}
//   pending   → {executing, cancelled, blocked}
//   executing → {completed, failed, cancelled}
//   completed → {} (terminal)
//   cancelled → {} (terminal)
//   failed    → {pending}
var ListingTaskStatusTransitions = map[string]map[string]bool{
	"blocked":   {"pending": true, "cancelled": true},
	"pending":   {"executing": true, "cancelled": true, "blocked": true},
	"executing": {"completed": true, "failed": true, "cancelled": true},
	"completed": {},
	"cancelled": {},
	"failed":    {"pending": true},
}

// NewListingTaskStateMachine creates a state machine for listing task status transitions.
func NewListingTaskStateMachine() *statemachine.StateMachine {
	return statemachine.New(ListingTaskStatusTransitions)
}
