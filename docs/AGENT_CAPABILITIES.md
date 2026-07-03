# Agent Capabilities

> **用途**: 该文档列出 LingMirror (MultiSell) 项目中所有对 Agent 可用的能力。
> **定位**: 每个开发 AI 在开始工作前应首先阅读此文档，了解自己有什么能力可用。
> **维护**: 添加新 API/工具/MCP 时同步更新此文档。

---

## Agent Onboarding 指引

1. 先读此文档了解有什么可用
2. 按需用 MCP 或 API 或 CLI 操作
3. 遇到不清楚的行为，优先用 CodeGraph 查代码
4. 需要调试 UI 时，启动 Chrome DevTools MCP 并在本地运行项目

---

## 一、MCP Servers

在当前 Claude 会话中可直接调用的 MCP 工具。所有 `mcp__chrome-devtools__*` 和 `mcp__codegraph__*` 工具在会话中自动可用。

### 1. Chrome DevTools（浏览器调试）

| 工具 | 用途 |
|------|------|
| `take_screenshot` | 截屏（支持整页/元素级） |
| `take_snapshot` | 获取可访问性树快照（比截图更有用） |
| `navigate_page` | 导航到指定 URL |
| `new_page` | 打开新标签页 |
| `evaluate_script` | 在页面执行 JS |
| `click` | 点击元素 |
| `fill` | 填写表单元素 |
| `fill_form` | 批量填表（多字段一次完成） |
| `select_page` | 切换当前选中的页面 |
| `list_console_messages` | 查看控制台日志 |
| `list_network_requests` | 查看网络请求 |
| `get_network_request` | 获取具体网络请求/响应的详情 |
| `list_pages` | 列出所有打开的页面 |
| `hover` | 悬停在元素上 |
| `press_key` | 键盘操作（Enter, Ctrl+ 等） |
| `type_text` | 在聚焦输入框中打字 |
| `upload_file` | 上传文件 |
| `wait_for` | 等待文本出现 |
| `handle_dialog` | 处理浏览器弹窗 |
| `emulate` | 模拟设备/网络/颜色模式/UA |
| `resize_page` | 调整浏览器窗口大小 |
| `lighthouse_audit` | 做无障碍/SEO/最佳实践评分 |
| `performance_start_trace` | 启动性能追踪（LCP/INP/CLS 分析） |
| `performance_stop_trace` | 停止性能追踪 |
| `take_heapsnapshot` | 抓取 JS 堆快照（内存泄漏分析） |

**使用场景**: 前端调试、UI 测试、页面分析、性能优化。
**启动方式**: 调用 `mcp__chrome-devtools__*`（会话中自动可用）。

### 2. CodeGraph（代码智能）

| 工具 | 用途 |
|------|------|
| `codegraph_explore` | **主要查询入口** — 自然语言问题或符号名，返回符号源码 + 调用路径 |
| `codegraph_node` | 读取文件/符号源码（替代 Read），附带依赖它的文件列表 |
| `codegraph_search` | 按名称快速搜索符号（仅返回位置） |
| `codegraph_callers` | 查询某个函数/方法的调用者 |

**使用场景**: 代码理解、定位文件/符号、编辑前检查影响范围、查调用链。
**优先 CodeGraph 而非手动的 grep + Read 循环** — 一次调用返回源码+关系，远快于手动搜索。

---

## 二、后端 API（Go / Gin）

所有 API 端点以 `/api/v1` 为前缀。非 auth 端点需要 JWT 认证（`Authorization: Bearer <token>`）。
响应格式统一为 `{ "code": ..., "message": "...", "data": ... }`。

### 公共端点（无需认证）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/v1/health` | API 版本健康检查 |
| GET | `/metrics` | Prometheus 指标（需配置开启） |
| GET | `/ws` | WebSocket 端点（AI 流式输出，需 JWT） |
| POST | `/api/v1/auth/login` | 登录（获取 access_token + refresh_token） |
| POST | `/api/v1/auth/register` | 注册用户 |
| POST | `/api/v1/auth/refresh` | 刷新 token |

