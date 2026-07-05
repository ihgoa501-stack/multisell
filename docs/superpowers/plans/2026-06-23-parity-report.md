> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror 路由 Parity 报告

> 生成时间：2026-06-23
> 旧后端：FastAPI (`/Users/lc/multisell/backend/app/`)，统一前缀 `/api`
> 新后端：Go Gin (`/Users/lc/multisell/backend-go/internal/`)，统一前缀 `/api/v1`
> 旧前端：Vue Router (`/Users/lc/multisell/frontend/src/router/`)
> 新前端：Next.js App Router (`/Users/lc/multisell/frontend-next/src/app/`)

---

## 关键发现摘要

| 维度 | 旧端 | 新端 | 精确匹配率 | 语义匹配率 | 缺失 high 路由 |
|------|------|------|-----------|-----------|---------------|
| 后端 API | 290 条 (FastAPI) | 287 条 (Go) | ~30% | ~57% | **25 条** |
| 前端页面 | 54 条 (Vue) | 49 条 (Next) | ~50% | ~65% | — |

**最大风险**：Go 后端将 `/api` 改为 `/api/v1`，同时大量路径做了单复数/命名重构（`/orders`→`/order`、`/settlements`→`/settlement`、`/1688-collect`→`/sourcing1688`、`/image-gen`→`/imagegen`、`/operation-logs`→`/operationlog` 等），导致前端 API 调用几乎全部需要更新。Agent/AgentOS 模块路由覆盖严重不足（旧 45 条→新 10 条）。

---

## 后端 API Parity

### 旧 FastAPI 路由总数：290
### 新 Go 路由总数：287
### 精确匹配率：~30%（88/290）
### 语义匹配率：~57%（165/290，含路径重构但有对应功能）

> **匹配率说明**：精确匹配指规范化后（剥离 `/api` vs `/api/v1`、`{param}`→`:param`）路径+method 完全一致；语义匹配含路径重构但有对应 Go 端点。完全无 Go 对应的旧路由约 125 条。

### 系统性差异（影响所有路由）

| 差异类型 | FastAPI | Go | 影响范围 |
|---------|---------|-----|---------|
| **API 前缀** | `/api` | `/api/v1` | 全局 — 前端 baseURL 需改 |
| **路径参数语法** | `{param}` | `:param` | 全局 — 框架差异，不影响匹配 |
| **单复数重构** | `/orders` | `/order` | order/settlement/notification/decision 等模块 |
| **连字符去除** | `/image-gen` | `/imagegen` | image-gen 模块 |
| **模块重命名** | `/1688-collect` | `/sourcing1688` | sourcing 模块 |
| **模块重命名** | `/operation-logs` | `/operationlog` | operation-log 模块 |
| **模块重命名** | `/import` | `/importbatch` | import-batch 模块 |

---

### Go 缺失的旧路由（必须补齐）

#### High 严重度（auth / order / listing / finance）

| 旧路由 | Method | 旧文件:行号 | 说明 |
|--------|--------|------------|------|
| /api/auth/register | POST | app/auth/router.py:27 | 用户注册完全缺失，Go 仅有 login+refresh |
| /api/orders/{order_id}/shipping-quote | POST | app/order/router.py:98 | 订单运费快照绑定缺失 |
| /api/orders/{order_id}/profit-inputs | PUT | app/order/router.py:122 | 订单利润输入更新缺失 |
| /api/products/{product_id}/publish/{platform_id} | POST | app/listing/router.py:26 | 商品发布到平台缺失 |
| /api/products/{product_id}/listings | GET | app/listing/router.py:75 | 商品发布状态查询缺失 |
| /api/listing-tasks/from-decisions | POST | app/listing/router.py:151 | 从决策创建上架任务缺失 |
| /api/listing-tasks/{task_id}/recheck | POST | app/listing/router.py:178 | 重新检查上架任务缺失 |
| /api/listing-tasks/{task_id}/cancel | POST | app/listing/router.py:192 | 取消上架任务缺失 |
| /api/listing-tasks/{task_id}/publish | POST | app/listing/router.py:206 | 发布上架任务缺失 |
| /api/listing-tasks/{task_id}/execute | POST | app/listing_task/router.py:143 | 执行上架任务缺失 |
| /api/listing-tasks/{task_id}/retry-failed | POST | app/listing_task/router.py:180 | 重试失败任务缺失 |
| /api/listing-tasks/{task_id}/items/{item_id}/retry | POST | app/listing_task/router.py:204 | 重试单条目缺失 |
| /api/exchange-rates | GET | app/exchange_rate/router.py:20 | 汇率列表完全缺失（整个模块无 Go 对应） |
| /api/exchange-rates | POST | app/exchange_rate/router.py:41 | 汇率创建缺失 |
| /api/exchange-rates/{from_currency}/{to_currency} | PUT | app/exchange_rate/router.py:29 | 汇率更新缺失 |
| /api/exchange-rates/{rate_id} | DELETE | app/exchange_rate/router.py:53 | 汇率删除缺失 |
| /api/finance/profit-summary | GET | app/finance/router.py:84 | 利润汇总报表缺失 |
| /api/finance/orders/{order_id}/ledger/rebuild | POST | app/finance/router.py:101 | 重建订单利润账本缺失 |
| /api/finance/orders/{order_id}/ledger | GET | app/finance/router.py:114 | 订单利润账本明细缺失 |
| /api/finance/orders/{order_id}/profit | GET | app/finance/router.py:127 | 订单利润汇总缺失 |
| /api/finance/mock | POST | app/finance/router.py:143 | 模拟财务数据缺失 |
| /api/settlements/{settlement_id}/items | POST | app/settlement/router.py:143 | 添加结算明细缺失 |
| /api/settlements/{settlement_id}/items | GET | app/settlement/router.py:173 | 结算明细列表缺失 |
| /api/settlements/items/{item_id}/reconciliation | PUT | app/settlement/router.py:225 | 更新明细对账状态缺失 |
| /api/settlements/mock | POST | app/settlement/router.py:258 | 模拟结算数据缺失 |

