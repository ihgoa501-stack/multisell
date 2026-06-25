# 凌镜 LingMirror Project Status

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

更新时间：2026-06-24

## 当前结论

凌镜已完成全站新技术栈迁移。当前唯一活跃开发线是：

- Backend: `backend-go/`，Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/`，Next.js / React / TypeScript / Ant Design
- API prefix: `/api/v1`
- Health check: `/api/health`

旧栈 `backend/`（Python / FastAPI）和 `frontend/`（Vue 3）已经暂停，只能用于行为对照、迁移参考、安全回滚或文档标注。新功能不得继续落到旧栈。

## 一句话说明

凌镜是跨境电商 AI AgentOS，核心流程是：

商品创建 -> SKU / 价格 / 库存维护 -> AI 优化与经营决策 -> 多平台发布 -> 订单、结算、财务、异常和 AgentOS 运营闭环。

## 入口与运行

| 项目 | 位置 |
|---|---|
| 后端入口 | `backend-go/cmd/server/main.go` |
| 后端路由汇总 | `backend-go/internal/httpx/router.go` |
| 后端配置 | `backend-go/configs/config.yaml`，支持环境变量覆盖 |
| 前端入口 | `frontend-next/src/app/` |
| 前端 API client | `frontend-next/src/lib/api-client.ts` |
| 前端菜单配置 | `frontend-next/src/config/menu.ts` |
| Docker 默认入口 | `docker-compose.yml` |

本地开发命令：

```bash
docker compose up -d db

cd backend-go
go run cmd/server/main.go

cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
```

## 当前覆盖

### 后端

`backend-go/internal/httpx/router.go` 在 `/api/v1` 下注册了认证、RBAC、Agent、AgentOS 和业务域路由。当前业务域包括：

- 商品基础：`category`、`brand`、`sku`、`price`、`inventory`、`supplier`
- 平台与发布：`platform`、`integrations`、`listing`、`listingtask`
- 订单履约：`order`、`orderimport`、`shipping`、`platformfee`、`aftersales`
- 财务经营：`finance`、`settlement`、`decision`、`allocation`、`report`、`exchangerate`
- 运营支撑：`dashboard`、`search`、`notification`、`exceptions`、`operationlog`、`importbatch`
- AI / AgentOS：`ai`、`agent`、`agentos`、`agentrule`、`entropy`、`evolution`、`trustscore`、`actionpolicy`
- 选品与生图：`sourcing1688`、`imagegen`
- 实时能力：WebSocket `/ws`

### 前端

`frontend-next/src/app/` 已按业务域迁移到 Next App Router。当前 build 输出覆盖：

- Dashboard / AI / AgentOS / Action Center
- 商品、SKU、分类、品牌、库存、供应商
- 平台、平台集成、刊登、刊登任务、刊登任务详情
- 订单、订单详情、订单导入、售后
- 物流、平台费用、结算、结算详情、财务、经营决策、成本分摊
- 异常、通知、生图、画布、批量导入、操作日志、搜索、报表
- 设置、LLM 配置、RBAC、审批策略

侧边栏菜单目前有 41 个入口，已确认都能匹配到 `frontend-next/src/app` 下的实际页面。

## 验证状态

2026-06-25 复核结果（本次并行修复后）：

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd backend-go && go test ./...` | 通过 | 34 个 Go 测试包，含新增 6 个 domain 模块测试（158 tests） |
| `cd backend-go && go vet ./...` | 通过 | 无 vet 输出 |
| `cd frontend-next && npm test` | 通过 | 12 个 test files，77 tests |
| `cd frontend-next && npm run build` | 通过 | Sentry auth token/source map 上传 warning 不阻塞 build |
| `cd frontend-next && npm run lint` | 1 error / 3 warnings | 剩余 1 error（AntdProvider.tsx setState in effect）和 3 个 unused var warning。较之前 16 errors 已大幅改善 |

## 本次修复内容（2026-06-25，4 Agent 并行执行）

### API 路径一致性 ✅ 已修复

之前在风险栏列出的缺失 `/v1` 前缀问题已全部修复：

- `/ai/actions` → `/v1/ai/actions`
- `/policy/rules` → `/v1/policy/rules`
- `/evolution/nudges/evaluate` → `/v1/evolution/nudges/evaluate`
- `/trust-scores/summary` → `/v1/trust-scores/summary`

共 17 处调用跨 6 个前端文件。

### 前端 lint

当前 `eslint` 剩余 1 error（AntdProvider.tsx setState in effect）和 3 个 unused var warning。较之前 16 errors / 22 warnings 已大幅改善，但仍有 1 个需修复。

### EventBus workerLoop 修复

`backend-go/internal/platform/eventbus/bus.go` 重构：

- 优先队列（`container/heap`）替代内联 `go func()` 分发
- 背压控制：队列满时返回 `ErrQueueFull`，不再无限增长 goroutine
- 13 个完整测试（含 race detector）

### 大文件拆分

- `logistics_ops.go`（1217 行）→ 5 个文件（max 381 行）
- `aftersales_mgmt.go`（1058 行）→ 6 个文件（max 288 行）

### 测试覆盖提升

新加 6 个 domain 模块测试（158 tests）：

| 模块 | tests |
|------|-------|
| price | 40 |
| finance | 34 |
| supplier | 34 |
| decision | 18 |
| trustscore | 24 |
| integrations | 28 |

### 文档清理

旧 FastAPI / Vue 阶段文档仍然存在，阅读时应先看：

- `README.md`
- `AGENTS.md`
- `docs/ACTIVE_STACK_POLICY.md`
- `backend-go/README.md`
- `frontend-next/README.md`

历史文档中出现 `backend/app/*`、`frontend/src/views/*`、`/api/*` 时，默认按旧栈参考处理，不能直接作为当前实现事实。