### 认证 & RBAC

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录（获取 access_token + refresh_token） |
| POST | `/api/v1/auth/register` | 注册用户 |
| POST | `/api/v1/auth/refresh` | 刷新 token |
| GET | `/api/v1/auth/me` | 获取当前用户信息（需 JWT） |
| GET/POST/PUT/DELETE | `/api/v1/rbac/*` | RBAC 角色权限管理 |

### 商品管理（SKU / Product）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/products` | 商品列表（分页+搜索） |
| GET | `/api/v1/products/:id` | 商品详情 |
| POST | `/api/v1/products` | 创建商品 |
| PUT | `/api/v1/products/:id` | 更新商品 |
| DELETE | `/api/v1/products/:id` | 删除商品 |
| GET | `/api/v1/products/:id/specs` | 商品规格列表 |
| POST | `/api/v1/products/:id/specs` | 创建规格 |
| PUT | `/api/v1/products/:id/specs/:spec_id` | 更新规格 |
| DELETE | `/api/v1/products/:id/specs/:spec_id` | 删除规格 |
| POST | `/api/v1/products/:id/specs/:spec_id/values` | 创建规格值 |
| GET | `/api/v1/products/:id/skus` | 商品下 SKU 列表 |
| PUT | `/api/v1/spec-values/:id` | 更新规格值 |
| DELETE | `/api/v1/spec-values/:id` | 删除规格值 |
| GET | `/api/v1/skus` | SKU 列表 |
| GET | `/api/v1/skus/:id` | SKU 详情 |
| POST | `/api/v1/skus` | 创建 SKU |
| PUT | `/api/v1/skus/:id` | 更新 SKU |
| DELETE | `/api/v1/skus/:id` | 删除 SKU |

### 分类 & 品牌

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/categories` | 分类 CRUD |
| GET/POST | `/api/v1/brands` | 品牌 CRUD |

### 库存管理

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/inventory` | 库存列表 |
| GET | `/api/v1/inventory/:id` | 库存详情 |
| PUT | `/api/v1/inventory/:id` | 更新库存 |
| GET | `/api/v1/inventory/logs` | 库存变动日志 |

### 价格管理

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/prices` | 价格列表 |
| POST | `/api/v1/prices` | 创建价格 |
| PUT | `/api/v1/prices/:id` | 更新价格 |
| DELETE | `/api/v1/prices/:id` | 删除价格 |

### 平台（Platform）& 店铺（Store）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/platforms` | 平台列表 |
| GET/POST/PUT/DELETE | `/api/v1/platforms/:id` | 平台 CRUD |
| GET | `/api/v1/stores` | 店铺列表 |
| GET/POST/PUT/DELETE | `/api/v1/stores/:id` | 店铺 CRUD |

### Listing（上架管理）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/listings` | Listing 列表 |
| GET/POST/PUT/DELETE | `/api/v1/listings/:id` | Listing CRUD |
| POST | `/api/v1/listings/:id/publish` | 发布到平台 |
| POST | `/api/v1/listings/:id/sync` | 同步平台状态 |
| POST | `/api/v1/listing` | 创建 Listing（兼容别名） |
| POST | `/api/v1/listing/products/:product_id/publish/:platform_id` | 将商品发布到指定平台 |
| GET | `/api/v1/listing/products/:product_id/listings` | 查询某商品在所有平台的 Listing |
| GET | `/api/v1/listing/listing-tasks` | 上架任务队列列表 |
| POST | `/api/v1/listing/listing-tasks/from-decisions` | 从决策创建上架任务 |
| POST | `/api/v1/listing/listing-tasks/:task_id/recheck` | 重新检查上架任务 |
| POST | `/api/v1/listing/listing-tasks/:task_id/cancel` | 取消上架任务 |
| POST | `/api/v1/listing/listing-tasks/:task_id/publish` | 执行上架任务 |

