# 凌镜 LingMirror 系统架构设计

> 跨境電商 AI AgentOS 完整系统架构设计
> 版本: v1.0 (迭代 3/3 终稿)
> 日期: 2026-06-28
> 状态: 待审核

---

## 目录

- [1. 设计总览](#1-设计总览)
- [2. 系统分层架构](#2-系统分层架构)
- [3. 核心数据流](#3-核心数据流)
- [4. AIOS 内核: 11 个基础设施模块](#4-aios-内核-11-个基础设施模块)
- [5. Agent 编排模式](#5-agent-编排模式)
- [6. 决策全生命周期状态机](#6-决策全生命周期状态机)
- [7. 跨 Agent 数据一致性](#7-跨-agent-数据一致性)
- [8. 错误恢复策略](#8-错误恢复策略)
- [9. 作业调度](#9-作业调度)
- [10. 四层 Agent 执行模型](#10-四层-agent-执行模型)
- [11. 端到端决策时序全景](#11-端到端决策时序全景)
- [12. 现有设施映射](#12-现有设施映射)
- [13. 实现优先级](#13-实现优先级)
- [14. 核心架构决策](#14-核心架构决策)

---

## 1. 设计总览

凌镜 LingMirror 是 **AI 原生跨境商品经营操作系统**，核心流程：

```
商品创建 → SKU/价格/库存维护 → AI 优化与经营决策 → 多平台发布 → 订单、结算、财务、异常和 AgentOS 运营闭环
```

### 关键设计原则

1. **Agent 不直接读写 DB** — 一切业务操作通过 Tool API 调用，Tool 内部执行 CRUD + 权限检查 + 审计
2. **AIOS 内核服务之间不直接依赖** — 通过 Event Bus 或 Tool Registry 互操作
3. **每个基础设施组件有明确的 Go interface** — 可独立测试、可替换实现
4. **已有地基不动** — Event Bus / Scheduler / Command Dispatcher 继续存在，AIOS 层看作扩展
5. **一套底座，两个入口** — 运营后台（手动操作）+ AgentOS 总控台（AI 管理）
6. **所有经营动作标准化** — 人和 Agent 都通过同一套动作中枢操作系统

---

## 2. 系统分层架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│ L9: 用户体验层 (UI/UX)                                                   │
│ ┌──────────────┐  ┌──────────────────┐  ┌──────────────────────────┐   │
│ │ 运营后台      │  │ AgentOS 总控台    │  │ 外部访问                  │   │
│ │ (企业管理人)  │  │ (老板/运营负责人)  │  │ 公开页面 / SEO / Landing │   │
│ └──────────────┘  └──────────────────┘  └──────────────────────────┘   │
│ ┌──────────────────────────────────────────────────────────────────┐   │
│ │ Next.js 16 App Router / React 19 / Ant Design 6 / Zustand 5     │   │
│ │ WebSocket Streaming / Reconnecting-WebSocket / TanStack Query 5 │   │
│ └──────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────┤
│ L8: 接入层 (API Gateway)                                                 │
│ ┌──────────────────────────────────────────────────────────────────┐   │
│ │ Gin HTTP Server / /api/v1/... / /ws / /metrics / /api/health    │   │
│ │ Middleware: CORS → RequestID → Metrics → Recovery → Audit → Auth│   │
│ │ Rate Limiting / RBAC 权限检查                                     │   │
│ └──────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────┤
│ L7: AIOS 内核服务层 (11 个基础设施模块)                                   │
│ ┌──────────┐  ┌────────┐  ┌────────┐  ┌───────────┐  ┌───────────┐   │
│ │Tool      │  │Agent   │  │LLM     │  │Memory     │  │Guardrails │   │
│ │Registry  │  │IPC     │  │Gateway │  │System     │  │(护栏)     │   │
│ └──────────┘  └────────┘  └────────┘  └───────────┘  └───────────┘   │
│ ┌──────────┐  ┌────────┐  ┌────────┐  ┌───────────┐  ┌───────────┐   │
│ │Decision  │  │Agent   │  │Know-   │  │Autonomous │  │Agent SDK  │   │
│ │Pipeline  │  │Observ. │  │ledge   │  │Evolution  │  │(开发框架) │   │
│ └──────────┘  └────────┘  └────────┘  └───────────┘  └───────────┘   │
│                                      所有模块在 internal/aios/ 下      │
├─────────────────────────────────────────────────────────────────────────┤
│ L6: 平台基础设施层 (已有, 加固)                                           │
│ ┌───────┐  ┌────────┐  ┌───────┐  ┌───────┐  ┌──────┐  ┌────────┐   │
│ │Event  │  │Sched-  │  │Command│  │RBAC+  │  │Web-  │  │操作日志│   │
│ │Bus    │  │uler    │  │Disp.  │  │Auth   │  │Socket│  │(Audit) │   │
│ └───────┘  └────────┘  └───────┘  └───────┘  └──────┘  └────────┘   │
│ ┌───────┐  ┌────────┐  ┌───────┐  ┌────────────┐                    │
│ │Tool-  │  │Approval│  │Agent  │  │Middleware  │                    │
│ │Bridge │  │Policies│  │Reg-   │  │Stack (已有)│                    │
│ └───────┘  └────────┘  │istry  │  └────────────┘                    │
│                        └───────┘                                    │
├─────────────────────────────────────────────────────────────────────────┤
│ L5: 领域业务模块层 (Domain Modules)                                      │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  │
│ │Product   │  │Order     │  │Inventory │  │Listing   │  │Finance │  │
│ │SKU/Price │  │Shipping  │  │Supplier  │  │Platform  │  │Settle  │  │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘  └────────┘  │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  │
│ │Sourcing  │  │Logistics │  │Platform  │  │Exchange  │  │Agent   │  │
│ │(选品)    │  │(物流费率) │  │Fee(费用) │  │Rate(汇率)│  │Rules   │  │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘  └────────┘  │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│ │TrustScore│  │Entropy   │  │Evolution │  │Action    │              │
│ │(信任分)  │  │(自净化)  │  │(演化)    │  │Policy    │              │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘              │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐                            │
│ │Decision  │  │Integ-    │  │Imagegen  │                            │
│ │(决策)    │  │rations   │  │(生图)    │                            │
│ └──────────┘  └──────────┘  └──────────┘                            │
├─────────────────────────────────────────────────────────────────────────┤
│ L4: Agent 层                                                           │
│ ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │
│ │G0: 系统健康 │  │G1: 看板总览│  │G2: 市场监测│  │G3: 折扣风控   │   │
│ └────────────┘  └────────────┘  └────────────┘  └────────────────┘   │
│ ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │
│ │A4: 供应商   │  │A5: 库存预警│  │A6: 利润监控│  │A7: 平台发布   │   │
│ └────────────┘  └────────────┘  └────────────┘  └────────────────┘   │
│ ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │
│ │A8: 选品    │  │A9: 刊登优化│  │A10: 物流费率│  │M1: 客服       │   │
│ └────────────┘  └────────────┘  └────────────┘  └────────────────┘   │
│ ┌────────────┐  ┌────────────┐                                         │
│ │A2: 经营复盘│  │A3: 异常处理│                                         │
│ └────────────┘  └────────────┘                                         │
├─────────────────────────────────────────────────────────────────────────┤
│ L3: 数据持久化层                                                       │
│ ┌───────────────┐  ┌──────────────┐  ┌────────────────────┐          │
│ │ PostgreSQL 15 │  │ Redis        │  │ Vector Store (规划) │          │
│ │ GORM ORM      │  │ 缓存/队列    │  │ 知识向量化存储     │          │
│ └───────────────┘  └──────────────┘  └────────────────────┘          │
├─────────────────────────────────────────────────────────────────────────┤
│ L2: 集成层 (External Integrations)                                      │
│ ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌──────────────────┐    │
│ │Shopee     │  │Ozon       │  │Lazada(规划)│  │1688 (采购)       │    │
│ └───────────┘  └───────────┘  └───────────┘  └──────────────────┘    │
│ ┌───────────┐  ┌───────────┐  ┌───────────┐                          │
│ │LLM        │  │Sentry     │  │Prometheus │                          │
│ │API(外部)  │  │(错误追踪) │  │(监控)    │                          │
│ └───────────┘  └───────────┘  └───────────┘                          │
├─────────────────────────────────────────────────────────────────────────┤
│ L1: 治理 & 运维层                                                       │
│ ┌────────────────┐  ┌────────────────┐  ┌──────────────────────────┐  │
│ │Owner-First协议  │  │平台宪法(规则)  │  │Agent 协作协议            │  │
│ └────────────────┘  └────────────────┘  └──────────────────────────┘  │
│ ┌────────────────┐  ┌────────────────┐                                │
│ │Kernel 契约文档  │  │Docker部署(生产) │                                │
│ └────────────────┘  └────────────────┘                                │
└─────────────────────────────────────────────────────────────────────────┘
```

### AIOS 内核模块依赖关系

```
                           ┌──────────────────────┐
                           │    Agent SDK          │ ← 开发框架
                           └──────────┬───────────┘
                                      │ 使用
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
     ┌────────────────┐   ┌──────────────────┐   ┌──────────────────┐
     │  Tool Registry │   │  Agent Runtime   │   │  Decision        │
     │   Agent发现工具  │   │  生命周期/资源/沙箱 │   │  Pipeline        │
     └───────┬────────┘   └────────┬─────────┘   │  决策拓扑编排     │
             │                     │              └────────┬─────────┘
             │                     │                       │
             ▼                     ▼                       ▼
     ┌──────────────────────────────────────────────────────────┐
     │                   核心依赖层                                 │
     │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
     │  │LLM       │  │ Memory   │  │Guardrails│  │ Agent    │ │
     │  │Gateway   │  │System    │  │分层护栏   │  │ IPC      │ │
     │  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
     └──────────────────────────────────────────────────────────┘
             │              │              │              │
             ▼              ▼              ▼              ▼
     ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐
     │Agent     │  │Knowledge │  │Autonomous│  │  Event Bus   │
     │Observ.   │  │Engine    │  │Evolution │  │  (已有)      │
     └──────────┘  └──────────┘  └──────────┘  └──────────────┘
```

---

## 3. 核心数据流

```
                ┌──────────────┐
                │  LLM 提供商   │
                │  (OpenAI/)   │
                └──────┬───────┘
                       │
               ┌───────▼───────┐
               │  LLM Gateway  │  ── 统一provider抽象、流式输出、token记账
               └───────┬───────┘
                       │
          ┌────────────┼────────────┐
          │            │            │
  ┌───────▼────┐ ┌────▼────┐ ┌────▼──────┐
  │ Agent 运行时│ │Agent    │ │Guardrails│  ── 分层护栏：政策/风险/权限
  │ 生命周期管理│ │IPC通信  │ │(策略拦截) │
  └───────┬────┘ └─────────┘ └───────────┘
          │
  ┌───────▼───────────────┐
  │  决策管道 (Pipeline)   │  ── 观察→提案→策略检查→审批→执行→审计→复盘
  └───────┬───────────────┘
          │
  ┌───────▼───────┐      ┌───────────────┐
  │ Tool Registry │      │ Memory System │  ── 短期(会话)/长期(向量)/共享(Agent间)
  └───────┬───────┘      └───────────────┘
          │
  ┌───────▼─────────────────────────────────────┐
  │  Domain Service (通过 Command/ToolBridge 调用)│
  │  Product → Order → Finance → ...            │
  └───────┬─────────────────────────────────────┘
          │
  ┌───────▼───┐     ┌───────────┐     ┌────────┐
  │PostgreSQL │     │  Redis    │     │ 外部API │
  └───────────┘     └───────────┘     └────────┘
```

### 当前 Agent 管道链 (EventBus 驱动)

```
                      ┌──────────────┐
                      │  G0 系统健康  │ ← 每个scheduler tick
                      └──────┬───────┘
                             │ anomaly > 3
                             ▼
              ┌──────────────────────────┐
              │  G1 看板总览 (dashboard)  │
              └──────┬───────────────────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
  ┌──────────┐ ┌──────────┐ ┌──────────┐
  │ A4 供应商  │ │ A5 库存   │ │ G3 折扣   │
  │ 评估     │ │ 预警     │ │ 风控     │
  └──────────┘ └─────┬────┘ └────┬─────┘
                     │ red       │ block
                     ▼           ▼
              ┌──────────┐ ┌──────────┐
              │ A6 利润   │ │ A2 经营   │
              │ 监控     │ │ 复盘     │
              └─────┬────┘ └──────────┘
                    │ loss/threshold
                    ▼
              ┌──────────┐
              │ A9 刊登   │
              │ 优化     │
              └──────────┘
```

---

## 4. AIOS 内核: 11 个基础设施模块

### 4.1 Tool Registry (工具注册表)

```go
// 核心数据结构
type Tool struct {
    Name        string            // "order.ship.create"
    Version     string            // "1.0.0"
    Description string            // LLM看到的人类描述
    Squad       string            // 所属Agent Squad
    Parameters  *jsonschema.Schema // JSON Schema
    Returns     *jsonschema.Schema // 返回Schema
    RequiredPermissions []string  // ["order.write"]
    RiskLevel           string    // low/medium/high/critical
    Handler func(ctx, map[string]any) (any, error)
    CostTokens     int
    MaxDuration    time.Duration
    CircuitBreaker *CircuitConfig
    SensitiveData  bool
}

type Registry interface {
    Register(ctx, tool) error
    Unregister(ctx, name string) error
    Get(name string) (*Tool, error)
    List() []*Tool
    Search(query string) []*Tool  // LLM-friendly发现
    Call(ctx, name string, input map[string]any) (any, error)
}
```

### 4.2 LLM Gateway

```go
type Provider interface {
    Chat(ctx, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx, req ChatRequest) (<-chan StreamEvent, error)
    Name() string
}

type ChatRequest struct {
    Model    string
    Messages []Message
    Tools    []Tool
    Config   ChatConfig       // temperature, max_tokens, thinking
    Budget   *TokenBudget
}

type Gateway interface {
    Send(ctx, req ChatRequest) (*ChatResponse, error)
    SendStream(ctx, req ChatRequest) (<-chan StreamEvent, error)
    EstimateTokens(req ChatRequest) TokenEstimate
    GetProvider(name string) (Provider, error)
    Fallback(ctx, req, primary, backup string) (*ChatResponse, error)
}
```

### 4.3 Memory System (记忆系统)

```go
type MemoryType string
const (
    ShortTerm  MemoryType = "short_term"    // 会话级
    LongTerm   MemoryType = "long_term"     // Agent持久记忆
    Shared     MemoryType = "shared"        // Agent间共享
    Episodic   MemoryType = "episodic"      // 决策事件记忆
)

type MemoryEntry struct {
    ID          string
    AgentID     string
    SquadID     string
    Type        MemoryType
    Key         string
    Content     string
    Embedding   []float64
    Metadata    map[string]any
    TTL         *time.Duration
    Importance  float64       // 0-1
    CreatedAt   time.Time
    AccessedAt  time.Time
}

type MemorySystem interface {
    Store(ctx, entry MemoryEntry) error
    Recall(ctx, agentID string, query string, opts RecallOpts) ([]MemoryEntry, error)
    Forget(ctx, entryID string) error
    Prune(ctx, agentID string, maxEntries int) (int, error)
    Consolidate(ctx, agentID string) error  // 短→长记忆固化
}
```

### 4.4 Decision Pipeline (决策管道)

```go
type DecisionStage string
const (
    StageObserve   DecisionStage = "observe"
    StagePropose   DecisionStage = "propose"
    StagePolicy    DecisionStage = "policy"
    StageApproval  DecisionStage = "approval"
    StageExecute   DecisionStage = "execute"
    StageAudit     DecisionStage = "audit"
    StageReview    DecisionStage = "review"
)

type Decision struct {
    ID              string
    AgentID         string
    Stage           DecisionStage
    ToolName        string
    Input           map[string]any
    Output          map[string]any
    RiskLevel       string
    PolicyVerdict   string     // pass/block/review
    ApprovalStatus  string     // pending/approved/rejected/skipped
    ParentID        string     // 父决策ID
    CreatedAt       time.Time
    CompletedAt     time.Time
}

type Pipeline interface {
    Execute(ctx, dec Decision) (*Decision, error)
    GetDecision(ctx, id string) (*Decision, error)
    GetChain(ctx, id string) ([]Decision, error)
    Cancel(ctx, id string) error
}
```

### 4.5 Guardrails (护栏系统)

```go
type GuardrailType string
const (
    PolicyGuardrail   GuardrailType = "policy"
    RiskGuardrail     GuardrailType = "risk"
    PermissionGuard   GuardrailType = "permission"
    CostGuardrail     GuardrailType = "cost"
    RateLimitGuard    GuardrailType = "rate"
)

type Guardrails interface {
    Check(ctx, agentID string, action Tool, input map[string]any) (*GuardrailResult, error)
    RegisterRule(ctx, rule GuardrailRule) error
    NeedsApproval(ctx, action Tool, input map[string]any) (bool, *ApprovalConfig)
}
```

### 4.6 Agent IPC (Agent间通信)

```go
type MessageType string
const (
    IPCRequest  MessageType = "request"
    IPCResponse MessageType = "response"
    IPCBroadcast MessageType = "broadcast"
    IPCEvent    MessageType = "event"
)

type IPC interface {
    Send(ctx, msg IPCMessage) error
    Request(ctx, msg IPCMessage) (*IPCMessage, error) // 同步请求-响应
    Subscribe(topic string, handler func(msg IPCMessage)) (func(), error)
    Broadcast(ctx, topic string, payload []byte) error
    Health() map[string]AgentIPCStatus
}
```

### 4.7 Agent Observability

```go
type AgentMetrics struct {
    AgentID        string
    SquadID        string
    DecisionsTotal int64
    DecisionsOK    int64
    DecisionsFail  int64
    AvgLatency     time.Duration
    TokenUsage     TokenUsage
    LastHeartbeat  time.Time
}

type AgentObservability interface {
    RecordDecision(ctx, decision Decision) error
    RecordTokenUsage(ctx, agentID string, usage TokenUsage) error
    GetAgentMetrics(ctx, agentID string) (*AgentMetrics, error)
    ReportHealth(ctx, agentID string) error
    StartTrace(ctx, decisionID string) error
    EndTrace(ctx, decisionID string) error
    GetTrace(ctx, decisionID string) ([]TraceSpan, error)
}
```

### 4.8 Knowledge Engine (知识引擎)

```go
type KnowledgeDocument struct {
    ID          string
    Title       string
    Content     string
    Source      string    // "policy_doc" | "market_report" | "user_faq"
    Tags        []string
    Embedding   []float64
    SquadScope  []string
}

type KnowledgeEngine interface {
    Index(ctx, doc KnowledgeDocument) error
    Search(ctx, query string, opts SearchOpts) ([]KnowledgeDocument, error)
    Get(ctx, id string) (*KnowledgeDocument, error)
    AutoExtract(ctx, agentID string) ([]KnowledgeDocument, error)
}
```

### 4.9 Autonomous Evolution (自治演进)

```go
type EvolutionExperiment struct {
    ID              string
    AgentID         string
    ExperimentName  string
    Variant         string    // "A" | "B"
    Parameter       string
    OldValue        any
    NewValue        any
    MetricsBefore   AgentMetrics
    MetricsAfter    AgentMetrics
    Improvement     float64
    RolledOut       bool
}

type AutonomousEvolution interface {
    ProposeExperiment(ctx, agentID string) (*EvolutionExperiment, error)
    RunExperiment(ctx, experiment *EvolutionExperiment) error
    AnalyzeResult(ctx, experimentID string) (*EvolutionAnalysis, error)
    RootCauseAnalysis(ctx, agentID string, window time.Duration) (*RCAResult, error)
}
```

### 4.10 Agent Runtime

```go
type AgentState string
const (
    StateInit    AgentState = "init"
    StateReady   AgentState = "ready"
    StateActive  AgentState = "active"
    StateSuspend AgentState = "suspend"
    StateStop    AgentState = "stop"
)

type Runtime interface {
    Start(ctx, agentID string) error
    Stop(ctx, agentID string) error
    Suspend(ctx, agentID string) error
    Resume(ctx, agentID string) error
    GetState(ctx, agentID string) (*AgentState, error)
    IsHealthy(ctx, agentID string) bool
    AllocateResources(ctx, agentID string, req ResourceRequest) (*ResourceAllocation, error)
    ReleaseResources(ctx, agentID string) error
}
```

### 4.11 Agent SDK

Agent 声明式定义:

```yaml
# manifest.yaml
name: "A5-stock-alert"
version: "1.0.0"
squad: "supply-chain"
schedule: "*/30 * * * *"   # 每30分钟
tools:
  - "inventory.query_current"
  - "inventory.query_history"
  - "purchase_order.create"
trigger:
  event: "scheduler.tick.A5"
risk_level: "medium"
requires_approval: true
autonomy:
  minimum_trust_score: 75
  levels:
    - "observation"
    - "suggestion"
    - "semi_autonomous"
    - "full_autonomous"
```

```go
type Agent struct {
    Manifest   Manifest
    Registry   toolregistry.Registry
    Memory     memory.MemorySystem
    Gateway    llmgateway.Gateway
    Pipeline   pipeline.Pipeline
    Observ     observability.AgentObservability
    Logger     *slog.Logger
}

func (a *Agent) Run(ctx Context) error { /* SDK 生命周期管理 */ }
func (a *Agent) Execute(ctx Context, input map[string]any) (any, error)
func (a *Agent) GetTools() []toolregistry.Tool
```

---

## 5. Agent 编排模式

### 5.1 管道链 (Chain)

```
┌──────┐   事件     ┌──────┐   事件     ┌──────┐
│  A5  │───►stock──►│  G3  │───►block──►│  A6  │
│ 库存  │  alert     │ 折扣  │  discount  │ 利润  │
│ 预警  │           │ 风控  │  risk      │ 监控  │
└──────┘           └──────┘           └──────┘
```

当前基于 EventBus 实现。AIOS 强化：增加超时窗口、熔断。

### 5.2 并行扇出 (Fan-out)

```
                    ┌──────────┐
                    │  G1 看板  │
                    └──────────┘
          ┌──────────┐
    ┌─────┤ A4 供应商 │
    │     └──────────┘
    │     ┌──────────┐
────┤─────┤ A5 库存   │
 事件    └──────────┘
          ┌──────────┐
    └─────┤ A3 异常   │
          └──────────┘
```

EventBus 广播到多个订阅者，各 Agent 独立判断。

### 5.3 汇聚 (Gather)

```
   ┌──────────────────┐
   │ A5: "建议补货30件"│
   ├──────────────────┤
   │ A6: "利润空间OK" │───► 裁决Agent ──► 最终建议
   ├──────────────────┤       (多数决)
   │ A3: "无异常"     │
   └──────────────────┘
```

通过 Agent IPC 同步请求-Response 聚合，裁决 Agent 做加权评分。

---

## 6. 决策全生命周期状态机

```
                    ┌─────────┐
                    │ OBSERVE │  ← Agent观测数据
                    └────┬────┘
                         │
                    ┌────▼────┐
                    │ PROPOSE │  ← 提出动作建议
                    └────┬────┘
                         │
                    ┌────▼────┐
                    │ POLICY  │  ← Guardrails检查
                    └────┬────┘
                         │
              ┌──────────┼──────────┐
              │ allow    │ review   │ deny
              ▼          ▼          ▼
        ┌──────────┐ ┌────────┐ ┌────────┐
        │ APPROVAL │ │EXECUTE │ │ REJECT │
        └────┬─────┘ └───┬────┘ └───┬────┘
    ┌────────┼───────┐   │          │
    ▼        ▼       ▼   │          │
  auto   pending  reject │          │
   go    wait user  back │          │
        ┌───────┐        │          │
        │EXECUTE│◄───────┘          │
        └───┬───┘                   │
            │                       │
        ┌───▼───┐                   │
        │AUDIT  │                   │
        └───┬───┘                   │
            │                       │
        ┌───▼───┐                   │
        │REVIEW │  ← Agent 自我复盘   │
        └───┬───┘                   │
            │                       │
        ┌───▼────┐                  │
        │COMPLETE│◄─────────────────┘
        └────────┘
```

### 并发控制

```go
type DecisionLock struct {
    AgentID    string
    Resource   string      // "product:123:replenish"
    DecisionID string
    AcquiredAt time.Time
    TTL        time.Duration  // 防死锁
}
// 粒度: product.category 级别
// 超时释放: 默认 5 分钟
// 死锁检测: 每 30 秒扫描过期锁
```

---

## 7. 跨 Agent 数据一致性

### SAGA 补偿事务

```go
type SagaStep struct {
    ToolName    string
    Input       map[string]any
    Compensator string        // 补偿操作名
    Commit      func() error
    Rollback    func() error
}

type SagaCoordinator struct {
    DecisionID string
    Steps      []SagaStep
    Completed  []int
    Status     string    // "running" | "committed" | "rolled_back"
}
```

### 幂等性

```go
type IdempotencyStore interface {
    IsProcessed(ctx, key string) (bool, error)
    MarkProcessed(ctx, key string, result any) error
}
// key = "decision:{decision_id}:{tool_name}"
// Tool 执行前检查 → 已存在则返回历史结果
```

---

## 8. 错误恢复策略

| 故障类型            | 恢复策略                        | AIOS 组件          |
|-------------------|--------------------------------|--------------------|
| LLM 调用超时       | 重试(最多3次，指数退避)           | LLM Gateway        |
| Tool 执行失败      | 补偿/SAGA回滚                   | Decision Pipeline   |
| Agent 崩溃(panic)  | Runtime 自动重启                | Agent Runtime       |
| 消息丢失(EventBus) | 持久化 + 重放                   | IPC/EventBus        |
| 死锁(锁竞争)       | TTL 超时自动释放                | Runtime DecisionLock|
| 数据不一致         | 定期 Reconciler 扫描修复         | Observability       |
| Token 预算超限     | 降级: FULL→SUMMARY→SKIP         | LLM Gateway         |

---

## 9. 作业调度

从单向 EventBus 到完整编排:

```
┌─────────────────────────────────────────────┐
│             Job Orchestrator                 │
│                                              │
│  1. Scheduler 触发 tick                       │
│  2. Orchestrator 创建 Job 实例 (持久化)        │
│  3. 检查 Job 依赖 + 并发限制                    │
│  4. 检查 Agent 资源 (Token/负载)               │
│  5. 发送 Decision 到 Agent Runtime            │
│  6. 跟踪 execution (超时、重试)                │
│  7. 记录 Job 结果 + 触发下游                    │
└─────────────────────────────────────────────┘
```

```go
type Job struct {
    ID           string
    Trigger      string       // "scheduler" | "event" | "manual"
    AgentID      string
    ToolName     string
    Input        map[string]any
    Priority     int
    Dependencies []string
    MaxRetries   int
    Timeout      time.Duration
    Status       string       // pending/running/completed/failed/cancelled
    ScheduleAt   *time.Time
}
```

---

## 10. 四层 Agent 执行模型

```
┌─────────────────────────────────────────────────────────────┐
│  L4: 决策层 (What to do)                                     │
│  Agent + LLM == 观察、判断、提案、复盘                         │
│  关键问题: 商品 X 应该上架吗？                                │
│  输入: 商品数据 + 市场数据 + 利润测算 + 历史决策               │
│  输出: {"decision": "上架", "confidence": 0.87}              │
├─────────────────────────────────────────────────────────────┤
│  L3: 编排层 (When to do it)                                  │
│  Job Orchestrator + Decision Pipeline == 调度、审批、跟踪      │
│  检查: Token预算、并发锁、审批状态、Agent负载                  │
│  转换: 决策 → 可执行的动作提案                                │
├─────────────────────────────────────────────────────────────┤
│  L2: 执行层 (How to do it)                                   │
│  Tool Registry + ToolBridge == 参数校验、调用幂等             │
│  执行: 参数校验 → 权限检查 → SAGA提交 → 幂等key标记 → 审计    │
│  补偿: 失败时SAGA回滚已执行步骤                               │
├─────────────────────────────────────────────────────────────┤
│  L1: 记录层 (What happened)                                  │
│  Audit + Observability + Memory == 全链路可追溯              │
│  持久化: Decision Trail + Token Usage + Memory + 操作日志    │
│  结构化: JSON Lines 存储 → 后续分析/复盘/Auto Extract         │
└─────────────────────────────────────────────────────────────┘
```

---

## 11. 端到端决策时序全景

以一个典型场景为例: **A5 库存预警 Agent 发现库存不足 → 触发补货提案 → 审批 → 执行**

```
Scheduler                      AIOS Kernel                      Domain/DB
   │                              │                                │
   │  1. tick "A5" (每30分钟)      │                                │
   ├─────────────────────────────►│                                │
   │                              │                                │
   │  2. Job Orchestrator 创建Job │                                │
   │     检查: Token预算/Agent负载  │                                │
   │                              │                                │
   │  3. Agent A5 启动            │                                │
   │     → Memory: 读取上次快照    │                                │
   │     → Tool: 调库存当前水位   │                                │
   │─────────────────────────────►│──────────────────────────────►│
   │                              │◄──────────────────────────────│
   │                              │                                │
   │  4. LLM Gateway: 分析判断    │                                │
   │     创建提案: 补货20件        │                                │
   │─────────────────────────────►│                                │
   │◄─────────────────────────────│                                │
   │                              │                                │
   │  5. Guardrails: 策略检查     │                                │
   │     补货=中风险 → 需审批     │                                │
   │─────────────────────────────►│                                │
   │                              │                                │
   │  6. Approval: 创建WorkItem   │                                │
   │     通知AgentOS总控台        │                                │
   │─────────────────────────────►│                                │
   │                              │                                │
   │  ←─── 用户批准 (via UI) ────│                                │
   │                              │                                │
   │  7. Decision Pipeline:       │                                │
   │     EXECUTE stage             │                                │
   │     配 idempotency_key        │                                │
   │─────────────────────────────►│                                │
   │                              │                                │
   │  8. Tool: purchase_order.    │                                │
   │     create                   │                                │
   │     校验 → 幂等检查 → 执行    │                                │
   │─────────────────────────────►│──── Command Disp. ──────────►│
   │                              │                                │
   │  9. Audit: 全链路记录        │                                │
   │     Trace + 操作日志          │                                │
   │─────────────────────────────►│                                │
   │                              │                                │
   │  10. Memory: 记录决策结果    │                                │
   │─────────────────────────────►│                                │
   │                              │                                │
   │  11. EventBus: 发布事件      │                                │
   │      stock.replenished        │                                │
   │─────────────────────────────►│ → 下游A6/G3监听                │
```

---

## 12. 现有设施映射

| 现有代码                              | AIOS 内核位置              | 改造策略                                |
|-------------------------------------|---------------------------|----------------------------------------|
| `internal/platform/eventbus/`       | `internal/aios/ipc/`      | 包装EventBus作为IPC底层                  |
| `internal/platform/command/`        | `internal/aios/toolregistry/` | Tool Handler 包装 Command Handler   |
| `internal/platform/scheduler/`      | `internal/aios/runtime/`  | Scheduler触发tick，Runtime接管生命周期    |
| `internal/platform/toolbridge/`     | `internal/aios/toolregistry/` | Plugin Driver 作为 Tool 的一种实现    |
| `internal/ai/`                      | `internal/aios/llmgateway/` | 抽取 Provider 接口，Gateway 做编排层   |
| `domain/entropy/`                   | `internal/aios/guardrails/` | Entropy 作为 Guardrails 规则源         |
| `domain/evolution/`                 | `internal/aios/evolution/` | 扩展为 A/B 实验框架                      |
| `domain/trustscore/`                | `internal/aios/runtime/`  | TrustScore 作为 Runtime 准入条件         |
| `domain/actionpolicy/`              | `internal/aios/guardrails/` | ActionPolicy 作为 Guardrails 策略层    |

---

## 13. 实现优先级

### P0 (高优先级 — 地基模块)

| 模块 | 最短路径 | 预估工时 |
|------|---------|---------|
| `internal/aios/llmgateway/` | 抽取 Provider 接口 → 当前 OpenAI 实现 → 重试/退避 → Fallback | 1-2 天 |
| `internal/aios/pipeline/` | 7 阶段核心 → 先在现有 Agent 执行流程上包一层 | 1-2 天 |
| `internal/aios/toolregistry/` | 包装现有 command.Dispatcher + ToolBridge → 补 Tool Schema | 1 天 |

### P1 (中优先级 — 核心能力)

| 模块 | 最短路径 | 预估工时 |
|------|---------|---------|
| `internal/aios/runtime/` | 生命周期 + 并发锁 + 健康检查 | 2 天 |
| `internal/aios/guardrails/` | 策略规则引擎 → 接入 entropy/actionpolicy | 2-3 天 |
| `internal/aios/observability/` | Agent 级 Metrics + 决策 Tracing | 1-2 天 |

### P2 (标准优先级)

| 模块 | 最短路径 | 预估工时 |
|------|---------|---------|
| `internal/aios/memory/` | 短记忆(Redis)先 → 长记忆(PostgreSQL+向量)后 | 2-3 天 |
| `internal/aios/ipc/` | 基于 EventBus 加同步 Request 封装 | 1 天 |

### P3 (低优先级 — 增强能力)

| 模块 | 最短路径 | 预估工时 |
|------|---------|---------|
| `internal/aios/knowledge/` | 文档索引 → 自动知识抽取 | 2-3 天 |
| `internal/aios/evolution/` | 扩展现有 domain/evolution → A/B实验 | 2-3 天 |
| `internal/aios/sdk/` | Agent 声明式 manifest 定义 + 注册 | 2 天 |

### 实施建议

1. **Phase 1 (P0)** — LLM Gateway + Pipeline + Tool Registry 三件套，1 周
2. **Phase 2 (P1)** — Runtime + Guardrails + Observability，1-1.5 周
3. **Phase 3 (P2)** — Memory + IPC，1 周
4. **Phase 4 (P3)** — Knowledge Engine + Evolution + SDK，1.5 周

总计约 **4-5 周** 完成 AIOS 内核完整建设。

---

## 14. 核心架构决策

| 决策 | 选择 | 替代方案 | 理由 |
|------|------|---------|------|
| AIOS 内核放 `internal/aios/` | 独立包 | 放在 domain/ | 不属于业务领域，是基础设施层 |
| Go interface 定义 | Go 原生 | proto/gRPC | 进程内调用，无需序列化 |
| Tool 是 Agent 操作外部世界的唯一通道 | 严格 | Agent 可直接调 service | 审计+权限+可观测性集中管控 |
| Pipeline 7 阶段 | 异步编排 | 直接同步执行 | 需要审批、审计、补偿，每步可暂停 |
| Memory 分三层 | 短/长/共享 | 单一存储 | 不同场景的访问模式不同 |
| 基于现有 EventBus 建 IPC | 包装 | 自建消息系统 | 复用已有基础设施 |
| SAGA 补偿 | 正向执行+反向补偿 | 强事务(2PC) | 跨服务无法用事务，最终一致性足够 |
| DecisionLock TTL | 5 分钟超时释放 | 手动释放 | 防死锁 |

---

## 附录: 设计自检

| 检查项 | 状态 |
|--------|------|
| 组件层次定义 | L1-L9 全部覆盖 |
| 模块接口契约 | 11 个 AIOS 模块全部定义接口+数据模型 |
| 核心数据流 | 端到端 Agent 决策时序完整画出 |
| 错误恢复 | 7 种故障×恢复方案 |
| Agent 编排 | Chain / Fan-out / Gather 三种模式 |
| 状态机 | 7 阶段决策全生命周期 |
| 并发控制 | DecisionLock + TTL |
| 治理文档映射 | 全部对齐平台宪法 + Kernel 契约 |
| 实现优先级 | P0-P3 分级，最短路径标注 |
