package listingtask

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// ListingTaskStatusTransitions defines the allowed listing task status transitions.
//
//	blocked            ──→ pending_approval
//	pending_approval   ──→ approved, rejected
//	approved           ──→ executing
//	executing          ──→ completed, failed
//	failed             ──→ pending_approval (retry)
//	completed / rejected / cancelled ── terminal
var ListingTaskStatusTransitions = map[string]map[string]bool{
	"blocked":           {"pending_approval": true},
	"pending_approval":  {"approved": true, "rejected": true},
	"approved":          {"executing": true},
	"executing":         {"completed": true, "failed": true},
	"failed":            {"pending_approval": true},
	"completed":         {},
	"rejected":          {},
	"cancelled":         {},
}

// NewListingTaskStateMachine creates a state machine for listing task status transitions.
func NewListingTaskStateMachine() *statemachine.StateMachine {
	return statemachine.New(ListingTaskStatusTransitions)
}