#### Medium 严重度（search / dashboard）

| 旧路由 | Method | 旧文件:行号 | 说明 |
|--------|--------|------------|------|
| /api/dashboard/stats | GET | app/dashboard/router.py:14 | Go 有 /dashboard/overview 但路径不同 |
| /api/reports/product-stats | GET | app/dashboard/router.py:23 | Go 有 /report/ 但路径结构不同 |
| /api/reports/platform-stats | GET | app/dashboard/router.py:32 | Go 有 /report/ 但路径结构不同 |
| /api/search | GET | app/search/router.py:14 | Go 有 /search 但需验证参数兼容性 |

#### Low 严重度（其他模块）

| 旧路由 | Method | 旧文件:行号 | 说明 |
|--------|--------|------------|------|
| /api/agents/decisions | GET | app/agent/router.py:48 | Agent 决策日志缺失 |
| /api/agents/decisions/{decision_id}/feedback | POST | app/agent/router.py:68 | 决策反馈缺失 |
| /api/agents/actions | GET | app/agent/router.py:87 | 待执行操作列表缺失 |
| /api/agents/actions/{action_id}/execute | POST | app/agent/router.py:120 | 确认执行操作缺失 |
| /api/agents/actions/{action_id}/reject | POST | app/agent/router.py:138 | 拒绝操作缺失 |
| /api/agents/dashboard | GET | app/agent/router.py:150 | 运营驾驶舱缺失 |
| /api/agents/rules | GET/POST | app/agent/router.py:159,173 | 个人规则 CRUD 缺失 |
| /api/agents/rules/{rule_id} | PUT/DELETE | app/agent/router.py:183,198 | 规则更新/删除缺失 |
| /api/agents/profile | GET/PUT | app/agent/router.py:210,219 | Honcho 用户模型缺失 |
| /api/agents/episodes | GET | app/agent/router.py:231 | Episode 列表缺失 |
| /api/agents/events/routes | GET | app/agent/router.py:253 | 事件路由列表缺失 |
| /api/agents/events/emit | POST | app/agent/router.py:260 | 手动触发事件缺失 |
| /api/agents/schedules | GET | app/agent/router.py:275 | Agent 调度配置缺失 |
| /api/agents/schedules/{agent_id} | GET/PUT | app/agent/router.py:282,293 | 单 Agent 调度缺失 |
| /api/agents/schedules/{agent_id}/trigger | POST | app/agent/router.py:305 | 手动触发调度缺失 |
| /api/agents/evolution/overview | GET | app/agent/router.py:320 | 自治等级总览缺失 |
| /api/agents/evolution/nudge/pending | GET | app/agent/router.py:329 | 待处理 Nudge 缺失 |
| /api/agents/evolution/nudge/{nudge_id}/respond | POST | app/agent/router.py:338 | 响应 Nudge 缺失 |
| /api/agents/evolution/generate-nudges | POST | app/agent/router.py:353 | 生成 Nudge 缺失 |
| /api/agents/evolution/{agent_id} | GET | app/agent/router.py:365 | Agent 进化详情缺失 |
| /api/agents/evolution/{agent_id}/stage | PUT | app/agent/router.py:377 | 变更自治阶段缺失 |
| /api/agents/{agent_id}/decide | POST | app/agent/router.py:404 | Agent 决策缺失 |
| /api/agentos/control-center | GET | app/agentos/router.py:28 | 总控台（Go 有 /agentos 但功能可能不同） |
| /api/agentos/squads | GET | app/agentos/router.py:71 | 团队列表缺失 |
| /api/agentos/templates | GET | app/agentos/router.py:81 | 模板列表缺失 |
| /api/agentos/operations | GET | app/agentos/router.py:91 | 操作审计日志缺失 |
| /api/agentos/action-proposals | POST | app/agentos/router.py:123 | 创建动作提案缺失 |
| /api/agentos/action-proposals/{proposal_id}/approve | POST | app/agentos/router.py:137 | 审批提案缺失 |
| /api/agentos/action-proposals/{proposal_id}/reject | POST | app/agentos/router.py:161 | 拒绝提案缺失 |
| /api/agentos/action-proposals/{proposal_id}/execute | POST | app/agentos/router.py:183 | 执行提案缺失 |
| /api/agentos/action-proposals/{proposal_id}/review | POST | app/agentos/router.py:205 | 复盘提案缺失 |
| /api/agentos/work-items/{item_id}/status | PATCH | app/agentos/router.py:233 | 更新 WorkItem 状态缺失 |
| /api/agentos/work-items/{item_id}/approve | POST | app/agentos/router.py:254 | 审批 WorkItem 缺失 |
| /api/agentos/work-items/{item_id}/reject | POST | app/agentos/router.py:275 | 拒绝 WorkItem 缺失 |
| /api/agentos/agents/upgrade-candidates | GET | app/agentos/router.py:299 | 升级候选缺失 |
| /api/agentos/agents/{agent_id}/upgrade | POST | app/agentos/router.py:309 | 执行升级缺失 |
| /api/agentos/agents/{agent_id}/downgrade | POST | app/agentos/router.py:326 | 执行降级缺失 |
| /api/agentos/agents/{agent_id}/detail | GET | app/agentos/router.py:346 | Agent 详情缺失 |
| /api/entropy/dashboard | GET | app/agent/entropy/router.py:16 | 熵驾驶舱缺失 |
| /api/entropy/defend | POST | app/agent/entropy/router.py:25 | 执行防守动作缺失 |
| /api/entropy/health | GET | app/agent/entropy/router.py:34 | 规则健康评分缺失 |
| /api/entropy/spc | GET | app/agent/entropy/router.py:43 | SPC 控制状态缺失 |
| /api/entropy/changes | GET | app/agent/entropy/router.py:52 | 变更日志缺失 |
| /api/agent-actions | POST/GET | app/agent_actions/router.py:24,38 | 动作提案 CRUD 缺失 |
| /api/agent-actions/{action_id} | GET | app/agent_actions/router.py:51 | 提案详情缺失 |
| /api/agent-actions/{action_id}/approve | POST | app/agent_actions/router.py:63 | 审批通过缺失 |
| /api/agent-actions/{action_id}/reject | POST | app/agent_actions/router.py:80 | 驳回缺失 |
| /api/agent-actions/{action_id}/mark-executed | POST | app/agent_actions/router.py:101 | 标记执行缺失 |
| /api/settings/llm | GET/PUT | app/agent/config_router.py:14,29 | LLM 配置缺失 |
| /api/upload | POST | app/common/upload.py:11 | 文件上传缺失 |
| /api/products/batch/status | POST | app/core/router.py:46 | 批量修改商品状态缺失 |
| /api/products/batch/delete | POST | app/core/router.py:69 | 批量删除商品缺失 |
| /api/products/export | GET | app/core/router.py:96 | 导出商品 Excel 缺失 |
| /api/products/export-template | GET | app/core/router.py:125 | 下载导入模板缺失 |
| /api/products/import | POST | app/core/router.py:139 | Excel 导入商品缺失 |
| /api/products/{product_id}/duplicate | POST | app/core/router.py:269 | 复制商品缺失 |
| /api/products/{product_id}/detail | GET | app/core/router.py:316 | 商品聚合详情缺失 |
| /api/products/{product_id}/ai-enhance | POST | app/core/router.py:332 | AI 优化商品缺失 |
| /api/products/{product_id}/skus/generate | POST | app/sku/router.py:95 | 生成 SKU 缺失 |
| /api/shipping/channels/{channel_id}/zones | GET/POST | app/shipping/router.py:184,194 | 物流区域（嵌套路径）缺失 |
| /api/shipping/zones/{zone_id} | DELETE | app/shipping/router.py:213 | 删除物流区域缺失 |
| /api/shipping/channels/{channel_id}/rules | GET/POST | app/shipping/router.py:236,246 | 报价规则（嵌套路径）缺失 |
| /api/shipping/rules/{rule_id} | PUT | app/shipping/router.py:265 | 更新报价规则缺失 |
| /api/shipping/import-rules | POST | app/shipping/router.py:309 | 导入物流报价表缺失 |
| /api/shipping/calculate | POST | app/shipping/router.py:339 | 运费计算（Go 有 /shipping/quote 替代） |
| /api/shipping/bills/import | POST | app/shipping/router.py:355 | 导入运费账单缺失 |
| /api/shipping/bills | GET | app/shipping/router.py:383 | 账单批次列表缺失 |
| /api/shipping/bills/{batch_id} | GET | app/shipping/router.py:395 | 账单批次详情缺失 |
| /api/shipping/bills/{batch_id}/items | GET | app/shipping/router.py:409 | 账单行列表缺失 |
| /api/shipping/reconciliation/summary | GET | app/shipping/router.py:422 | 对账汇总缺失 |
| /api/shipping/bills/{batch_id}/reconcile | POST | app/shipping/router.py:433 | 对账缺失 |
| /api/shipping/bills/items/{item_id}/resolve | POST | app/shipping/router.py:451 | 手动解决差异缺失 |
| /api/platforms/{platform_id}/backfill-orders | POST | app/platform/router.py:118 | 回填历史订单缺失 |
| /api/notifications/check | POST | app/notification/router.py:76 | 触发预警检查缺失 |
| /api/alert-rules/initialize | POST | app/notification/router.py:89 | 初始化预警规则缺失 |
| /api/inventory/alerts | GET | app/inventory/router.py:23 | 库存预警列表缺失 |
| /api/inventory/check | POST | app/inventory/router.py:171 | 库存预占检查缺失 |
| /api/decisions/prelisting | POST | app/decision/router.py:25 | 上架前决策缺失（Go /decision 是 CRUD 完全不同） |
| /api/decisions/prelisting/compare | POST | app/decision/router.py:38 | 多平台决策对比缺失 |
| /api/decisions/prelisting/batch | POST | app/decision/router.py:105 | 批量决策缺失 |
| /api/decisions/prelisting/batch/template | GET | app/decision/router.py:163 | 下载模板缺失 |
| /api/decisions/prelisting/batch/preview | POST | app/decision/router.py:177 | 预览缺失 |
| /api/decisions/prelisting/batch/export | POST | app/decision/router.py:192 | 导出缺失 |
| /api/operation-logs/modules | GET | app/operation_log/router.py:47 | 获取模块列表缺失 |
| /api/brands/all | GET | app/brand/router.py:93 | 所有品牌（下拉）缺失 |
| /api/rbac/users | GET | app/rbac/router.py:224 | 用户列表缺失 |
| /api/rbac/users/{user_id}/permissions | GET | app/rbac/router.py:213 | 获取用户权限缺失 |
| /api/products/{product_id}/suppliers | GET | app/supplier/router.py:158 | 查询商品供应商缺失 |
| /api/order-import/csv | POST | app/order_import/router.py:46 | CSV 导入缺失 |
| /api/order-import/mock | POST | app/order_import/router.py:82 | 模拟订单数据缺失 |
| /api/platform-integrations/adapters | GET | app/platform_integrations/router.py:30 | 列出适配器缺失 |
| /api/inventory/warehouse/{sku_id} | GET | app/allocation/router.py:118 | SKU 各仓库库存缺失 |
| /api/inventory/allocate | POST | app/allocation/router.py:128 | 分配库存缺失 |
| /api/inventory/auto-allocate/{sku_id} | POST | app/allocation/router.py:151 | 自动分配缺失 |
| /api/warehouses/mock | POST | app/allocation/router.py:167 | 模拟仓库数据缺失 |

