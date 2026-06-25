# 凌镜 LingMirror Development Guide

> 最后更新：2026-06-24
> 适用范围：当前活跃新栈 `backend-go/` + `frontend-next/`

## 本地启动

### 1. 启动数据库

```bash
docker compose up -d db
```

默认开发数据库：

- 数据库：`multisell`
- 用户名：`postgres`
- 密码：`postgres`

### 2. 启动后端

```bash
cd backend-go
go run cmd/server/main.go
```

默认地址：

- Health check: `http://localhost:8080/api/health`
- API base: `http://localhost:8080/api/v1`

### 3. 启动前端

```bash
cd frontend-next
npm install
npm run dev -- --hostname 127.0.0.1 --port 3000
```

前端地址：

```text
http://localhost:3000
```

### 4. Docker 一键启动

```bash
docker compose up -d
```

旧栈只用于参考或回滚：

```bash
docker compose -f docker-compose.legacy.yml up -d
```

## 测试与验证

后端：

```bash
cd backend-go
go test ./...
go vet ./...
go build -o bin/server cmd/server/main.go
```

前端：

```bash
cd frontend-next
npm test
npm run build
npm run lint
```

当前已知状态：

- `go test ./...` 通过
- `go vet ./...` 通过
- `npm test` 通过
- `npm run build` 通过
- `npm run lint` 仍有错误，需要修复后才能作为绿色质量门禁

## 项目结构

```text
backend-go/
  cmd/server/main.go          后端入口
  configs/config.yaml         默认配置
  internal/
    auth/                     JWT 认证
    rbac/                     角色和权限
    httpx/                    Gin router 和 middleware
    domain/                   业务模块
    ai/ agent/ agentos/       AI 和 AgentOS
    platform/                 event bus / command / scheduler
    realtime/                 WebSocket
  migrations/                 SQL migrations

frontend-next/
  src/
    app/                      Next App Router 页面
    components/               共享组件
    config/menu.ts            侧边栏菜单
    lib/api-client.ts         API client
    stores/                   Zustand stores
    types/                    TypeScript 类型

docs/
  PROJECT_STATUS.md
  FRONTEND_PAGES_AND_ROUTING.md
  FUNCTION_INVENTORY.md
  PERMISSIONS_AND_AUDIT.md
  ROADMAP.md
```

## 后端模块约定

每个 Go 业务模块优先遵守这个结构：

```text
backend-go/internal/domain/<module>/
  routes.go
  handler.go
  service.go
  model.go
  *_test.go
```

约定：

- `routes.go` 只注册 Gin route group。
- `handler.go` 处理绑定、参数解析、HTTP 状态和响应。
- `service.go` 放业务规则、事务和 GORM 查询。
- `model.go` 放 GORM model、request、response。
- 新路由通过 `backend-go/internal/httpx/router.go` 挂到 `/api/v1`。
- 写操作必须能被审计：优先使用全局 audit middleware，必要时写 `operationlog`。

## 前端模块约定

页面放在：

```text
frontend-next/src/app/(main)/<module>/page.tsx
```

详情页示例：

```text
frontend-next/src/app/(main)/orders/[id]/page.tsx
frontend-next/src/app/(main)/agents/[id]/trace/[traceId]/page.tsx
```

共享组件放在：

```text
frontend-next/src/components/
```

API 调用统一走：

```text
frontend-next/src/lib/api-client.ts
```

路径规则：

- `NEXT_PUBLIC_API_URL` 默认是 `http://localhost:8080/api`
- 业务调用必须传 `/v1/*`
- 最终请求应是 `/api/v1/*`

示例：

```ts
apiClient.get('/v1/dashboard/overview')
apiClient.post('/v1/ai/chat', payload)
```

## 新功能开发流程

推荐流程：

1. 先确认功能应落在 `backend-go/`、`frontend-next/`，不要落到旧栈。
2. 阅读相关模块的 `routes.go`、`handler.go`、`service.go` 和页面。
3. 为后端业务规则加 focused Go tests。
4. 实现后端接口并确认 `/api/v1` contract。
5. 实现前端页面或组件，统一使用 `/v1/*` API path。
6. 跑局部测试。
7. 跑相关全量验证。
8. 更新文档。

## 交给其他模型时的提示词

可以把下面这段直接发给其他模型：

```text
你接手的是 LingMirror / MultiSell 项目。当前全站已迁移到 Go + Next 新技术栈。

先阅读：
- AGENTS.md
- docs/PROJECT_STATUS.md
- docs/ACTIVE_STACK_POLICY.md
- docs/DEVELOPMENT_GUIDE.md
- docs/FRONTEND_PAGES_AND_ROUTING.md

要求：
- 新功能只写到 backend-go/ 和 frontend-next/。
- backend/ 和 frontend/ 是旧栈，只能作参考、迁移、回滚或文档用途。
- 后端遵守 routes.go / handler.go / service.go / model.go 模块边界。
- 新 API 统一挂在 /api/v1。
- 前端 apiClient 调用必须使用 /v1/* 路径。
- 完成后至少运行相关 focused tests；共享面改动运行 go test ./...、npm test、npm run build，并尽量修复 npm run lint。
```
