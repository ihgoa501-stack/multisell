# 凌镜 LingMirror — 模块目录 (Module Catalog)

> **这是模块、路由和前端页面的唯一事实源。**
> 此文档合并了以下已废弃文档的内容：FUNCTION_INVENTORY.md、FRONTEND_PAGES_AND_ROUTING.md、api-inventory.md、PROJECT_STATUS.md（模块清单部分）。
> 添加新模块或路由时，只更新此文件即可。
>
> 更新日期: 2026-07-12
> 技术名: MultiSell

---

## 验证状态

本文件是模块、路由和页面清单的事实源，不再作为当前测试结果的事实源。

当前验证状态以 [PROJECT_STATUS.md](PROJECT_STATUS.md) 的“当前事实快照”为准。
历史测试报告保留在 [TEST_SUMMARY.md](TEST_SUMMARY.md) 和
[FRONTEND_TEST_REPORT.md](FRONTEND_TEST_REPORT.md) 中。

最后一次写入本目录的验证快照来自 2026-06-30，已降级为历史参考；
在重新运行最新检查前，不应把旧的 build/lint/test 结果当作当前事实。

---

## 总览

后端分为三大层，共 60+ 领域模块和 19 个内部包：

```
backend-go/internal/
├── domain/          → 领域业务模块（60+）
├── platform/        → 平台基础设施（4 个）
├── aios/            → AIOS 内核（建设中）
├── ai/              → LLM 编排 & Agent 运行时
├── agent/           → Agent 注册和执行
├── agentos/         → AgentOS 总控台
├── auth/            → JWT 认证
├── rbac/            → 权限角色
├── response/        → 标准响应信封
├── httpx/           → HTTP 引擎 + 中间件 + 路由
├── common/          → 通用工具
├── config/          → 配置解析
├── database/        → 数据库连接
├── dbtest/          → 测试用内存 SQLite
├── migration/       → 迁移框架
├── realtime/        → WebSocket 实时 Hub
├── prismadapter/    → Prism 图片服务客户端
└── feedback/        → 内部反馈系统
```

### 入口与运行

| 项目 | 位置 |
|------|------|
| 后端入口 | `backend-go/cmd/server/main.go` |
| 后端路由汇总 | `backend-go/internal/httpx/router.go` |
| 后端配置 | `backend-go/configs/config.yaml`，支持环境变量覆盖 |
| 前端入口 | `frontend-next/src/app/` |
| 前端 API client | `frontend-next/src/lib/api-client.ts` |
| 前端菜单配置 | `frontend-next/src/config/menu.ts` |
| Docker 默认入口 | `docker-compose.yml` |

### 网关与中间件

| 层 | 说明 |
|----|------|
| CORS | `middleware.CORS` — 全栈跨域 |
| Request ID | `middleware.RequestID` — 每个请求分配 UUID |
| Metrics | `middleware.Metrics` — Prometheus 请求计数/耗时（opt-in） |
| Recovery | `middleware.RecoveryWithSentry` — panic 恢复 + Sentry 上报 |
| Audit | `middleware.Audit` — 所有写操作记录到操作日志表 |
| Auth | `middleware.Auth` — JWT 认证（仅 `/api/v1/*` 组） |
| Rate Limit | `ratelimit.go` — 速率限制 |

---

## 1. 平台基础设施 (`internal/platform/`)

四进程内协调原语，Agent 间、Agent 系统间通信基础。

