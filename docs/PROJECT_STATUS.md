# 凌镜 LingMirror Project Status

> **用途边界**：本文只保存带日期的工程与验证事实，不是产品方向、预算或开发优先级授权。当前产品方向见 [PRODUCT.md](PRODUCT.md)，当前管理状态见 [CURRENT.md](CURRENT.md)。历史自用经营提案未获当前确认。

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

当前事实复核：2026-07-11

## 当前完成度裁决

| 声明 | 当前裁决 |
|---|---|
| 代码、迁移、API 或页面存在 | 只能标记 `implemented` |
| 自动测试或构建通过 | 只能标记 `automated_verified`，不能代替真实经营验收 |
| 候选市场比较与 Owner 选定市场 | 未完成 |
| 实验案卷、证据等级和顺序闸门 | 基础实现完成 |
| 同订单最终利润与现金关联校验 | 部分完成 |
| 非关联买家付款、签收和售后窗口闭合 | 未形成确定性事实链 |
| 正的最终贡献利润 | 未见真实经营证据；代码仍存在负利润可 `continue` 的漏口 |
| 生产可用 | 未验证 |

以下 2026-07-07 及更早的 “Dev Done / Test Green” 记录是历史工程交付记录，不代表当前经营主线完成，也不代表 Business Verified。

> 2026-07-07 direction note:
> 商品出海决策与执行层 Phase 1 + Phases 2-6 已交付。Phase 1 修复审批-任务-执行断点并打通单商品 dry-run 闭环；Phase 2-6 在此基础上新增 ProductHub 证据链、Execution Mode 统一与 Sandbox 支持、批量评估与 Owner 决策队列、执行结果回流与复盘、多平台对比。
> 详见 [features/phase1-dry-run-closed-loop-spec.md](features/phase1-dry-run-closed-loop-spec.md) 和 [features/product-export-decision-execution-layer.md](features/product-export-decision-execution-layer.md)。

## 2026-07-07 商品出海决策与执行层 Phase 2-6 交付

当前北极星保持不变：

```text
商品能不能卖，系统能说清楚；
订单和履约会不会亏，系统能说清楚；
高风险动作是否可执行，Owner 能看懂并审批。
```

### 本期交付汇总

| Phase | 能力 | 状态 |
|-------|------|------|
| Phase 2 | ProductHub 证据链聚合 API + 前端生命周期时间线 | ✅ Dev Done / Test Green |
| Phase 3 | Execution Mode 统一 (dry_run/sandbox/production) + PublishHook 重构 + 外部引用 ID | ✅ Dev Done / Test Green |
| Phase 4 | 批量评估 API + Owner 决策队列排序/过滤/搜索/摘要统计 | ✅ Dev Done / Test Green |
| Phase 5 | 执行结果回流 + 推荐反馈汇总 API + 复盘卡片 | ✅ Dev Done / Test Green |
| Phase 6 | 平台对比 API + 跨平台 Listing 记录查询 | ✅ Dev Done / Test Green |

当前北极星：

```text
商品能不能卖，系统能说清楚；
订单和履约会不会亏，系统能说清楚；
高风险动作是否可执行，Owner 能看懂并审批。
```

### 当前已完成的优先级项

| 优先级 | 方向 | 状态 | 交付物 |
|--------|------|------|--------|
| P0 | EventBus/Scheduler 生命周期验证 | ✅ 已完成 | EventBus 生命周期测试覆盖 start→publish→receive→stop→no-more-deliveries |
| P0 | 统一执行门禁 `/ai/actions/:id/execute` | ✅ 已完成 | ExecuteAction 审计写入 + 幂等守卫 + RBAC 权限路由 |
| P0 | 审批/执行绑定登录用户和 RBAC | ✅ 已完成 | ActionDecisionInput 移除 operator 字段；approve/execute/reject 路由需 `ai.action` 权限 |
| P1 | 外部平台写 dry-run/sandbox 模式 | ✅ 已完成 | ExecutionMode 类型 + context 传递 + PublishToOzon dry-run 守卫 |
| P1 | 商品出海决策与执行层 Phase 1 | ✅ 已完成 (2026-07-07) | 修复5个断点: approval topic统一→事件写回approval_id+状态→dry_run mode传播→防止重复审批→publishHook门禁; PR #318 |
| P1 | 审计日志敏感字段脱敏 | ✅ 已完成 | `operationlog.RedactSensitive` 正则脱敏；Log 和 LogStructured 自动应用 |
| P1 | 前端高风险操作确认 UX | ✅ 已完成 | `HighRiskConfirmDialog` 组件含风险等级/目标/前后值/环境模式/审计去向/回滚说明 |

### 当前建议优先推进

