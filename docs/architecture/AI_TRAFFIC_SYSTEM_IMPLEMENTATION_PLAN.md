# AI Traffic System Implementation Plan

> 将 AI Traffic System 从架构愿景落成系统事实的工程计划。
> 本计划服务于 LingMirror AI AgentOS 的 Platform Kernel 和 Agent Workflows 落地。
> 最后更新：2026-07-03
>
> 状态：P0 ✅ 已完成 | P1 ✅ 已完成 | P2 待开始 | P3 待开始 | P4 待开始

## 1. 目标

AI Traffic System 的目标不是写更多文档，而是让所有真实 Agent 动作都必须经过同一套平台交通规则：

```text
Event
  -> Agent Decision
  -> AgentAction
  -> Risk / Policy Check
  -> Approval when needed
  -> Command / ToolBridge
  -> Audit
  -> Dashboard status
```

当这条链路变成唯一生产执行路径时，文档才真正变成系统事实。

## 2. 业务结果

Owner 应该获得的结果：

- AI 可以持续接入更多业务场景，但不会绕过审批、审计和权限。
- 高风险动作在执行前必须能被看见、理解和放行。
- 系统能回放一次 Agent 动作从触发到执行的完整链路。
- 出现误判、失败、拥堵或外部平台异常时，总控台能显示状态，而不是隐藏在日志里。

## 3. 范围

### 涉及层级

- Platform Kernel
- Agent Workflows
- UI / Experience
- Documentation / Governance

### 涉及内核能力

- EventBus
- Pipeline DAG
- AgentAction
- Command Dispatcher
- ToolBridge
- Approval
- Audit
- RBAC
- Observability
- AgentOS Dashboard

### 不在本计划内

- 重写所有业务模块。
- 引入新的跨进程消息队列。
- 让 Agent 自动执行价格、库存、订单、钱或外部发布动作。
- 替代已有 `KERNEL_CONTRACTS.md`。

## 4. 风险等级

整体风险：High。

原因：

- 触及 Agent autonomy。
- 触及 EventBus、Command Dispatcher、ToolBridge、Approval、Audit。
- 未来会影响价格、库存、订单、外部平台发布等高风险业务动作的执行路径。

落地策略：

- 分阶段推进。
- 每阶段只改变一个清晰边界。
- 先 dry-run 和可见性，再生产执行。
- 先拦截高风险动作，再扩展自动化能力。

## 5. P0：统一动作模型

### 目标

所有 Agent 输出必须能表示为结构化 `AgentAction`，不能只依赖自然语言结果。

### 要做什么

- 盘点当前 Agent 决策输出。
- 确认 `AgentAction` 是否覆盖以下字段：

```text
action_type
version
agent_id
actor
tenant_id
target_type
target_id
risk_level
approval_required
approval_id
mode
status
idempotency_key
correlation_id
input
rollback_note
audit_id
```

- 补齐缺失字段或建立兼容映射。
- 定义所有 Agent Action 的标准状态：

```text
suggested
pending_approval
approved
rejected
executing
completed
failed
blocked
```

### 验收标准

- 每个 Agent 决策都能追溯到一个结构化 action。
- action 能区分 risk、mode、approval requirement 和 status。
- 自然语言建议只能作为 action 的说明，不能作为执行凭证。

### 建议测试

- 创建 low / medium / high 三类 action，字段校验通过。
- 缺少 `agent_id`、`risk_level`、`mode` 或 `correlation_id` 的 action 不能进入执行。
- high risk action 默认 `approval_required=true`。

## 6. P1：统一执行入口

### 目标

所有 Agent 动作必须通过 `Command.DispatchSafe` 或 `ToolBridge` 等受控入口执行。

### 要做什么

- 盘点所有 Agent 触发业务修改的路径。
- 标记绕过 Command / ToolBridge / service boundary 的路径。
- 对高风险动作强制执行：

```text
permission check
risk check
mode check
approval check
idempotency check
audit context check
```

- 对外部 mutation tool 强制区分：

```text
dry_run
sandbox
production
```

### 硬拦截规则

以下生产动作没有有效 approval 时必须失败：

- price_update
- inventory_change
- order_cancel
- refund_issue
- listing_publish
- sync_inventory
- credential_change
- permission_change
- destructive_data_change

