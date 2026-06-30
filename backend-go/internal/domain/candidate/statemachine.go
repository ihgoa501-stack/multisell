package candidate

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// CandidateStatusTransitions defines allowed candidate product status transitions.
// draft → in_review → approved/rejected
// Rejected and in_review can go back to draft for rework.
var CandidateStatusTransitions = map[string]map[string]bool{
	"draft":     {"in_review": true},
	"in_review": {"approved": true, "rejected": true, "draft": true},
	"approved":  {},
	"rejected":  {},
}

// NewCandidateStateMachine creates a state machine for candidate product status transitions.
func NewCandidateStateMachine() *statemachine.StateMachine {
	return statemachine.New(CandidateStatusTransitions)
}
