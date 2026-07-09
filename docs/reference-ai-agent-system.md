# AI & Agent System 参考

> 凌镜 LingMirror 的 AI 和 Agent 系统参考文档。
> 涵盖 LLM 编排、Agent 注册/执行/Pipeline、AgentOS 控制台、Trace 系统、门禁系统。
> 最后更新: 2026-07-09

---

## 系统总览

AI & Agent 层分为三个包：

| 层 | 包 | 职责 |
|----|------|--------|
| LLM 编排 | `internal/ai/` | Provider 抽象、Chat/Stream、Trace 记录、Orchestrator 协调、Guardrails 集成 |
| Agent 注册与执行 | `internal/agent/` | Agent 注册表、HTTP 路由、Pipeline DAG 引擎 |
| AgentOS 控制台 | `internal/agentos/` | Work Items 管理、自主运营总览、SLA 升级 |

各层的依赖关系：

```
internal/ai/
  ├── llm_provider.go     → LLM API 调用抽象（OpenAI/Anthropic/Stub）
  ├── orchestrator.go     → Agent 决策编排核心（调度 Agent、调用 LLM、执行 Action）
  ├── registry.go         → Agent 注册表（AgentSpec 定义）
  ├── model.go            → AI 数据模型（Action/Trace/决策请求）
  ├── service.go          → AI 业务逻辑（Action CRUD、执行审批）
  ├── handler.go          → HTTP 请求/响应映射
  ├── routes.go           → Gin 路由注册
  ├── trace.go            → Trace 写入/查询
  ├── streaming.go        → SSE/WebSocket 流式输出
  └── moa.go              → Mixture of Agents 多模型编排

internal/agent/
  ├── routes.go           → Agent HTTP 路由
  ├── handler.go          → HTTP handler
  ├── service.go          → Agent 查询/列表服务
  └── impl/               → Agent 具体实现
       ├── agents.go      → Agent 接口定义 + All() 注册函数
       ├── coordinator.go → G0 健康/协调
       ├── sourcing.go    → A8 选品
       ├── logistics.go   → A10 物流费率
       └── ...            → 各 Agent 实现

internal/agentos/
  ├── routes.go           → AgentOS 控制台路由
  ├── handler.go          → HTTP handler
  └── service.go          → Work Item 管理
```

---

## 1. LLM 编排 (`internal/ai/`)

### 1.1 LLM Provider

`llm_provider.go` 定义统一的 LLM Provider 接口：

```go
type LLMProvider interface {
    Name() string
    Chat(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
    ChatStream(ctx context.Context, req *LLMRequest) (<-chan LLMChunk, error)
}
```

**支持的 Provider：** 通过 `LLM_PROVIDER` 环境变量选择。

| Provider | 值 | 默认模型 | 默认 Base URL |
|----------|-----|-----------|----------------|
| OpenAI 兼容 | `openai` | `gpt-4o-mini` | `https://api.openai.com/v1` |
| Anthropic | `anthropic` | 环境变量指定 | `https://api.anthropic.com` |
| 通义千问 | `qwen` | `qwen-plus` | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| DeepSeek | `deepseek` | `deepseek-chat` | `https://api.deepseek.com/v1` |
| Azure | `azure` | 环境变量指定 | 环境变量指定 |
| Stub (开发用) | `stub` 或空 | — | — |

**环境变量：**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `LLM_PROVIDER` | 选择 Provider | `stub`（开发模式） |
| `LLM_API_KEY` | API 密钥 | 生产环境必填 |
| `LLM_MODEL` | 模型名称 | Provider 特定默认 |
| `LLM_BASE_URL` | API Base URL | Provider 特定默认 |

生产环境（`ENV=production` 或 `GIN_MODE=release`）下，未设置 `LLM_PROVIDER` 或无 `LLM_API_KEY` 导致进程退出。

### 1.2 数据模型

`model.go` 中的核心类型：