---

### Go 多出的新路由（新增，OK）

> Go 后端新增了大量 CRUD 标准化路由和 AI 编排路由。以下列出主要新增类别。

| 新路由模式 | Method | Go 文件 | 说明 |
|-----------|--------|---------|------|
| /api/v1/auth/refresh | POST | auth/routes.go:19 | Token 刷新（新增） |
| /api/v1/ai/chat | POST | ai/routes.go:20 | AI 对话（全新模块） |
| /api/v1/ai/run | POST | ai/routes.go:21 | 运行 Agent |
| /api/v1/ai/traces | GET | ai/routes.go:22 | 追踪列表 |
| /api/v1/ai/traces/{trace_id} | GET | ai/routes.go:29 | 追踪详情 |
| /api/v1/ai/actions | GET/POST | ai/routes.go:23,26 | 动作列表/创建 |
| /api/v1/ai/actions/{id} | GET | ai/routes.go:30 | 动作详情 |
| /api/v1/ai/actions/{id}/approve | POST | ai/routes.go:31 | 审批动作 |
| /api/v1/ai/actions/{id}/reject | POST | ai/routes.go:32 | 拒绝动作 |
| /api/v1/ai/actions/{id}/execute | POST | ai/routes.go:33 | 执行动作 |
| /api/v1/ai/actions/{id}/review | POST | ai/routes.go:34 | 复盘动作 |
| /api/v1/ai/agents | GET | ai/routes.go:24 | Agent 花名册 |
| /api/v1/ai/agents/specs | GET | ai/routes.go:25 | Agent 规格 |
| /api/v1/agents (POST) | POST | agent/routes.go:23 | 创建 Agent |
| /api/v1/agents/{id}/actions | POST | agent/routes.go:27 | 执行 Agent 动作 |
| /api/v1/agentos/status | GET | agentos/routes.go:17 | AgentOS 状态 |
| /api/v1/agentos/autonomy | GET | agentos/routes.go:19 | 自治概览 |
| /api/v1/stores (CRUD) | GET/POST/PUT/DELETE | platform/routes.go:23-29 | 店铺管理（全新） |
| /api/v1/listings (CRUD) | GET/POST/PUT/DELETE | listing/routes.go:16-20 | 发布 CRUD 标准化 |
| /api/v1/listings/{id}/sync | POST | listing/routes.go:22 | 同步发布状态 |
| /api/v1/listing-tasks/{id} (PUT) | PUT | listingtask/routes.go:19 | 更新任务 |
| /api/v1/listing-tasks/{id}/items (CRUD) | GET/POST/PUT/DELETE | listingtask/routes.go:23-26 | 任务条目管理 |
| /api/v1/shipping/quote | POST | shipping/routes.go:17 | 运费报价（替代 calculate） |
| /api/v1/shipping/providers/{id} (GET) | GET | shipping/routes.go:21 | 供应商详情 |
| /api/v1/shipping/channels/{id} (GET) | GET | shipping/routes.go:28 | 渠道详情 |
| /api/v1/shipping/zones (GET/POST) | GET/POST | shipping/routes.go:34,35 | 区域列表/创建（顶层化） |
| /api/v1/shipping/rules (GET/POST) | GET/POST | shipping/routes.go:39,40 | 规则列表/创建（顶层化） |
| /api/v1/shipping/bill-batches (CRUD) | GET/POST/DELETE | shipping/routes.go:44-48 | 账单批次（重构） |
| /api/v1/order/summary | GET | order/routes.go:17 | 订单汇总 |
| /api/v1/order/{id} (PUT/DELETE) | PUT/DELETE | order/routes.go:20,21 | 更新/删除订单 |
| /api/v1/order-import/{id}/process | POST | orderimport/routes.go:22 | 处理导入 |
| /api/v1/order-import/{id}/complete | POST | orderimport/routes.go:23 | 完成导入 |
| /api/v1/settlement/summary | GET | settlement/routes.go:17 | 结算汇总 |
| /api/v1/settlement/{id}/reconcile | POST | settlement/routes.go:24 | 执行对账 |
| /api/v1/finance/summary | GET | finance/routes.go:17 | 财务汇总 |
| /api/v1/finance/ledger | GET | finance/routes.go:18 | 账本列表 |
| /api/v1/finance/accounts/{id} (GET/PUT/DELETE) | — | finance/routes.go:23-25 | 账户 CRUD 补齐 |
| /api/v1/decision (CRUD) | GET/POST/PUT/DELETE | decision/routes.go:16-21 | 决策 CRUD（新语义） |
| /api/v1/decision/{id}/approve | POST | decision/routes.go:22 | 审批决策 |
| /api/v1/decision/{id}/reject | POST | decision/routes.go:23 | 拒绝决策 |
| /api/v1/allocation/cost/batches (CRUD) | GET/POST | allocation/routes.go:29-31 | 成本批次（全新） |
| /api/v1/exceptions (POST) | POST | exceptions/routes.go:18 | 创建异常 |
| /api/v1/exceptions/{id} (PUT/DELETE) | PUT/DELETE | exceptions/routes.go:19,22 | 更新/删除异常 |
| /api/v1/notification (POST) | POST | notification/routes.go:20 | 创建通知 |
| /api/v1/notification/{id} (GET) | GET | notification/routes.go:19 | 通知详情 |
| /api/v1/notification/alert-rules (POST/DELETE) | POST/DELETE | notification/routes.go:26,28 | 预警规则创建/删除 |
| /api/v1/dashboard/overview | GET | dashboard/routes.go:16 | 总览（替代 stats） |
| /api/v1/dashboard/orders | GET | dashboard/routes.go:17 | 订单看板 |
| /api/v1/dashboard/inventory | GET | dashboard/routes.go:18 | 库存看板 |
| /api/v1/dashboard/exceptions | GET | dashboard/routes.go:19 | 异常看板 |
| /api/v1/search/recent | GET | search/routes.go:17 | 最近搜索 |
| /api/v1/imagegen (CRUD) | GET/POST/PUT/DELETE | imagegen/routes.go:17-21 | 生图记录 CRUD |
| /api/v1/imagegen/canvas (CRUD) | GET/POST/PUT/DELETE | imagegen/routes.go:24-28 | 画布 CRUD 标准化 |
| /api/v1/imagegen/templates/{id}/use | POST | imagegen/routes.go:35 | 使用模板 |
| /api/v1/importbatch (CRUD) | GET/POST/PUT/DELETE | importbatch/routes.go:16-20 | 导入批次 CRUD |
| /api/v1/importbatch/{id}/rows | GET | importbatch/routes.go:21 | 批次行列表 |
| /api/v1/operationlog (POST) | POST | operationlog/routes.go:18 | 创建日志 |
| /api/v1/operationlog/{id} (GET) | GET | operationlog/routes.go:17 | 日志详情 |
| /api/v1/platform-integrations (CRUD) | GET/POST/PUT/DELETE | integrations/routes.go:17-23 | 集成 CRUD 标准化 |
| /api/v1/platform-integrations/{id}/sync | POST | integrations/routes.go:25 | 同步集成 |
| /api/v1/platform-integrations/{id}/categories | GET/POST | integrations/routes.go:26,27 | 类目映射（重构） |
| /api/v1/platform-integrations/{id}/attributes | GET/POST | integrations/routes.go:28,29 | 属性映射（重构） |
| /api/v1/aftersales/summary | GET | aftersales/routes.go:17 | 售后汇总 |
| /api/v1/aftersales/{id} (PUT/DELETE) | PUT/DELETE | aftersales/routes.go:20,21 | 更新/删除退货 |
| /api/v1/sourcing1688/summary | GET | sourcing1688/routes.go:17 | 1688 汇总 |
| /api/v1/sourcing1688/{id} (PUT) | PUT | sourcing1688/routes.go:20 | 更新候选 |
| /api/v1/report/sales | GET | report/routes.go:16 | 销售报表 |
| /api/v1/report/profit | GET | report/routes.go:17 | 利润报表 |
| /api/v1/report/inventory | GET | report/routes.go:18 | 库存报表 |
| /api/v1/report/settlement | GET | report/routes.go:19 | 结算报表 |
| /api/v1/report/platform-fee | GET | report/routes.go:20 | 平台费用报表 |
| /api/v1/inventory (GET) | GET | inventory/routes.go:17 | 库存列表 |
| /api/v1/inventory/{id} (GET) | GET | inventory/routes.go:18 | 库存详情 |
| /api/v1/inventory/{id}/lock | POST | inventory/routes.go:20 | 锁定库存 |
| /api/v1/inventory/{id}/unlock | POST | inventory/routes.go:21 | 解锁库存 |
| /api/v1/inventory/logs | GET | inventory/routes.go:22 | 库存日志 |
| /api/v1/inventory/warehouses (CRUD) | GET/POST/PUT/DELETE | inventory/routes.go:25-29 | 仓库管理（从 allocation 移入） |
| /api/v1/inventory/sku/{sku_id}/warehouses | GET | inventory/routes.go:32 | SKU 仓库库存 |
| /api/v1/products/{product_id}/specs/{id} (PUT/DELETE) | PUT/DELETE | sku/routes.go:26,27 | 规格更新/删除 |
| /api/v1/products/{product_id}/specs/{spec_id}/values | POST | sku/routes.go:30 | 创建规格值 |
| /api/v1/spec-values/{id} (PUT/DELETE) | PUT/DELETE | sku/routes.go:39,40 | 规格值更新/删除 |
| /api/v1/skus (POST) | POST | sku/routes.go:48 | 创建 SKU |
| /api/v1/skus/{id} (DELETE) | DELETE | sku/routes.go:50 | 删除 SKU |
| /api/v1/prices/{id} (GET/PUT/DELETE) | — | price/routes.go:18,20,21 | 价格 CRUD 补齐 |
| /api/v1/product-suppliers (PUT/DELETE) | PUT/DELETE | supplier/routes.go:29,30 | 商品-供应商更新/删除 |
| /api/v1/platform-fee (GET/POST) | GET/POST | platformfee/routes.go:19,20 | 费用规则（顶层化） |
| /api/v1/platform-fee/{id} (GET/PUT/DELETE) | — | platformfee/routes.go:21-23 | 费用规则 CRUD |
| /ws | GET | httpx/router.go:118 | WebSocket（全新） |

