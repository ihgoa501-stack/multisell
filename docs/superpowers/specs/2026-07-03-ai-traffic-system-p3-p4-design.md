# AI Traffic System — P3 总控台 & P4 交通健康 设计文档

> 基于现有 AgentOS 驾驶舱 (unified_action / ai_trace / operation_log) 的增量开发。
> 最后更新：2026-07-03

## 1. 目标

在现有 AI Traffic System 的 P0/P1/P2（统一动作模型、统一执行入口、审批审计闭环）之上，补充：

- **P3**：Owner 能从 AgentOS 总控台看到完整的交通状态——流量漏斗、拦截记录、审计回放。
- **P4**：系统能采集 Agent 指标、限流、追踪外部平台失败、并通过 Entropy/TrustScore 降权异常 Agent。

## 2. 范围

### 涉及模块

| 层 | 模块 | 改动类型 |
|----|------|---------|
| 后端 Platform | `command/` | 新增 `ratelimit.go` (P4) |
| 后端 Platform | `toolbridge/` | 新增 `tracker.go` 外部调用追踪 (P4) |
| 后端 Domain | `agentos/` | handler + service 新增 4 个 API (P3+P4) |
| 后端 | `realtime/` | WS event 推送 action status 变更 (P3) |
| 后端 | `domain/entropy/` | 消费 Agent metrics 做降权 (P4) |
| 前端 | `agentos/page.tsx` | 加 Traffic Section + Agent 健康 (P3+P4) |
| 测试 | `command/ action_test.go catalog_test.go` | 补未覆盖分支 (P0/P1 剩余缺口) |

### 不在范围内

- 重写现有 AgentOS 驾驶舱（只增区块，不改现有区域）
- 引入 Redis/消息队列（限流和数据停留在内存或 DB）
- 跨进程消息系统

## 3. P3 — 总控台可见

### 3.1 新增 API

| 端点 | 方法 | 返回 | 用途 |
|------|------|------|------|
| `/v1/agentos/traffic-summary` | GET | status 分布计数 + 拦截计数 + funnel 统计 | 流量漏斗卡片 |
| `/v1/agentos/intercepted-actions` | GET | 最近被策略/权限/审批拦截的动作列表 | Blocked 区 |
| `/v1/agentos/audit-replay/:correlation_id` | GET | 按 correlation_id 聚合的完整动作+审批+审计链路 | 审计回放 drawer |

**traffic-summary 返回结构：**

```json
{
  "status_distribution": {
    "suggested": 12,
    "pending_approval": 5,
    "approved": 3,
    "rejected": 1,
    "executing": 2,
    "completed": 87,
    "failed": 4,
    "blocked": 2
  },
  "intercepted_total": 7,
  "funnel": {
    "produced": 116,
    "approved": 3,
    "executed": 87,
    "blocked_by_policy": 7,
    "rejected_by_owner": 1
  },
  "by_risk": {
    "low": {"suggested": 5, "pending": 1, "blocked": 0, ...},
    "medium": {...},
    "high": {...}
  }
}
```

**intercepted-actions 返回：**

```json
{
  "items": [
    {
      "id": 101,
      "action_type": "price_update",
      "agent_id": "A6",
      "risk_level": "high",
      "block_reason": "approval_required",
      "blocked_at": "2026-07-03 12:00:00",
      "target_summary": "SKU-1001"
    }
  ],
  "total": 7
}
```

**audit-replay 返回：**

```json
{
  "correlation_id": "corr-abc-123",
  "events": [
    {"type": "event", "name": "scheduler.tick.A5", "at": "..."},
    {"type": "agent_decision", "agent_id": "A5", "action_type": "stock_alert", "at": "..."},
    {"type": "action", "id": 42, "status": "suggested", "risk_level": "low", "at": "..."},
    {"type": "action", "id": 43, "status": "pending_approval", "risk_level": "high", "at": "..."},
    {"type": "approval", "approval_id": 7, "status": "approved", "reviewer": "owner", "at": "..."},
    {"type": "execution", "command": "price_update", "result": "success", "at": "..."},
    {"type": "audit", "audit_id": 99, "detail": "price 19.99 → 24.99", "at": "..."}
  ]
}
```

### 3.2 WebSocket 事件推送

在现有 `realtime/hub.go` 中注册 `agent.action.status_changed` 事件类型。当 DispatchSafe 执行后 action status 变化时，通过 EventBus 发布事件 → realtime hub 推送到前端。

事件 payload：

```json
{
  "type": "agent.action.status_changed",
  "payload": {
    "action_id": 42,
    "action_type": "price_update",
    "agent_id": "A6",
    "risk_level": "high",
    "old_status": "approved",
    "new_status": "executing",
    "correlation_id": "corr-abc-123",
    "timestamp": "2026-07-03T12:00:00Z"
  }
}
```

### 3.3 前端新增区块

在 AgentOS 页面 (`agentos/page.tsx`) 新增以下区块（保留现有内容不变）：

**区块 1 — 流量漏斗卡片**（顶部 StatCard 行下方）

```
┌─────────────┬──────────────┬─────────────┬──────────────┬──────────────┐
│ 已产生 116   │ 待审批 5     │ 已执行 87    │ 被拦截 7     │ 被拒绝 1     │
│ [produced]  │ [pending]    │ [completed] │ [blocked]    │ [rejected]   │
└─────────────┴──────────────┴─────────────┴──────────────┴──────────────┘
┌────────────────────────────────────────────────────────────────────────┐
│ Mini funnel bar:  ██████████████████████████████████████░░░░░░░░░░░   │
│                   suggested→approved→executed 转化率: 75%             │
└────────────────────────────────────────────────────────────────────────┘
```