| 模块 | 文件 | 说明 |
|------|------|------|
| **EventBus** | `eventbus/bus.go` | 发布/订阅，glob 主题匹配 (`order.*`)。~15 订阅。可选 outbox 持久化。 |
| **Command** | `command/command.go` | 类型化处理器注册。命令: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check` |
| **Scheduler** | `scheduler/scheduler.go` | 周期性任务（5min-6h），发布 `scheduler.tick.{agent_id}` 事件。 |
| **ToolBridge** | `toolbridge/bridge.go` | 插件驱动工具执行桥接，Agent 通过它调用外部工具。 |

领域分类的机器可校验唯一合同位于 `backend-go/internal/domain/platformtruth/`，受保护只读 API 为 `GET /api/v1/platform-truth`，Owner 页面为 `/platform-truth`。合同覆盖当前 `internal/domain/` 全部目录；新增领域而未分类会导致测试失败。

**EventBus 核心 API**:
```go
type EventBus interface {
    Publish(ctx, topic, source string, payload Map) error
    Subscribe(topic string, handler HandlerFunc) Subscription
    Start(ctx)
    Stop()
}
```

**事件主题命名规则**:
- `<domain>.<action>`: `order.created`, `inventory.low`, `scheduler.tick.A5`
- Agent 决策链: `agent.decided.<agent_id>.<decision_point>`
- 通配符订阅: `agent.decided.**`

---

## 2. AI & Agent 层

| 包 | 说明 |
|----|------|
| `internal/ai/` | LLM 编排，chat/streaming/trace，provider 抽象（OpenAI/Anthropic/stub），Guardrails 集成，预算控制 |
| `internal/agent/` | Agent 注册表 + 执行入口 + Pipeline DAG 引擎 |
| `internal/agentos/` | AgentOS 总控台：work items，自主运营概览，SLA 升级 |
| `internal/aios/` | AIOS 内核 11 模块（施工中） |

### Agent 清单

| ID | 名称 | 决策点 | 间隔 | 说明 |
|----|------|--------|------|------|
| G0 | 系统健康 | `system_health` | 5min | 协调仲裁健康检查 |
| G1 | 看板总览 | `dashboard_overview` | 5min | 驾驶舱聚合 |
| G2 | 仓储报关 | `warehouse_routing` | 1h | 仓储报关 |
| G3 | 折扣风控 | `discount_risk_check` | 30min | 折扣风险扫描 |
| A1 | 选品调研 | sourcing_discovery | — | 选品调研 |
| A2 | 经营复盘 | `listing_optimize` | 事件驱动 | 刊登优化（链式触发） |
| A3 | 广告分析 | `acos_analysis` | 1h | ACOS 分析 |
| A4 | 客服回复 | `auto_reply` | 5min | 待处理消息检查 |
| A5 | 库存预警 | `stock_alert` | 15min | 库存检查 |
| A6 | 利润看护 | `profit_watch` | 1h | 利润监控 |
| A7 | 合规检测 | `compliance_check` | 2h | 合规检测 |
| A8 | 选品扫描 | `sourcing_scan` | 1h | 1688 选品扫描 |
| A9 | 批量运维 | — | — | 批量更新/同步/导入 |
| A10 | 物流费率 | (链式触发) | — | 跨境物流费率 |
| A11 | 售后管理 | — | — | 退货分析/退费决策/纠纷管理 |
| M1 | 代谢评分 | `excretion_scoring` | 1h | 实体排泄评分 |

### Agent 决策链

```
A5 stock_alert (red)           → G3 discount_risk_check
G3 discount_risk_check (block)  → A6 profit_watch
A6 profit_watch (loss/threshold)→ A2 listing_optimize
G0 system_health (anomaly > 3) → G1 dashboard_overview
```

所有定时 Agent：G0/A4/G1/A5/G3/A6/A3/G2/A7/M1/trustscore/entropy

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **AI** | | | |
| POST | `/api/v1/ai/chat` | ✅ | `ai/page.tsx`, `copilot/CopilotPanel.tsx` |
| POST | `/api/v1/ai/run` | ✅ | `ai/page.tsx` |
| GET | `/api/v1/ai/traces` | ✅ | `ai/page.tsx`, `agents/[id]/page.tsx` |
| GET | `/api/v1/ai/actions` | ✅ | `ai/page.tsx`, `actions/page.tsx`, `lib/actions-api.ts` |
| GET | `/api/v1/ai/agents` | ✅ | `ai/page.tsx` |
| GET | `/api/v1/ai/agents/specs` | ✅ | — |
| POST | `/api/v1/ai/actions` | ✅ | — |
| GET | `/api/v1/ai/traces/:trace_id` | ✅ | `agents/[id]/trace/[traceId]/page.tsx` |
| GET | `/api/v1/ai/actions/:id` | ✅ | `actions/[id]/page.tsx` |
| POST | `/api/v1/ai/actions/:id/approve` | ✅ | `ai/page.tsx`, `agentos/page.tsx`, `actions/page.tsx` |
| POST | `/api/v1/ai/actions/:id/reject` | ✅ | 同上 |
| POST | `/api/v1/ai/actions/:id/execute` | ✅ | 同上 |
| POST | `/api/v1/ai/actions/:id/review` | ✅ | `actions/[id]/page.tsx` |
| **Agents** | | | |
| GET/POST | `/api/v1/agents[/:id]` | ✅ | `agents/page.tsx`, `agents/[id]/page.tsx` |
| GET/POST/PUT/DELETE | `/api/v1/agents/rules[/:id]` | ✅ | — |
| POST | `/api/v1/agents/rules/apply` | ✅ | — |
| GET | `/api/v1/agents/evolution` | ✅ | — |
| GET | `/api/v1/agents/entropy` | ✅ | `agents/entropy/page.tsx` |
| POST | `/api/v1/agents/:id/actions` | ✅ | — |
| **AgentOS** | | | |
| GET | `/api/v1/agentos` | ✅ | `agentos/page.tsx` |
| GET | `/api/v1/agentos/status` | ✅ | — |
| GET | `/api/v1/agentos/work-items` | ✅ | `agentos/page.tsx`, `agentos/work-items/page.tsx` |
| GET | `/api/v1/agentos/autonomy` | ✅ | `agentos/page.tsx` |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/ai` | AI 指挥中心 | `ai/page.tsx` |
| `/agents` | Agent 列表 | `agents/page.tsx` |
| `/agents/[id]` | Agent 详情 | `agents/[id]/page.tsx` |
| `/agents/[id]/trace/[traceId]` | Trace 详情 | `agents/[id]/trace/[traceId]/page.tsx` |
| `/agents/actions` | Agent Action 中心 | `agents/actions/page.tsx` |
| `/agents/entropy` | 熵监控 | `agents/entropy/page.tsx` |
| `/agents/evolution` | 进化建议 | `agents/evolution/page.tsx` |
| `/agents/trust` | 信任与自主度 | `agents/trust/page.tsx` |
| `/actions` | 统一 Action 列表 | `actions/page.tsx` |
| `/actions/[id]` | Action 详情 | `actions/[id]/page.tsx` |
| `/agentos` | AgentOS 控制台 | `agentos/page.tsx` |
| `/agentos/work-items` | 工作队列 | `agentos/work-items/page.tsx` |