---

### Method 不一致

| 旧路由 | 旧 Method | Go 路由 | Go Method | 说明 |
|--------|----------|---------|----------|------|
| /api/exceptions/{exception_id}/assign | POST | /api/v1/exceptions/{id}/assign | **PUT** | POST→PUT |
| /api/exceptions/{exception_id}/resolve | POST | /api/v1/exceptions/{id}/resolve | **PUT** | POST→PUT |
| /api/agentos/work-items/{item_id}/status | PATCH | （缺失） | — | Go 无 PATCH 路由 |
| /api/notifications/{notification_id}/read | PUT | /api/v1/notification/{id}/read | PUT | 一致（但路径 notifications→notification） |
| /api/notifications/read-all | PUT | /api/v1/notification/read-all | PUT | 一致（但路径不同） |
| /api/products/{product_id}/publish/{platform_id} | POST | /api/v1/listings/{id}/publish | POST | 语义不同（Go 按 listing ID 发布，旧按 product+platform） |

---

## 前端路由 Parity

### 旧 Vue 路由总数：54（含基础路由 + 模块路由，不含 /redesign 变体）
### 新 Next 路由总数：49（含 (main) 组 + (auth) 组 + 根页面）

> **说明**：Vue 还有 ~46 条 `/redesign/*` 变体路由（redesign 预览版），未计入总数。Next 使用文件系统路由，`(main)` 为路由组（不影响 URL）。