```go
// Agent 注册表条目
type AgentSpec struct {
    ID             string
    Name           string
    Squad          string          // governance | insight | fulfillment | ops
    Autonomy       string          // advisory | guided | supervised | autonomous
    DecisionPoints []string        // 决策点列表
    Description    string
    ModelHint      string
    Tools          []string
}

// LLM 请求 / 响应
type LLMRequest struct {
    System      string
    Messages    []LLMMessage
    Model       string
}

type LLMResponse struct {
    Answer       string
    Model        string
    TokensIn     int
    TokensOut    int
    LatencyMs    int
    FinishReason string
}

// 统一 Action（Agent 决策输出）
type UnifiedAction struct {
    ID            uint
    TraceID       string
    AgentID       string
    DecisionPoint string
    ActionType    string
    Title         string
    Description   string
    RiskLevel     string    // low / medium / high / critical
    Status        string    // suggested / pending / approved / rejected / executed / failed
    TargetType    string    // product / order / listing / …
    TargetID      uint
    ProposedAt    time.Time
    ExpiresAt     *time.Time
    ApprovedBy    *string
    ApprovedAt    *time.Time
    ExecutedBy    *string
    ExecutedAt    *time.Time
    // ...
}
```

### 1.3 Agent 注册表

`registry.go` 中的 `AgentRegistry` 管理所有 Agent 定义：

```go
type AgentRegistry struct {
    Agents map[string]*AgentSpec
}

func DefaultRegistry() *AgentRegistry    // 注册所有预定义 Agent
func (r *AgentRegistry) Get(id string) (*AgentSpec, bool)
func (r *AgentRegistry) All() []*AgentSpec
```

**预注册的 15 个 Agent：**

| ID | 名称 | Squad | 决策点 |
|----|------|-------|--------|
| G0 | 系统健康 | governance | system_health, anomaly_escalation, cross_squad_coordinate, agent_audit |
| G1 | 看板总览 | governance | dashboard_overview |
| G2 | 仓储报关 | governance | warehouse_routing, customs_declaration |
| G3 | 折扣风控 | governance | discount_risk_check |
| A1 | 选品调研 | insight | sourcing_discovery |
| A2 | 经营复盘 | insight | listing_optimize |
| A3 | 广告分析 | insight | acos_analysis |
| A4 | 客服回复 | ops | auto_reply |
| A5 | 库存预警 | fulfillment | stock_alert |
| A6 | 利润看护 | fulfillment | profit_watch |
| A7 | 合规检测 | fulfillment | compliance_check |
| A8 | 选品扫描 | insight | sourcing_scan |
| A9 | 批量运维 | ops | batch_price_update, batch_inventory_sync, batch_listing_update |
| A10 | 物流费率 | ops | carrier_compare, shipping_bill_audit, carrier_performance |
| A11 | 售后管理 | ops | return_analysis, refund_decision, dispute_manage |

### 1.4 Orchestrator 执行模型

`orchestrator.go` 中的 `Orchestrator` 是 Agent 决策编排的核心：

```go
type Orchestrator struct {
    db         *gorm.DB
    registry   *AgentRegistry
    provider   LLMProvider
    agentImpls map[string]impl.Agent
    traces     *TraceWriter
    hub        *realtime.Hub          // WebSocket 推送
    bus        EventPublisher         // EventBus 事件发布
    guardrails *guardrails.Chain      // AIOS 护栏链（可选）
    budget     *costcontrol.Controller // 成本控制（可选）
    cmd        *command.Dispatcher     // Action 执行命令分发
    cat        *actioncatalog.Catalog  // 生产验证 Action Catalog
}
```

**建造者模式配置：**

```go
orch := ai.NewOrchestrator(db, logger)
    .WithProvider(provider)
    .WithHub(hub)
    .WithBus(bus)
    .WithDispatcher(cmd)
    .WithCatalog(cat)
    .WithGuardrails(guardChain)
    .WithBudget(budgetCtrl)
    .WithDecisionCache(5 * time.Minute)
```

**三种执行路径：**

