> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# 凌镜 LingMirror 经营闭环 MVP — 可信闭环收口实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把凌镜收口成一条可信经营闭环 MVP：候选商品 → 资料完整度 → 利润/成本/风险 → Agent 建议 → Owner 审批 → Listing Task → 执行前门禁 → 执行/阻断 → 审计 → 结果回流

**Architecture:** 基于已有 platform/statemachine 通用 FSM 框架，为 ListingTask/Approval 建立状态机；在 listingtask 执行入口加统一门禁（auth/RBAC/approval/state/idempotency/audit）；补 Agent 建议反馈回流字段；升 Owner 工作台为决策队列 UI；补 full-loop 集成测试。

**Tech Stack:** Go 1.25, Gin, GORM, PostgreSQL 15; Next.js 16, React 19, TypeScript, Ant Design 6; testing/integrationtest for full-stack tests

**Risk Level:** MEDIUM — 影响 listing 任务执行流程、审批门禁、Agent 反馈，但不改价格/库存/订单/资金数据，不涉及外部平台真实发布。

## Global Constraints

- 只使用 backend-go/ 和 frontend-next/，不碰旧栈
- 不碰 .kilo/worktrees/
- 不做无关重构
- 不提交、不推送、不提 PR，除非明确要求
- 不真实发布到外部平台
- 不自动改价、改库存、改订单状态、改资金相关数据
- 高风险动作必须经过 RBAC、Approval、Audit、State Machine
- loop 当前是本地模拟经营闭环，不要描述为真实外部平台自动化
- 所有新 API 受 JWT 保护
- 遵循现有模块 pattern（model/handler/service/routes）

---
## Current State (before changes)

### What works
- Candidate Product CRUD (`candidate/`) — 基本增删改查，状态是自由字符串
- Completeness Check (`completeness/`) — 12 维资料评分
- Profit Calculation (`profit/`) — 利润/成本/风险计算
- Loop Evaluate (`loop/`) — 编排完整度→利润→建议→listingtask
- Listing (ProductListing) State Machine (`listing/statemachine.go`) — draft → submitted → approved → active 已完整
- Approval CRUD (`approval/`) — pending → approved/rejected
- Owner Dashboard (`owner/`) — 风险摘要 + 建议列表（只读 API）
- ListingTask CRUD + Execute (`listingtask/`) — 含 Prism 合规门禁
- OperationLog (`operationlog/`) — 通用审计日志
- Platform State Machine (`platform/statemachine/`) — 通用 FSM 框架

### Critical gaps
1. ❌ ListingTask 无状态机 — 状态更新无校验
2. ❌ Approval 缺少 expired/canceled/superseded 状态
3. ❌ ExecuteTask 不检查 approval — 无审批门禁
4. ❌ ExecuteTask 无幂等校验 — 可重复执行
5. ❌ ExecuteTask 不写审计 — 关键操作不记录
6. ❌ Agent 建议无反馈 — 采纳/拒绝/结果不回流
7. ❌ Owner 页面不在菜单中 — 靠直接 URL 访问
8. ❌ Owner 页面的审批按钮调用错误 API — 需改为 proper approval flow

---
## Task Breakdown

### Task 1: ListingTask 状态机

**Files:**
- Create: `backend-go/internal/domain/listingtask/statemachine.go`
- Modify: `backend-go/internal/domain/listingtask/service.go`
- Test: `backend-go/internal/domain/listingtask/listingtask_test.go`

**Interfaces:**
- Consumes: `statemachine.New(transitions)` from `internal/platform/statemachine`
- Produces: `NewListingTaskStateMachine()` factory function

定义 ListingTask 状态机，并在 Service.Update、PublishTask、ExecuteTask 中集成校验。

状态转换规则：
```
blocked   → {pending, cancelled}
pending   → {executing, cancelled, blocked}
executing → {completed, failed, cancelled}
completed → {}  (terminal)
cancelled → {}  (terminal)
failed    → {pending}
```

- [ ] Create `statemachine.go` with `ListingTaskStatusTransitions` and `NewListingTaskStateMachine()`
- [ ] Integrate state machine into `Service.Update()` — replace direct status assignment
- [ ] Integrate into `Service.PublishTask()` — replace `if task.Status != "pending"`
- [ ] Add state validation to `Service.ExecuteTask()` — block if can't transition to executing
- [ ] Write tests: valid transitions, invalid transitions, terminal detection

---

### Task 2: Approval 状态机增强

**Files:**
- Create: `backend-go/internal/domain/approval/statemachine.go`
- Modify: `backend-go/internal/domain/approval/model.go` — 加状态常量
- Modify: `backend-go/internal/domain/approval/service.go` — 加 Cancel/Expire/Supersede
- Test: `backend-go/internal/domain/approval/approval_test.go`

**Interfaces:**
- Produces: `NewApprovalStateMachine()`, `Cancel()`, `ExpirePending()`, `Supersede()`