### Next 缺失的旧路由

| 旧 Vue 路由 | 旧文件:行号 | 严重度 | 说明 |
|------------|------------|--------|------|
| /products/:id/edit | router/index.ts:37 | medium | 编辑商品页缺失（Next 仅有 /products/[id]） |
| /products/:id/skus | router/index.ts:55 | medium | SKU 管理页缺失（Next 有 /sku 但非嵌套） |
| /products/:id/prices | router/index.ts:61 | medium | 价格管理页缺失 |
| /products/:id/inventory | router/index.ts:67 | medium | 库存管理页缺失 |
| /inventory/alerts | router/index.ts:103 | low | 库存预警页缺失（Next 仅有 /inventory） |
| /listing-tasks/:id | modules/listing.ts:16 | high | 上架任务详情页缺失 |
| /settlements/:id | modules/settlement.ts:16 | high | 结算详情页缺失 |
| /decisions/prelisting | modules/decision.ts:5 | high | 决策工作台缺失（Next 有 /decision 但语义不同） |
| /decisions/prelisting/batch | modules/decision.ts:16 | high | 批量决策页缺失 |
| /1688-sourcing | modules/sourcing1688.ts:5 | medium | 1688 选品页缺失（Next 有 /sourcing1688 命名不同） |
| /orders/:id | modules/order.ts:20 | high | 订单详情页缺失 |
| /image-gen-canvas | modules/imageGenCanvas.ts:5 | low | 画布编辑器缺失（Next 有 /image-gen/canvas 路径不同） |
| /agents/dashboard | modules/agent.ts:17 | low | Agent 看板缺失 |
| /agents/llm-settings | modules/agent.ts:23 | low | LLM 设置缺失（Next 有 /settings/llm 路径不同） |
| /agents/rules | modules/agent.ts:35 | low | Agent 规则页缺失 |
| /users | modules/rbac.ts:8 | medium | 用户管理缺失（Next 有 /settings/rbac 路径不同） |
| /roles | modules/rbac.ts:14 | medium | 角色管理缺失（Next 有 /settings/rbac 路径不同） |
| /shipping/manage | modules/shipping.ts:5 | low | 物流管理缺失（Next 有 /shipping 合并） |
| /shipping/calculator | modules/shipping.ts:11 | low | 运费计算缺失（Next 有 /shipping 合并） |
| /shipping/bill-reconciliation | modules/shipping.ts:17 | low | 运费对账缺失（Next 有 /shipping 合并） |
| /agentos/control-center | modules/agentos.ts:16 | low | 总控台缺失（Next 有 /agentos 合并） |
| /agentos/squads | modules/agentos.ts:38 | low | Agent 团队页缺失 |
| /agentos/autonomy | modules/agentos.ts:49 | low | 自治管理页缺失 |
| /agentos/agents/:agentId | modules/agentos.ts:60 | low | AgentOS Agent 详情缺失 |
| /listings/ai-workbench | modules/listingTask.ts:5 | medium | AI 上架台缺失（Next 有 /listing-tasks/workbench 路径不同） |