---

## 3. 商品与供应链模块

总路由前缀: `/api/v1/products`, `/api/v1/skus`, etc.

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `sku` | `/api/v1/products`, `/api/v1/skus` | 商品 CRUD、规格、SKU 笛卡尔积生成 |
| `category` | `/api/v1/categories` | 无限级分类树 |
| `brand` | `/api/v1/brands` | 品牌管理 |
| `price` | `/api/v1/prices`, `/api/v1/skus/:id/prices` | 多类型价格、批量调价、记录 |
| `inventory` | `/api/v1/inventory` | 库存更新、安全库存预警、变动记录 |
| `supplier` | `/api/v1/suppliers` | 供应商档案、商品-供应商绑定 |
| `purchase` | `/api/v1/purchase` | 采购订单 |
| `supplychain` | `/api/v1/supplychain` | 供应链编排（A8→A10联动） |
| `supplyevent` | — | 供应链事件模型 |
| `tariff` | `/api/v1/tariffs` | 关税规则 |
| `candidate` | `/api/v1/candidates` | 候选商品管理 |
| `experiment` | `/api/v1/experiments` | 待迁移的经营事实核验案卷；对象关联只支持追踪，不证明因果或反馈闭环 |
| `platformtruth` | `/api/v1/platform-truth` | 唯一方向、事实/声明等级、系统边界、对象身份、来源规则与全领域处置合同（只读） |
| `demandcase` | `/api/v1/demand-cases` | 候选市场八维证据、独立反证、确定性裁决与 Owner 六行决策卡 |
| `demandcase`（商品机会） | `/api/v1/product-opportunities` | 由最新 Owner selected 市场决定派生的商品机会、完整性检查与 Owner 批准 |
| `completeness` | `/api/v1/completeness` | 商品完整度检查 |
| `profit` | `/api/v1/profit` | 利润计算 |
| `cost` | `/api/v1/costs` | 成本分摊 |
| `landedcost` | `/api/v1/landed-costs` | 到岸成本 |
| `productanalysis` | `/api/v1/product-analysis` | 商品分析 |
| `producthub` | `/api/v1/product-hub` | 商品中心 Hub |
| `consolidation` | `/api/v1/consolidation` | 商品聚合/合并 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **经营事实核验案卷（技术路径暂保留 experiments）** | | | |
| GET/POST | `/api/v1/experiments` | ✅ | `experiments/page.tsx` |
| GET/PUT | `/api/v1/experiments/:experimentId` | ✅ | `experiments/[experimentId]/page.tsx` |
| POST | `/api/v1/experiments/:experimentId/evidence` | ✅ | `experiments/[experimentId]/page.tsx` |
| POST | `/api/v1/experiments/:experimentId/evidence/:evidenceId/verify` | ✅ | `experiments/[experimentId]/page.tsx` |
| POST | `/api/v1/experiments/:experimentId/links` | ✅ | `experiments/[experimentId]/page.tsx` |
| POST | `/api/v1/experiments/:experimentId/gates/evaluate` | ✅ | `experiments/[experimentId]/page.tsx` |
| GET | `/api/v1/experiments/:experimentId/owner-summary` | ✅ | `experiments/[experimentId]/page.tsx` |
| **候选市场案件** | | | |
| GET/POST | `/api/v1/demand-cases` | ✅ | `demand-cases/page.tsx` |
| GET | `/api/v1/demand-cases/:id` | ✅ | `demand-cases/[id]/page.tsx` |
| POST | `/api/v1/demand-cases/:id/evidence` | ✅ | Owner 页面待建 |
| POST | `/api/v1/demand-cases/:id/falsifications` | ✅ | Owner 页面待建 |
| POST | `/api/v1/demand-cases/:id/evaluate` | ✅ | Owner 页面待建 |
| GET | `/api/v1/demand-cases/:id/decision-card` | ✅ | `demand-cases/[id]/page.tsx` |
| POST | `/api/v1/demand-cases/research/import` | ✅ | 受保护 AI 研究契约入口 |
| POST | `/api/v1/demand-cases/research/first-public-batch` | ✅ | `demand-cases/page.tsx` |
| GET/POST | `/api/v1/demand-cases/:id/owner-decision[s]` | ✅ | `demand-cases/[id]/page.tsx` |
| GET/POST | `/api/v1/product-opportunities[/:id]` | ✅ | `product-opportunities/page.tsx` |
| POST | `/api/v1/product-opportunities/:id/evaluate` | ✅ | `product-opportunities/page.tsx` |
| POST | `/api/v1/product-opportunities/:id/owner-decisions` | ✅ | `product-opportunities/page.tsx` |
| **Products & SKU** | | | |
| GET/POST/PUT/DELETE | `/api/v1/products[/:id]` | ✅ | `products/page.tsx`, `products/[id]/page.tsx`, `products/create/page.tsx` |
| GET/POST/PUT/DELETE | `/api/v1/products/:id/specs[/:spec_id]` | ✅ | — |
| POST | `/api/v1/products/:id/specs/:spec_id/values` | ✅ | — |
| GET | `/api/v1/products/:id/skus` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/skus[/:id]` | ✅ | — |
| PUT | `/api/v1/spec-values/:id` | ✅ | — |
| DELETE | `/api/v1/spec-values/:id` | ✅ | — |
| GET | `/api/v1/products/:id/supplier-comparison` | ✅ | `products/[id]/suppliers/page.tsx` |
| **Pricing** | | | |
| GET/POST/PUT/DELETE | `/api/v1/prices[/:id]` | ✅ | — |
| GET | `/api/v1/skus/:id/prices` | ✅ | — |
| GET | `/api/v1/skus/:id/current-price` | ✅ | — |
| GET | `/api/v1/skus/:id/price-history` | ✅ | — |
| **Inventory** | | | |
| GET/PUT | `/api/v1/inventory[/:id]` | ✅ | — |
| POST | `/api/v1/inventory/:id/lock` | ✅ | — |
| POST | `/api/v1/inventory/:id/unlock` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/inventory/warehouses[/:id]` | ✅ | — |
| GET | `/api/v1/inventory/logs` | ✅ | — |
| GET | `/api/v1/inventory/sku/:sku_id/warehouses` | ✅ | — |
| **Supplier** | | | |
| GET/POST/PUT/DELETE | `/api/v1/suppliers[/:id]` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/product-suppliers[/:id]` | ✅ | — |
| **Categories** | | | |
| GET/POST/PUT/DELETE | `/api/v1/categories[/:id]` | ✅ | — |
| GET | `/api/v1/categories/tree` | ✅ | — |
| **Brands** | | | |
| GET/POST/PUT/DELETE | `/api/v1/brands[/:id]` | ✅ | — |
| **Candidate** | | | |
| GET/POST | `/api/v1/candidates` | ✅ | — |
| **Completeness** | | | |
| POST | `/api/v1/completeness/check/:productId` | ✅ | — |
| **Profit** | | | |
| GET | `/api/v1/profit/summary/:productId` | ✅ | — |
| **Product Analysis** | | | |
| POST | `/api/v1/product-analysis/analyze` | ✅ | — |
| GET | `/api/v1/product-analysis/analyses` | ✅ | — |
| GET | `/api/v1/product-analysis/analyses/:id` | ✅ | — |
| POST | `/api/v1/product-analysis/analyses/:id/feedback` | ✅ | — |
| **Purchase** | | | |
| GET/POST | `/api/v1/purchase/orders[/:id]` | ✅ | — |
| GET/POST | `/api/v1/purchase/suggestions` | ✅ | — |
| CRUD | `/api/v1/purchase/suppliers[/:id]` | ✅ | — |
| **Sourcing 1688** | | | |
| GET/POST/PUT/DELETE | `/api/v1/sourcing1688[/:id]` | ✅ | — |
| GET | `/api/v1/sourcing1688/summary` | ✅ | — |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/products` | 商品列表 | `products/page.tsx` |
| `/products/create` | 新建商品 | `products/create/page.tsx` |
| `/products/[id]` | 商品详情 | `products/[id]/page.tsx` |
| `/products/[id]/suppliers` | 商品供应商管理 | `products/[id]/suppliers/page.tsx` |
| `/categories` | 分类管理 | `categories/page.tsx` |
| `/brands` | 品牌管理 | `brands/page.tsx` |
| `/sku` | SKU 管理 | `sku/page.tsx` |
| `/inventory` | 库存管理 | `inventory/page.tsx` |
| `/suppliers` | 供应商管理 | `suppliers/page.tsx` |
| `/sourcing` | AI 选品 | `sourcing/page.tsx` |
| `/sourcing1688` | 1688 采购 | `sourcing1688/page.tsx` |