### 订单管理

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/order` | 订单列表 |
| GET | `/api/v1/order/summary` | 订单汇总 |
| GET/POST/PUT/DELETE | `/api/v1/order/:id` | 订单 CRUD |
| POST | `/api/v1/order/:id/status` | 更新订单状态 |

### 订单导入

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/orderimport` | 导入批次列表 |
| POST | `/api/v1/orderimport` | 创建导入批次 |
| GET | `/api/v1/orderimport/:id` | 导入批次详情 |
| DELETE | `/api/v1/orderimport/:id` | 删除导入批次 |

### 物流（Shipping）

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/v1/shipping/quote` | 运费报价 |
| GET/POST/PUT/DELETE | `/api/v1/shipping/providers` | 物流商管理 |
| GET/POST/PUT/DELETE | `/api/v1/shipping/channels` | 物流渠道管理 |
| GET/POST/DELETE | `/api/v1/shipping/zones` | 物流区域管理 |
| GET/POST/DELETE | `/api/v1/shipping/rules` | 报价规则管理 |
| GET/POST/DELETE | `/api/v1/shipping/bill-batches` | 账单批次管理 |
| GET | `/api/v1/shipping/bill-batches/:id/items` | 账单条目列表 |

### 平台费用（Platform Fee）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/platform-fee` | 平台费用列表 |
| POST/PUT/DELETE | `/api/v1/platform-fee/:id` | 平台费用 CRUD |

### 结算（Settlement）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/settlement` | 结算列表 |
| GET | `/api/v1/settlement/summary` | 结算汇总 |
| GET/POST/PUT/DELETE | `/api/v1/settlement/:id` | 结算 CRUD |
| POST | `/api/v1/settlement/:id/reconcile` | 对账操作 |
| POST | `/api/v1/settlement/:id/items` | 添加结算条目 |
| GET | `/api/v1/settlement/:id/items` | 结算条目列表 |
| PUT | `/api/v1/settlement/items/:item_id/reconciliation` | 更新条目对账状态 |

### 财务（Finance）

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/v1/finance/profit/calculate` | 计算单条利润 |
| POST | `/api/v1/finance/profit/batch-calculate` | 批量计算利润 |
| GET | `/api/v1/finance/profit/summary` | 利润汇总 |
| GET | `/api/v1/finance/profit/ranking` | SKU 利润排行 |
| GET | `/api/v1/finance/summary` | 财务汇总 |
| GET | `/api/v1/finance/ledger` | 账本列表 |
| POST | `/api/v1/finance/mock` | 模拟数据 |
| GET | `/api/v1/finance/orders/:order_id/profit` | 订单利润 |
| POST | `/api/v1/finance/orders/:order_id/ledger/rebuild` | 重建订单账本 |
| GET/POST/PUT/DELETE | `/api/v1/finance/accounts` | 账户管理 |
| GET/POST | `/api/v1/finance/transactions` | 交易流水 |

### 供应商（Supplier）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/suppliers` | 供应商列表 |
| PUT/DELETE | `/api/v1/suppliers/:id` | 供应商 CRUD |

### 售后（After-sales）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/aftersales` | 售后单 CRUD |
| GET/PUT/DELETE | `/api/v1/aftersales/:id` | 售后单管理 |

### 1688 采购 & AI 选品

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/v1/sourcing-1688` | 1688 采购 CRUD |
| POST | `/api/v1/sourcing/fetch` | AI 选品：抓取并分析 1688 商品（⚠️ 已定义但未在 router.go 接线） |
| GET | `/api/v1/sourcing/recommendations` | 获取 AI 选品推荐列表（⚠️ 同上） |

### 分配（Allocation）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/allocation` | 分配管理 |

### 异常（Exceptions）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/exceptions` | 异常列表 |

