package listing

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// ListingStatusTransitions defines allowed listing status transitions.
// draft → submitted → approved → active → paused → ended
// Rejected and failed are reachable from submitted.
var ListingStatusTransitions = map[string]map[string]bool{
	"draft":      {"submitted": true, "cancelled": true},
	"submitted":  {"approved": true, "rejected": true, "failed": true},
	"approved":   {"active": true},
	"active":     {"paused": true, "ended": true},
	"paused":     {"active": true, "ended": true},
	"ended":      {},
	"cancelled":  {},
	"rejected":   {},
	"failed":     {},
}

// NewListingStateMachine creates a state machine for listing status transitions.
func NewListingStateMachine() *statemachine.StateMachine {
	return statemachine.New(ListingStatusTransitions)
}