---

## 4. 平台与发布模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `platform` | `/api/v1/platforms` | 平台配置（Ozon/Shopee API 密钥） |
| `integrations` | `/api/v1/platform-integrations` | 平台集成适配器（PlatformAdapter 接口） |
| `listing` | `/api/v1/listings`, `/api/v1/listing` | 刊登记录 + 发布链 |
| `listingtask` | `/api/v1/listing-tasks`, `/api/v1/listing-task` | 刊登任务 + 工作台 |
| `loop` | `/api/v1/loop` | 经营循环 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **Platform** | | | |
| GET/POST/PUT/DELETE | `/api/v1/platforms[/:id]` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/stores[/:id]` | ✅ | — |
| **Platform Integrations** | | | |
| GET/POST/PUT/DELETE | `/api/v1/platform-integrations[/:id]` | ✅ | — |
| POST | `/api/v1/platform-integrations/:id/test` | ✅ | — |
| POST | `/api/v1/platform-integrations/:id/sync` | ✅ | — |
| GET/POST | `/api/v1/platform-integrations/:id/categories` | ✅ | — |
| GET/POST | `/api/v1/platform-integrations/:id/attributes` | ✅ | — |
| **Listings** | | | |
| GET/POST/PUT/DELETE | `/api/v1/listings[/:id]` | ✅ | `listings/create/page.tsx` |
| POST | `/api/v1/listings/:id/publish` | ✅ | — |
| POST | `/api/v1/listings/:id/sync` | ✅ | — |
| POST | `/api/v1/listing/products/:product_id/publish/:platform_id` | ✅ | — |
| GET | `/api/v1/listing/products/:product_id/listings` | ✅ | — |
| POST | `/api/v1/listing/listing-tasks/from-decisions` | ✅ | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/recheck` | ✅ | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/cancel` | ✅ | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/publish` | ✅ | — |
| POST | `/api/v1/listing` | ❌ | 前端调用 `POST /v1/listing` 但后端只有 `POST /v1/listings` |
| **Listing Tasks** | | | |
| GET/POST/PUT/DELETE | `/api/v1/listing-tasks[/:id]` | ✅ | `listing-tasks/[id]/page.tsx` |
| GET/POST/PUT/DELETE | `/api/v1/listing-tasks/:id/items[/:item_id]` | ✅ | — |
| POST | `/api/v1/listing-task/:task_id/execute` | ✅ | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/retry-failed` | ✅ | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/items/:item_id/retry` | ✅ | — |
| GET | `/api/v1/listing-task/stats` | ❌ | `listing-tasks/workbench/page.tsx` 调用但无 handler |
| POST | `/api/v1/listing-task/retry-all` | ❌ | `listing-tasks/workbench/page.tsx` 调用但无 handler |
| **Loop** | | | |
| POST | `/api/v1/loop/evaluate/:productId` | ✅ | — |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/platforms` | 平台管理 | `platforms/page.tsx` |
| `/platform-integrations` | 平台集成 | `platform-integrations/page.tsx` |
| `/platform-integrations/[id]/ozon-products` | Ozon 商品列表 | `platform-integrations/[id]/ozon-products/page.tsx` |
| `/listings` | 刊登管理 | `listings/page.tsx` |
| `/listings/create` | 创建刊登 | `listings/create/page.tsx` |
| `/listing-tasks` | 上架任务 | `listing-tasks/page.tsx` |
| `/listing-tasks/[id]` | 上架任务详情 | `listing-tasks/[id]/page.tsx` |
| `/listing-tasks/workbench` | 上架任务工作台 | `listing-tasks/workbench/page.tsx` |

---

## 5. 订单与物流模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `order` | `/api/v1/order` | 订单管理 |
| `orderimport` | `/api/v1/order-import` | 订单导入（CSV） |
| `aftersales` | `/api/v1/aftersales` | 售后、退货 Rate Tracker |
| `shipping` | `/api/v1/shipping` | 运费 |
| `logistics` | `/api/v1/logistics` | 物流费率引擎（A10）：四种定价模式、YAML 费率表、承运商绩效 |
| `platformfee` | `/api/v1/platform-fee` | 平台费用规则 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **Order** | | | |
| GET/POST/PUT/DELETE | `/api/v1/order[/:id]` | ✅ | `orders/[id]/page.tsx` |
| GET | `/api/v1/order/summary` | ✅ | — |
| POST | `/api/v1/order/:id/status` | ✅ | — |
| **Order Import** | | | |
| GET/POST/PUT/DELETE | `/api/v1/order-import[/:id]` | ✅ | — |
| GET | `/api/v1/order-import/summary` | ✅ | — |
| POST | `/api/v1/order-import/:id/process` | ✅ | — |
| POST | `/api/v1/order-import/:id/complete` | ✅ | — |
| **Aftersales** | | | |
| GET/POST/PUT/DELETE | `/api/v1/aftersales[/:id]` | ✅ | — |
| GET | `/api/v1/aftersales/summary` | ✅ | — |
| **Shipping** | | | |
| POST | `/api/v1/shipping/quote` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/shipping/providers[/:id]` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/shipping/channels[/:id]` | ✅ | — |
| GET/POST/DELETE | `/api/v1/shipping/zones[/:id]` | ✅ | — |
| GET/POST/DELETE | `/api/v1/shipping/rules[/:id]` | ✅ | — |
| GET/POST/DELETE | `/api/v1/shipping/bill-batches[/:id]` | ✅ | — |
| GET | `/api/v1/shipping/bill-batches/:id/items` | ✅ | — |
| **Platform Fee** | | | |
| POST | `/api/v1/platform-fee/calculate` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/platform-fee[/:id]` | ✅ | — |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/orders` | 订单列表 | `orders/page.tsx` |
| `/orders/[id]` | 订单详情 | `orders/[id]/page.tsx` |
| `/order-import` | 订单导入 | `order-import/page.tsx` |
| `/shipping` | 物流管理 | `shipping/page.tsx` |
| `/platform-fees` | 平台费用 | `platform-fees/page.tsx` |
| `/aftersales` | 售后 | `aftersales/page.tsx` |
| `/metabolism` | 代谢评分（M1） | `metabolism/page.tsx` |

---

## 6. 财务与经营模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `finance` | `/api/v1/finance` | 财务总览（需 `finance.read` 权限） |
| `settlement` | `/api/v1/settlement` | 结算管理 |
| `decision` | `/api/v1/decision` | 经营决策 |
| `allocation` | `/api/v1/allocation` | 成本分摊 |
| `report` | `/api/v1/report` | 报表（需 `report.read` 权限） |
| `exchangerate` | `/api/v1/exchange-rates` | 汇率管理 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **Finance** | | | |
| POST | `/api/v1/finance/profit/calculate` | ✅ | — |
| POST | `/api/v1/finance/profit/batch-calculate` | ✅ | — |
| GET | `/api/v1/finance/profit/summary` | ✅ | — |
| GET | `/api/v1/finance/profit/ranking` | ✅ | — |
| GET | `/api/v1/finance/summary` | ✅ | — |
| GET | `/api/v1/finance/profit-summary` | ✅ | — |
| GET | `/api/v1/finance/ledger` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/finance/accounts[/:id]` | ✅ | — |
| GET/POST | `/api/v1/finance/transactions` | ✅ | — |
| POST | `/api/v1/finance/mock` | ✅ | — |
| GET | `/api/v1/finance/orders/:order_id/ledger` | ✅ | — |
| GET | `/api/v1/finance/orders/:order_id/profit` | ✅ | — |
| POST | `/api/v1/finance/orders/:order_id/ledger/rebuild` | ✅ | — |
| **Settlement** | | | |
| GET/POST/PUT/DELETE | `/api/v1/settlement[/:id]` | ✅ | `settlement/[id]/page.tsx` |
| GET | `/api/v1/settlement/summary` | ✅ | — |
| POST | `/api/v1/settlement/:id/reconcile` | ✅ | `settlement/[id]/page.tsx` |
| GET/POST | `/api/v1/settlement/:id/items` | ✅ | — |
| PUT | `/api/v1/settlement/items/:item_id/reconciliation` | ✅ | — |
| **Decision** | | | |
| GET/POST/PUT/DELETE | `/api/v1/decision[/:id]` | ✅ | `decision/prelisting/page.tsx` |
| GET | `/api/v1/decision/summary` | ✅ | — |
| POST | `/api/v1/decision/:id/approve` | ✅ | `decision/prelisting/page.tsx` |
| POST | `/api/v1/decision/:id/reject` | ✅ | `decision/prelisting/page.tsx` |
| **Allocation** | | | |
| CRUD | `/api/v1/allocation/warehouses[/:id]` | ✅ | — |
| CRUD | `/api/v1/allocation/rules[/:id]` | ✅ | — |
| CRUD | `/api/v1/allocation/cost/batches[/:id]` | ✅ | — |
| **Report** | | | |
| GET | `/api/v1/report/sales` | ✅ | `reports/page.tsx` |
| GET | `/api/v1/report/profit` | ✅ | `reports/page.tsx` |
| GET | `/api/v1/report/inventory` | ✅ | `reports/page.tsx` |
| GET | `/api/v1/report/settlement` | ✅ | `reports/page.tsx` |
| GET | `/api/v1/report/platform-fee` | ✅ | `reports/page.tsx` |
| **Exchange Rates** | | | |
| GET/POST/DELETE | `/api/v1/exchange-rates[/:id]` | ✅ | — |
| PUT | `/api/v1/exchange-rates/:from_currency/:to_currency` | ✅ | — |
| GET | `/api/v1/exchange-rates/:from_currency/:to_currency/latest` | ✅ | — |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/finance` | 财务总览 | `finance/page.tsx` |
| `/settlement` | 结算列表 | `settlement/page.tsx` |
| `/settlement/[id]` | 结算详情 | `settlement/[id]/page.tsx` |
| `/decision` | 决策总览 | `decision/page.tsx` |
| `/decision/prelisting` | 上架前决策 | `decision/prelisting/page.tsx` |
| `/allocation` | 分配 | `allocation/page.tsx` |
| `/allocation/cost` | 成本分摊 | `allocation/cost/page.tsx` |
| `/reports` | 报表 | `reports/page.tsx` |

---

## 7. 运营支撑模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `dashboard` | `/api/v1/dashboard` | 数据总览（/overview, /orders, /inventory, /exceptions） |
| `search` | `/api/v1/search` | 全局搜索 |
| `notification` | `/api/v1/notification` | 通知管理 |
| `exceptions` | `/api/v1/exceptions` | 异常中心 |
| `importbatch` | `/api/v1/importbatch` | 批量导入记录 |
| `operationlog` | `/api/v1/operationlog` | 操作审计日志 |
| `imagegen` | `/api/v1/imagegen` | 商品图片生成（Prism） |
| `sourcing` | `/api/v1/sourcing` | 1688 选品引擎（A8） |
| `mock` | `/api/v1/mock` | Mock 数据（启动时自动 seed） |
| `owner` | `/api/v1/owner` | Owner 面板 |
| `support` | `/api/v1/support` | 客服支持 |
| `feedback` | — | 反馈系统 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **Dashboard** | | | |
| GET | `/api/v1/dashboard/overview` | ✅ | `dashboard/page.tsx` |
| GET | `/api/v1/dashboard/orders` | ✅ | — |
| GET | `/api/v1/dashboard/inventory` | ✅ | — |
| GET | `/api/v1/dashboard/exceptions` | ✅ | — |
| **Search** | | | |
| GET | `/api/v1/search` | ✅ | `search/page.tsx` |
| GET | `/api/v1/search/recent` | ✅ | — |
| **Notification** | | | |
| GET/POST/PUT/DELETE | `/api/v1/notification[/:id]` | ✅ | — |
| GET | `/api/v1/notification/unread-count` | ✅ | — |
| PUT | `/api/v1/notification/read-all` | ✅ | — |
| CRUD | `/api/v1/notification/alert-rules[/:id]` | ✅ | — |
| **Exceptions** | | | |
| GET/POST/PUT/DELETE | `/api/v1/exceptions[/:id]` | ✅ | — |
| **Operation Log** | | | |
| GET/POST | `/api/v1/operationlog[/:id]` | ✅ | — |
| **Image Gen** | | | |
| GET/POST/PUT/DELETE | `/api/v1/imagegen[/:id]` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/imagegen/canvas[/:id]` | ✅ | — |
| GET/POST/PUT/DELETE | `/api/v1/imagegen/templates[/:id]` | ✅ | — |
| POST | `/api/v1/imagegen/templates/:id/use` | ✅ | — |
| PUT | `/api/v1/imagegen/:id/status` | ✅ | — |
| **Import Batch** | | | |
| GET/POST/PUT/DELETE | `/api/v1/importbatch[/:id]` | ✅ | — |
| GET | `/api/v1/importbatch/:id/rows` | ✅ | — |
| **Sourcing** | | | |
| POST | `/api/v1/sourcing/fetch` | ✅ | `sourcing/page.tsx` |
| GET | `/api/v1/sourcing/recommendations` | ✅ | `sourcing/page.tsx` |
| **Mock** | | | |
| POST | `/api/v1/mock/seed` | ✅ | — |
| **Owner** | | | |
| GET | `/api/v1/owner/risk-summary` | ✅ | — |
| GET | `/api/v1/owner/suggestions` | ✅ | — |
| GET | `/api/v1/owner/decision-queue` | ✅ | — |
| **Support** | | | |
| CRUD | `/api/v1/support/conversations[/:id]` | ✅ | — |
| CRUD | `/api/v1/support/templates[/:id]` | ✅ | — |
| CRUD | `/api/v1/support/blacklist` | ✅ | — |