| 方法 | 用途 | LLM 调用 | Action 创建 |
|------|------|----------|-------------|
| `RunAgent()` | 业务逻辑 Agent 立即执行 | 否 | 否（直接返回结果） |
| `RunAgentWithLLM()` | AI 增强的 Agent 决策 | 是 | 是（创建 UnifiedAction） |
| `ExecuteAction()` | 执行已审批的 Action | 否 | 否（执行状态转换 + 审计） |

**Lazy 执行流程：**

```
1. RunAgentWithLLM → UnifiedAction(status=suggested)
2. Owner 在 AgentOS / Owner 面板审查
3. POST /actions/:id/approve → status=approved
4. POST /actions/:id/execute
   → 门禁链：RBAC → 审批验证 → 幂等检查 → Action Catalog → 审计日志
   → Command Dispatcher 执行
5. 结果 → Trace 记录 → EventBus 发布 → WebSocket 推送
```

### 1.5 Trace 系统

`trace.go` 记录 Agent 每次决策的执行轨迹：

```go
type Trace struct {
    ID              string
    TraceID         string
    AgentID         string
    DecisionPoint   string
    Status          string         // running / completed / failed
    Input           map[string]interface{}
    Output          map[string]interface{}
    Confidence      float64
    RiskLevel       string
    TokensUsed      int
    StartedAt       time.Time
    CompletedAt     *time.Time
    Error           string
}

type TraceWriter struct{ db *gorm.DB; logger *zap.Logger }

func NewTraceWriter(db *gorm.DB, logger *zap.Logger) *TraceWriter
func (w *TraceWriter) StartTrace(traceID, agentID, decisionPoint string, input map[string]interface{}) error
func (w *TraceWriter) CompleteTrace(traceID string, output map[string]interface{}, confidence float64, riskLevel string, tokens int) error
func (w *TraceWriter) FailTrace(traceID string, err error) error
```

### 1.6 Guardrails 集成

AIOS 护栏链通过 `WithGuardrails()` 可选注入。当前已集成的门禁系统：

| 门禁 | 位置 | 触发点 |
|------|------|--------|
| JWT 认证 | `middleware.Auth` | 所有 `/api/v1/*` |
| RBAC 权限 | `middleware.RequirePermission` | 财务/报表等敏感模块 |
| 审批检查 | `ai/service.go` | `ExecuteAction` 执行前 |
| 幂等守卫 | `ai/service.go` | 重复执行 blocking |
| Action Catalog | `actioncatalog.Catalog` | 高风险动作标记为 `AutonomousBlocked: true` |
| 平台执行模式 | `integrations/types.go` | `live` / `dry_run` / `sandbox` |
| 审计日志 | `operationlog` | 执行/审批/反馈写入 |
| 状态机 | `listingtask/statemachine.go` | ListingTask 状态转换 |

---

## 2. Agent 系统 (`internal/agent/`)

### 2.1 Agent 接口

`internal/agent/impl/agents.go` 定义 Agent 接口和注册函数：

```go
type Agent interface {
    Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (
        output map[string]interface{},
        confidence float64,
        riskLevel string,
        err error,
    )
}

// All 返回所有已注册的 Agent 实现
func All(db *gorm.DB, logger *zap.Logger) map[string]Agent
```

每个 Agent 按 decisionPoint 分发到不同的决策逻辑：

```go
func (a *CoordinatorAgent) Decide(ctx context.Context, dp string, params map[string]interface{}) (...) {
    switch dp {
    case "system_health":
        return a.systemHealth(ctx, params)
    case "anomaly_escalation":
        return a.anomalyEscalation(ctx, params)
    // ...
    }
}
```

### 2.2 Agent 实现一览

