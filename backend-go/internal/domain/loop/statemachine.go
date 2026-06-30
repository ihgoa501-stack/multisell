package loop

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// RecommendationStatusTransitions defines the allowed state transitions for ListingRecommendation status.
var RecommendationStatusTransitions = map[string]map[string]bool{
	RecStatusPending:  {RecStatusAccepted: true, RecStatusRejected: true, RecStatusExpired: true},
	RecStatusAccepted: {},
	RecStatusRejected: {},
	RecStatusExpired:  {},
}

// NewRecommendationStateMachine creates a StateMachine for ListingRecommendation status transitions.
func NewRecommendationStateMachine() *statemachine.StateMachine {
	return statemachine.New(RecommendationStatusTransitions)
}