### Next 多出的新路由（新增，OK）

| 新 Next 路由 | 文件路径 | 说明 |
|------------|---------|------|
| / | src/app/page.tsx | 根页面 |
| /settings | src/app/(main)/settings/page.tsx | 设置中心 |
| /settings/rbac | src/app/(main)/settings/rbac/page.tsx | RBAC 设置（合并 users+roles） |
| /settings/llm | src/app/(main)/settings/llm/page.tsx | LLM 设置（从 agents 移出） |
| /listings/create | src/app/(main)/listings/create/page.tsx | 创建发布 |
| /import-batches | src/app/(main)/import-batches/page.tsx | 导入批次管理 |
| /agents/evolution | src/app/(main)/agents/evolution/page.tsx | Agent 进化 |
| /agents/[id]/trace/[traceId] | src/app/(main)/agents/[id]/trace/[traceId]/page.tsx | Agent 追踪详情 |
| /ai | src/app/(main)/ai/page.tsx | AI 对话 |
| /actions/[id] | src/app/(main)/actions/[id]/page.tsx | 动作详情 |
| /listing-tasks/workbench | src/app/(main)/listing-tasks/workbench/page.tsx | 上架工作台 |
| /sku | src/app/(main)/sku/page.tsx | SKU 独立管理页 |
| /decision | src/app/(main)/decision/page.tsx | 决策管理（新语义） |
| /sourcing1688 | src/app/(main)/sourcing1688/page.tsx | 1688 选品（重命名） |
| /platform-fees | src/app/(main)/platform-fees/page.tsx | 平台费用 |
| /allocation/cost | src/app/(main)/allocation/cost/page.tsx | 成本分摊 |
| /search | src/app/(main)/search/page.tsx | 全局搜索 |
| /exceptions | src/app/(main)/exceptions/page.tsx | 异常工作台 |
| /platform-integrations | src/app/(main)/platform-integrations/page.tsx | 平台集成 |
| /operation-logs | src/app/(main)/operation-logs/page.tsx | 操作日志 |