| ID | 实现文件 | 核心逻辑 |
|----|----------|----------|
| A1 | `scout.go` | ProductScoutAgent — 选品发现 |
| A2 | `listing_optimizer.go` | ListingOptimizerAgent — 刊登优化 |
| A3 | `ad_advice.go` | AdAdviceAgent — 广告建议 |
| A4 | `customer_service.go` | CustomerServiceAgent — 客服自动回复 |
| A5 | `inventory.go` | InventoryAlertAgent — 库存水位检查 |
| A6 | `profit_watch.go` | ProfitWatchAgent — 利润监控 |
| A7 | `compliance.go` | ComplianceGuardAgent — 合规检测 |
| A8 | `sourcing.go` | SourcingAgent — 1688 选品扫描 |
| A9 | `batch_ops.go` | BatchOpsAgent — 批量运维 |
| A10 | `logistics.go` | LogisticsOpsAgent — 物流费率引擎 |
| A11 | `aftersales.go` | AftersalesMgmtAgent — 售后管理 |
| G0 | `coordinator.go` | CoordinatorAgent — 系统健康/协调 |
| G1 | `dashboard.go` | DashboardAgent — 看板总览 |
| G2 | `warehouse_customs.go` | WarehouseCustomsAgent — 仓储报关 |
| G3 | `discount_risk.go` | DiscountRiskAgent — 折扣风控 |

### 2.3 Agent 决策链

EventBus 驱动的链式触发：

```
Scheduler tick → publish "scheduler.tick.{agent_id}"
  ↓
Agent 订阅收到事件 → 运行 Decide()
  ↓
Agent 决策结果 → "agent.decided.{agent_id}" 事件
  ↓
下游 Agent 订阅 → 链式触发
```

决策链拓扑：

```
A5 stock_alert（库存低）       → G3 discount_risk_check（折扣风控）
G3 discount_risk_check（阻断）  → A6 profit_watch（利润监控）
A6 profit_watch（亏损）        → A2 listing_optimize（刊登优化）
G0 system_health（异常>3）     → G1 dashboard_overview（看板聚合）
```

### 2.4 HTTP API

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/agents` | Agent 列表 |
| GET | `/api/v1/agents/:id` | Agent 详情 |
| POST | `/api/v1/agents` | 创建 Agent |
| PUT | `/api/v1/agents/:id` | 更新 Agent |
| DELETE | `/api/v1/agents/:id` | 删除 Agent |
| GET | `/api/v1/agents/specs` | 完整 Agent Spec 列表 |
| GET | `/api/v1/agents/evolution` | 演化建议 |
| GET | `/api/v1/agents/entropy` | 熵监控数据 |
| POST | `/api/v1/agents/:id/actions` | 创建 Agent Action |
| CRUD | `/api/v1/agents/rules[/:id]` | Agent 行为规则 |
| POST | `/api/v1/agents/rules/apply` | 应用规则批量变更 |

---

## 3. AgentOS 控制台 (`internal/agentos/`)

### 3.1 Work Item 模型

```go
type WorkItem struct {
    ID          uint
    AgentID     string
    Type        string    // approval / review / exception
    Title       string
    Description string
    Priority    string    // low / medium / high / critical
    Status      string    // pending / in_progress / completed / cancelled
    DecisionID  *uint
    AssignedTo  *string
    SLA         *time.Time
    Escalated     bool
    EscalatedAt   *time.Time
    CompletedAt   *time.Time
}
```

### 3.2 HTTP API

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/agentos` | AgentOS 总控台聚合 |
| GET | `/api/v1/agentos/status` | Agent 状态聚合 |
| GET | `/api/v1/agentos/work-items` | 工作队列（分页/筛选） |
| GET | `/api/v1/agentos/autonomy` | 自主运营概览 |

### 3.3 SLA 升级策略

| 优先级 | SLA | 超时动作 |
|--------|-----|----------|
| Low | 无 | 仅队列显示 |
| Medium | 4h | 升级提醒通知 |
| High | 2h | 升级到管理员 |
| Critical | 30min | 紧急升级 + 通知 |

---

## 4. AI HTTP API