### 决策管理（Decision）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/decision` | 决策列表 |
| GET | `/api/v1/decision/summary` | 决策汇总 |
| GET/POST/PUT/DELETE | `/api/v1/decision/:id` | 决策 CRUD |
| POST | `/api/v1/decision/:id/approve` | 审批通过 |
| POST | `/api/v1/decision/:id/reject` | 审批拒绝 |

### 客服（Support）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/support/conversations` | 会话列表/创建 |
| GET/PUT/DELETE | `/api/v1/support/conversations/:id` | 会话管理 |
| POST | `/api/v1/support/conversations/:id/reply` | 发送回复 |
| POST | `/api/v1/support/conversations/:id/close` | 关闭会话 |
| GET | `/api/v1/support/conversations/:id/messages` | 会话消息列表 |
| GET/POST/PUT/DELETE | `/api/v1/support/templates` | 回复模板管理 |
| GET/POST/DELETE | `/api/v1/support/blacklist` | 黑名单管理 |

### 通知（Notification）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/notification` | 通知列表 |

### 搜索

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/search` | 全局搜索 |
| GET | `/api/v1/search/recent` | 最近搜索 |

### 仪表盘（Dashboard）& 报表

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/dashboard` | 仪表盘数据 |
| GET | `/api/v1/report` | 报表列表 |
| GET | `/api/v1/report/summary` | 报表汇总 |

### 设置

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/setting` | 系统设置 |
| PUT | `/api/v1/setting` | 更新设置 |

### 图片生成

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/imagegen` | 图片生成列表 |
| POST | `/api/v1/imagegen` | 生成图片 |

### 导入批次 & 操作日志

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/importbatch` | 导入批次管理 |
| GET | `/api/v1/operationlog` | 操作日志列表 |

### 汇率 & 产品分析

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/exchange-rates` | 汇率列表 |
| GET | `/api/v1/product-analysis` | 产品分析 |

### 平台集成（Integrations）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/v1/platform-integrations` | 平台集成管理 |

### 采购（Purchase）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/purchase` | 采购单列表 |

### AI & Agent 系统

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/v1/agents/:id/actions` | 触发 Agent 决策执行 |
| GET | `/api/v1/agents` | Agent 列表 |
| GET | `/api/v1/agents/:id` | Agent 详情 |
| GET | `/api/v1/agents/evolution` | Agent 演化配置 |
| GET | `/api/v1/agents/entropy` | Agent 熵防御状态 |
| POST | `/api/v1/ai/chat` | AI 聊天 |
| POST | `/api/v1/ai/run` | 运行 Agent（指定 agent_id + decision_point） |
| GET | `/api/v1/ai/traces` | 追踪记录列表 |
| GET | `/api/v1/ai/traces/:trace_id` | 追踪详情 |
| GET | `/api/v1/ai/actions` | AI Action 列表 |
| GET/POST | `/api/v1/ai/actions/:id` | Action 详情/操作 |
| POST | `/api/v1/ai/actions/:id/approve` | 批准 Action |
| POST | `/api/v1/ai/actions/:id/reject` | 拒绝 Action |
| POST | `/api/v1/ai/actions/:id/execute` | 执行 Action |
| POST | `/api/v1/ai/actions/:id/review` | 审核 Action |
| GET | `/api/v1/ai/agents` | Agent 花名册 |
| GET | `/api/v1/ai/agents/specs` | Agent 定义明细 |

### AgentOS 驾驶舱

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/agentos` | AgentOS 概览 |
| GET | `/api/v1/agentos/status` | AgentOS 状态 |
| GET | `/api/v1/agentos/work-items` | 工作项列表 |
| GET | `/api/v1/agentos/autonomy` | 自主化状态 |

### 信任分 & 熵 & 演化 & 策略

| 方法 | 路径 | 用途 |
|------|------|------|
| GET/POST | `/api/v1/trustscore/*` | 信任分管理 |
| GET/POST | `/api/v1/entropy/*` | 熵防御 |
| GET/POST | `/api/v1/evolution/*` | 演化规则 |
| GET/POST | `/api/v1/action-policy/*` | Action 审批策略 |
| GET/POST | `/api/v1/agent-rules/*` | 个人规则 |

