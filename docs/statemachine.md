# State Machine Validation Framework

## Overview

The `internal/platform/statemachine/` package provides a generic, zero-dependency
finite state machine for status transition validation. It follows the same
map-based pattern used by the aftersales and purchase modules, but extracted
into a reusable framework with support for pre-transition guards and
post-transition hooks.

## Package Location

```
backend-go/internal/platform/statemachine/
├── statemachine.go       # Core framework
└── statemachine_test.go  # Unit tests (17 tests)
```

## Usage

### 1. Define transitions

Define your state machine as a `map[string]map[string]bool` where the outer key
is the current status and the inner keys are the allowed target statuses. A
status with no outgoing transitions (empty map or absent from the map) is
considered **terminal**.

```go
var OrderTransitions = map[string]map[string]bool{
    "pending":   {"confirmed": true, "cancelled": true},
    "confirmed": {"shipped": true, "cancelled": true},
    "shipped":   {"delivered": true},
    "delivered": {"completed": true},
    "completed": {},
    "cancelled": {},
}
```

### 2. Create a state machine

```go
import "github.com/lingmirror/backend-go/internal/platform/statemachine"

sm := statemachine.New(OrderTransitions)
```

### 3. Validate transitions

**`CanTransition`** — boolean check without guards:

```go
if !sm.CanTransition(current, target) {
    return fmt.Errorf("cannot transition from %s to %s", current, target)
}
```

**`MustTransition`** — full validation with guards and hooks:

```go
if err := sm.MustTransition(ctx, current, target, entity); err != nil {
    return err  // returns descriptive error
}
```

`MustTransition` returns an error if:
- Current status is terminal (no outgoing transitions)
- Transition is not defined in the map
- Any guard function returns an error

### 4. Add guards (pre-transition validation)

```go
sm.AddGuard("pending", "confirmed", func(ctx context.Context, entity interface{}) error {
    order, ok := entity.(*Order)
    if !ok {
        return nil
    }
    if order.TotalAmount <= 0 {
        return errors.New("order amount must be positive")
    }
    return nil
})
```

Multiple guards can be registered for the same transition. They run in
registration order. If any guard returns an error, the transition is blocked
and subsequent guards and hooks are skipped.

### 5. Add hooks (post-transition callbacks)

```go
sm.AddHook("shipped", "delivered", func(ctx context.Context, entity interface{}) error {
    order, ok := entity.(*Order)
    if ok {
        log.Printf("Order %d delivered", order.ID)
    }
    return nil
})
```

Hooks only execute if all guards pass. If a hook returns an error, the
transition is still considered to have failed (caller decides rollback).

### 6. Utility methods

```go
targets := sm.AllowedTargets("pending")  // returns ["confirmed", "cancelled"]
terminal := sm.IsTerminal("completed")   // returns true
```

## Module Integration Pattern

Each domain module that needs state machine validation follows this pattern:

### Define transitions in `statemachine.go`

```go
package order

import "github.com/lingmirror/backend-go/internal/platform/statemachine"

var OrderStatusTransitions = map[string]map[string]bool{
    "pending":   {"confirmed": true, "cancelled": true},
    "confirmed": {"shipped": true, "cancelled": true},
    "shipped":   {"delivered": true},
    "delivered": {"completed": true},
    "completed": {},
    "cancelled": {},
}

func NewOrderStateMachine() *statemachine.StateMachine {
    return statemachine.New(OrderStatusTransitions)
}
```

### Validate in service methods

```go
func (s *Service) UpdateStatus(id int64, from, to, operator, remark string) error {
    sm := NewOrderStateMachine()
    if err := sm.MustTransition(context.Background(), from, to, nil); err != nil {
        return err
    }
    // ... proceed with status update
}
```

## Currently Integrated Modules

| Module | State Machine | Key Statuses |
|--------|--------------|--------------|
| `order` | `OrderStatusTransitions` | pending → confirmed → shipped → delivered → completed (cancelled any step) |
| `listing` | `ListingStatusTransitions` | draft → submitted → approved → active → paused → ended (rejected/failed from submitted) |
| `listingtask` | `ListingTaskStatusTransitions` | blocked → pending → executing → completed/failed → pending (retry); cancellable from blocked/pending/executing |
| `approval` | `ApprovalStatusTransitions` | pending → approved/rejected/expired/canceled/superseded; approved → superseded |

## Design Principles

- **No persistence** — the state machine validates but does not update the entity
- **No new dependencies** — uses only `context` and `fmt` from stdlib
- **Guard-first, hooks-second** — guards run before hooks; guard failure skips hooks
- **Multiple per transition** — any number of guards/hooks can be registered; run in order
- **Generic entity** — the `entity interface{}` is passed to guards/hooks for type assertion

## Testing

```bash
# Framework tests
cd backend-go && go test ./internal/platform/statemachine/... -v

# Module integration tests
cd backend-go && go test ./internal/domain/order/... -v
cd backend-go && go test ./internal/domain/listing/... -v
```

The framework tests cover:
- Valid transitions
- Invalid transitions
- Terminal status detection
- Guard acceptance and rejection
- Hook execution and error propagation
- Entity type passing
- Multiple guard ordering
