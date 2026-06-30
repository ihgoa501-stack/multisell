package candidate

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// CandidateProductStatusTransitions defines allowed candidate product status transitions.
//   draft     → {in_review}
//   in_review → {approved, rejected, draft}
//   approved  → {} (terminal)
//   rejected  → {} (terminal)
var CandidateProductStatusTransitions = map[string]map[string]bool{
	"draft":     {"in_review": true},
	"in_review": {"approved": true, "rejected": true, "draft": true},
	"approved":  {},
	"rejected":  {},
}

// NewCandidateStateMachine creates a state machine for candidate product status transitions.
func NewCandidateStateMachine() *statemachine.StateMachine {
	return statemachine.New(CandidateProductStatusTransitions)
}