---

## 三、测试用外部 API

用于开发测试的外部服务接口。**API Key 和 Secret 不走文档，走环境变量。**

| API | 模型 | 用途 | 默认 endpoint |
|-----|------|------|---------------|
| **DeepSeek** | `deepseek-v4-flash` | AI 模型测试（替代昂贵的生产模型） | 见环境变量 `ANTHROPIC_BASE_URL` |

```bash
# .env 或 shell profile — 非对称到 git 的配置
export ANTHROPIC_BASE_URL=http://127.0.0.1:15721
export ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME=deepseek-v4-flash
export ANTHROPIC_DEFAULT_SONNET_MODEL_NAME=deepseek-v4-flash
export ANTHROPIC_DEFAULT_OPUS_MODEL_NAME=deepseek-v4-flash
# API Key 在 .claude/settings.json 的 env 段里管理
```

> **注意**: 生产模型走 Claude（Opus/Sonnet/Haiku），deepseek-v4-flash 仅用于开发测试和低成本试跑。密钥配置在 `.claude/settings.json` 的 `env.ANTHROPIC_AUTH_TOKEN` 中（目前已配置为 `PROXY_MANAGED`）。

---

## 四、平台集成适配器

支持的电商平台（通过 `internal/domain/integrations/` 的 `PlatformAdapter` 接口）：

| 平台 | 适配器状态 | 能力 |
|------|-----------|------|
| **Ozon** | ✅ 已实现 | 发布商品、同步状态、验证凭证、同步库存、推追踪号、拉订单 |
| **Shopee** | ✅ 已实现 | 同上 |

同步任务在 `router.go` 中以 scheduler tick 形式注册：
- Ozon 订单同步：每 15 分钟（`scheduler.tick.ozon_sync`）

---

## 五、AI Agent 系统

### Agent 花名册

| ID | 名称 | Squad | 决策点 | 自主化等级 |
|----|------|-------|--------|-----------|
| A1 | 选品助理 | insight | product_scout, market_analysis, product_research, supplier_discovery | advisory |
| A2 | 商品优化师 | insight | listing_optimize, keyword_research | advisory |
| A3 | 广告分析师 | insight | acos_analysis, ad_optimization | advisory |
| A4 | 客服助理 | ops | auto_reply, intent_classify | guided |
| A5 | 库存助理 | ops | stock_alert | guided |
| A6 | 利润看护 | ops | profit_watch, profit_check | supervised |
| A7 | 合规专员 | ops | compliance_check, certification_lookup | supervised |
| A8 | 选品盈利分析 | insight | sourcing_recommend | advisory |
| A9 | 批量运维 | ops | batch_price_update, batch_inventory_sync, batch_listing_update, import_validation | guided |
| A10 | 物流运费引擎 | ops | carrier_compare, shipping_bill_audit, carrier_performance, logistics_route_opt | guided |
| A11 | 售后管理 | ops | return_analysis, refund_decision, dispute_manage, aftersales_report | guided |
| G0 | 系统健康员 | governance | system_health | supervised |
| G1 | 驾驶舱 | governance | dashboard_overview | advisory |
| G2 | 仓储专员 | governance | warehouse_routing, customs_declare | supervised |
| G3 | 折扣风控 | governance | discount_risk_check, promotion_validation | supervised |

### 调度周期

| Agent | 周期 | 决策点 |
|-------|------|--------|
| G0 | 5 min | system_health |
| A4 | 5 min | auto_reply |
| G1 | 5 min | dashboard_overview |
| A5 | 15 min | stock_alert |
| G3 | 30 min | discount_risk_check |
| A6 | 1 hr | profit_watch |
| A3 | 1 hr | acos_analysis |
| G2 | 1 hr | warehouse_routing |
| A7 | 2 hr | compliance_check |
| trustscore | 1 hr | recalculate |
| entropy | 6 hr | defend |
| M1 | 1 hr | excretion_scoring |
| ozon_sync | 15 min | sync_orders |

