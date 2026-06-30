# 模块目录 (Module Catalog)

> 凌镜 LingMirror（技术名: MultiSell）后端领域模块完整目录
> 更新日期: 2026-06-30
> 源码路径: `backend-go/internal/domain/`

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

---

## 1. 平台基础设施 (`internal/platform/`)

四进程内协调原语，Agent 间、Agent 系统间通信基础。

| 模块 | 文件 | 说明 |
|------|------|------|
| **EventBus** | `eventbus/bus.go` | 发布/订阅，glob 主题匹配 (`order.*`)。~15 订阅。可选 outbox 持久化。 |
| **Command** | `command/command.go` | 类型化处理器注册。命令: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check` |
| **Scheduler** | `scheduler/scheduler.go` | 周期性任务（5min-6h），发布 `scheduler.tick.{agent_id}` 事件。 |
| **ToolBridge** | `toolbridge/bridge.go` | 插件驱动工具执行桥接，Agent 通过它调用外部工具。 |

### EventBus 核心 API

```go
type EventBus interface {
    Publish(ctx, topic, source string, payload Map) error
    Subscribe(topic string, handler HandlerFunc) Subscription
    Start(ctx)
    Stop()
}
```

### 事件主题命名规则

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
| A2 | 经营复盘 | `listing_optimize` | 事件驱动 | 刊登优化（链式触发） |
| A3 | 广告分析 | `acos_analysis` | 1h | ACOS 分析 |
| A4 | 客服回复 | `auto_reply` | 5min | 待处理消息检查 |
| A5 | 库存预警 | `stock_alert` | 15min | 库存检查 |
| A6 | 利润看护 | `profit_watch` | 1h | 利润监控 |
| A7 | 合规检测 | `compliance_check` | 2h | 合规检测 |
| A8 | 选品扫描 | `sourcing_scan` | 1h | 1688 选品扫描 |
| A10 | 物流费率 | (链式触发) | — | 跨境物流费率 |
| M1 | 代谢评分 | `excretion_scoring` | 1h | 实体排泄评分 |

### Agent 决策链

```
A5 stock_alert (red)           → G3 discount_risk_check
G3 discount_risk_check (block)  → A6 profit_watch
A6 profit_watch (loss/threshold)→ A2 listing_optimize
G0 system_health (anomaly > 3) → G1 dashboard_overview
```

---

## 3. 商品与供应链模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `sku` | `/api/v1/products`, `/skus` | 商品 CRUD、规格、SKU 笛卡尔积生成 |
| `category` | `/api/v1/categories` | 无限级分类树 |
| `brand` | `/api/v1/brands` | 品牌管理 |
| `price` | `/api/v1/prices`, `/skus/:id/prices` | 多类型价格、批量调价、记录 |
| `inventory` | `/api/v1/inventory` | 库存更新、安全库存预警、变动记录 |
| `supplier` | `/api/v1/suppliers` | 供应商档案、商品-供应商绑定 |
| `purchase` | `/api/v1/purchases` | 采购订单 |
| `supplychain` | `/api/v1/supplychain` | 供应链编排（A8→A10 联动） |
| `supplyevent` | — | 供应链事件模型 |
| `tariff` | `/api/v1/tariffs` | 关税规则 |
| `candidate` | `/api/v1/candidates` | 候选商品管理 |
| `completeness` | `/api/v1/completeness` | 商品完整度检查 |
| `profit` | `/api/v1/profit` | 利润计算 |
| `cost` | `/api/v1/costs` | 成本分摊 |
| `landedcost` | `/api/v1/landed-costs` | 到岸成本 |
| `productanalysis` | `/api/v1/product-analysis` | 商品分析 |
| `producthub` | `/api/v1/product-hub` | 商品中心 Hub |
| `consolidation` | `/api/v1/consolidation` | 商品聚合/合并 |

## 4. 平台与发布模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `platform` | `/api/v1/platforms` | 平台配置（Ozon/Shopee API 密钥） |
| `integrations` | `/api/v1/integrations` | 平台集成适配器（PlatformAdapter 接口） |
| `listing` | `/api/v1/listings` | 刊登记录 |
| `listingtask` | `/api/v1/listing-tasks` | 刊登任务 + 工作台 |
| `loop` | `/api/v1/loop` | 经营循环 |

## 5. 订单与物流模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `order` | `/api/v1/orders` | 订单管理 |
| `orderimport` | `/api/v1/order-import` | 订单导入（CSV） |
| `shipping` | `/api/v1/shipping` | 运费 |
| `logistics` | `/api/v1/logistics` | 物流费率引擎（A10）：四种定价模式、YAML 费率表、承运商绩效 |
| `aftersales` | `/api/v1/aftersales` | 售后、退货 Rate Tracker |
| `platformfee` | `/api/v1/platform-fees` | 平台费用规则 |

## 6. 财务与经营模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `finance` | `/api/v1/finance` | 财务总览（需 `finance.read` 权限） |
| `settlement` | `/api/v1/settlement` | 结算管理 |
| `decision` | `/api/v1/decisions` | 经营决策 |
| `allocation` | `/api/v1/allocation` | 成本分摊 |
| `report` | `/api/v1/reports` | 报表（需 `report.read` 权限） |
| `exchangerate` | `/api/v1/exchange-rates` | 汇率管理 |

## 7. 运营支撑模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `dashboard` | `/api/v1/dashboard` | 数据总览（/overview, /orders, /inventory, /exceptions） |
| `search` | `/api/v1/search` | 全局搜索 |
| `notification` | `/api/v1/notifications` | 通知管理 |
| `exceptions` | `/api/v1/exceptions` | 异常中心 |
| `importbatch` | `/api/v1/import-batches` | 批量导入记录 |
| `operationlog` | `/api/v1/operation-logs` | 操作审计日志 |
| `settings` | `/api/v1/settings` | 系统设置、LLM 配置 |
| `imagegen` | `/api/v1/image-gen` | 商品图片生成（Prism） |
| `sourcing` | `/api/v1/sourcing` | 1688 选品引擎（A8） |
| `sourcing1688` | `/api/v1/sourcing1688` | 1688 采集端 |
| `mock` | `/api/v1/mock` | Mock 数据（启动时自动 seed） |
| `owner` | `/api/v1/owner` | Owner 面板 |
| `agentlearning` | `/api/v1/agent-learning` | Agent 学习记录 |
| `approval` | `/api/v1/approvals` | 审批管理 |
| `orchestration` | `/api/v1/orchestration` | Agent 编排 |
| `personalrule` | `/api/v1/personal-rules` | 个人规则 |
| `sentiment` | `/api/v1/sentiment` | 情感分析 |
| `support` | `/api/v1/support` | 客服支持 |
| `feedback` | — | 反馈系统 |

## 8. AI 治理模块

| 模块 | 路由前缀 | 能力 |
|------|----------|------|
| `agentrule` | `/api/v1/agent-rules` | Agent 行为规则 |
| `entropy` | `/api/v1/entropy` | 自净化：SPC 控制、健康评分、防御 |
| `evolution` | `/api/v1/evolution` | Agent 演化推送 |
| `trustscore` | `/api/v1/trust-scores` | 信任分计算、自主权门控 |
| `actionpolicy` | `/api/v1/action-policies` | 动作审批策略 |
| `metabolism` | `/api/v1/metabolism` | 代谢系统：排泄评分 (M1) |

## 9. 新加入 / 施工中模块

| 模块 | 位置 | 状态 |
|------|------|------|
| `content` | `internal/domain/content/` | 多语言内容本地化 |
| `prismadapter` | `internal/prismadapter/` | Prism 外部生图服务客户端 |
| `schemadrift` | `internal/schemadrift/` | Schema drift 检测（迁移安全网） |

---

## 相关文档

- [API 快速参考](reference-api-quick.md) — 路由、权限、响应格式
- [配置参考](reference-configuration.md) — config.yaml + 环境变量
- [How to 添加新领域模块](howto-add-domain-module.md)
- [系统架构 v1](system-architecture-design-v1.md)
