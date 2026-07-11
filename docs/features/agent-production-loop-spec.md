# Spec: 商品上架 Agent 生产闭环

> 添加时间：2026-07-10
> 提出人：Owner / Agent
> 优先级：P1
> 阶段：SPECIFY（待 Owner 评审）

## Objective

为现有商品上架链路补齐可上线的 Agent 执行闭环，覆盖任务拆解、工具调用、评测与可观测性、审批后执行和断点恢复。

目标用户是商品运营人员。用户提交一个商品上架目标后，系统应生成可解释的执行计划，依次完成数据检查、利润/合规检查和上架建议；涉及外部平台写入时必须等待人工审批。审批通过后，系统应从中断节点继续执行，并保留完整的执行、工具、审批和审计记录。

本阶段不建设通用工作流画布，也不引入“自动创建任意多 Agent”的开放式编排器；先用一条真实业务链路验证平台能力。

### User stories

- 运营人员可以发起一次商品上架评估，并看到当前处于哪个步骤。
- 运营人员可以看到 Agent 使用了哪些数据、调用了哪些工具、得出了什么结论以及为什么需要审批。
- 运营人员可以批准或拒绝高风险上架动作。
- 审批通过后，系统可以从待审批步骤继续，而不是重新执行整条链路。
- 系统异常或外部平台失败时，运营人员可以看到失败原因和可重试状态。
- 技术和产品人员可以按 execution ID 回放一次运行的步骤、工具调用、审批和最终结果。

## Scope

### In scope

- 商品上架链路的持久化 execution / step / checkpoint 状态。
- 固定的阶段拆解：输入校验 → 商品数据完整性检查 → 利润与风险评估 → 生成上架建议 → 人工审批 → 平台上架执行 → 结果审计。
- ToolBridge 工具调用的统一 correlation ID、幂等键、超时、错误分类和结果记录。
- 高风险外部平台写入的审批门禁。
- 审批通过后的断点恢复和失败重试。
- 按 execution ID 查询执行时间线和关键 trace。
- 基础评测样例，至少覆盖正常、缺数据、高风险、工具失败、审批拒绝五类场景。

### Out of scope

- 通用拖拽式工作流设计器。
- 任意业务域的全量迁移。
- 无审批的价格、库存、订单、资金或平台写入自动化。
- 新增 LLM 厂商或替换现有模型网关。
- 重新设计现有 AgentOS 前端信息架构。

## Tech Stack

- Backend: Go 1.25, Gin, GORM, PostgreSQL 15。
- Agent platform: existing Orchestrator, Agent Registry, EventBus, Command Dispatcher, Scheduler, ToolBridge。
- Frontend: Next.js 16, React 19, TypeScript, Ant Design 6。
- Auth and safety: existing JWT, RBAC, Approval, Audit / OperationLog。
- Observability: existing trace, Prometheus metrics, structured logs, Sentry。

## Commands

```bash
# Backend focused tests
cd backend-go && go test -v ./internal/agent/... ./internal/agentos/... ./internal/domain/approval/... ./internal/domain/listingtask/...

# Backend full verification
cd backend-go && go build ./...
cd backend-go && go vet ./...
cd backend-go && go test ./...

# Frontend verification when UI changes
cd frontend-next && npm run lint
cd frontend-next && npm run build

# E2E when the approval/recovery UI flow changes
cd frontend-next/e2e && npx playwright test
```

## Project Structure

```text
backend-go/internal/agent/              Agent orchestration and execution entry points
backend-go/internal/agentos/             Execution/work-item APIs and cockpit views
backend-go/internal/platform/            EventBus, ToolBridge, Scheduler and kernel contracts
backend-go/internal/domain/approval/     Approval policy, review and approval events
backend-go/internal/domain/listingtask/  Product listing task state and execution result
backend-go/migrations/                   Durable execution/checkpoint schema changes
frontend-next/src/app/(main)/agentos/    Owner-facing execution, approval and replay views
frontend-next/src/lib/api-client.ts     Authenticated API access
backend-go/internal/*/*_test.go          Focused backend tests near changed behavior
frontend-next/e2e/                       Approval and resume browser tests
docs/features/                           Living feature specifications
```

