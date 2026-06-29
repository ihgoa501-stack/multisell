# 凌镜 LingMirror Project Status

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

更新时间：2026-06-29

## 当前结论

凌镜已完成全站新技术栈迁移，并完成一人Agent公司7天MVP最小经营闭环的核心接线。

当前唯一活跃开发线是：

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
- 订单履约：`order`、`orderimport`、`shipping`、`logistics`、`platformfee`、`aftersales`
- 财务经营：`finance`、`settlement`、`decision`、`allocation`、`report`、`exchangerate`
- 运营支撑：`dashboard`、`search`、`notification`、`exceptions`、`operationlog`、`importbatch`
- AI / AgentOS：`ai`、`agent`、`agentos`、`agentrule`、`entropy`、`evolution`、`trustscore`、`actionpolicy`
- 选品与生图：`sourcing`、`sourcing1688`、`imagegen`
- 实时能力：WebSocket `/ws`

### 前端

`frontend-next/src/app/` 已按业务域迁移到 Next App Router。当前 build 输出覆盖：

- Dashboard / AI / AgentOS / Action Center
- 商品、SKU、分类、品牌、库存、供应商
- 平台、平台集成、刊登、刊登任务、刊登任务详情
- 订单、订单详情、订单导入、售后、代谢评分
- 物流、平台费用、结算、结算详情、财务、经营决策、成本分摊
- 选品分析、1688 采购
- 异常、通知、生图、画布、批量导入、操作日志、搜索、报表
- 设置、LLM 配置、RBAC、审批策略

侧边栏菜单目前有 47 个入口，已确认都能匹配到 `frontend-next/src/app` 下的实际页面。

## 验证状态

2026-06-26 复核结果（July gap-fill P1 追加后）：

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd backend-go && go test ./...` | 通过 | 38 个 Go 测试包，新增 logistics、sourcing、toolbridge 等模块测试 |
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

## 本次新增内容（2026-06-26，July gap-fill P1）

### 新领域模块

| 模块 | 位置 | 说明 |
|------|------|------|
| **sourcing** | `internal/domain/sourcing/` | A8 选品盈利分析引擎：利润公式计算、Eval 评估、Handler/Service/Routes 已定义（`POST /api/v1/sourcing/fetch`、`GET /api/v1/sourcing/recommendations`），⚠️ 尚未在 `router.go` 接线 |
| **logistics** | `internal/domain/logistics/` | 全新运费费率引擎（独立于 shipping 包），支持四种定价模式（first_additional / tiered / fixed / per_kg），YAML 配置加载 |
| **toolbridge** | `internal/platform/toolbridge/` | 插件驱动的工具执行桥接，允许 Agent 通过已注册插件执行外部工具 |
| **echo_ext** | `internal/realtime/extension_handler.go` | WebSocket 扩展处理器，支持实时连接扩展 |

### 新增 Agent（A8–A11）

`internal/agent/impl/agents.go` 中已注册 15 个 Agent（A1-A11 + G0-G3）：

| ID | 名称 | Squad | 决策点 |
|----|------|-------|--------|
| A8 | 选品盈利分析 | insight | sourcing_recommend |
| A9 | 批量运维 | ops | batch_price_update, batch_inventory_sync, batch_listing_update, import_validation |
| A10 | 物流运费引擎 | ops | carrier_compare, shipping_bill_audit, carrier_performance, logistics_route_opt |
| A11 | 售后管理 | ops | return_analysis, refund_decision, dispute_manage, aftersales_report |

### 新增前端页面

- `/sourcing` — AI 选品面板，对接 `POST /api/v1/sourcing/fetch`
- `/metabolism` — M1 代谢评分引擎 UI

### Chrome 扩展

- `chrome-extension/` — 全新的浏览器扩展，支持内容脚本注入、侧边栏面板、实时 WebSocket 通信。协议定义在 `shared/protocol.ts`。

### 其他改进

- aftersales 同步：新增 `sync.go` 实现平台售后单同步
- allocation 扩展：细化成本分摊维度
- importbatch 增强：新增 YAML/JSON parser 和异步 processor
- inventory 扩展：库存字段扩展和预警规则
- AGENT_CAPABILITIES.md 新增

### 文档清理（本次）

- `AGENTS.md` — 新增 ToolBridge 和 logistics/sourcing 模块
- `docs/AGENT_CAPABILITIES.md` — 新增 Agent 花名册 A8-A11、sourcing API、新前端页面、ToolBridge、Chrome 扩展
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — 新增 `/sourcing`、`/metabolism` 路由
- `docs/PROJECT_STATUS.md` — 本次更新

## 本次新增内容（2026-06-29，One-Person Agent Company MVP）

### 新领域模块

| 模块 | 位置 | Issues | 说明 |
|------|------|--------|------|
| **candidate** | `internal/domain/candidate/` | #57 | 候选商品CRUD管理 |
| **completeness** | `internal/domain/completeness/` | #58 | 12维资料完整度评分引擎 |
| **profit** | `internal/domain/profit/` | #59 | 利润汇总（采购+物流+平台费+关税） |
| **loop** | `internal/domain/loop/` | #60 | 评估链路：完整度→利润→建议→listingtask |
| **mock** | `internal/domain/mock/` | #62 | Mock订单/结算/同步状态数据 |
| **owner** | `internal/domain/owner/` | #61 | Owner总控台聚合数据API |

### 新增迁移与种子数据

- `migrations/000006_candidate_tables.up.sql` — candidate_product/completeness_check/profit_summary/listing_recommendation 表 + 20条种子商品
- `migrations/000007_mock_tables.up.sql` — mock_order/mock_settlement/mock_sync_status 表
- Mock数据在服务启动时自动注入

### 新增前端页面

- `/owner` — Owner经营总控台（风险摘要/Agent建议/审批操作/平台同步状态）
- 菜单新增"经营闭环"组
- `/candidates` 已接入后端API

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./...` | ✅ 通过（含新包测试） |
| `go vet ./...` | ✅ 通过 |
| `npm test` | ✅ 77 tests 通过 |
| `npm run build` | ✅ 通过 |
| `npm run lint` | 1 error / 8 warnings（均为遗留文件） |

### 关键API

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/candidates | 候选商品列表 |
| POST | /api/v1/completeness/check/:productId | 完整度检查 |
| GET | /api/v1/profit/summary/:productId | 利润汇总 |
| POST | /api/v1/loop/evaluate/:productId | 全链路评估 |
| GET | /api/v1/owner/risk-summary | 风险汇总 |
| GET | /api/v1/owner/suggestions | Agent建议 |
| POST | /api/v1/mock/seed | Mock数据注入 |

### 安全边界

- 所有新API受JWT保护
- Listingtask初始为`blocked`状态，需Owner审批
- 无真实外部发布/改价/改库存代码
- 所有操作可通过operationlog追溯