### 4.1 Chat & 流式

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/ai/chat` | AI Chat（支持流式 SSE） |
| POST | `/api/v1/ai/run` | 运行 Agent 决策 |

Chat 请求：

```json
{
  "message": "查询SKU 123的库存状态",
  "agent_id": "A5",
  "decision_point": "stock_alert",
  "stream": true
}
```

流式输出通过 WebSocket 端点 `ws://host/ws` 推送。Agent 运行期间推送 Progress 事件和最终结果。

### 4.2 Action 管理

| Method | Path | 说明 | 权限 |
|--------|------|------|------|
| GET | `/api/v1/ai/actions` | Action 列表 | 认证用户 |
| GET | `/api/v1/ai/actions/:id` | Action 详情 | 认证用户 |
| POST | `/api/v1/ai/actions` | 创建 Action | 认证用户 |
| POST | `/api/v1/ai/actions/:id/approve` | 审批 | `ai.action` |
| POST | `/api/v1/ai/actions/:id/reject` | 拒绝 | `ai.action` |
| POST | `/api/v1/ai/actions/:id/execute` | 执行 | `ai.action` |
| POST | `/api/v1/ai/actions/:id/review` | 回顾评论 | 认证用户 |

### 4.3 Agent 数据

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/ai/agents` | Agent 列表（含状态） |
| GET | `/api/v1/ai/agents/specs` | Agent 完整 Spec |
| GET | `/api/v1/ai/traces` | Trace 列表 |
| GET | `/api/v1/ai/traces/:trace_id` | Trace 详情（含 input/output/confidence） |

### 4.4 Trace 响应结构

```json
{
  "trace_id": "trc_abc123",
  "agent_id": "A5",
  "decision_point": "stock_alert",
  "status": "completed",
  "confidence": 0.92,
  "risk_level": "medium",
  "tokens_used": 1234,
  "started_at": "2026-07-09T10:00:00Z",
  "completed_at": "2026-07-09T10:00:05Z",
  "output": {
    "stock_level": 12,
    "safety_stock": 20,
    "alert": "below_safety",
    "suggested_replenish": 30
  }
}
```

---

## 5. AIOS LLM Gateway（施工中）

`internal/aios/llmgateway/` 提供高级 LLM Gateway。当前为增量采纳状态，与 `internal/ai/` 共存。

| 特性 | 状态 | 说明 |
|------|------|------|
| `Gateway.Chat()` | ✅ 已实现 | 路由 + 缓存 + 降级 |
| `DefaultRouter` | ✅ 已实现 | 规则驱动的模型选择 |
| 语义缓存 | ✅ 已实现 | 哈希缓存 + TTL |
| 降级链 | ✅ 已实现 | Opus → Sonnet → Haiku → 规则引擎 |
| 成本归属 | ✅ 已实现 | Token 追踪 + 成本估算 |
| `aiProviderAdapter` | ✅ 已实现 | 桥接 `internal/ai/` 的 Provider |

Gateway 接口：

```go
type Provider interface {
    Name() string
    Chat(ctx context.Context, req *Request) (*Response, error)
}

type Router interface {
    Select(ctx context.Context, req *Request) ModelTarget
}
```

模型选择策略：

| 条件 | 路由到 | 原因 |
|------|--------|------|
| `MaxLatency < 3s` | Haiku | 最快的响应 |
| `Sensitive=true` (财务/风控) | Opus | 最审慎 |
| 决策/分析 | Sonnet | 平衡 |
| 分类/抽取 | Haiku | 成本优选 |
| 复杂推理 | Opus | 深度思考 |

---

## 相关文档

- [AIOS 基础设施架构](aios-architecture.md) — 11 个 AIOS 内核模块设计蓝图
- [Agent 能力清单](AGENT_CAPABILITIES.md) — 每个 Agent 的详细能力
- [How-to: Agent 规则](howto-agent-rules.md) — 控制 Agent 决策边界和触发条件
- [权限与审计](PERMISSIONS_AND_AUDIT.md) — 鉴权规则和审计日志接入
- [系统架构设计 v1](system-architecture-design-v1.md) — 九层架构总览
