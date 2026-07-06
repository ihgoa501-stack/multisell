# 动作限制清单

> 基于运行时代码数据生成， Last updated: 2026-07-06
> 来源：`actioncatalog.Default()` 目录、`forbidden_action` 表逻辑、`approval_policy_rule` 引擎、`operation_log` 审计日志、`DispatchSafe` 执行门禁

---

## 第一节：自动放行动作（低风险，无需审批）

以下动作不需要 Owner 审批，Agent 可以直接执行。包括所有 **L1（只读/告警）** 和 **L2（草稿/建议）** 级别动作，以及系统内部确定性状态转换。

### L1 — 只读 / 告警（RiskLow）

| 动作类型 | 名称 | 用途 | 目标 |
|---------|------|------|------|
| `stock_alert` | 库存预警 | 创建库存预警通知，不改库存 | sku |
| `system_health` | 系统健康检查 | 只读系统健康检查 | system |
| `dashboard_overview` | 仪表盘概览 | 只读仪表盘聚合数据 | dashboard |
| `auto_reply` | 客服自动回复 | 生成客服自动回复内容 | order, message |
| `profit_watch` | 利润监控 | SKU 利润分析监控 | sku |

### L2 — 草稿 / 建议（RiskMedium）

| 动作类型 | 名称 | 用途 | 目标 |
|---------|------|------|------|
| `listing_optimize` | Listing 优化 | 创建 listing draft，但不发布到外部平台 | listing, sku |
| `compliance_check` | 合规检测 | 标记不合规商品但不阻断刊登。命中高风险时升级至 L3 | product, sku |
| `replenish` | 补货建议 | 创建补货建议/采购请求 | sku |
| `discount_risk_check` | 折扣风控检查 | 分析折扣风险但不修改价格 | price_rule |

### 系统内部动作（L3 但免审批）

这些是 EventBus 事件触发的确定性系统状态转换，绕过 Agent 发起的审批路径，但仍受 mutation guard 审计层约束。

| 动作类型 | 名称 | 触发事件 | 用途 |
|---------|------|---------|------|
| `system.inventory.receive` | 系统库存接收 | `supplychain.order.received` | 采购入库确认后自动增加对应 SKU 库存 |
| `system.inventory.aftersale_restock` | 系统售后入库 | `supplychain.aftersale.completed` | 退货完成后自动增加对应 SKU 库存 |

---

## 第二节：需要审批的动作（中高风险）

以下所有 **L3 生产变更** 动作默认 `RequireApproval: true`，必须通过 `approval_request` 表提交审批流程，经 Owner 批准后才能执行。`DispatchSafe` 在 production 模式下会强制校验 `ApprovalID`。

| 动作类型 | 名称 | 风险等级 | 用途 | 目标 |
|---------|------|---------|------|------|
| `price_update` | 调价 | RiskHigh | 修改商品价格，高风险，必须审批 | sku |
| `price_review` | 价格审查（遗留） | RiskHigh | 审查并修改价格，与 price_update 同语义 | sku |
| `listing_publish` | 发布 Listing | RiskHigh | 将 listing draft 发布到外部平台 | listing |
| `inventory_change` | 库存调整 | RiskHigh | 修改库存数量 | sku, inventory |
| `order_cancel` | 取消订单 | RiskHigh | 取消客户订单 | order |
| `refund_issue` | 发起退款 | RiskHigh | 向客户发起退款 | order |
| `sync_inventory` | 同步库存 | RiskHigh | 将库存同步到外部平台 | sku, inventory |
| `credential_change` | 修改凭证 | RiskHigh | 修改平台 API 凭证或密钥 | credential |
| `permission_change` | 修改权限 | RiskHigh | 修改用户或 Agent 权限 | permission, rbac |
| `destructive_data_change` | 破坏性数据修改 | RiskHigh | 删除或批量修改关键业务数据 | * |

### 审批流程

1. **提交请求** — 系统创建 `approval_request` 记录（状态 `pending`）。
2. **Owner 审批** — 通过接受 API `POST /api/v1/approvals/:id/approve` 或拒绝 `POST /api/v1/approvals/:id/reject`。
3. **审批事件** — 审批通过后发布 `approval.approved.{request_type}` 事件，触发后续闭环流程（如 listing 创建）。
4. **执行验证** — `DispatchSafe` 在 production 模式下检查 `action.ApprovalID != nil` 并调用 `policy.IsApproved()` 验证审批仍有效。
5. **审计记录** — 审批操作记录到 `operation_log`（TriggerType: `owner_approval`）。

### 审批规则引擎

`approval_policy_rule` 表支持按以下维度配置自动审批/阻断/升级策略：