### 前端页面

| 路由 | 说明 | 文件 |
|------|------|------|
| `/` | 根入口（→ dashboard） | `page.tsx` |
| `/dashboard` | 运营数据总览 | `dashboard/page.tsx` |
| `/exceptions` | 异常工作台 | `exceptions/page.tsx` |
| `/notifications` | 通知中心 | `notifications/page.tsx` |
| `/image-gen` | AI 生图 | `image-gen/page.tsx` |
| `/image-gen/canvas` | 生图画布 | `image-gen/canvas/page.tsx` |
| `/import-batches` | 批量导入 | `import-batches/page.tsx` |
| `/operation-logs` | 操作日志 | `operation-logs/page.tsx` |
| `/search` | 搜索 | `search/page.tsx` |
| `/reports` | 报表 | `reports/page.tsx` |
| `/settings` | 系统设置 | `settings/page.tsx` |
| `/settings/llm` | LLM 配置 | `settings/llm/page.tsx` |
| `/settings/rbac` | 权限管理 | `settings/rbac/page.tsx` |
| `/settings/policy` | 审批策略 | `settings/policy/page.tsx` |
| `/owner` | Owner 经营总控台 | `owner/page.tsx` |
| `/experiments` | 经营实验案件列表与创建 | `experiments/page.tsx` |
| `/experiments/[experimentId]` | 证据、反证、闸门、对象关联、利润与现金终态 | `experiments/[experimentId]/page.tsx` |

