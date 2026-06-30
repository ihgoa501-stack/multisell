package approval

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// ApprovalStatusTransitions defines allowed approval status transitions.
//   pending   → {approved, rejected, expired, canceled, superseded}
//   approved  → {superseded}
//   rejected  → {} (terminal)
//   expired   → {} (terminal)
//   canceled  → {} (terminal)
//   superseded → {} (terminal)
var ApprovalStatusTransitions = map[string]map[string]bool{
	"pending":   {"approved": true, "rejected": true, "expired": true, "canceled": true, "superseded": true},
	"approved":  {"superseded": true},
	"rejected":  {},
	"expired":   {},
	"canceled":  {},
	"superseded": {},
}

// NewApprovalStateMachine creates a state machine for approval status transitions.
func NewApprovalStateMachine() *statemachine.StateMachine {
	return statemachine.New(ApprovalStatusTransitions)
}