## Code Style

Persist workflow transitions explicitly and make side effects idempotent. A transition should record the business state, correlation ID, and failure information together:

```go
func (s *Service) Advance(ctx context.Context, executionID int64, next StepStatus) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&Execution{}).
			Where("id = ? AND status IN ?", executionID, allowedTransitions[next]).
			Updates(map[string]interface{}{
				"status":         next,
				"correlation_id": CorrelationIDFromContext(ctx),
				"updated_at":     time.Now(),
			}).Error
	})
}
```

Conventions:

- Use existing module boundaries and service/handler/model patterns.
- Use explicit states such as `pending`, `running`, `waiting_approval`, `failed`, `completed`, and `rejected`.
- Do not mutate critical business data directly from an Agent; use command, service, approval, and audit contracts.
- Every external side effect must carry an idempotency key and correlation ID.
- Errors should be classified as validation, approval, transient tool, permanent tool, or system failure.
- Preserve existing response envelopes, JWT authentication, RBAC, and audit middleware.

## Testing Strategy

### Unit tests

- Valid and invalid execution state transitions.
- Checkpoint creation and resume from each resumable step.
- Idempotent replay of a completed tool call.
- Approval required, approval accepted, approval rejected, and expired approval.
- Retry classification and maximum retry behavior.
- Trace event ordering and correlation ID propagation.

### Integration tests

- Full listing execution through a SQLite test database where possible.
- Transaction rollback when checkpoint or audit persistence fails.
- EventBus notification from approval approval to execution resume.
- External platform adapter failure does not mark the listing successful.

### Evaluation cases

Store deterministic fixtures for at least:

1. Complete product data and low-risk suggestion.
2. Missing SKU or cost data.
3. Margin below threshold and approval required.
4. Tool timeout or provider error.
5. Approval rejection.

Each case must assert final status, risk decision, required approval, tool calls, and audit linkage. “感觉效果不错” is not an acceptance criterion.

### UI / E2E tests

When the UI is changed, verify that an Owner can inspect a running execution, review an approval, see failure details, and resume an approved execution.

## Boundaries

- Always: preserve JWT/RBAC; require approval for high-risk platform writes; persist state transitions; include correlation and idempotency identifiers; add focused tests; keep docs synchronized.
- Ask first: schema migrations that affect existing production data; changing approval policy; adding a new external platform; changing autonomous execution defaults; adding dependencies; changing CI or deployment configuration.
- Never: bypass approval or audit; execute platform writes directly from prompt output; use in-memory state as the only source of truth; delete failed execution history; commit secrets; touch `.kilo/worktrees/`; rewrite git history.

## Success Criteria

- A single execution ID identifies the complete listing run from request through final result.
- The system exposes the current step and status to the Owner.
- High-risk external platform writes cannot execute without a valid approval.
- Approval rejection terminates the execution safely and records the reason.
- Approval acceptance resumes from the checkpoint without repeating completed side effects.
- Transient tool failures retry within a bounded policy; permanent failures become visible and actionable.
- Every step and tool call has a persisted trace event with timestamp, correlation ID, status, and error/result summary.
- A run can be queried and replayed as an ordered timeline.
- The five evaluation cases pass automatically.
- Existing backend build, vet, and full test suite remain green.
- Documentation states clearly which parts are implemented and which remain planned.

## Open Questions

- Should the first external platform target be Ozon or Shopee?
- Should approved executions resume automatically from the approval event, or require the Owner to press “继续执行”?
- What is the maximum retry count and backoff policy for platform API failures?
- Which execution and trace fields may contain product/customer data, and what must be redacted?
- Does the Owner need a frontend replay timeline in phase one, or is a backend/API replay sufficient for the first increment?

## Review Gate

This document is the source of truth for the first implementation increment. Do not enter PLAN or modify implementation code until the Owner confirms the scope, especially the target platform, resume behavior, retry policy, data redaction, and phase-one replay surface.
