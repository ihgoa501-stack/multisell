# 凌镜 LingMirror 前端页面与路由

> **注意**: 本文档的内容已合并到 [reference-module-catalog.md](reference-module-catalog.md)（前端页面章节）。添加新前端路由时，请更新该文件。
> 本文保留仅用于向后兼容参考，不再作为前端路由的事实源。
>
> 最后更新：2026-06-26
> 框架：Next.js / React / TypeScript / Ant Design
> 入口：`frontend-next/src/app/`

---

## 当前路由架构

当前活跃前端是 `frontend-next/`，使用 Next.js App Router：

```text
frontend-next/src/app/
├── (auth)/login/page.tsx
├── (main)/layout.tsx
├── (main)/**/page.tsx
└── page.tsx
```

主应用布局：

- `frontend-next/src/app/(main)/layout.tsx`
- `frontend-next/src/components/layout/AppSidebar.tsx`
- `frontend-next/src/components/layout/AppHeader.tsx`
- `frontend-next/src/components/layout/CommandPalette.tsx`
- `frontend-next/src/components/copilot/CopilotPanel.tsx`
- `frontend-next/src/components/auth/AuthGuard.tsx`

菜单来源：

- `frontend-next/src/config/menu.ts`

API client：

- `frontend-next/src/lib/api-client.ts`

旧 Vue 前端 `frontend/` 已暂停，不再参与当前路由判断。

## 页面清单

### 认证

| 路由 | 文件 | 说明 |
|---|---|---|
| `/login` | `src/app/(auth)/login/page.tsx` | 登录页 |

### 总览

| 路由 | 说明 |
|---|---|
| `/` | 根入口 |
| `/dashboard` | 运营数据总览 |
| `/ai` | AI 指挥中心 |

### 商品管理

| 路由 | 说明 |
|---|---|
| `/products` | 商品列表 |
| `/products/create` | 新建商品 |
| `/products/[id]` | 商品详情 |
| `/products/[id]/suppliers` | 商品供应商管理 |
| `/categories` | 分类管理 |
| `/brands` | 品牌管理 |
| `/sku` | SKU 管理 |
| `/inventory` | 库存管理 |
| `/suppliers` | 供应商管理 |

### 平台与发布

| 路由 | 说明 |
|---|---|
| `/platforms` | 平台管理 |
| `/platform-integrations` | 平台集成 |
| `/platform-integrations/[id]/ozon-products` | Ozon 商品列表 |
| `/listings` | 刊登管理 |
| `/listings/create` | 创建刊登 |
| `/listing-tasks` | 上架任务 |
| `/listing-tasks/[id]` | 上架任务详情 |
| `/listing-tasks/workbench` | 上架任务工作台 |

### 订单、物流与售后

| 路由 | 说明 |
|---|---|
| `/orders` | 订单列表 |
| `/orders/[id]` | 订单详情 |
| `/order-import` | 订单导入 |
| `/shipping` | 物流管理 |
| `/platform-fees` | 平台费用 |
| `/aftersales` | 售后 |
| `/metabolism` | 代谢评分（M1） |

### 财务与经营决策

| 路由 | 说明 |
|---|---|
| `/finance` | 财务总览 |
| `/experiments` | 经营实验案件列表与创建 |
| `/experiments/[experimentId]` | 端到端经营实验事实链与 Owner 决策 |
| `/settlement` | 结算列表 |
| `/settlement/[id]` | 结算详情 |
| `/decision` | 决策总览 |
| `/decision/prelisting` | 上架前决策 |
| `/business-decisions/[id]` | Owner 经营决策案卷；AI 建议与 Owner 决定分开保存 |
| `/allocation` | 分配 |
| `/allocation/cost` | 成本分摊 |

### AgentOS

| 路由 | 说明 |
|---|---|
| `/agentos` | AgentOS 控制台 |
| `/agentos/work-items` | 工作队列 |
| `/agents` | Agent 列表 |
| `/agents/[id]` | Agent 详情 |
| `/agents/[id]/trace/[traceId]` | Trace 详情 |
| `/agents/actions` | Agent Action 中心 |
| `/agents/entropy` | 熵监控 |
| `/agents/evolution` | 进化建议 |
| `/agents/trust` | 信任与自主度 |
| `/actions` | 统一 Action 列表 |
| `/actions/[id]` | Action 详情 |

### 运营工具

| 路由 | 说明 |
|---|---|
| `/exceptions` | 异常工作台 |
| `/notifications` | 通知中心 |
| `/image-gen` | AI 生图 |
| `/image-gen/canvas` | 生图画布 |
| `/import-batches` | 批量导入 |
| `/operation-logs` | 操作日志 |
| `/search` | 搜索 |
| `/reports` | 报表 |
| `/sourcing1688` | 1688 采购 |
| `/sourcing` | AI 选品 |

### 设置

| 路由 | 说明 |
|---|---|
| `/settings` | 系统设置 |
| `/settings/llm` | LLM 配置 |
| `/settings/rbac` | 权限管理 |
| `/settings/policy` | 审批策略 |

## 菜单覆盖

`frontend-next/src/config/menu.ts` 当前定义 47 个菜单入口。2026-06-26 复核结果：所有菜单入口均有对应 `page.tsx`。额外详情页面（如 `/[id]`、`/[id]/suppliers`、`/[id]/ozon-products`、`/templates`）不出现在菜单中但路由正常。

菜单入口仅覆盖主导航页面；详情页、创建页和工作台子页面不一定出现在菜单中。

## 路由保护

主应用页面由 `AuthGuard` 包裹：

- 无 token 时跳转 `/login`
- 有 token 时加载 RBAC 权限
- 侧边菜单按 `frontend-next/src/stores/permission-store.ts` 的权限结果过滤

## API 路径规则

后端业务 API 统一注册在 `/api/v1`。前端默认：

```text
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

因此 `apiClient` 调用应传入 `/v1/*` 路径，例如：

```ts
apiClient.get('/v1/dashboard/overview')
apiClient.post('/v1/ai/chat', payload)
```

不要在新代码中使用旧 FastAPI 风格的 `/api/*` 或 Vue 前端路径。

## 验证

```bash
cd frontend-next
npm run build
npm test
npm run lint
```

2026-06-24 状态：

- `npm run build` 通过
- `npm test` 通过，75 tests
- `npm run lint` 未通过，需修复 `any`、未使用变量和 `AuthGuard` hooks 规则