**区块 2 — 被拦截动作列表**（Squad 健康图下方）

- Table 展示：action_type, agent_id, risk_level, block_reason, blocked_at
- 可展开查看详情（什么政策拦截的）

**区块 3 — 审计回放 Drawer**

- 复用现有 WorkItemDrawer 模式
- 从 work item 或拦截记录点击「审计回放」
- 按时间轴展示完整链路：事件 → 决策 → 动作 → 审批 → 执行 → 审计

## 4. P4 — 交通健康与拥堵治理

### 4.1 Rate Limiter (`command/ratelimit.go`)

滑动窗口计数器，按 `(agent_id, action_type)` 限流：

```go
type RateLimiter struct {
    mu      sync.Mutex
    windows map[string]*slidingWindow  // key = "agent_id:action_type"
    limit   int                        // max actions per window
    window  time.Duration              // window size (default 1 minute)
}

func (rl *RateLimiter) Allow(agentID, actionType string) bool
```

配置：默认 20 动作/小时/agent（和现有 `MaxActionsPerHour` 一致）。超出时 `DispatchSafe` 返回 `ErrRateLimited`。

### 4.2 外部调用追踪 (`toolbridge/tracker.go`)

记录 ToolBridge 调用的外部平台结果到内存或 DB（轻量级，暂不建新表，用 `operation_log` 加标记）：

```go
type ExternalCallTracker struct {
    mu   sync.Mutex
    stats map[string]*platformStats  // key = platform name
}

type platformStats struct {
    TotalCalls    int
    FailedCalls   int
    LastFailureAt time.Time
    LastError     string
    ConsecutiveFailures int
}
```

当连续失败 >= 3 次时，标记该平台为 `degraded` 并触发 Entropy 告警。

### 4.3 Agent 指标 API

新端点 `GET /v1/agentos/agent-metrics`：

```json
{
  "agents": [
    {
      "agent_id": "A5",
      "run_count": 42,
      "success_count": 38,
      "failure_count": 2,
      "blocked_count": 2,
      "approval_rate": 0.85,
      "owner_acceptance_rate": 0.78,
      "avg_latency_ms": 3200,
      "external_failure_rate": 0.05,
      "health": "ok"
    }
  ]
}
```

数据来源：聚合 `unified_action` 和 `ai_trace` 表（无需新表）。

### 4.4 Entropy 集成

现有 `domain/entropy/` 已支持 SPC 控制图和健康评分。P4 补充：

- Entropy 消费 `agent-metrics` 数据
- 当 `failure_rate > 0.2` 或 `consecutive_failures >= 5` 时触发异常标记
- `TrustScore` 据此降权 Agent 的 autonomy level

### 4.5 前端新增区块

**区块 4 — Agent 健康卡片**（Squad 健康图下方 / 右侧栏）

每张卡片展示：
- Agent ID + 名称
- 运行数 / 成功 / 失败 / 被拦截
- 批准采纳率
- 平均延迟
- 外部失败率
- 健康状态 (ok / warn / critical)

**区块 5 — 外部平台健康面板**（底部）

Table 展示每个平台的调用总数、失败数、连续失败数、状态（ok / degraded / down）。

**区块 6 — 拥堵指示器**

在顶部区域加一个小横幅/Alert，当有 Agent 被限流、平台降级、或异常 Agent 时显示。

## 5. 测试覆盖率缺口

当前覆盖 (`go test -cover`):

| Package | Current | Target | 缺口 |
|---------|---------|--------|------|
| `command` | 79.7% | 85%+ | `ActionStatus.String()` edge cases, `RiskLevel.String()` unknown, `ActionMode.String()` unknown, `HandlerNotFoundError.Error()` |
| `actioncatalog` | 64.5% | 80%+ | `AutonomyLevel.String()` unknown, `Must()` panic path, `AutonomousBlocked` false branch |

P3/P4 新代码要求覆盖率至少 80%。

## 6. 实现顺序

```
Phase 1（并行三路）:
  Stream A: 补测试缺口 (action_test.go, catalog_test.go)
  Stream B: P3 后端 API (traffic-summary, intercepted, audit-replay, WS)
  Stream D: P4 治理 (ratelimit, tracker, agent-metrics, entropy)

Phase 2（并行两路，等 Phase 1 API 就绪）:
  Stream C: P3 前端 (Traffic Section + Audit Replay)
  Stream E: P4 前端 (Agent Health + External Health + Congestion)

Phase 3: 集成验证 + 收口
```

独立测试：Stream A 无依赖，可最先启动。Stream B 和 D 互不依赖，可并行。Stream C 和 E 依赖对应后端 API 就绪后接。

## 7. 验证标准

- [ ] 现有 50 个测试保持通过
- [ ] P3 新增 API 有测试覆盖（>=80%）
- [ ] 前端新区块不复用现有行为
- [ ] Rate limiter 能拦截超出阈值的 action
- [ ] External tracker 连续 3 次失败标记 degraded
- [ ] Entropy 能消费 metrics 并影响 TrustScore
- [ ] WebSocket 推送 action status 变更到前端
