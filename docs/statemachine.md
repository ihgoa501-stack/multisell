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
| **`listingtask`** | `ListingTaskStatusTransitions` | 见下方详细说明 |
| **`approval`** | 内联校验 | pending → approved / rejected（无正式状态机，直接校验） |

## 可信经营闭环状态机

### ListingTask 状态转换

代码位置：`internal/domain/listingtask/statemachine.go`

```
                    +--→ rejected (terminal)
                    |
blocked ──→ pending_approval ──→ approved ──→ executing ──→ completed (terminal)
                                                              |
                                                              +--→ failed ──→ pending_approval (retry)

cancelled (terminal)
```

**状态说明：**

| 状态 | 说明 | 可转换到 |
|------|------|----------|
| `blocked` | 初始状态，需 Owner 采纳建议 | `pending_approval` |
| `pending_approval` | 等待 Owner 审批 | `approved`, `rejected` |
| `approved` | 审批通过，可执行 | `executing` |
| `executing` | 正在执行中 | `completed`, `failed` |
| `completed` | 执行完成（terminal） | — |
| `failed` | 执行失败 | `pending_approval`（重试） |
| `rejected` | 被拒绝（terminal） | — |
| `cancelled` | 已取消（terminal） | — |

**转换定义（Go 代码）：**

```go
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
```

### Approval 状态转换

代码位置：`internal/domain/approval/service.go`（Review 方法内联校验）

```
pending ──→ approved（审批通过）
     └───→ rejected（审批拒绝）
```

Approval 状态没有使用通用状态机框架，而是直接校验：
- 只有 `pending` 状态的请求可以被 `Review` 处理
- 已审批或已拒绝的请求不能重复审批

**审计集成：**
- 创建审批请求时记录 `operation_log`（如果启用了 `oplogSvc`）
- 审批/拒绝时记录操作日志

### Agent Recommendation 反馈状态

代码位置：`internal/domain/loop/model.go`

```
pending ──→ adopted（Owner 采纳）
     │          └──→ executed（执行成功）
     │          └──→ execution_failed（执行失败）
     └──→ rejected（Owner 拒绝）
```

反馈状态由两部分控制：
1. **Owner 反馈**（`owner/service.go` RecordFeedback）— 设置 `adopted` 或 `rejected`
2. **执行结果回写**（`loop/service.go` RecordExecutionResult）— 设置 `executed` 或 `execution_failed`

## 执行门禁检查顺序

ListingTask 的 `ExecuteTask` 方法（`listingtask/service.go`）在真正执行前通过 `validateExecutePreconditions` 进行 6 层检查：

```
1. 任务存在 ──→ First(&task, taskID) — 隐含在 ExecuteTask 开头
2. 幂等性(completed) ──→ task.Status == "completed" → 直接返回成功
3. 幂等性(executing) ──→ task.Status == "executing" → 返回错误
4. 状态机校验 ──→ sm.CanTransition(status, "executing") → 仅 approved 可执行
5. ApprovalID 存在 ──→ task.ApprovalID == nil → 返回错误
6. 审批记录校验 ──→ approvalSvc.Get(approvalID):
   a. 审批记录存在
   b. 状态为 "approved"
   c. EntityType 为 "listing_task"
   d. EntityID 与 task.ID 匹配
```

所有检查通过后：
1. 写入审计日志（operation_log）
2. 执行状态转换为 `executing`
3. 运行 Prism 合规检查（可选，通过配置启用）
4. 更新任务项状态为 `completed`
5. 更新任务主状态为 `completed`
6. 写入审计日志
7. 执行审计状态变更日志
8. 回写 Loop 反馈状态为 `executed`

## 审计集成点

执行门禁在以下节点写入 operation_log：

| 审计点 | Action | 条件 |
|--------|--------|------|
| 执行开始 | `listing_task.execute` (started) | 通过门禁后立即写入 |
| Prism 阻断 | `listing_task.execute` (blocked) | 当 Prism 合规检查失败且 prismStrict=true |
| 执行失败 | `listing_task.execute` (failure) | 事务回滚/平台错误 |
| 执行完成 | `listing_task.execute` (success) | 执行成功 |

此外，每次状态变更写入 `listing_task.status_change` 审计事件，记录 `old → new` 转换。

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
