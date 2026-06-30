package order

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

// OrderStatusTransitions defines allowed order status transitions.
// pending → confirmed → shipped → delivered → completed
// Cancelled is reachable from pending, confirmed, and shipped.
var OrderStatusTransitions = map[string]map[string]bool{
	"pending":   {"confirmed": true, "cancelled": true},
	"confirmed": {"shipped": true, "cancelled": true},
	"shipped":   {"delivered": true, "cancelled": true},
	"delivered": {"completed": true},
	"completed": {},
	"cancelled": {},
}

// NewOrderStateMachine creates a state machine for order status transitions.
func NewOrderStateMachine() *statemachine.StateMachine {
	return statemachine.New(OrderStatusTransitions)
}
