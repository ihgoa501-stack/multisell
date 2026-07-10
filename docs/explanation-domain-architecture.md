# 领域模块架构

> 凌镜 60+ 领域模块如何组织，它们之间如何协作。
> 最后更新: 2026-07-09

---

## 为什么需要领域模块

当系统有 60+ 模块时，需要一致的组织方式。

凌镜的领域模块集中在 `backend-go/internal/domain/` 下，每个模块对应一个独立的业务能力。模块之间通过 **EventBus 事件** 和 **API 调用** 通信，不直接依赖于彼此的内部实现。

---

## 模块结构规范

每个领域模块通常遵循四文件模式：

```
internal/domain/{module}/
├── routes.go    → Gin 路由注册
├── handler.go   → HTTP 请求/响应映射
├── service.go   → 业务逻辑
└── model.go     → GORM 模型 + 请求/响应 DTO
```

例外：
- 简单 CRUD 模块（brand, category）可能将 routes + handler 合并
- 复杂模块（logistics, aftersales）可能拆成多个文件（`_ops.go`, `_mgmt.go`）

### 标准响应信封

所有 handler 使用统一的响应格式：

```go
response.Success(c, data)                           // 200 OK
response.Error(c, http.StatusBadRequest, msg)        // 400
response.Paginated(c, data, total, page, size)       // 分页
response.InternalError(c, err)                       // 500
```

### 测试模式

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})
    svc := NewService(db, logger)
}
```

`dbtest` 使用内存 SQLite，每次测试自动建表、自动清理。不需要外部数据库。

---

## 模块分类体系

```
internal/domain/
├── 商品与供应链     → sku, category, brand, price, inventory, supplier, purchase
│                    → supplychain, supplyevent, tariff
│                    → candidate, completeness, profit, cost, landedcost
│                    → productanalysis, producthub, consolidation
│
├── 平台与发布       → platform, integrations, listing, listingtask, loop
│
├── 订单与物流       → order, orderimport, aftersales
│                    → shipping, logistics, platformfee
│
├── 财务与经营       → finance, settlement, decision, allocation
│                    → report, exchangerate
│
├── 运营支撑         → dashboard, search, notification, exceptions
│                    → importbatch, operationlog
│                    → imagegen, sourcing, mock, owner, support
│
├── AI 治理          → agentrule, agentlearning, actionpolicy
│                    → entropy, evolution, trustscore
│                    → metabolism, sentiment, personalrule
│                    → orchestration, approval
│
└── 施工中           → content (多语言本地化), compliance, competitor
                      → reliability, workflow
```

---

## 模块间的协作模式

### 1. 同步 API 调用

A 模块的 service 调用 B 模块的 service（通过依赖注入）：

```go
type ServiceA struct {
    svcB *ServiceB
}

func NewServiceA(db, logger, svcB) *ServiceA { ... }
```

典型场景：
- `loop` → 调用 `candidate` + `completeness` + `profit` + `listingtask`
- `producthub` → 调用 `sku` + `price` + `inventory`

### 2. EventBus 事件驱动

模块间用事件解耦：

```go
bus.Publish(ctx, "inventory.low", "inventory", map[string]any{"sku_id": 123})
// → 下游: stock_alert 订阅者 → A5 Agent 决策
```

当前 ~15 个 EventBus 订阅。事件主题命名规则：

```
<domain>.<action>           → "order.created", "inventory.low"
agent.decided.<agent_id>     → "agent.decided.A5"
agent.decided.<decision>     → "agent.decided.stock_alert"
scheduler.tick.<agent_id>    → "scheduler.tick.A5"
```

### 3. Scheduler 定时触发

Scheduler 每 N 分钟发布 tick 事件，Agent 借此触发定时检查：

```go
// 注册: 每5分钟触发 A5
scheduler.Add("A5", 5*time.Minute, func(ctx) {
    bus.Publish(ctx, "scheduler.tick.A5", "scheduler", nil)
})
```

当前定时 Agent 调度间隔：

| Agent | 间隔 | 决策点 |
|-------|------|--------|
| G0 | 5min | system_health |
| A4 / A5 | 5min / 15min | auto_reply / stock_alert |
| G1 / G3 | 5min / 30min | dashboard_overview / discount_risk |
| A6 / A3 | 1h / 1h | profit_watch / acos_analysis |
| A7 / A8 | 2h / 1h | compliance_check / sourcing_scan |
| G2 / M1 | 1h / 1h | warehouse_routing / excretion_scoring |

### 4. Command Dispatcher 动作执行

Agent 决策产生的 Action 通过 Command Dispatcher 执行：

```go
cmd.Register("stock_alert", handler)
cmd.Dispatch(ctx, "stock_alert", params)
```

已注册的命令：`stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`

### 5. ToolBridge 外部工具桥接

Agent 通过 ToolBridge 调用 Chrome 扩展等外部工具：

```go
bridge.FetchPage(ctx, url)  // → Chrome 扩展采集页面数据 → 返回 PageData
```

ToolBridge 支持三种驱动模式：
- **Chrome 扩展** — WebSocket 连接浏览器扩展，采集实时页面数据
- **Mock** — 测试用固定数据
- **AI** — 通过 LLM 解析页面描述

---

## 三层基础设施

模块依赖自下而上的三层基础设施：

```
┌─────────────────────────────────────┐
│  领域业务模块 (60+)                  │
│  sku, order, listing, finance, ... │
├─────────────────────────────────────┤
│  平台基础设施 (4个)                   │
│  EventBus / Scheduler / Command  /  │
│  ToolBridge                         │
├─────────────────────────────────────┤
│  AIOS 内核 (施工中, 11个模块)        │
│  ToolRegistry / Runtime / LLM       │
│  Gateway / Guardrails / IPC / ...   │
└─────────────────────────────────────┘
```

---

## 关键模块依赖关系图

以**商品上架**流程为例：

```
candidate
  └→ completeness ─→ profit ─→ loop ─→ decision ─→ listingtask
                       │
                       ├→ landedcost (海运/空运到岸成本)
                       ├→ tariff (关税规则)
                       ├→ platformfee (平台费率)
                       └→ logistics (物流费率)
```

以**订单履约**流程为例：

```
order ─→ shipping ─→ logistics ─→ settlement ─→ finance
           │                                    │
           ├→ provider (承运商)                 ├→ allocation (成本分摊)
           ├→ channel (物流渠道)                 ├→ exchangerate (汇率)
           └→ zone (区域)                       └→ decision (经营决策)
```

---

## 各维独立管理

以下模块独立于业务闭环，不参与复杂编排：

| 模块 | 职责 |
|------|------|
| `brand` | 品牌增删改查 |
| `category` | 无限级分类树（CRUD） |
| `supplier` | 供应商档案、商品-供应商绑定、供应商对比 |
| `search` | 全局搜索（/search + /search/recent） |
| `operationlog` | 操作审计日志（写操作自动记录） |
| `importbatch` | CSV/Excel 批量导入记录 + 异步处理 |
| `imagegen` | AI 商品图生成（Prism 服务桥接） |

---

## 相关文档

- [模块目录](reference-module-catalog.md) — 完整的模块、路由和页面清单
- [Two Business Loops](explanation-business-loops.md) — 两个核心业务闭环详解
- [How-to: 添加新领域模块](howto-add-domain-module.md) — 分步指南
- [系统架构设计 v1](system-architecture-design-v1.md) — 九层架构总览
- [AIOS 架构](aios-architecture.md) — AIOS 内核模块设计