---

## 8. AI 治理模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `agentrule` | `/api/v1/agent-rules` | Agent 行为规则 |
| `entropy` | `/api/v1/entropy` | 自净化：SPC 控制、健康评分、防御 |
| `evolution` | `/api/v1/evolution` | Agent 演化推送 |
| `trustscore` | `/api/v1/trust-scores` | 信任分计算、自主权门控 |
| `actionpolicy` | `/api/v1/policy` | 动作审批策略 |
| `metabolism` | `/api/v1/metabolism` | 代谢系统：排泄评分 (M1) |
| `approval` | `/api/v1/approvals` | 审批管理 |
| `agentlearning` | `/api/v1/agent-learning` | Agent 学习记录 |
| `orchestration` | `/api/v1/orchestration` | Agent 编排 |
| `personalrule` | `/api/v1/personal-rules` | 个人规则 |
| `sentiment` | `/api/v1/sentiment` | 情感分析 |

### API 路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| **Trust Scores** | | | |
| GET/POST | `/api/v1/trust-scores[/:agent_id]` | ✅ | `agents/trust/page.tsx` |
| POST | `/api/v1/trust-scores/recalculate` | ✅ | `agents/trust/page.tsx` |
| POST | `/api/v1/trust-scores/eligible` | ✅ | — |
| PUT | `/api/v1/trust-scores/:agent_id/level` | ✅ | — |
| POST | `/api/v1/trust-scores/auto-upgrade` | ✅ | `agents/trust/page.tsx` |
| GET | `/api/v1/trust-scores/summary` | ✅ | `agents/trust/page.tsx` |
| **Evolution** | | | |
| GET | `/api/v1/evolution/nudges` | ✅ | — |
| POST | `/api/v1/evolution/nudges/evaluate` | ✅ | `agents/evolution/page.tsx` |
| POST | `/api/v1/evolution/nudges/:id/accept` | ✅ | — |
| POST | `/api/v1/evolution/nudges/:id/dismiss` | ✅ | — |
| **Entropy** | | | |
| GET | `/api/v1/entropy` | ✅ | — |
| POST | `/api/v1/entropy/defense` | ✅ | — |
| GET | `/api/v1/entropy/health` | ✅ | — |
| GET | `/api/v1/entropy/spc` | ✅ | — |
| GET | `/api/v1/entropy/changelog` | ✅ | — |
| **Action Policy** | | | |
| CRUD | `/api/v1/policy/rules[/:id]` | ✅ | `settings/policy/page.tsx` |
| POST | `/api/v1/policy/evaluate` | ✅ | — |
| **Agent Rules** | | | |
| CRUD + Toggle | `/api/v1/agent-rules[/:id]` | ✅ | — |
| POST | `/api/v1/agent-rules/evaluate` | ✅ | — |
| **RBAC** | | | |
| CRUD | `/api/v1/rbac/roles[/:id]` | ✅ | — |
| GET/POST | `/api/v1/rbac/roles/:id/permissions` | ✅ | — |
| CRUD | `/api/v1/rbac/permissions[/:id]` | ✅ | — |
| GET | `/api/v1/rbac/current/permissions` | ✅ | `stores/permission-store.ts` |
| GET/POST | `/api/v1/rbac/users/:id/roles` | ✅ | — |