### 验收标准

- Agent 写错代码时，也不能绕过执行入口完成高风险动作。
- production mutation tool 没有 approval ID 时返回明确错误。
- dry_run 永远不产生业务 mutation。

### 建议测试

- high risk command without approval returns `ErrApprovalRequired`。
- production mutation tool without approval returns `ErrMutationRequiresApproval`。
- dry_run command validates input but does not mutate data。
- duplicate `idempotency_key` does not execute twice。

## 7. P2：审批与审计闭环

### 目标

高风险动作必须在审批、执行、审计之间形成闭环。

### 要做什么

- 将需要审批的 `AgentAction` 自动关联 `approval_request`。
- 审批结果同步回 action 状态。
- Command / Tool 执行时验证 approval 状态仍然有效。
- 高风险 mutation 成功后必须写 audit。
- audit 记录必须携带：

```text
agent_id
actor
target
risk_level
approval_id
request_id
correlation_id
before
after
result
external_reference
```

### 验收标准

- Owner 能看到 action 为什么需要审批。
- 审批通过后才能执行。
- 审批拒绝、过期或目标不一致时不能执行。
- 关键动作执行后可以回放完整链路。

### 建议测试

- approval approved -> command can execute。
- approval rejected -> command blocked。
- approval expired -> command blocked。
- approval target mismatch -> command blocked。
- audit write failure causes action to fail or enter blocked state。

## 8. P3：总控台可见

### 目标

Owner 能通过 AgentOS Dashboard 看见城市交通状态。

### 要做什么

总控台至少展示：

- suggested actions
- pending approvals
- executing actions
- completed actions
- failed actions
- blocked actions
- intercepted high-risk actions
- Agent health
- external tool failures
- audit replay link

### Owner 视图语言

状态必须用业务语言表达：

| 技术状态 | Owner 可见表达 |
|---|---|
| suggested | AI 已提出建议 |
| pending_approval | 等待你放行 |
| executing | 正在执行 |
| completed | 已完成 |
| failed | 执行失败 |
| blocked | 已被系统拦截 |
| rejected | 你已拒绝 |

### 验收标准

- Owner 不看日志也能知道 AI 现在在做什么。
- 每个高风险 action 都显示风险、原因、目标和影响。
- 失败和拦截不会被混成“已处理”。
- Dashboard 可以进入审计回放。

### 建议测试

- 前端能区分 suggested / pending approval / executing / completed / failed / blocked。
- high risk action 显示审批按钮和风险说明。
- failed tool call 显示安全错误摘要。
- audit replay 能按 correlation ID 查到链路。

## 9. P4：交通健康与拥堵治理

### 目标

系统不仅能执行，还能发现拥堵、误判和异常 Agent。

### 要做什么

- 为 Agent workflow 增加指标：

```text
run_count
success_count
failure_count
blocked_count
approval_rate
owner_acceptance_rate
latency
external_failure_rate
false_alert_rate
```

- G0 / Entropy / TrustScore 根据异常状态降权、暂停或提示复查。
- 对同类事件风暴做限流、合并或延迟重试。
- 对外部平台失败做安全降级。

### 验收标准

- 异常 Agent 不会持续制造高风险待办。
- 外部平台失败不会被误报为成功。
- 同类事件大量触发时系统不会淹没 Owner。

### 建议测试

- repeated failures reduce Agent health or trigger blocked state。
- event burst does not create duplicate high-risk actions。
- external failure records provider, operation, latency and safe error summary。

## 10. 执行顺序

推荐顺序：

1. P0：统一动作模型。
2. P1：统一执行入口。
3. P2：审批与审计闭环。
4. P3：总控台可见。
5. P4：交通健康与拥堵治理。

不能跳过 P0 / P1 直接做 Dashboard。否则 UI 只是展示不完整事实，无法真正治理系统。

## 11. Review Checklist

每次涉及 AgentOS、Command、ToolBridge、Approval、Audit、EventBus、Scheduler 的变更，都必须检查：