---

## 建议

### 必须补齐的 High 严重度后端路由清单（25 条）

#### 1. Auth（1 条）
- `POST /api/v1/auth/register` — 用户注册

#### 2. Order（2 条）
- `POST /api/v1/orders/{order_id}/shipping-quote` — 订单运费快照绑定
- `PUT /api/v1/orders/{order_id}/profit-inputs` — 订单利润输入

#### 3. Listing（9 条）
- `POST /api/v1/products/{product_id}/publish/{platform_id}` — 商品发布到平台
- `GET /api/v1/products/{product_id}/listings` — 商品发布状态
- `POST /api/v1/listing-tasks/from-decisions` — 从决策创建上架任务
- `POST /api/v1/listing-tasks/{task_id}/recheck` — 重新检查
- `POST /api/v1/listing-tasks/{task_id}/cancel` — 取消任务
- `POST /api/v1/listing-tasks/{task_id}/publish` — 发布任务
- `POST /api/v1/listing-tasks/{task_id}/execute` — 执行任务
- `POST /api/v1/listing-tasks/{task_id}/retry-failed` — 重试失败
- `POST /api/v1/listing-tasks/{task_id}/items/{item_id}/retry` — 重试单条目

#### 4. Finance（13 条）
- `GET/POST/PUT/DELETE /api/v1/exchange-rates` — 汇率管理（整个模块缺失，4 条）
- `GET /api/v1/finance/profit-summary` — 利润汇总
- `POST /api/v1/finance/orders/{order_id}/ledger/rebuild` — 重建账本
- `GET /api/v1/finance/orders/{order_id}/ledger` — 账本明细
- `GET /api/v1/finance/orders/{order_id}/profit` — 利润汇总
- `POST /api/v1/finance/mock` — 模拟数据
- `POST/GET /api/v1/settlements/{settlement_id}/items` — 结算明细（2 条）
- `PUT /api/v1/settlements/items/{item_id}/reconciliation` — 对账状态
- `POST /api/v1/settlements/mock` — 模拟结算

### 其他关键建议

1. **统一 API 前缀**：决定 Go 是否保留 `/api/v1` 还是回退到 `/api`。如果保留 `/v1`，前端所有 API 调用必须更新 baseURL。建议保留 `/api/v1` 作为版本化策略，但需同步更新前端。

2. **路径命名统一**：Go 的单复数不一致（`/order` vs `/orders`、`/settlement` vs `/settlements`、`/notification` vs `/notifications`）需统一。建议全部使用复数（RESTful 惯例）。

3. **Agent/AgentOS 模块严重不足**：旧 FastAPI 有 45 条 Agent+AgentOS+Entropy 路由，Go 仅 10 条。这是功能覆盖最大的缺口，需要专项补齐。

4. **Decision 模块语义不同**：旧 `/decisions/prelisting` 是上架前经营决策（含批量、Excel 导入导出），Go `/decision` 是通用 CRUD。两者语义完全不同，需重新设计。

5. **文件上传缺失**：`POST /api/upload` 在 Go 中完全缺失，前端图片/文件上传功能将不可用。