### Agent 管道链（Pipeline Chain）

```
A5 stock_alert (red) → G3 discount_risk_check
G3 discount_risk_check (block) → A6 profit_watch
A6 profit_watch (loss/threshold) → A2 listing_optimize
G0 system_health (anomaly > 3) → G1 dashboard_overview
```

### Action 风险等级

| 等级 | 示例 | 需审批 |
|------|------|--------|
| **low** | listing_optimize, keyword_research, product_scout | 否 |
| **medium** | stock_alert, profit_check, replenishment_plan | 是 |
| **high** | profit_watch, discount_check, compliance_check | 是 |

---

## 六、前端应用（Next.js 16）

可用 `mcp__chrome-devtools__navigate_page` 直接访问。

### 本地开发 URL

```
http://localhost:3000
```

### 前端页面路由

| 路径 | 对应 API |
|------|----------|
| `/login` | POST /api/v1/auth/login |
| `/dashboard` | GET /api/v1/dashboard |
| `/products` | GET /api/v1/products |
| `/products/:id` | GET /api/v1/products/:id |
| `/categories` | GET /api/v1/categories |
| `/brands` | GET /api/v1/brands |
| `/sku` | GET /api/v1/sku |
| `/listings` | GET /api/v1/listings |
| `/listing-tasks` | GET /api/v1/listing/listing-tasks |
| `/inventory` | GET /api/v1/inventory |
| `/prices` | GET /api/v1/prices |
| `/orders` | GET /api/v1/orders |
| `/order-import` | GET /api/v1/order-import |
| `/shipping` | GET /api/v1/shipping/* |
| `/settlement` | GET /api/v1/settlement |
| `/finance` | GET /api/v1/finance/* |
| `/platform-fees` | GET /api/v1/platform-fee |
| `/platforms` | GET /api/v1/platforms |
| `/platform-integrations` | GET /api/v1/platform-integrations |
| `/platform-integrations/:id/ozon-products` | GET /api/v1/platform-integrations/:id |
| `/suppliers` | GET /api/v1/suppliers |
| `/aftersales` | GET /api/v1/aftersales |
| `/sourcing1688` | GET /api/v1/sourcing-1688 |
| `/allocation` | GET /api/v1/allocation |
| `/exceptions` | GET /api/v1/exceptions |
| `/decision` | GET /api/v1/decision |
| `/ai` | POST /api/v1/ai/* |
| `/agents` / `/agentos` | GET /api/v1/agents / /agentos |
| `/settings` | GET /api/v1/setting |
| `/support` | GET /api/v1/support/* |
| `/support/templates` | GET /api/v1/support/templates |
| `/notifications` | GET /api/v1/notification |
| `/operation-logs` | GET /api/v1/operationlog |
| `/search` | GET /api/v1/search |
| `/reports` | GET /api/v1/report |
| `/image-gen` | GET /api/v1/imagegen |
| `/import-batches` | GET /api/v1/importbatch |
| `/purchase` | GET /api/v1/purchase |
| `/purchase/suggestions` | — |
| `/exchange-rates` | GET /api/v1/exchange-rates |
| `/product-analysis` | GET /api/v1/product-analysis |
| `/sourcing` | POST /api/v1/sourcing/fetch, GET /api/v1/sourcing/recommendations |
| `/metabolism` | GET /api/v1/metabolism |

### 前端开发命令

```bash
cd frontend-next
npm run dev                           # 启动开发服务器（localhost:3000）
npm run build                         # 生产构建
npm run lint                          # ESlint 检查（已知有未修复的 lint errors）
npm test                              # Vitest 单元测试
npx playwright test                   # E2E 测试
```

### 前端关键依赖

| 依赖 | 用途 |
|------|------|
| React 19 + Next.js 16 | 框架 |
| Ant Design 6 | UI 组件库 |
| TanStack React Query 5 | 服务端状态管理 |
| Zustand 5 | 客户端全局状态 |
| dayjs | 日期处理 |
| cmdk | 命令面板 |
| reconnecting-websocket | WebSocket 重连 |

---

## 七、后端开发（Go 1.25，Gin + GORM）

### 项目结构

```
backend-go/
├── cmd/server/main.go        # 入口
├── internal/
│   ├── auth/                 # JWT 认证
│   ├── rbac/                 # 角色权限
│   ├── ai/                   # AI 编排（Orchestrator + AgentRegistry + Trace）
│   ├── agent/                # Agent 服务 & 动作
│   ├── agentos/              # AgentOS 驾驶舱
│   ├── aios/                 # AIOS 基础设施
│   │   ├── toolregistry/     # 工具注册中心（核心！）
│   │   ├── llmgateway/       # LLM 网关（路由/缓存/回退）
│   │   ├── runtime/          # Agent 运行时（生命周期/资源/健康检查）
│   │   ├── knowledge/        # 知识引擎
│   │   ├── memory/           # 记忆系统
│   │   ├── ipc/              # Agent 间通信
│   │   ├── guardrails/       # 安全护栏
│   │   ├── observability/    # 可观测性
│   │   ├── pipeline/         # 管道链
│   │   └── sdk/              # Agent 定义 SDK（YAML manifest）
│   ├── common/               # 工具函数（分页/排序等）
│   ├── config/               # 配置管理
│   ├── domain/               # 所有业务领域模块
│   ├── httpx/                # 路由注册 + 中间件
│   ├── platform/             # 基础设施
│   │   ├── eventbus/         # 事件总线（pub/sub）
│   │   ├── command/          # 命令分发器
│   │   ├── toolbridge/       # 工具执行桥接（plugin driver）
│   │   └── scheduler/        # 定时任务
│   ├── realtime/             # WebSocket Hub + extension handler
│   └── response/             # 统一响应格式
├── migrations/               # SQL 迁移
└── configs/config.yaml       # 配置文件
```

### 全部领域模块（`internal/domain/`）

```
actionpolicy     aftersales       agentrule        allocation       brand
category         dashboard        decision         entropy          evolution
exceptions       exchangerate     finance          imagegen         importbatch
integrations     inventory        listing          listingtask      notification
operationlog     order            orderimport      personalrule     platform
platformfee      price            productanalysis  purchase         report
search           settings         settlement       shipping         sku
logistics        sourcing         sourcing1688     supplier         support          trustscore
```

### 开发命令

```bash
cd backend-go

# 运行开发服务器
go run cmd/server/main.go

# 编译
go build -o bin/server cmd/server/main.go

# 测试（全部）
go test ./...

# 测试（单个包）
go test -v ./internal/domain/order/

# 静态分析
go vet ./...
```

### 测试数据库辅助

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})  // in-memory SQLite, per-call isolation
    svc := NewService(db, logger)
}
```

### 配置

| 环境变量 | 配置路径 |
|----------|----------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `database.*` |
| `JWT_SECRET` | `jwt.secret` |
| `SERVER_PORT` | `server.port` |
| `REDIS_ADDR` / `REDIS_PASSWORD` | `redis.*` |
| `SENTRY_DSN` | `sentry.dsn` |
| `CORS_ALLOWED_ORIGINS` | `cors.allowed_origins` |
| `METRICS_ENABLED` | `metrics.enabled` |

---

## 八、基础设施

### 测试/生产服务器

| 项目 | 信息 |
|------|------|
| IP | `118.196.42.156` |
| OS | Ubuntu 24.04 LTS |
| 部署方式 | Docker + systemd |
| 域名 | http://118.196.42.156 |

**访问地址**

| 服务 | 地址 |
|------|------|
| 前端 | http://118.196.42.156 |
| API | http://118.196.42.156/api |
| 健康检查 | http://118.196.42.156/api/health |

**部署的服务**

| 服务 | 方式 | 端口 |
|------|------|------|
| PostgreSQL 15 | Docker | 5432 |
| Go Backend | systemd | 8080 |
| Next.js | Docker | 3000 |
| Nginx (反向代理) | Docker | 80 |

**更新后端流程**

```bash
# 1. 本机交叉编译 AMD64
cd backend-go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/lingmirror-server ./cmd/server/main.go

# 2. 上传到服务器（密钥登录，无需密码）
scp /tmp/lingmirror-server lingmirror:/opt/lingmirror/backend/server

# 3. 重启
ssh lingmirror "systemctl restart lingmirror-backend"
```

> **SSH 配置**: `~/.ssh/config` 中已配置 `Host lingmirror`，指向 `root@118.196.42.156`，密钥登录。

### Docker 本地开发

```bash
# 启动全部服务
docker compose up -d

# 仅启动数据库
docker compose up -d db

# 遗留栈（Python + Vue，已冻结）
docker compose -f docker-compose.legacy.yml up -d
```

### PostgreSQL 15

- Host: `localhost:5432`（本地）/ `118.196.42.156:5432`（服务器）
- 数据库名: `multisell`
- 迁移文件: `backend-go/migrations/`
- 生产服务器 SSH 部署路径: `/opt/lingmirror/deploy/`

---

## 九、技能 (Skills)

Slash command 形式的技能，可调用：

| 命令 | 用途 |
|------|------|
| `/browse` | 启动浏览器做 QA/页面分析 |
| `/plan` | 架构规划 |
| `/design-review` | 设计评审 |
| `/review` | 代码审查 |
| `/code-review` | 代码审查（针对性更高） |
| `/investigate` | 调查问题 |
| `/qa` | 系统 QA |
| `/ship` | 发布流程（PR → merge → deploy） |
| `/context-save` / `/context-restore` | 保存/恢复工作上下文 |
| `/spec` | 写 Spec |
| `/office-hours` | 产品思路梳理 |

---

## 十、代码规范

### 模块模式

所有 domain 模块遵循统一布局：

```
internal/domain/xxx/
├── routes.go     # Gin 路由注册
├── handler.go    # HTTP 请求/响应映射
├── service.go    # 业务逻辑
└── model.go      # GORM 模型 + 请求/响应结构
```

### 中间件栈（执行顺序）

```
CORS → RequestID → Metrics（可选）→ RecoveryWithSentry → Audit → Auth（仅 /api/v1 组）
```

### 响应格式

```go
response.Success(c, data)                       // {"code":0, "message":"ok", "data":...}
response.Error(c, http.StatusBadRequest, msg)   // {"code":400, "message":msg}
response.Paginated(c, data, total, page, size)  // + pagination fields
response.InternalError(c, err)                  // 500
```

### 平台基础设施

| 组件 | 用途 |
|------|------|
| Event Bus | pub/sub with glob topic matching（`order.*`） |
| Command Dispatcher | 桥接 Agent 决策到 domain service |
| ToolBridge | 插件驱动的工具执行桥接（plugin driver 模式） |
| Scheduler | 定期任务（5min - 6hr） |
| WebSocket Hub | 实时推送 AI 输出 |

### 遗留代码

- `backend/`（Python/FastAPI）— **已冻结**，仅参考
- `frontend/`（Vue 3）— **已冻结**，仅参考
- 不在遗留代码中做新功能

---

## 开始工作 Checklist

- [ ] 读了 AGENT_CAPABILITIES.md，知道所有可用能力
- [ ] 需要用 CodeGraph 时直接调 `mcp__codegraph__codegraph_explore`
- [ ] 需要调试/测试 UI 时用 Chrome DevTools MCP
- [ ] 需要调 API 时查上方端点列表
- [ ] 需要加新工具时，通过 `toolregistry.Tool` 注册
- [ ] 需要改新功能前确认是 active stack（Go + Next.js），不是 legacy（Python + Vue）