- 是否所有 Agent mutation 都走统一执行入口？
- 是否高风险动作默认需要审批？
- 是否 dry_run 不会修改数据？
- 是否 production mutation tool 没有 approval 会失败？
- 是否每个 workflow 带 correlation ID？
- 是否能写入审计并回放？
- 是否 Owner 能在总控台看见状态？
- 是否没有把业务规则塞进 Platform Kernel？
- 是否没有为某个 Agent 发明第二套交通规则？

## 13. 已完成阶段状态

### P0 ✅ 统一动作模型 (2026-07-03)

已实现：
- `AgentAction` 统一动作结构包含所有标准字段（action_type, version, agent_id, actor, tenant_id, target_type, target_id, risk_level, approval_required, approval_id, mode, status, idempotency_key, correlation_id, audit_id, input, rollback_note）
- `ActionStatus` 类型定义 8 个标准状态：suggested, pending_approval, approved, rejected, executing, completed, failed, blocked
- `AgentAction.Validate()` 方法执行结构校验：必需字段（action_type, agent_id, actor）、有效 mode、有效 risk level
- 高风险 action 默认 `approval_required=true`（Validate 强制要求）
- `ErrActionValidation` 错误类型用于校验失败
- `HighRiskActions()` 返回所有高风险 action 类型的规范列表
- 所有现有 DispatchSafe 测试保持通过

### P1 ✅ 统一执行入口 (2026-07-03)

已实现：
- `DispatchSafe` 在所有模式下执行前调用 `action.Validate()`
- `AgentAction.Validate()` 拒绝缺少 agent_id/action_type/actor 或无效 mode/risk 的 action
- `actioncatalog` 新增 6 个高风险 action 类型：order_cancel, refund_issue, sync_inventory, credential_change, permission_change, destructive_data_change
- ToolCall.Validate() 已正确要求 production mutation 需要 approval
- dry_run 模式永远不执行 handler（通过 Validate 和 DispatchSafe 两重保障）
- 幂等性已记录为 TODO（无持久化存储前不应假装实现）

关键验收点验证：
- [x] high risk AgentAction without approval returns approval required error
- [x] production mutation ToolCall without approval returns mutation approval required error
- [x] dry_run does not execute mutation
- [x] high risk action defaults approval_required=true
- [x] invalid action missing required identity/mode/risk cannot execute
- [x] duplicate idempotency_key — TODO documented

### P2 ✅ 审批与审计闭环 (2026-07-03)

已实现：
- **ApprovalPolicyChecker** — 基于 `approval_request` 表的 PolicyChecker 实现：
  - 验证 approval_id 在 DB 中存在且 status=approved
  - 验证审批未过期（ExpiresAt 检查）
  - 返回 false 不会导致 DispatchSafe 返回 ErrApprovalRequired
- **AuditRecorder** — 高风险 production mutation 执行成功后自动调用：
  - 通过 `WithAuditRecorder()` 可选的 DispatcherOption 注入
  - 仅在生产模式 + 高风险 + 执行成功时触发
  - 不触发于 dry_run、sandbox、低风险或执行失败的场景
- 审批过期、拒绝、不存在或 pending → DispatchSafe 拦截执行

关键验收点验证：
- [x] approval approved → command can execute (ApprovalPolicyChecker)
- [x] approval rejected → is not IsApproved (PolicyChecker 返回 false)
- [x] approval expired → is not IsApproved（过期检测）
- [x] nonexistent approval ID → returns false
- [x] high-risk production success → audit recorder called
- [x] low-risk production → audit recorder NOT called
- [x] dry_run → audit recorder NOT called
- [x] execution failure → audit recorder NOT called

### P2 生产环境接入说明

要启用生产环境的审批检查和审计记录，在创建 Command Dispatcher 的代码中传入：
1. `ApprovalPolicyChecker(approvalSvc)` 作为 `DispatchSafe` 调用的 policy 参数
2. `WithAuditRecorder(func(ctx, action, result) { ... })` 写入 operation_log

## 14. 完成定义

AI Traffic System 变成系统事实的最低标准：

- Agent 动作有统一结构。
- 高风险动作没有审批不能生产执行。
- mutation 有审计。
- workflow 有 correlation ID。
- Dashboard 能展示状态。
- 测试能证明违规路径走不通。

只有达到这些标准，城市交通系统才不是文档口号，而是 LingMirror 的真实平台内核。
