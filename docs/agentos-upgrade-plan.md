# LingMirror AI AgentOS — 全面升级规划

> 版本：v1.0 | 日期：2026-06-24 | 作者：Architect

---

## 目录

1. [现状评估：够用但脆弱](#1-现状评估够用但脆弱)
2. [架构缺口全景](#2-架构缺口全景)
3. [底层基础建设（P0）](#3-底层基础建设p0)
4. [新Agent设计（P0-P1）](#4-新agent设计p0-p1)
5. [基础设施补全（P1-P2）](#5-基础设施补全p1-p2)
6. [数据库迁移清单](#6-数据库迁移清单)
7. [执行路线图](#7-执行路线图)
8. [附录：与Python Hermes功能对比](#8-附录与python-hermes功能对比)

---

## 1. 现状评估：够用但脆弱

### 1.1 Go版已有能力（完整）

| 组件 | 状态 | 说明 |
|------|------|------|
| Agent框架 | ✅ 10个Agent已全部移植 | A1-A7 + G1-G3，含真实业务逻辑 |
| Orchestrator | ✅ 完整 | LLM集成为核心路径，含trace/action联动 |
| Trace系统 | ✅ 完整 | `ai_trace` / `ai_trace_event` / `ai_evidence_ref` |
| Action生命周期 | ✅ 完整 | `suggested → approved → executed → reviewed` |
| 信任分系统 | ✅ 完整 | 40%采纳率+30%执行成功+30%置信度四级阈值 |
| AgentOS驾驶舱 | ✅ 完整 | 小队概览、工作队列、自主度配置 |
| WebSocket实时 | ✅ 完整 | 动作广播 + SSE流式 |
| 前端Action Center | ✅ 完整 | 审批/拒绝/执行，风险过滤 |

### 1.2 冰山以下

Go版完成的是 **触达用户的表层**（10个Agent跑起来了，Action能创建了），但核心基础设施有 **三大结构性缺口**：

```
                    ┌──────────────────────┐
                    │   10 Agents 跑通了    │  ← 水面以上（完成）
                    │  Trace/Action/Trust   │
                    └──────┬───────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────┴────┐    ┌──────┴──────┐    ┌─────┴────┐
    │ 无事件总线 │    │ 无命令执行器 │    │  无调度器  │  ← 水面以下（缺失）
    │ Agent孤岛 │    │ Action不落地 │    │ 无人触发  │
    └─────────┘    └─────────────┘    └──────────┘
    ┌─────────┐    ┌─────────────┐    ┌──────────┐
    │ 无策略引擎 │    │ 无进化/Nudge │    │ 无熵系统  │
    │ 决策靠代码 │    │ 不会自主进化  │    │ 规则腐烂  │
    └─────────┘    └─────────────┘    └──────────┘
```

---

## 2. 架构缺口全景

### 2.1 基础设施缺口

| # | 缺口 | 严重度 | 当前后果 |
|---|------|--------|---------|
| **I1** | **事件总线** | 🔴 P0 | Agent之间无法通信；无法响应业务事件；链式协作（A5→G3→A6→A2）断裂 |
| **I2** | **命令执行器** | 🔴 P0 | Action审批通过后**不执行任何业务逻辑**，只改状态，Agent成了「建议箱」 |
| **I3** | **定时调度器** | 🔴 P0 | 没有周期性Agent执行；信任分从不自动重算；熵防御从不运行 |
| I4 | 策略引擎 | 🟠 P1 | `actionpolicy` 目录不存在，自动化审批逻辑嵌入在 orchestrator 的 if-else 中 |
| I5 | Agent协作管线 | 🟠 P1 | Python的 `pipeline.py`（链式触发）未移植，链深度限制无法配置 |
| I6 | 个人规则系统 | 🟠 P1 | 用户无法为Agent设置 veto/threshold/strategy 规则 |
| I7 | 进化+Nudge | 🟡 P2 | 信任分算出来了但没有Nudge提示升级；缺少prompt进化机制 |
| I8 | 熵防御系统 | 🟡 P2 | 规则过期、规则冲突、SPC控制、回归分析全都没有 |
| I9 | 多租户SLA | 🟡 P2 | SLA检测是硬编码1小时，没有升级流程 |

### 2.2 Agent覆盖缺口

| # | 缺口 | 严重度 | 当前后果 |
|---|------|--------|---------|
| **A8** | **结算对账** | 🔴 P0 | settlement差异无人发现；回款异常靠人工；资金链盲区 |
| **G0** | **协调仲裁** | 🔴 P0 | Agent之间无协调者；突发事件（汇率+物流+封号并发）无响应；跨Squad协作靠脆弱链式规则 |
| A11 | 售后管理 | 🟠 P1 | 退货/退款/A-to-Z claim无人主动管理；A4只覆盖咨询FAQ |
| A10 | 物流优化 | 🟠 P1 | 承运商比价/运费审计/物流商绩效无人管理 |
| A9 | 采购供应链 | 🟡 P2 | 1688 sourcing/供应商管理/采购单无Agent覆盖 |

### 2.3 顶层视图

```
当前架构（10 Agents）：
┌─────────────────────────────────────────────────────────┐
│ autonomous Squad               governance Squad         │
│  A1 选品 → A2 优化 → A3 广告     G1 仪表盘              │
│                     ↘           G2 仓储海关              │
│                       A4 客服    G3 风控                 │
│  A5 库存 + A6 利润 + A7 合规                             │
└─────────────────────────────────────────────────────────┘
                          ↕ 无协作（链式规则深度=1）
                    业务事件 → 无人响应（无Event Bus）

目标架构（15 Agents + 完整基础设施）：
┌──────────────────────────────────────────────────────────────┐
│ growth Squad          fulfillment Squad   risk Squad         │
│  A1 选品 → A2 优化     A5 库存 + A10 物流   A6 利润 + A7 合规 │
│  → A3 广告 → A4 客服   G2 仓储海关          G3 风控 + G1 仪表盘│
│                                                             │
│ settle Squad          governance Squad                       │
│  A8 结算对账 + A11 售后  G0 协调仲裁                          │
└──────────────────────────┬───────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              │     基础设施层            │
              │  EventBus + Scheduler    │
              │  Command + Policy Engine │
              │  Evolution + Entropy     │
              └─────────────────────────┘
```

---

## 3. 底层基础建设（P0）

### 3.1 事件总线 — `internal/platform/eventbus/`

**目标**：Agent之间、业务模块与Agent之间的异步解耦通信。

#### 设计

```
┌──────────────┐    Publish(topic)   ┌─────────────────┐
│业务模块        │ ─────────────────→  │   Event Bus      │
│(order,sku,    │                    │                  │
│ inventory)    │                    │  topic → []Handler│
└──────────────┘                    └────────┬──────────┘
                                              │ dispatch
┌──────────────┐    Publish(topic)           ▼
│ Agent         │ ─────────────────→  ┌──────────────┐
│ (decide完)   │                     │ A5 stock_alert│
└──────────────┘                     │ A4 auto_reply │
                                     │ G0 system_health
┌──────────────┐    Publish(topic)   └──────────────┘
│ Command       │ ─────────────────→
│ Handler完成   │
└──────────────┘
```

**核心类型**：

```go
package eventbus

type Event struct {
    ID        string                 // uuid
    Topic     string                 // "order.created", "inventory.stock_low"
    Source    string                 // "orderimport", "agent:a5"
    Payload   map[string]interface{}
    Priority  int                    // 0=normal, 1=high, 2=critical
    CreatedAt time.Time
}

type Handler func(ctx context.Context, event Event) error

type Bus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(topic string, handler Handler) (subscriptionID string)
    Unsubscribe(subscriptionID string)
    Close() error
}
```

**实施要点**：
- **内存通道**优先，可选DB持久化（`event_outbox`表，outbox pattern）
- **Topic模式**：支持 glob 匹配（`order.*` 匹配 `order.created`, `order.refund`）
- **订阅注册表**：应用启动时由各模块注册
- **与Scheduler关系**：Scheduler 定期发布 `scheduler.tick.{agent_id}` 事件

**话题定义**：

| 事件模式 | 订阅者 | 说明 |
|---------|--------|------|
| `order.*` | A4, G0 | 订单事件 |
| `inventory.*` | A5, A10 | 库存事件 |
| `price.*` | G3, A6 | 价格变更 |
| `finance.*` | A6, G0, A8 | 财务异常 |
| `settlement.*` | A8, A6 | 结算事件 |
| `compliance.*` | A7, G0 | 合规告警 |
| `scheduler.tick.*` | 各Agent | 定时触发 |
| `agent.decided.*` | Pipeline链 | Agent决策完成通知 |
| `action.executed.*` | 各Agent | Action执行完成通知 |

**迁移文件**：`000007_event_outbox`

```sql
CREATE TABLE event_outbox (
    id BIGSERIAL PRIMARY KEY,
    topic VARCHAR(100) NOT NULL,
    source VARCHAR(50) NOT NULL,
    payload JSONB,
    priority INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now(),
    delivered_at TIMESTAMPTZ
);
CREATE INDEX idx_event_outbox_status ON event_outbox(status, created_at);
```

---

### 3.2 命令执行器 — `internal/platform/command/`

**目标**：Agent创建的Action审批通过后，**实际执行业务逻辑**，而不是仅仅改变状态。

**现状**：

```
UnifiedAction (status=approved)
    → ExecuteAction()
    → db.Update(status="executed")   // 不调用任何业务逻辑！
    → return
```

**目标**：

```
UnifiedAction (status=approved)
    → ExecuteAction()
    → CommandDispatcher.Dispatch(actionType, payload)
    → 匹配到 Handler
    → 调用真实业务服务
    → 记录 before_snapshot / after_snapshot
    → db.Update(status="executed", after_snapshot)
    → eventBus.Publish("action.executed.{type}")
    → return
```

**核心类型**：

```go
package command

type Handler func(ctx context.Context, input map[string]interface{}) (*Result, error)

type Result struct {
    Success       bool
    AfterSnapshot map[string]interface{}
    BusinessID    string
    Error         string
}

type Dispatcher struct {
    handlers map[string]Handler
}

func NewDispatcher() *Dispatcher
func (d *Dispatcher) Register(actionType string, handler Handler)
func (d *Dispatcher) Dispatch(ctx context.Context, actionType string, payload map[string]interface{}) (*Result, error)
```

**初始5个Handler（Phase 1）**：

| Action Type | Handler | 目标服务 |
|-------------|---------|---------|
| `stock_alert` | CreateAlert | `notification.Service.CreateAlert` |
| `replenish` | InventoryReplenish | `inventory.Service.ReplenishOrder` |
| `price_review` | PriceAdjust | `price.Service.AdjustPrice` |
| `listing_optimize` | ListingDraft | `listing.Service.CreateDraft` |
| `compliance_check` | FlagProduct | `compliance.Service.FlagNonCompliant` |

---

### 3.3 定时调度器 — `internal/platform/scheduler/`

**核心类型**：

```go
package scheduler

type Task struct {
    ID            string
    AgentID       string
    DecisionPoint string
    Interval      time.Duration
    Squad         string
}

type Scheduler struct {
    bus    *eventbus.Bus
    tasks  []Task
    logger *zap.Logger
}

func New(bus *eventbus.Bus, logger *zap.Logger) *Scheduler
func (s *Scheduler) Register(task Task)
func (s *Scheduler) Start(ctx context.Context) error
func (s *Scheduler) Shutdown() error
```

**调度表**：

| Agent | 决策点 | 间隔 | 说明 |
|-------|--------|------|------|
| G0 | `system_health` | 5min | 协调仲裁：系统健康检查 |
| G1 | `dashboard_overview` | 5min | 驾驶舱聚合 |
| A4 | `auto_reply` | 5min | 客服待处理消息 |
| A5 | `stock_alert` | 15min | 库存检查 |
| A6 | `profit_watch` | 1hr | 利润看护 |
| A3 | `acos_analysis` | 1hr | 广告分析 |
| G3 | `discount_risk_check` | 30min | 折扣风控扫描 |
| G2 | `warehouse_routing` | 1hr | 仓储报关 |
| A7 | `compliance_check` | 2hr | 合规检测 |
| A8 | `settlement_reconcile` | 4hr | 结算对账 |
| A10 | `logistics_audit` | 6hr | 物流审计 |
| TrustScore | `recalculate` | 1hr | 信任分重算 |
| Entropy | `defense_cycle` | 6hr | 熵防御周期 |

---

### 3.4 基础设施集成图

```
┌────────────────────────────────────────────────────────────┐
│ 应用启动                                                     │
│  main.go                                                    │
│   ├─ Init DB (GORM)                                         │
│   ├─ Init EventBus ─────────────────────────────────────┐  │
│   ├─ Init Command Dispatcher ── register 5 handlers ──┐ │  │
│   ├─ Init Scheduler ─── register 10+ tasks ───────┐   │ │  │
│   ├─ Init Orchestrator (Provider, Registry, Impls)  │   │ │  │
│   └─ Init HTTP Router                               │   │ │  │
│                                                      │   │ │  │
│   Scheduler (goroutines)                             │   │ │  │
│    ├─ taskLoop(A5, 15m) ─── tick ───────────────────┼───┼─┘  │
│    ├─ taskLoop(G0, 5m)  ─── tick ───────────────────┼───┘    │
│    └─ taskLoop(...)                                  │        │
│                                                      ▼        ▼
│  ┌──────────────┐   Publish(topic)  ┌─────────────────────────┘
│  │ 业务模块      │ ───────────────→ │  Event Bus
│  │ (order,price) │                  │  topic → []Handler
│  └──────────────┘                   │
│  ┌──────────────┐   Publish         │  ┌──────────────────────┐
│  │ Agent Decide  │ ───────────────→ │  │ order.created        │
│  └──────────────┘                   │  │  → A4 auto_reply     │
│                                     │  │  → G0 system_health  │
│  ┌──────────────┐   Publish         │  ├──────────────────────┤
│  │ Command       │ ───────────────→ │  │ scheduler.tick.a5    │
│  │ Handler完成   │                  │  │  → Orchestrator.Run   │
│  └──────────────┘                   │  └──────────────────────┘
└─────────────────────────────────────┴──────────────────────────┘
                                                │ dispatch
                                        ┌───────▼──────────┐
                                        │ Orchestrator.Run  │
                                        └───────┬──────────┘
                                                │
                                        ┌───────▼──────────┐
                                        │ Agent.Decide()    │
                                        └───────┬──────────┘
                                                │
                                        ┌───────▼──────────────┐
                                        │ Create UnifiedAction  │
                                        └───────┬──────────────┘
                                                │ 用户审批通过
                                        ┌───────▼──────────────┐
                                        │  Command.Dispatch()   │
                                        │  → 真实业务逻辑        │
                                        │  → after_snapshot      │
                                        │  → emit executed event │
                                        └───────────────────────┘
```

---

## 4. 新Agent设计（P0-P1）

### 4.1 G0 — 协调仲裁 Agent（Supervisor）

**角色**：整个Agent生态的「大脑」—— 监控异常、协调跨Squad协作、仲裁冲突。

| 属性 | 值 |
|------|-----|
| ID | G0 |
| 名称 | 协调仲裁 Agent |
| Squad | governance |
| 默认自主度 | supervised |
| 风险基线 | high |
| Model | gpt-4o |

**决策点**：

| 决策点 | 输入 | 输出 | 触发方式 |
|--------|------|------|---------|
| `system_health` | 所有Agent信任分、pending action数、异常追踪、SLA | 健康评分、预警列表 | Scheduler 5min |
| `anomaly_escalation` | 业务事件（汇率暴跌、物流大面积延误、平台封号） | 升级建议、协调指令 | Event Bus |
| `cross_squad_coordinate` | 跨Squad请求（库存告急+利润告警+物流涨价同时发生） | 优先级排序、资源分配建议 | Event Bus |
| `agent_audit` | Agent决策历史、信任分趋势、用户反馈 | Agent健康评估、回滚建议 | Scheduler 24h |

**G0与G1的区别**：
- G1 仪表盘：**被动展示** — 聚合数据给用户看（OBSERVATION级别）
- G0 协调仲裁：**主动决策** — 检测到异常后给出协调指令（SUPERVISED级别），能创建需审批的Action

---

### 4.2 A8 — 结算对账 Agent

| 属性 | 值 |
|------|-----|
| ID | A8 |
| 名称 | 结算对账 Agent |
| Squad | settle（新增） |
| 默认自主度 | supervised |
| 风险基线 | critical |
| Model | gpt-4o |

**决策点**：

| 决策点 | 输入 | 输出 | 触发方式 |
|--------|------|------|---------|
| `settlement_import` | 平台结算文件（CSV/API） | 导入结果、异常记录 | Event Bus: `platform.settlement_ready` |
| `reconciliation_check` | 平台结算 vs 系统订单 vs 银行到账 | 差异报告、差异分类 | Scheduler 4h |
| `discrepancy_resolve` | 差异记录、历史处理模式 | 处理建议 | 链式: reconciliation完成 |
| `cash_flow_watch` | 各平台回款进度、汇率损益 | 资金预警、回款预测 | Scheduler 24h |

---

### 4.3 A10 — 物流优化 Agent（P1）

| 属性 | 值 |
|------|-----|
| ID | A10 |
| 名称 | 物流优化 Agent |
| Squad | fulfillment |
| 默认自主度 | supervised |
| 风险基线 | medium |
| Model | gpt-4o-mini |

**决策点**：`carrier_compare`、`shipping_bill_audit`、`carrier_performance`

### 4.4 A11 — 售后管理 Agent（P1）

| 属性 | 值 |
|------|-----|
| ID | A11 |
| 名称 | 售后管理 Agent |
| Squad | settle |
| 默认自主度 | supervised |
| 风险基线 | medium |
| Model | gpt-4o |

**决策点**：`return_analysis`、`refund_decision`、`dispute_manage`

### 4.5 A9 — 采购供应链 Agent（P2 / Q4）

| 属性 | 值 |
|------|-----|
| ID | A9 |
| 名称 | 采购供应链 Agent |
| Squad | supply（Q4新增） |
| 默认自主度 | supervised |
| 风险基线 | high |

**决策点**：`sourcing_recommend`、`supplier_evaluate`、`purchase_plan`

### 4.6 Squad重构

| Squad | Agent | 领域 |
|-------|-------|------|
| growth | A1, A2, A3, A4 | 商品增长 |
| fulfillment | A5, A10, G2 | 订单履约+物流 |
| risk | A6, A7, G3, G1 | 风控+仪表盘 |
| settle | A8, A11 | 结算+售后 |
| governance | G0 | 协调仲裁 |

---

## 5. 基础设施补全（P1-P2）

### 5.1 策略引擎 — `internal/domain/actionpolicy/`

```go
type ApprovalPolicy struct {
    ID            int64
    Name          string
    SquadID       string
    AgentID       string
    ActionType    string
    RiskLevel     string
    ConditionExpr string
    Effect        string    // auto_approve | escalate | block
    Priority      int
    Enabled       bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**迁移文件**：`000008_approval_policy`

### 5.2 个人规则 — `internal/domain/agentrule/`

**迁移文件**：`000009_personal_rule`

### 5.3 Agent协作管线 — Event Bus扩展

```go
eventBus.Subscribe("agent.decided.a5.stock_alert", func(ctx, event) {
    output := event.Payload
    if output["stock_status"] == "red" {
        eventBus.Publish(ctx, Event{
            Topic: "agent.trigger.g3.discount_risk_check",
            Payload: output,
        })
    }
})
```

### 5.4 进化+Nudge — `internal/domain/evolution/`
### 5.5 熵防御 — `internal/domain/entropy/`

---

## 6. 数据库迁移清单

| 文件 | 内容 | 优先级 |
|------|------|--------|
| 000004_approval_policy | ✅ 已有（确认是否落地） | - |
| 000005_trust_score | ✅ 已有 | - |
| 000006_pre_listing_decision | ✅ 已有 | - |
| **000007_event_outbox** | `event_outbox` 表 | P0 |
| **000008_approval_policy** | `approval_policy` 表 | P1 |
| **000009_personal_rule** | `personal_rule` 表 | P1 |
| **000010_agent_nudge** | `agent_nudge` 表 | P2 |
| **000011_entropy** | `rule_conflict`, `spc_control_limit` | P2 |
| **000012_squad_refactor** | `squad_id`更新 | P1 |

---

## 7. 执行路线图

### Phase 1：基础设施打底 + G0（P0 — 2周）

```
Week 1                     Week 2
├── Event Bus               ├── Command Executor
│   Pub/Sub 实现            │   5个核心Handler
│   Topic 定义              │   Orchestrator集成
│   Outbox 表               │   before/after snapshot
│   订阅注册表               ├── Scheduler
└──                         │   10个task注册
                            │   tick → Orchestrator
                            ├── G0 实现
                            │   system_health
                            │   anomaly_escalation
                            │   cross_squad_coordinate
                            └── Registry + API
```

**检查清单**：
- [ ] `internal/platform/eventbus/` — 事件总线 + outbox表
- [ ] `internal/platform/command/` — 命令执行器 + 5个Handler
- [ ] `internal/platform/scheduler/` — 定时调度器
- [ ] `internal/agent/impl/coordinator.go` — G0
- [ ] 迁移：000007_event_outbox
- [ ] Agent通过Event Bus + Scheduler自动触发
- [ ] Action执行后真正调用业务逻辑

### Phase 2：新Agent + Pipeline（P1 — 2周）

```
Week 3                     Week 4
├── A8 结算对账             ├── A11 售后管理
│   settlement_import      │   return_analysis
│   reconciliation         │   refund_decision
│   discrepancy_resolve    │   dispute_manage
├── A10 物流优化            ├── Pipeline链式
│   carrier_compare        │   A5→G3→A6→A2
│   bill_audit             │   A8→A6
│   carrier_performance    │   G0→G1
├── Squad名重构             ├── 策略引擎(基础)
└──                        └── Personal Rules(基础)
```

**检查清单**：
- [ ] `internal/agent/impl/settlement.go` — A8
- [ ] `internal/agent/impl/logistics.go` — A10
- [ ] `internal/agent/impl/aftersales.go` — A11
- [ ] Event Bus链式注册（5条管道规则）
- [ ] Policy Engine + API CRUD
- [ ] 迁移：000008, 000009

### Phase 3：进化 + 熵（P2 — 2周）

```
Week 5                     Week 6
├── Nudge系统               ├── 熵防御系统
│   升级阈值计算            │   TTL Sweeper
│   Nudge推送              │   Budget Enforcer
│   用户操作处理            │   Merge Detector
├── 策略引擎完整             │   Regret Analyzer
│   条件表达式引擎           │   SPC Control
│   前端策略管理页           ├── SLA监控
├── Personal Rules完整      ├── 信任分看板前端
└──                        └── 升级推荐前端UI
```

**检查清单**：
- [ ] `internal/domain/evolution/` — Nudge
- [ ] `internal/domain/entropy/` — 熵防御
- [ ] Policy Engine完整（表达式引擎）
- [ ] SLA监控 + 升级流程
- [ ] 迁移：000010, 000011

### Phase 4：A9 + 完全自主（Q4）

- [ ] A9 采购供应链 Agent
- [ ] 1688集成对接
- [ ] FULL_AUTONOMOUS (L4) 扩展
- [ ] 多Agent对话推理
- [ ] Python归档

---

## 8. 附录：与Python Hermes功能对比

| 功能 | Python Hermes | Go当前 | Go目标 | 优先级 |
|------|:---:|:---:|:---:|:------:|
| Agent注册 | ✅ | ✅ | ✅ | - |
| 10个Agent实现 | ✅ | ✅ | ✅ | - |
| Orchestrator | ✅ | ✅ | ✅ | - |
| LLM Provider | ✅ | ✅ | ✅ | - |
| Trace系统 | ✅ | ✅ | ✅ | - |
| Action生命周期 | ✅ | ✅ | ✅ | - |
| Trust Score | ✅ | ✅ | ✅ | - |
| Event Bus | ✅ | ❌ | ✅ | P0 |
| Scheduler | ✅ | ❌ | ✅ | P0 |
| Command Handlers | ✅ | ❌ | ✅ | P0 |
| Pipeline/Chains | ✅ | ❌ | ✅ | P1 |
| Action Policy Engine | ❌ | ❌ | ✅ | P1（新） |
| Personal Rules | ✅ | ❌ | ✅ | P1 |
| Evolution Service | ✅ | ❌ | ✅ | P2 |
| Nudge Mechanism | ✅ | ❌ | ✅ | P2 |
| Entropy System | ✅ | ❌ | ✅ | P2 |
| SPC Control | ✅ | ❌ | ✅ | P2 |
| SLA Monitoring | ⚠️ | ❌(stub) | ✅ | P2 |
| G0 Coordinator | ❌ | ❌ | ✅ | P0 |
| A8 Settlement | ❌ | ❌ | ✅ | P0 |
| A11 Aftersales | ❌ | ❌ | ✅ | P1 |
| A10 Logistics | ❌ | ❌ | ✅ | P1 |
| A9 Procurement | ❌ | ❌ | ✅ | P2 |
| FULL_AUTONOMOUS(L4) | ✅ | ❌ | ✅ | Q4 |

---

## 关键决策日志

| 决策 | 选项 | 决定 | 理由 |
|------|------|------|------|
| Pipeline实现 | 独立包 vs Event Bus扩展 | **Event Bus扩展** | 避免重复调度逻辑 |
| 命令执行时机 | 同步 vs 异步 | **同步+after_snapshot** | 保持Action状态一致性 |
| 调度器实现 | robfig/cron vs time.Ticker | **time.Ticker** | Agent固定间隔，简洁可靠 |
| squad命名 | 保留 vs 业务命名 | **业务命名** | 对用户友好，与Python一致 |
| settle Squad | 合并到risk vs 独立 | **独立** | 结算和售后有独立风险特征 |
| G0风险基线 | medium vs high | **high** | 协调决策误判代价高 |

---

> **一句话总结**：10个Agent跑通了表面流程，但缺少的事件总线、命令执行器和调度器让它们实际上是五个孤岛。**Phase 1先把这三个基础设施打牢，再把G0和A8这两个P0新Agent加上**，AgentOS才算真正「能用」。Phase 2-3再把剩下的补全，到Q4达到完整自主进化AgentOS。
