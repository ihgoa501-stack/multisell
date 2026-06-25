# AgentOS Development Guide

> 最后更新：2026-06-24
> 适用范围：当前 Go + Next 新栈

## 当前入口

后端：

- `backend-go/internal/agentos/`
- `backend-go/internal/agent/`
- `backend-go/internal/ai/`
- `backend-go/internal/domain/actionpolicy/`
- `backend-go/internal/domain/agentrule/`
- `backend-go/internal/domain/entropy/`
- `backend-go/internal/domain/evolution/`
- `backend-go/internal/domain/trustscore/`
- `backend-go/internal/platform/eventbus/`
- `backend-go/internal/platform/scheduler/`
- `backend-go/internal/platform/command/`

前端：

- `frontend-next/src/app/(main)/agentos/page.tsx`
- `frontend-next/src/app/(main)/agentos/work-items/page.tsx`
- `frontend-next/src/app/(main)/agents/page.tsx`
- `frontend-next/src/app/(main)/agents/[id]/page.tsx`
- `frontend-next/src/app/(main)/agents/[id]/trace/[traceId]/page.tsx`
- `frontend-next/src/app/(main)/agents/actions/page.tsx`
- `frontend-next/src/app/(main)/agents/entropy/page.tsx`
- `frontend-next/src/app/(main)/agents/evolution/page.tsx`
- `frontend-next/src/app/(main)/agents/trust/page.tsx`
- `frontend-next/src/app/(main)/actions/page.tsx`
- `frontend-next/src/app/(main)/actions/[id]/page.tsx`

旧 `backend/app/agentos/`、`frontend/src/views/agentos/` 只作为历史参考。

## 后端路由

所有业务 API 通过 `/api/v1` 暴露。

主要路由：

| 模块 | 路由前缀 | 文件 |
|---|---|---|
| AI runtime | `/api/v1/ai` | `backend-go/internal/ai/routes.go` |
| Agent | `/api/v1/agents` | `backend-go/internal/agent/routes.go` |
| AgentOS | `/api/v1/agentos` | `backend-go/internal/agentos/routes.go` |
| Action policy | `/api/v1/policy` | `backend-go/internal/domain/actionpolicy/routes.go` |
| Agent rules | `/api/v1/agent-rules` | `backend-go/internal/domain/agentrule/routes.go` |
| Entropy | `/api/v1/entropy` | `backend-go/internal/domain/entropy/routes.go` |
| Evolution | `/api/v1/evolution` | `backend-go/internal/domain/evolution/routes.go` |
| Trust score | `/api/v1/trust-scores` | `backend-go/internal/domain/trustscore/routes.go` |

## AgentOS 后端职责

### `ai`

负责：

- chat
- run agent
- trace list/detail
- action list/detail
- action approve/reject/execute/review
- streaming response

### `agent`

负责：

- Agent 列表
- Agent 详情
- Agent 动作入口
- evolution / entropy 聚合入口

### `agentos`

负责：

- 总控台 overview
- status
- work items
- autonomy overview

### `platform/eventbus` + `scheduler` + `command`

负责：

- 周期任务 tick
- Agent 决策链路触发
- action command dispatch
- outbox/event 可靠性演进基础

## 前端页面职责

| 页面 | 说明 |
|---|---|
| `/agentos` | AgentOS 控制台 |
| `/agentos/work-items` | 工作队列 |
| `/agents` | Agent 列表 |
| `/agents/[id]` | Agent 详情 |
| `/agents/[id]/trace/[traceId]` | trace 详情 |
| `/agents/actions` | Agent action 中心 |
| `/agents/entropy` | 熵监控 |
| `/agents/evolution` | 进化建议 |
| `/agents/trust` | 信任与自主度 |
| `/actions` | 统一 action 列表 |
| `/actions/[id]` | action 详情 |

## API 调用规则

前端默认 API base：

```text
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

因此调用必须使用 `/v1/*`：

```ts
apiClient.get('/v1/agentos')
apiClient.get('/v1/agentos/work-items')
apiClient.get('/v1/ai/actions')
apiClient.post('/v1/ai/actions/123/approve', payload)
```

不要新增无 `/v1` 的调用。

## 当前已知风险

### API 前缀漂移

当前仍有少量前端调用使用 `/ai/actions`、`/evolution/nudges`、`/trust-scores/summary` 等无 `/v1` 路径。应统一修正。

### 测试覆盖不足

AgentOS 当前缺少足够 focused tests 和 E2E：

- action approve / reject / execute
- work item lifecycle
- scheduler tick -> AI run
- event bus chained triggers
- trust score recalculation
- entropy dashboard

### 前端 lint 未通过

AgentOS 相关页面也应跟随全站 lint 修复。

## 新能力开发流程

1. 明确能力属于 `ai`、`agent`、`agentos` 还是 domain 模块。
2. 后端先补 focused Go tests。
3. 在对应 `routes.go` 注册 `/api/v1/*` 路由。
4. 在 service 中保持业务规则和事务边界。
5. 前端通过 `apiClient` 使用 `/v1/*` 路径。
6. 如需要菜单入口，更新 `frontend-next/src/config/menu.ts`。
7. 验证：

```bash
cd backend-go && go test ./...
cd frontend-next && npm test
cd frontend-next && npm run build
```

## 设计原则

- 高风险动作必须可审计。
- 自动执行必须受 trust score、policy、权限和预算约束。
- Agent 建议和执行动作要能追溯到 trace / evidence / action。
- 前端页面应优先展示可执行状态和风险，而不是单纯展示模型输出。