| 维度 | 字段 | 说明 |
|------|------|------|
| 风险等级 | `risk_level` | low、medium、high |
| 动作类型 | `action_type` | 精确匹配或 `*` |
| Agent | `agent_id` | 精确匹配或 `*` |
| Squad | `squad_id` | 精确匹配或 `*` |
| 业务对象类型 | `business_object_type` | 精确匹配或 `*` |
| 金额上限 | `max_amount` | 超过则跳过此规则 |
| 数量上限 | `max_quantity` | 超过则跳过此规则 |
| 最低置信度 | `min_confidence` | 低于则跳过此规则 |

规则匹配结果（`outcome`）：
- `auto_approve` — 自动通过（high-risk 动作自动降级为 `escalate`，不可绕过）
- `escalate` — 需要 Owner 审批（默认）
- `block` — 直接否决

**安全门禁**：高风险的 action（`RiskLevel == "high"`）即使命中 `auto_approve` 规则，也会被强制降级为 `escalate`。

### API 端点

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/policy/rules` | 列出审批规则 |
| GET | `/api/v1/policy/rules/:id` | 获取单条规则 |
| POST | `/api/v1/policy/rules` | 创建审批规则 |
| PUT | `/api/v1/policy/rules/:id` | 更新审批规则 |
| DELETE | `/api/v1/policy/rules/:id` | 软删除审批规则 |
| POST | `/api/v1/policy/rules/:id/toggle` | 启用/禁用审批规则 |
| POST | `/api/v1/policy/evaluate` | 评估动作是否需要审批 |

---

## 第三节：禁止动作（永远不允许执行）

### 硬禁止（代码层直接阻断）

**L4 自动执行** — `auto_publish` 动作 `AutonomousBlocked: true`，在 `actioncatalog.ValidateProduction` 中直接拒绝，`DispatchSafe` 生产模式无法绕过。

| 动作类型 | 名称 | 阻塞原因 |
|---------|------|---------|
| `auto_publish` | 自动发布 | L4，当前阶段强制阻止自动发布到外部平台 |

任何不在 `actioncatalog.DefaultEntries()` 注册的未知动作类型，在 production 模式下 `DispatchSafe` 和 `ValidateProduction` 都会返回 `ErrUnknownAction`，拒绝执行。

### `forbidden_action` 表规则

`forbidden_action` 表支持按 action_type、agent_id、risk_level 维度定义不可覆盖的禁止规则：

| 字段 | 说明 |
|------|------|
| `action_type` | 动作类型（支持 `*` 通配所有动作） |
| `agent_id` | 适用 Agent（空值或 `*` 匹配任意 Agent） |
| `risk_level` | 风险等级（留空或 `*` 匹配任意等级） |
| `reason` | 禁止原因描述 |
| `enabled` | 是否启用 |

查找逻辑（`CheckForbidden`）：
```
WHERE enabled = true
  AND (action_type = ? OR action_type = '*')
  AND (agent_id = ? OR agent_id = '' OR agent_id = '*')
  AND (risk_level = ? OR risk_level = '' OR risk_level = '*')
```

如果 SQLite/PostgreSQL 表不存在，跳过检查（向上兼容未迁移的环境）。

### 平台宪法规定的禁止行为

来自 `PLATFORM_CONSTITUTION.md` 第 11 节：

- Agent 不得要求 Owner 做出技术架构决策。
- Agent 不得在未说明风险的情况下修改高风险行为。
- Agent 不得在没有审批和审计的情况下实施自动价格、库存、订单、资金、外部发布或权限变更。
- Agent 不得绕过 Auth、RBAC、Approval 或 Audit。
- Agent 不得将业务特定逻辑放入 Platform Kernel。
- Agent 不得复制已有领域概念替代扩展规范定义。
- Agent 不得在活动功能中使用遗留栈，除非明确要求。
- Agent 不得在未有明确 Owner 指令的情况下运行破坏性 git 或数据库命令。
- Agent 不得撤销无关用户或其他 Agent 的工作。
- Agent 不得在无具体业务或平台验收目标的情况下进行大规模重构。

---

## 第四节：审计记录查看位置

### middleware 自动审计

全局 Audit middleware 记录以下所有操作到 `operation_log` 表：

- 所有 POST / PUT / PATCH / DELETE 请求（跳过 `GET /api/health`, `GET /api/healthz`）
- 敏感路径的 GET 请求：`/api/v1/finance`、`/api/v1/orders`、`/api/v1/settlement`、`/api/v1/user`、`/api/v1/rbac`

审计字段清单：

| 字段 | 说明 |
|------|------|
| `module` | 从请求路径提取（如 `/api/v1/order/123` → `order`） |
| `action` | HTTP method + route template 生成（如 `create_order`） |
| `resource_id` | path 参数（优先 `:id`） |
| `content` | JSON 快照：path、query、status、截断 body |
| `operator` | JWT 中的 username 或 user_id |
| `user_id` | 整型用户 ID |
| `result` | `success`（status < 400）或 `failure` |
| `ip` | 客户端 IP |
| `duration` | handler 执行耗时（毫秒） |
| `trigger_type` | 未设置时为空；Agent 场景为 `agent`、`system`、`owner_approval` |

审计写入使用 `go` 后台 goroutine，超时 3 秒，失败只打 warning 不阻塞业务。

### EventBus mutation guard 审计

系统内部确定性变更（`system.inventory.receive`、`system.inventory.aftersale_restock`）记录到 `operation_log`：

- `trigger_type = 'eventbus'`
- `module` = domain 名称
- `action` = SystemAction 名称
- `resource_id` = event entity ID
- `operator` = `system:{domain}`（无 actor 时的回退值）
- `result` = `pending` → `executed` 或 `failed`

### 审批审计

Owner 审批/拒绝操作记录到 `operation_log`：

- `trigger_type = 'owner_approval'`
- `action = 'approval.review'`
- `result` = `approved` 或 `rejected`
- 附加 `action = 'approval.review.note'` 记录审批备注

### 高频生产变更审计

Command Dispatcher 的 `DispatchSafe` 在 high-risk production 动作成功执行后调用 `auditRecorder` 回调记录至 `operation_log`。

### 查询 API

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/operation-log` | 分页查询操作日志，支持 module、action、operator、时间范围过滤 |
| GET | `/api/v1/operation-log/:id` | 查询单条操作日志 |
| POST | `/api/v1/operation-log` | 手动写入操作日志 |

