# 前端测试报告

> 测试日期：2026-06-24
> 范围：`frontend-next/`
> 框架：Next.js / React / TypeScript / Ant Design

## 测试概览

| 检查 | 命令 | 状态 |
|---|---|---:|
| 单元测试 | `npm test` | 通过 |
| 生产构建 | `npm run build` | 通过 |
| 静态检查 | `npm run lint` | 失败 |

## 单元测试

```bash
cd frontend-next
npm test
```

结果：

- Test Files: 11 passed
- Tests: 75 passed
- Runner: Vitest

当前测试主要覆盖共享 UI / CRUD 组件：

- `CrudListPage`
- `BatchActionBar`
- `ConfirmDialog`
- `DetailDrawer`
- `EmptyState`
- `FilterBar`
- `LoadingSkeleton`
- `PageContainer`
- `PageHeader`
- `StatCard`
- `StatusTag`

## 生产构建

```bash
cd frontend-next
npm run build
```

结果：通过。

构建输出确认当前 App Router 页面覆盖 dashboard、商品、订单、结算、财务、物流、发布、AgentOS、设置等主业务页面，以及以下动态详情页：

- `/products/[id]`
- `/orders/[id]`
- `/settlement/[id]`
- `/listing-tasks/[id]`
- `/agents/[id]`
- `/agents/[id]/trace/[traceId]`
- `/actions/[id]`

Sentry warning：

- 未配置 Sentry auth token，跳过 release/source map 上传。
- 这不阻塞 build，但生产发布需要按部署策略配置或显式关闭上传。

## Lint

```bash
cd frontend-next
npm run lint
```

结果：失败。

主要问题：

- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-unused-vars`
- `react-hooks/set-state-in-effect`

优先修复文件：

- `frontend-next/src/components/auth/AuthGuard.tsx`
- `frontend-next/src/app/(auth)/login/page.tsx`
- `frontend-next/src/app/(main)/decision/prelisting/page.tsx`
- `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx`
- `frontend-next/src/app/(main)/settlement/[id]/page.tsx`

## 页面覆盖

当前菜单入口来自：

- `frontend-next/src/config/menu.ts`

2026-06-24 复核：

- 菜单入口数：41
- App Router 页面数：50+
- 菜单入口缺失目标页面：0

详见：

- `docs/FRONTEND_PAGES_AND_ROUTING.md`
- `docs/UI_FRAMEWORK_GAP_ANALYSIS.md`

## 下一步

1. 修复 lint。
2. 为关键业务页面增加 API mock 测试。
3. 补 Playwright/E2E：登录、Dashboard、商品列表、AI chat、订单详情、Agent action 审批。
4. 增加 API path smoke test，确保前端请求命中 `/api/v1/*`。
