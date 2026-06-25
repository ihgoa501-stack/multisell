# Permissions And Audit Guide

> 最后更新：2026-06-24
> 适用范围：当前 Go 后端 `backend-go/`

## 目标

凌镜的权限和审计目标是：

- 未登录用户不能访问受保护接口。
- 普通用户只能执行自己角色拥有权限的操作。
- 管理员和系统级账号可以按业务规则获得更高权限。
- 所有关键写操作都留下操作日志，便于追责、排查和 Agent 行为复盘。

## 核心文件

| 能力 | 文件 |
|---|---|
| JWT middleware | `backend-go/internal/httpx/middleware/auth.go` |
| 全局审计 middleware | `backend-go/internal/httpx/middleware/audit.go` |
| RBAC routes / service / model | `backend-go/internal/rbac/` |
| 认证 routes / service / model | `backend-go/internal/auth/` |
| 操作日志 domain | `backend-go/internal/domain/operationlog/` |
| 路由挂载 | `backend-go/internal/httpx/router.go` |

## 认证规则

公共接口：

- `/api/health`
- `/api/v1/health`
- `/api/v1/auth/login`
- `/api/v1/auth/register`
- `/api/v1/auth/refresh`

受保护接口：

- `backend-go/internal/httpx/router.go` 中 `protected := api.Group("")`
- `protected.Use(middleware.Auth(cfg))`
- 所有注册到 `protected` 的 domain、RBAC、AI、AgentOS 路由都要求 `Authorization: Bearer <token>`

JWT middleware 行为：

- 缺少 `Authorization` header 返回 401
- 非 `Bearer <token>` 格式返回 401
- token 无效或过期返回 401
- token 有效时把 `user_id` 写入 Gin context

## RBAC 数据模型

当前 RBAC 表：

- `role`
- `permission`
- `user_role`
- `role_permission`

Go model 定义在：

- `backend-go/internal/rbac/model.go`

RBAC service 支持：

- 角色 CRUD
- 权限 CRUD
- 用户角色分配
- 角色权限分配
- 聚合用户权限：`GetUserPermissions(userID)`

当前权限聚合逻辑通过：

```text
user_role -> role_permission -> permission.code
```

## 前端权限入口

前端权限 store：

- `frontend-next/src/stores/permission-store.ts`

菜单过滤：

- `frontend-next/src/components/layout/AppSidebar.tsx`
- `frontend-next/src/config/menu.ts`

页面保护：

- `frontend-next/src/components/auth/AuthGuard.tsx`

## 审计日志规则

全局审计 middleware：

- 文件：`backend-go/internal/httpx/middleware/audit.go`
- 挂载位置：`backend-go/internal/httpx/router.go`
- 记录对象：POST / PUT / PATCH / DELETE
- 跳过对象：GET / HEAD / OPTIONS、health checks

审计字段：

| 字段 | 来源 |
|---|---|
| `module` | 从 request path 提取，例如 `/api/v1/order/123` -> `order` |
| `action` | HTTP method + route template 生成 |
| `resource_id` | path param，优先 `:id`、`:trace_id`、`:action_id` 等 |
| `content` | path、query、status、截断后的 body |
| `operator` | Gin context 中的 `username` 或 `user_id` |
| `ip` | `c.ClientIP()` |
| `duration` | handler 执行耗时 |

审计写入是后台 goroutine，失败只打 warning，不阻塞业务响应。

## 接入新模块的步骤

1. 新路由挂到 `backend-go/internal/httpx/router.go` 的 protected group，除非明确是 public endpoint。
2. 在业务 handler/service 中保持清晰的状态变化边界。
3. 对复杂或高风险写操作，除全局 audit 外可显式写 operation log。
4. 新增 RBAC 权限时同步 `permission` seed / migration / 管理页面展示。
5. 前端菜单项如需权限控制，在 `frontend-next/src/config/menu.ts` 填 `permission`。
6. 添加 focused Go tests，至少覆盖未授权、授权成功和关键状态变化。

## 当前验证

2026-06-24 复核：

```bash
cd backend-go && go test ./...
cd backend-go && go vet ./...
```

结果均通过。

## 注意事项

- 不要只在前端隐藏按钮，后端必须强制校验鉴权和关键权限。
- 不要把平台 API key、用户密码 hash、token 等敏感数据写入审计 `content`。
- 不要为读请求写审计日志，除非业务上需要单独的安全审计。
- 不要在新代码中使用旧 FastAPI 的 `require_permission(...)` 或 `OperationLogService.log(...)` 写法；那只适用于 legacy `backend/`。