- 运行并修复主业务链路浏览器 / E2E 验证，证明 Owner 可从界面完成候选商品 -> 建议 -> 审批 -> 受控执行链路
- 将 Owner 工作台仍直接调用审批 / listing-task 执行的流程继续收口到统一 Action 门禁
- 清理和归档并行开发产生的未合并工作，避免后续 Agent 误判当前事实

### 当前不建议优先推进

- 继续堆新的 CRUD 页面（除非能改进两个业务闭环之一）。
- 做通用无代码 Agent 搭建平台。
- 宣传或实现全自动生产执行。
- 在审批、审计、RBAC 和可观测性未统一前扩大 Agent 自主权。——**本期已完成统一**
- 真实外部平台写回早于只读、sandbox、审批和失败处理闭环。——**dry-run/sandbox 模式已就绪**

### 本期执行门禁收口清单（2026-07-05）

| # | 交付物 | 文件变更 | 测试 |
|---|--------|----------|------|
| 0.1 | EventBus 生命周期测试 | `eventbus/bus_test.go` | ✅ TestBusLifecycle |
| 0.2/1.2 | 移除 ActionDecisionInput operator | `ai/model.go`, `ai/ai_test.go` | ✅ TestActionOperator_BoundToServer |
| 1.1 | ExecuteAction 审计日志 + 幂等守卫 | `ai/service.go`, `ai/routes.go`, `ai/ai_test.go`, `httpx/router.go` | ✅ TestExecuteAction_AuditLog, TestExecuteAction_Idempotent |
| 2.1 | RBAC 集成到 approve/execute 路由 | `ai/routes.go` | 新增 `ai.action` 权限路由 |
| 3.1 | 审计日志敏感字段脱敏 | `operationlog/redact.go`, `operationlog/service.go` | ✅ 15 tests (含 TestLogRedactsContent, TestLogStructuredRedactsContent) |
| 4.1 | 平台写 dry-run/sandbox 模式 | `integrations/types.go`, `integrations/service.go` | ✅ 4 tests (含 TestCheckWriteMode_DryRun_ReturnsMockResult) |
| 5.1 | 高风险确认弹窗组件 | `frontend/ui/HighRiskConfirmDialog.tsx` | ✅ 8 vitest tests |

## 当前事实快照

| 项目 | 当前口径 |
|------|----------|
| 活跃代码栈 | `backend-go/` + `frontend-next/` |
| 产品方向 | `unknown / 待 Owner 确认`，见 [PRODUCT.md](PRODUCT.md) |
| 模块 / API / 页面事实源 | [reference-module-catalog.md](reference-module-catalog.md) |
| 最新审计验证 | 2026-07-11：后端全量测试通过；前端 test / build 通过；lint 失败；E2E 未运行 |
| 历史验证记录 | 本文下方各日期段落、[TEST_SUMMARY.md](TEST_SUMMARY.md)、[FRONTEND_TEST_REPORT.md](FRONTEND_TEST_REPORT.md) |

### 当前验证状态

| 检查 | 当前状态 | 说明 |
|------|----------|------|
| `cd backend-go && go test ./...` | ✅ 通过（2026-07-11） | 116 个包、约 2900 个测试通过 |
| `cd backend-go && go vet ./...` | ⚪ 本次未重跑 | 2026-07-09 历史记录通过，不冒充本次验证 |
| `cd frontend-next && npm run lint` | ❌ 失败（2026-07-11） | ESLint 10.7 与 `react/display-name` 插件接口不兼容；当前质量门不全绿 |
| `cd frontend-next && npx tsc --noEmit --pretty false` | ⚪ 本次未单独重跑 | Next build 的类型检查通过，但不替代独立命令记录 |
| `cd frontend-next && npm test` | ✅ 通过（2026-07-11） | 20 test files / 105 tests |
| `cd frontend-next && npm run build` | ✅ 通过（2026-07-11） | Next production build 通过，生成约 89 个路由 |
| `cd frontend-next/e2e && npx playwright test` | ⏳ 待验证 | 需要运行中的后端、前端和测试数据 |

## 当前结论

凌镜已完成全站新技术栈迁移，旧栈（Python/FastAPI + Vue 3）已于 2026-06-30 删除。

历史发布标记：v0.5.0.0 — 2026-07-07，商品出海决策与执行层 Phase 1-6 工程交付；该方向已经冻结，且“工程交付”不等于当前经营闭环完成。
历史规格见 [features/product-export-decision-execution-layer.md](features/product-export-decision-execution-layer.md)。

当前唯一活跃开发线：