---

## 9. 新加入 / 施工中模块

| 模块 | 位置 | 状态 |
|------|------|------|
| `content` | `internal/domain/content/` | 多语言内容本地化 |
| `prismadapter` | `internal/prismadapter/` | Prism 外部生图服务客户端 |
| `schemadrift` | `internal/schemadrift/` | Schema drift 检测（迁移安全网） |

---

## 10. 认证与公共路由

| Method | Path | Status | 前端引用 |
|--------|------|--------|----------|
| GET | `/api/health` | ✅ | — |
| GET | `/api/v1/health` | ✅ | — |
| GET | `/metrics` | ✅ | — |
| POST | `/api/v1/auth/login` | ✅ | `(auth)/login/page.tsx` |
| POST | `/api/v1/auth/register` | ✅ | — |
| POST | `/api/v1/auth/refresh` | ✅ | `lib/api-client.ts` |
| GET | `/api/v1/auth/me` | ✅ | — |
| WS | `/ws` | ✅ | — |

---

## 11. 缺少的前端端点 (无后端路由)

| Method | Path | 前端文件 | 说明 |
|--------|------|----------|------|
| GET | `/v1/listing-task/stats` | `listing-tasks/workbench/page.tsx` | 返回 `{pending, total}` — 无 handler |
| POST | `/v1/listing-task/retry-all` | `listing-tasks/workbench/page.tsx` | 批量重试 — 无 handler |
| GET | `/v1/settings/llm` | `settings/llm/page.tsx` | LLM 配置 — 无 settings 模块 |
| POST | `/v1/listing` | `listings/create/page.tsx` | **路径不匹配** — 后端是 `POST /v1/listings` |