示例查询：
```
GET /api/v1/operation-log?module=order&action=create_order&from=2026-07-01T00:00:00Z
```

---

## 第五节：如何变更限制

### 修改动作分类（代码层）

编辑 `backend-go/internal/platform/actioncatalog/catalog.go` 中的 `DefaultEntries()` 函数：

- 新增动作：添加新的 `Entry` 结构体
- 修改风险等级：改 `RiskLevel`（`RiskNone`/`RiskLow`/`RiskMedium`/`RiskHigh`）
- 修改自治等级：改 `Level`（`Level1`/`Level2`/`Level3`/`Level4`）
- 切换审批要求：改 `RequireApproval`
- 切换自动执行阻塞：改 `AutonomousBlocked`

重新编译后生效：`go build -o bin/server cmd/server/main.go`。

### 配置 forbidden_action 表

直接通过数据库插入或管理接口（需开发）修改 `forbidden_action` 表记录。

示例：禁止 Agent `A5` 执行任何高风险动作：
```sql
INSERT INTO forbidden_action (action_type, agent_id, risk_level, reason, enabled)
VALUES ('*', 'A5', 'high', 'A5 当前不允许执行高风险动作', true);
```

### 配置审批规则

通过 `/api/v1/policy/rules` API 管理 `approval_policy_rule` 规则。

示例 — 配置 `auto_reply` 自动审批：
```json
POST /api/v1/policy/rules
{
  "name": "客服自动回复自动审批",
  "description": "自动回复内容不超过 500 字时自动通过",
  "action_type": "auto_reply",
  "risk_level": "low",
  "outcome": "auto_approve",
  "priority": 10
}
```

示例 — 配置高金额调价必须升级：
```json
POST /api/v1/policy/rules
{
  "name": "大额调价升级审批",
  "description": "调价金额超过 1000 元时必须 Owner 审批",
  "action_type": "price_update",
  "risk_level": "high",
  "max_amount": 1000.00,
  "outcome": "escalate",
  "priority": 20
}
```

示例 — 配置 A5 完全禁止调价：
```json
POST /api/v1/policy/rules
{
  "name": "A5 禁止调价",
  "description": "Agent A5 不允许执行调价动作",
  "agent_id": "A5",
  "action_type": "price_update",
  "outcome": "block",
  "priority": 30
}
```

### 重要设计约束

- **高风险自动审批门禁**：`RiskLevel == "high"` 的 actions，即使匹配到 `auto_approve` 规则，`Evaluate()` 也会强制降级为 `escalate`。此门禁在 `service.go:67` 硬编码，不可通过规则配置绕过。
- **未知动作拒绝**：不在 `actioncatalog` 注册的动作，production 模式下 `DispatchSafe` 会返回 `ErrUnknownAction`，安全关闭。
- **L4 自动执行阻塞**：`AutonomousBlocked = true` 的动作（如 `auto_publish`），生产环境无法执行，直到代码层解除阻塞。
- **系统动作免审批**：`system.inventory.*` 类型的 EventBus 触发动作 `RequireApproval: false`，因为它们是确定性系统内部状态转换，不受审批策略约束。