- Backend: `backend-go/`，Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/`，Next.js / React / TypeScript / Ant Design
- API prefix: `/api/v1`
- Health check: `/api/health`

历史文档中出现 `backend/app/*`、`frontend/src/views/*`、`/api/*` 时，按旧栈参考处理（已归档在 git history 中）。


## 历史更新记录

以下内容按日期保留，用于追溯项目演进。除非重新验证并写入“当前事实快照”，
否则不要把历史段落中的测试结果、缺陷数量或完成状态当作今天的事实。

## 2026-07-03 并行修复收口

~20 个 Agent 在 worktree 中并行执行，完成安全修复、Bug 修复、基础设施加固、文档体系建设和测试脚本开发。

| 类别 | 内容 |
|------|------|
| **安全修复** | WebSocket CheckOrigin 配置化验证（不再 `return true`）；JWT 中间件校验 token type=access，拒绝 refresh token；Webhook HMAC 验签（Ozon HMAC-SHA256 + Shopee HMAC）；密码最小长度 6→8；注册 5/min + refresh 20/min 限流；Rate limiter 启动 CleanupPeriodic goroutine |
| **Bug 修复** | M1 Metabolism Execute() 在 ScoringAdapter=nil 时返回 500 而非崩溃；Canvas 页面 useEffect+setState 改为 key-based remount |
| **基础设施** | docker-compose 8 处硬编码凭证替换为 `${VAR:-default}`；AIOS setup.go 标记 7 个未用模块（ponytail:）；Prometheus 新增 6 条 AI 管道告警规则（EventBus 队列积压、scheduler 错误、agent 心跳、5xx 尖峰） |
| **文档体系** | ADR-001~006 架构决策记录（`docs/adr/`）；INCIDENT_RESPONSE.md（SEV1-3 定义 + 5 场景流程）；DISASTER_RECOVERY.md（24h RPO / 2h RTO + 恢复步骤）；RUNBOOK.md 更新；Swagger/OpenAPI（swaggo 安装 + 44 核心端点标注，`/swagger/index.html` 在线）；api-inventory.md 27→71 模块更新；5 份重叠文档合并入 reference-module-catalog.md 作为单一事实源 |
| **测试脚本** | `backend-go/scripts/smoke_test.sh` + `smoke_test_setup.sh` — 10 步端到端管道验证 |

### 验证摘要

| 检查 | 状态 | 说明 |
|------|------|------|
| WebSocket CheckOrigin | ✅ 已实现 | `cors.allowed_origins` 配置化验证 |
| JWT token type 校验 | ✅ 已实现 | 中间件拒绝 refresh token 访问受保护端点 |
| Webhook HMAC 验签 | ✅ 已实现 | Ozon（HMAC-SHA256）+ Shopee（HMAC）双平台 |
| docker-compose 凭证 | ✅ 已替换 | 8 处硬编码改为 env var 默认值 |
| Swagger 端点 | ✅ 已配置 | `GET /swagger/index.html` — 44 核心端点 |
| smoke_test.sh | ✅ 已编写 | 10 步骤管道验证，需在有效数据库上运行 |
| Prometheus 告警 | ✅ 已配置 | `deploy/prometheus/rules/lingmirror_alerts.yml` |
| `go test ./...` / `go vet ./...` | ⏳ 待验证 | 安全修复 + 新脚本引入后需重新运行 |


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

## 模块与路由覆盖

> 完整的模块、路由和前端页面清单请参见 [reference-module-catalog.md](reference-module-catalog.md)（唯一事实源）。以下仅保留项目状态独有的内容。

`backend-go/internal/httpx/router.go` 在 `/api/v1` 下注册了认证、RBAC、Agent、AgentOS 和业务域路由，覆盖商品、平台、订单、财务、AI/AgentOS、选品生图等全套领域。`frontend-next/src/app/` 侧边栏菜单目前有 47 个入口，均已匹配到实际页面。

## 历史验证状态

2026-07-03 工作台 + 并行修复收口（Issue #64 + 20-Agent 并行执行）：

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd backend-go && go test ./...` | 通过 | Go 测试全绿 |
| `cd backend-go && go vet ./...` | 通过 | 无 vet 输出 |
| `cd frontend-next && npm test` | 通过 | 77 tests |
| `cd frontend-next && npm run build` | **失败** | `src/config/menu.ts` 存在未解决的合并冲突标记，3 个 Turbopack 构建错误 |
| `cd frontend-next && npm run lint` | **12 errors, 22 warnings** | 34 problems；含 merge conflict 解析错误、react-hooks/set-state-in-effect、@typescript-eslint/no-unused-vars 等 |
| 安全基线加固（6 项） | ✅ 已实现 | WebSocket CheckOrigin / JWT type 校验 / Webhook HMAC / 密码 8+ / 限流 / rate limiter cleanup |
| Swagger / OpenAPI | ✅ 已配置 | swaggo 安装 + 44 核心端点标注，`/swagger/index.html` 在线 |
| 运营文档体系 | ✅ 已建立 | ADR-001~006 / INCIDENT_RESPONSE.md / DISASTER_RECOVERY.md / RUNBOOK.md 更新 |
| smoke_test 脚本 | ✅ 已编写 | `backend-go/scripts/smoke_test.sh` + `smoke_test_setup.sh` |
| Prometheus 告警规则 | ✅ 已配置 | `deploy/prometheus/rules/lingmirror_alerts.yml` — 6 条 AI 管道规则 |

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

### 历史验证状态

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
## 本次更新内容（2026-06-29，7 Agent 并行执行）

### P1 缺口修复
- **importbatch** — 3个TODO stub替换为真实GORM操作：product+SKU创建、order创建、inventory upsert
- **feedback** — TS类型安全修复 6文件：Widget.tsx/WidgetButton.tsx/FeedbackCard/feedback 4页面
- **TS/lint** — TS errors 41→0！vitest全局类型声明（13个TS error清零）

### P2 stash 落地（Supply Chain Orchestrator + AIOS）
- **Supply Chain Orchestrator** — 10/10 issues：供应链价值链AI重构
- **Aftersales** — return_tracker 退货追踪模块
- **Logistics** — consolidation（合并发货）+ flywheel（物流飞轮）
- **Tariff** — 关税计算引擎（handler/model/routes）
- **Sourcing** — profit分析引擎
- **AIOS Phase 1-3** — 16 agents全部迁移到ToolRegistry
- **Marketing** — Ad Pilot、Home Feed
- **A9 批量运营前端** — CSV/XLSX上传、类型选择（product/order/inventory）、状态轮询、详情弹窗
- **Research docs** — 7份竞争调研报告（AI生图/广告优化/Listing工具/客服/利润引擎等）

### 验证
- `go test ./...` — 全绿
- `go vet ./...` — 全绿
- `tsc --noEmit` — 0 errors
- PR #29 — body 已更新，comment 已追加

### 待办
- 剩余 lint 34 problems（12 errors / 22 warnings）均为无关模块遗留
- **需修复** `src/config/menu.ts` 合并冲突标记（导致 build 失败）
- 清理 stale worktrees

## 本次更新内容（2026-06-30，可信经营闭环收口）

### 新增功能
- **ListingTask 状态机** — 定义 blocked→pending→executing→completed/failed→pending 状态转换，在执行入口统一校验。新增 `listingtask/statemachine.go`
- **Approval 状态机增强** — 增加 expired/canceled/superseded 状态及转换定义，新增 Cancel/ExpirePending/Supersede 方法。新增 `approval/statemachine.go`
- **统一执行门禁** — `ExecuteTask` 执行前增加 approval 审批检查、idempotency 幂等守卫、audit 审计写入口。无 valid approval 无法执行
- **Agent 建议反馈回流** — `ListingTask` 模型增加 `agent_feedback_status`（accepted/rejected）和 `agent_feedback_note` 字段，提供 `POST /listing-task/:task_id/feedback` API
- **Owner 决策队列** — 新增 `GET /owner/decision-queue` API，返回完善度/利润/风险/任务状态/审批状态/阻断原因/Agent反馈状态
- **Owner 工作台菜单** — 新增"经营闭环"侧边栏分组（经营总控台 + 候选商品）

### 平台门禁列表
| 门禁 | 触发点 | 实现 |
|------|--------|------|
| Auth/JWT | 所有 API | middleware.Auth（已有） |
| RBAC | 财务等敏感模块 | middleware.RequirePermission（已有） |
| State Machine | ListingTask 状态更新 | listingtask/statemachine.go（本期新增） |
| Approval | ExecuteTask 执行前 | checkApproval()（本期新增） |
| Idempotency | 重复执行 blocking | status=completed 明确报错（本期新增） |
| Audit | 执行/反馈关键操作 | operationlog.Service.Log（本期新增） |

### 历史验证状态
| 检查 | 结果 | 说明 |
|------|------|------|
| `go test ./internal/domain/listingtask/...` | 通过 | 21 tests（含状态机/门禁/反馈） |
| `go test ./internal/domain/approval/...` | 通过 | 28 tests（含取消/过期/替代） |
| `go test ./internal/domain/loop/...` | 通过 | 4 tests |
| `go test ./internal/domain/listing/...` | 通过 | 13 tests |
| `go test ./internal/domain/order/...` | 通过 | 43 tests |
| `go vet ./...` | 通过 | 无输出 |
| `npm run build` | ⚠️ 需 npm install | worktree 缺 node_modules |
| `npm test` | ⚠️ 需 npm install | 同上 |

### 安全边界（维持不变）
- 所有 API 受 JWT 保护
- ListingTask 初始为 `blocked` 状态，需审批后执行
- ExecuteTask 前必检查 approval
- 无真实外部发布/改价/改库存代码
- 所有状态转换通过 operationlog 可追溯
- loop 是本地模拟经营闭环，不触发真实外部操作