6. **Shipping 模块路径重构**：旧嵌套路径 `/shipping/channels/{id}/zones` 被 Go 扁平化为 `/shipping/zones`，但丢失了 `PUT /shipping/rules/{id}` 等路由。需补齐。

7. **Method 规范化**：Go 将 `POST /exceptions/{id}/assign` 改为 `PUT`，需确认前端同步更新。建议统一使用 POST for actions（非幂等操作）。

8. **前端缺失页面**：Next 缺少订单详情、结算详情、上架任务详情等关键详情页。`/products/[id]/edit`、`/products/[id]/skus` 等嵌套管理页也需补齐。

9. **前端路由合并策略**：Next 将 `/shipping/manage`、`/shipping/calculator`、`/shipping/bill-reconciliation` 合并为 `/shipping` 单页。需确认是否使用 Tab 切换，否则功能不可用。同理 `/agentos` 合并了 5 个子页面。

10. **WebSocket**：Go 新增 `/ws` 端点，前端 Next 需接入实时通信。

---

## 附录：路由统计明细

### FastAPI 各模块路由数

| 模块 | 路由数 | 文件 |
|------|--------|------|
| agent | 28 | app/agent/router.py |
| agentos | 17 | app/agentos/router.py |
| shipping | 24 | app/shipping/router.py |
| core (products) | 13 | app/core/router.py |
| platform_integrations | 10 | app/platform_integrations/router.py |
| settlement | 10 | app/settlement/router.py |
| rbac | 14 | app/rbac/router.py |
| notification | 9 | app/notification/router.py |
| finance | 9 | app/finance/router.py |
| allocation | 11 | app/allocation/router.py |
| image_gen | 14 | app/image_gen/router.py |
| image_gen/canvas | 4 | app/image_gen/canvas_router.py |
| listing | 8 | app/listing/router.py |
| listing_task | 7 | app/listing_task/router.py |
| order | 6 | app/order/router.py |
| sku | 6 | app/sku/router.py |
| platform | 6 | app/platform/router.py |
| import_batch | 6 | app/import_batch/router.py |
| agent_actions | 6 | app/agent_actions/router.py |
| brand | 6 | app/brand/router.py |
| aftersales | 7 | app/aftersales/router.py |
| price | 5 | app/price/router.py |
| decision | 6 | app/decision/router.py |
| exceptions | 5 | app/exceptions/router.py |
| platform_fee | 5 | app/platform_fee/router.py |
| inventory | 5 | app/inventory/router.py |
| entropy | 5 | app/agent/entropy/router.py |
| supplier | 8 | app/supplier/router.py |
| order_import | 4 | app/order_import/router.py |
| exchange_rate | 4 | app/exchange_rate/router.py |
| category | 4 | app/category/router.py |
| auth | 3 | app/auth/router.py |
| dashboard | 3 | app/dashboard/router.py |
| operation_log | 2 | app/operation_log/router.py |
| agent/config | 2 | app/agent/config_router.py |
| search | 1 | app/search/router.py |
| upload | 1 | app/common/upload.py |
| health | 1 | app/main.py:163 |
| **合计** | **290** | |

### Go 各模块路由数

| 模块 | 路由数 | 文件 |
|------|--------|------|
| ai | 13 | internal/ai/routes.go |
| sku (products+spec-values+skus) | 18 | internal/domain/sku/routes.go |
| shipping | 22 | internal/domain/shipping/routes.go |
| rbac | 14 | internal/rbac/routes.go |
| imagegen | 16 | internal/domain/imagegen/routes.go |
| inventory | 12 | internal/domain/inventory/routes.go |
| integrations | 11 | internal/domain/integrations/routes.go |
| allocation | 11 | internal/domain/allocation/routes.go |
| notification | 11 | internal/domain/notification/routes.go |
| platform (platforms+stores) | 10 | internal/domain/platform/routes.go |
| aftersales | 10 | internal/domain/aftersales/routes.go |
| listingtask | 9 | internal/domain/listingtask/routes.go |
| supplier (suppliers+product-suppliers) | 9 | internal/domain/supplier/routes.go |
| finance | 9 | internal/domain/finance/routes.go |
| orderimport | 8 | internal/domain/orderimport/routes.go |
| price (prices+skus) | 8 | internal/domain/price/routes.go |
| decision | 8 | internal/domain/decision/routes.go |
| exceptions | 7 | internal/domain/exceptions/routes.go |
| order | 7 | internal/domain/order/routes.go |
| settlement | 7 | internal/domain/settlement/routes.go |
| listing | 7 | internal/domain/listing/routes.go |
| sourcing1688 | 8 | internal/domain/sourcing1688/routes.go |
| platformfee | 6 | internal/domain/platformfee/routes.go |
| category | 6 | internal/domain/category/routes.go |
| agent | 6 | internal/agent/routes.go |
| importbatch | 6 | internal/domain/importbatch/routes.go |
| auth | 3 | internal/auth/routes.go |
| operationlog | 3 | internal/domain/operationlog/routes.go |
| dashboard | 4 | internal/domain/dashboard/routes.go |
| agentos | 4 | internal/agentos/routes.go |
| search | 2 | internal/domain/search/routes.go |
| report | 5 | internal/domain/report/routes.go |
| brand | 5 | internal/domain/brand/routes.go |
| health | 1 | internal/httpx/router.go:59 |
| ws | 1 | internal/httpx/router.go:118 |
| **合计** | **287** | |