---

## 12. 关键架构说明

1. **Products 和 SKUs 共享同一模块** (`internal/domain/sku/`)。`sku` 包同时处理 `/products` 和 `/skus` 路由。没有独立的 `domain/products/` 目录。
2. **Listings 拆分为两个前缀**: CRUD 在 `/listings` 下，发布链在 `/listing` 下。这是有意的设计（常规 CRUD vs. 运维操作）。
3. **Settings 模块不存在** 于 backend-go。前端 `settings/llm/page.tsx` 调用 `/v1/settings/llm` 但 router.go 中未注册。可能是 Phase 2 添加。
4. **调度驱动 Agent**（A1-A11, G0-G3）没有 HTTP 端点 — 它们作为 EventBus 订阅在 cron tick 上运行，而不是 REST API 调用。
5. **前端路径不匹配**: 前端 `POST /v1/listing` (单数) vs. 后端 `POST /v1/listings` (复数)。
6. **WebSocket** 端点为 `/ws`，由 `internal/realtime/` 处理，支持 AI 流式输出和实时更新。

---

## 路由统计

| 类别 | 计数 |
|------|------|
| 注册路由总数 | ~225 |
| 完整 CRUD 模块 | 38 of 38 |
| 仅 handler stub 模块 | 0 |
| 前端引用端点 | ~65 |
| 前端无后端路由 | 3 + 1 路径不匹配 |

---

## 相关文档

- [API 快速参考](reference-api-quick.md) — 路由、权限、响应格式
- [配置参考](reference-configuration.md) — config.yaml + 环境变量
- [How to 添加新领域模块](howto-add-domain-module.md)
- [系统架构 v1](system-architecture-design-v1.md)