状态转换规则：
```
pending   → {approved, rejected, expired, canceled, superseded}
approved  → {superseded}
rejected  → {} (terminal)
expired   → {} (terminal)
canceled  → {} (terminal)
superseded → {} (terminal)
```

- [ ] Add status constants to model.go (StatusPending, StatusApproved, StatusRejected, StatusExpired, StatusCanceled, StatusSuperseded)
- [ ] Create statemachine.go with transitions map
- [ ] Add Cancel/ExpirePending/Supersede methods to service.go
- [ ] Write tests for all new methods and transitions

---

### Task 3: 统一执行门禁

**Files:**
- Modify: `backend-go/internal/domain/listingtask/service.go`
- Modify: `backend-go/internal/domain/listingtask/routes.go`
- Modify: `backend-go/internal/domain/loop/service.go`
- Modify: `backend-go/internal/domain/loop/routes.go`

**Interfaces:**
- Consumes: `operationlog.Service` for audit writes
- Produces: `checkApproval()` gate, idempotency guard in ExecuteTask

在 ExecuteTask 执行前统一校验：
1. State machine check (Task 1) — 当前状态允许执行
2. Approval check — 必须有 approved approval（publish 类型，未过期）
3. Idempotency — completed 任务不能重复执行
4. Audit — 执行前写 operationlog

- [ ] Add operationlog.Service dependency to listingtask.Service
- [ ] Add checkApproval() method — queries approval_request for valid approval
- [ ] Add idempotency guard — explicit error on duplicate execution
- [ ] Add audit logging on execute start
- [ ] Update all callers of NewService (listingtask routes.go, loop service.go and routes.go)
- [ ] Verify existing tests pass

---

### Task 4: Agent 建议反馈回流

**Files:**
- Modify: `backend-go/internal/domain/listingtask/model.go`
- Modify: `backend-go/internal/domain/listingtask/service.go`
- Modify: `backend-go/internal/domain/listingtask/handler.go`
- Modify: `backend-go/internal/domain/listingtask/routes.go`
- Test: `backend-go/internal/domain/listingtask/listingtask_test.go`

- [ ] Add `AgentFeedbackStatus` (*string, accepted/rejected) and `AgentFeedbackNote` (string) fields to ListingTask model
- [ ] Add `SubmitFeedback(taskID, status, note, updatedBy)` to service — validates status, persists, writes audit
- [ ] Add handler + route: `POST /listing-tasks/:taskId/feedback`
- [ ] Write tests: valid/invalid feedback, audit write

---

### Task 5: Owner 工作台升级

**Files:**
- Modify: `backend-go/internal/domain/owner/service.go`
- Modify: `backend-go/internal/domain/owner/handler.go`
- Modify: `backend-go/internal/domain/owner/routes.go`
- Modify: `frontend-next/src/app/(main)/owner/page.tsx`
- Modify: `frontend-next/src/config/menu.ts`

- [ ] Backend: Enrich suggestions with agent_feedback_status, task_status, approval_status, blocking_reasons in a new `GET /owner/decision-queue` endpoint
- [ ] Frontend: Switch data source from `/v1/owner/suggestions` to `/v1/owner/decision-queue`
- [ ] Frontend: Add columns: task_status, approval_status, blocking_reasons, agent_feedback_status
- [ ] Frontend: Wire approve/reject through proper approval API flow
- [ ] Add menu entry for /owner in menu.ts
- [ ] Verify frontend build

---

### Task 6: 集成测试

**Files:**
- Create: `backend-go/internal/domain/listingtask/routes_test.go`

- [ ] Write auth gate test (401 without token)
- [ ] Write approval gate test (can't execute without approval)
- [ ] Write blocked status gate test
- [ ] Write idempotency guard test
- [ ] Write feedback integration test

---

### Task 7: 文档更新

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/OWNER_WORKROOM.md`
- Modify: `docs/statemachine.md`
- Modify: `docs/testing.md`

- [ ] Update PROJECT_STATUS.md — document this cycle's changes
- [ ] Update OWNER_WORKROOM.md — what Owner can do, what's mock/sandbox
- [ ] Update statemachine.md — add ListingTask and Approval state machines
- [ ] Update testing.md — integration test examples for gates

---

## Verification

```bash
cd backend-go && go test ./... -v -count=1
cd backend-go && go vet ./...
cd frontend-next && npm run build
cd frontend-next && npm test
cd frontend-next && npm run lint  # known failures are OK
```

| # | Criteria |
|---|----------|
| 1 | 未登录返回 401 |
| 2 | 没 approval 阻止执行 |
| 3 | blocked 状态不能直接发布 |
| 4 | approval 通过后状态转换正确 |
| 5 | completed 任务不会重复执行 |
| 6 | Agent 建议能记录采纳/拒绝 |
| 7 | Owner 决策队列和后端一致 |
| 8 | go test 通过 |
| 9 | frontend build 通过 |
