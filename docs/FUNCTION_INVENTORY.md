# 凌镜 LingMirror — 全站功能清单

> **注意**: 本文档的内容已合并到 [reference-module-catalog.md](reference-module-catalog.md)。添加新模块或路由时，请更新该文件。
> 本文保留仅用于向后兼容参考，不再作为功能清单的事实源。
>
> 更新时间：2026-06-30
> 范围：当前活跃新栈 `backend-go/` + `frontend-next/`

---

## 说明

本文档记录当前新技术栈的页面与后端能力边界，用于需求对齐、测试覆盖和新人 onboarding。

当前事实源：

- 前端页面：`frontend-next/src/app/`
- 前端菜单：`frontend-next/src/config/menu.ts`
- 前端 API client：`frontend-next/src/lib/api-client.ts`
- 后端路由汇总：`backend-go/internal/httpx/router.go`
- 后端业务模块：`backend-go/internal/domain/`

旧 `backend/` 和 `frontend/` 目录仅作历史参考，不再作为功能清单依据。

## 页面覆盖

### 主导航入口

| 分组 | 页面 |
|---|---|
| 总览 | `/dashboard`、`/ai` |
| 商品管理 | `/products`、`/categories`、`/brands`、`/sku`、`/inventory`、`/suppliers` |
| 销售管理 | `/platforms`、`/platform-integrations`、`/listings`、`/listing-tasks` |
| 订单物流 | `/orders`、`/order-import`、`/shipping`、`/platform-fees` |
| 财务 | `/finance`、`/settlement`、`/decision`、`/allocation`、`/allocation/cost` |
| AgentOS | `/agentos`、`/agents`、`/agents/actions`、`/agents/evolution`、`/agents/entropy`、`/agentos/work-items`、`/agents/trust` |
| 运营 | `/exceptions`、`/notifications`、`/image-gen`、`/import-batches`、`/operation-logs`、`/search`、`/reports`、`/aftersales`、`/sourcing1688` |
| 设置 | `/settings`、`/settings/llm`、`/settings/rbac`、`/settings/policy` |

### 详情与工作台页面

| 路由 | 说明 |
|---|---|
| `/products/[id]` | 商品详情 |
| `/products/create` | 新建商品 |
| `/orders/[id]` | 订单详情 |
| `/settlement/[id]` | 结算详情 |
| `/listing-tasks/[id]` | 上架任务详情 |
| `/listing-tasks/workbench` | 上架任务工作台 |
| `/listings/create` | 创建刊登 |
| `/agents/[id]` | Agent 详情 |
| `/agents/[id]/trace/[traceId]` | Agent trace 详情 |
| `/actions/[id]` | Action 详情 |
| `/image-gen/canvas` | 生图画布 |

2026-06-24 复核：`frontend-next/src/config/menu.ts` 的 41 个菜单入口均有对应页面。

## 后端能力

后端统一入口为 `/api/v1`，公共健康检查为 `/api/health`。受保护业务路由通过 JWT middleware 注册。

### 平台基础

| 能力 | 模块 |
|---|---|
| 认证与 token refresh | `auth` |
| RBAC 角色与权限 | `rbac` |
| 请求审计 | `httpx/middleware`、`operationlog` |
| 指标与健康检查 | `httpx/middleware` |
| WebSocket 实时广播 | `realtime` |

### 商品与供应链

| 能力 | 模块 |
|---|---|
| 商品、规格、SKU | `sku` |
| 分类 | `category` |
| 品牌 | `brand` |
| 价格 | `price` |
| 库存、锁定、仓库库存 | `inventory` |
| 供应商、商品供应商关系 | `supplier` |
| 1688 选品 | `sourcing1688` |

### 发布与平台集成

| 能力 | 模块 |
|---|---|
| 平台 / 店铺管理 | `platform` |
| 平台集成与映射 | `integrations` |
| 刊登记录 | `listing` |
| 上架任务与任务条目 | `listingtask` |
| ⇨ 平台发布钩子 | `listingtask.PublishHook` — ExecuteTask 成功后调用 adapter.Publish 推送到 Ozon/Shopee 等平台。失败时 task 状态转 failed 可重试，成功时 publish 结果合并到 item result（不覆盖 Prism 数据）。通过 auditSvc.LogStructured 记录发布审计。 |
| 平台费用规则 | `platformfee` |

### 订单、物流与售后

| 能力 | 模块 |
|---|---|
| 订单列表、详情、状态流转 | `order` |
| 订单导入 | `orderimport` |
| 物流供应商、渠道、区域、报价规则、账单批次 | `shipping` |
| 售后 | `aftersales` |
| 异常工作台 | `exceptions` |

### 财务与经营决策

| 能力 | 模块 |
|---|---|
| 财务账户、流水、利润汇总 | `finance` |
| 结算与对账 | `settlement` |
| 上架前决策 | `decision` |
| 库存与成本分配 | `allocation` |
| 汇率 | `exchangerate` |
| 报表 | `report` |
| Dashboard 聚合 | `dashboard` |

### AI / AgentOS

| 能力 | 模块 |
|---|---|
| AI chat、run、trace、action | `ai` |
| Agent 列表、详情、执行入口 | `agent` |
| AgentOS 总控台、工作队列、自治概览 | `agentos` |
| Agent 规则 | `agentrule` |
| Action policy | `actionpolicy` |
| 熵监控与防御 | `entropy` |
| 进化建议 | `evolution` |
| 信任分 | `trustscore` |

### 运营工具

| 能力 | 模块 |
|---|---|
| 通知与预警规则 | `notification` |
| 全局搜索 | `search` |
| 批量导入 | `importbatch` |
| 生图记录、画布、模板 | `imagegen` |

## 当前联调风险

### API 前缀（已修复 ✅）

此前列出的 17 处缺失 `/v1` 前缀的 API 调用（`/ai/actions`、`/policy/rules`、`/evolution/nudges`、`/trust-scores/summary` 等）已于 2026-06-25 修复，跨 6 个前端文件。

当前前端所有 `apiClient` 调用均统一使用 `/api/v1` 前缀。

### 路径命名

新后端中部分模块同时存在复数资源路由和单数动作路由，例如：

- `/api/v1/listing-tasks`：任务 CRUD 与 item CRUD
- `/api/v1/listing-task/:task_id/execute`：任务执行链路动作

前端新增调用时应以 Go route 文件为准，不要沿用旧 Vue/FastAPI 路径。

## 验证快照

2026-07-03：

| 命令 | 结果 |
|---|---|
| `cd backend-go && go test ./...` | 通过 |
| `cd backend-go && go vet ./...` | 通过 |
| `cd frontend-next && npm test` | 通过 |
| `cd frontend-next && npm run build` | 通过 |
| `cd frontend-next && npm run lint` | 通过 |
