# Spec: 商品出海决策与执行层 — Phase 1 Dry-Run 闭环修复

## Objective

打通一条单商品、非真实外写的商品出海可信闭环：

```
candidate -> loop evaluate -> listing_recommendation
-> owner approval -> listing_task approved
-> ExecuteTask dry-run -> execution result -> recommendation feedback
```

让 Owner 能在 `/owner` 看懂建议并批准，系统在 dry-run 模式下不触发真实平台写操作，
审批记录、任务状态、执行结果、审计日志可互相追溯。

## Scope

Phase 1 仅修复断点，不新增功能、不接 production 平台发布、不扩展 CRUD 页面。

## Assumptions

1. 开发环境已有 mock 数据库和种子数据。
2. Loop 路径是唯一商品出海决策主路径（已确认 `pre_listing_decision` 不是当前方向）。
3. 系统当前处于 dry-run 模式，不具备任何平台的真实 production 发布能力。
4. EventBus 已设置并运行。

## Current Gaps (Verification-Based)

通过代码调查，确认以下断点：

### Critical (Must Fix for Acceptance)

| # | Issue | File | Impact |
|---|-------|------|--------|
| C1 | Approval RequestType 不统一：loop 创建 `"publish"`，订阅器只接收 `"listing_task"` | `loop/service.go:92` → event topic `approval.approved.publish` vs `router.go:844` sub `approval.approved.listing_task` | 审批通过后事件从不触发，listing_task 永远停在 `blocked` |
| C2 | Approval_id 从不回写 listing_task：loop 创建 task 时不设 approval_id，审批 review 不写回 | `loop/service.go:280-292`、`approval/service.go:96-145` | ExecuteTask 的 precondition 检查 approval_id==nil 阻断执行 |
| C3 | Owner feedback "adopt" 创建重复审批请求 | `owner/service.go:316-332` | 同一条 listing_task 出现两个 pending approval |
| C4 | DecisionSnapshot 缺 `mode: "dry_run"`，ExecuteTask 不进入 dry-run 路径 | `loop/service.go:260-268` | dry-run 路径永远不走 |
| C5 | publishHook 中 adapter.Publish 无 dry-run guard | `router.go:788` | 非 dry-run 时 publishHook 直接调真实平台 API |

### Needs Confirmation / Gap

| # | Issue | File | Status |
|---|-------|------|--------|
| G1 | 前端 `/listing-tasks/:id` 是否展示 approval 状态和 dry-run 结果 | 待检查 | 需确认 |
| G2 | 前端 `/owner` 是否允许直接 approve/reject（而非只跳转） | 待检查 | 需确认 |
| G3 | approval 路由是否注册在 router.go 中 | `router.go` | 待确认 |

## Success Criteria

验收测试可用一个 demo candidate 完整跑通以下链路：

1. `POST /v1/loop/evaluate/:productId` → 返回 recommendation + listing_task created
2. Owner 在 `/owner` 看到建议（what / why / risk / expected action / execution mode）
3. Owner 在 `/owner` 批准建议（或通过 `/approval/:id/review`）
4. 审批通过后系统自动: listing_task.approval_id 已设 + listing_task.status = "approved"
5. `POST /v1/listing-task/:task_id/execute` → dry-run 执行成功
6. 获取 listing_task 详情: 显示 approval 状态、execution 状态、dry-run 结果
7. 未批准的任务执行被阻断（status not approved + no approval_id）
8. 事件 topic 统一使用 `approval.*.listing_task` 语义
9. 关键操作可追溯到 operation_log

## Non-Goals (Phase 1)

- 不接 production 平台发布
- 不扩展前端 CRUD 页面
- 不改 AGENTS.md 或 CLAUDE.md 的业务内容
- 不新增数据库迁移（复用现有列）

## Implementation Tasks

### Task 1: 统一 approval RequestType → "listing_task"
- **Change**: `loop/service.go` — `RequestType: "publish"` → `"listing_task"`
- **Why**: 匹配事件订阅 `approval.approved.listing_task`
- **Verify**: `loop.Service.Evaluate` 创建的 approval 使用 `listing_task`

### Task 2: 事件订阅器写回 approval_id + status→approved
- **Change**: `router.go:843-857` — 订阅器从读 event payload 写回 listing_task:
  - `approval_id = event.approval_id`
  - `status = "approved"`
  - 写 audit log
- **Why**: 审批通过后任务状态必须推进

### Task 3: 设置 dry_run mode + publishHook guard
- **Change**:
  - `loop/service.go:createListingTask` — 在 DecisionSnapshot 加 `"mode": "dry_run"`
  - `router.go:publishHook` — 读 DecisionSnapshot mode，dry-run 时跳过 adapter.Publish
- **Why**: ExecuteTask 的 dry-run 路径和发布 hook 安全

### Task 4: 修复 Owner feedback 重复 approval
- **Change**: `owner/service.go:RecordFeedback` — 当 adopt 时查找已有 pending approval，不新建
- **Why**: 避免一条 listing_task 出现两个批准请求

### Task 5: 补测试
- Loop evaluate: approval topic 使用 listing_task
- Owner feedback: adopt 不创建重复 approval
- listingtask execute: approval_id 和状态校验
- Dry-run mode 传播
- PublishHook dry-run guard

### Task 6: 前端/文档确认
- 确认 `/owner` 能展示建议和执行模式的可见性
- 确认 `/listing-tasks/:id` 展示 approval 状态和 dry-run 结果
- 更新文档索引

## Boundaries

- **Always:** 修复前置确认当前状态；写测试；文档同步
- **Ask first:** 数据库迁移；新增路由；修改高风险审批逻辑
- **Never:** 接 production 平台发布；修改 AGENTS.md/CLAUDE.md 业务内容；做无关重构
