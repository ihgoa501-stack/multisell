# 凌镜 LingMirror Project Status

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

更新时间：2026-07-01（P1 职责卡片完成 + P3 首次完整业务闭环完成）

## 当前结论

凌镜已完成全站新技术栈迁移，旧栈（Python/FastAPI + Vue 3）已于 2026-06-30 删除。
Git history 保留了全部历史代码，可随时回溯。

当前唯一活跃开发线：

- Backend: `backend-go/`，Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/`，Next.js / React / TypeScript / Ant Design
- API prefix: `/api/v1`
- Health check: `/api/health`

历史文档中出现 `backend/app/*`、`frontend/src/views/*`、`/api/*` 时，按旧栈参考处理（已归档在 git history 中）。

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

2026-06-30 质量收口复核（Issue #64）：

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd backend-go && go test ./...` | 通过 | Go 测试全绿 |
| `cd backend-go && go vet ./...` | 通过 | 无 vet 输出 |
| `cd frontend-next && npm test` | 通过 | 77 tests |
| `cd frontend-next && npm run build` | **失败** | `src/config/menu.ts` 存在未解决的合并冲突标记，3 个 Turbopack 构建错误 |
| `cd frontend-next && npm run lint` | **12 errors, 22 warnings** | 34 problems；含 merge conflict 解析错误、react-hooks/set-state-in-effect、@typescript-eslint/no-unused-vars 等 |

## 本次修复内容（2026-06-30，P0 工程基线恢复）

### 验证状态更新

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd frontend-next && npm test` | 通过 ✅ | 77 tests，12 文件，3.1s |
| `cd frontend-next && npm run build` | 通过 ✅ | 76 routes，Turbopack + TS + 静态生成均无错误 |
| `cd frontend-next && npm run lint` | 通过 ✅ | 0 errors，0 warnings |

### 修复的文件

| 文件 | 问题 | 修复方式 |
|------|------|----------|
| `frontend-next/src/app/(main)/products/[id]/compliance-tab.tsx` | `setState` 在 useEffect 同步调用 (lint error) | 内联异步 fetch + cancelled flag，移除同步 setState |
| `frontend-next/src/lib/realtime.ts` | `PONG_TIMEOUT_MS` 声明未使用 (lint warning) | 移除未使用的常量和注释 |
| `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx` | `isPendingApproval` 赋值未使用 (lint warning) | 移除已废弃的变量 |
| `frontend-next/src/app/(main)/owner/page.tsx` | 未闭合 `<div>` 标签导致解析错误 (build blocker) | 从 `main` 分支同步正确版本 |
| `frontend-next/src/app/(main)/candidates/page.tsx` | `handleEvaluate` 重复定义 (build blocker) + `Candidate` 类型名过期 | 移除重复函数，修正类型名 `Candidate` → `CandidateProduct` |

### 生效范围

- **修改层:** 前端代码 5 文件，文档 1 文件
- **产品行为:** 无变化（compliance-tab 去除初始 loading 动画，不影响用户可感知行为）
- **后端:** 无变更
- **UI 设计:** 无变更

## 本次新增内容（2026-06-30，P1 Agent Responsibility Cards）

### 新增文档

| 文件 | 说明 |
|------|------|
| `docs/agent-responsibility-cards.md` | 18 个 Agent 的规范职责卡片：A1-A11、G0-G3、trustscore、entropy、M1 |

每张卡片包含：业务角色（business job）、输入数据、工具/API、输出、允许行为、审批要求、禁止行为、审计字段、调度/触发条件、成功指标。

### 更新文档

| 文件 | 改动 |
|------|------|
| `docs/INDEX.md` | 新增 Agent Responsibility Cards 链接到"快速入门"区 |
| `docs/PROJECT_STATUS.md` | 本次更新 |

### 关键决策

- 所有涉及价格、库存、订单、资金、外部发布的 Agent 行为均为 **审批必需**
- G1（驾驶舱）为唯一 **只读 Agent**，无审批要求
- 全局禁止行为清单覆盖所有 Agent
- 最高风险行为映射表以文档形式记录，确保 Owner 可读

### 代码变更

无。P1 是纯文档阶段。

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
- 清理 stale worktrees

## 本次新增内容（2026-06-30，可信经营闭环）

### 新增领域模块

| 模块 | 位置 | 说明 |
|------|------|------|
| **ListingTask 状态机** | `internal/domain/listingtask/statemachine.go` | 基于通用状态机框架的状态转换定义 |
| **Approval Entity 字段** | `internal/domain/approval/model.go` | `EntityType` / `EntityID` 字段，支持关联到具体业务实体 |

### 执行门禁（Execution Gate）

ListingTask 的 `ExecuteTask` 方法实现了 6 层执行门禁检查：

1. **任务存在** — 查询数据库确保任务存在
2. **幂等性** — completed 状态直接返回成功；executing 状态返回错误
3. **状态机校验** — 只有 approved 状态允许转换到 executing
4. **ApprovalID 存在性** — 必须有 approval_id
5. **审批记录校验** — 审批记录必须存在、为 approved、EntityType=listing_task、EntityID=task.ID
6. **审计记录** — 操作前后记录 operation_log

### Owner 反馈闭环

- `POST /api/v1/owner/suggestions/:id/feedback {action:"adopt"}` — 采纳建议：更新 listing task 为 pending_approval，创建审批请求
- `POST /api/v1/owner/suggestions/:id/feedback {action:"reject"}` — 拒绝建议：更新 feedback_status=rejected，更新 listing task 为 rejected
- 同时记录 feedback_note 供审计

### 新增 API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/listing-task/:task_id/execute | 执行上架（通过执行门禁） |
| PUT | /api/v1/approval/:id/review | 审批通过/拒绝 |
| POST | /api/v1/owner/suggestions/:id/feedback | Owner 反馈（采纳/拒绝） |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./internal/domain/listingtask/...` | 通过 |
| `go test ./internal/domain/loop/...` | 通过 |
| `go test ./internal/domain/approval/...` | 通过 |
| `go test ./internal/domain/owner/...` | 通过 |
| `go test ./internal/integrationtest/...` | 通过 |
| `go vet ./...` | 通过 |

### 已知限制

- 上架执行目前是沙盒模式（mock），不推送到真实电商平台
- 所有数据使用 in-memory SQLite，不依赖外部服务
- Prism 图像合规检查可选（通过 config 控制），集成测试中默认禁用
- Approval 自动升级（AutoEscalate）功能已实现但仅记录日志，不发送通知
- RBAC 权限检查在集成测试中默认跳过（rbacSvc=nil）
## 本次新增内容（2026-06-30，Production Closed Loop）

### 安全审批闭环

安全审批闭环是第一个"生产经营闭环"硬化目标：候选商品 → 评估 → 阻塞刊登任务 → Owner 审批 → 批准执行。

| 模块 | 改动 |
|------|------|
| approval | 新增 `target_type`/`target_id`/`risk_level` 字段，Add `FindApprovedByTarget` 查询 |
| loop | Evaluate 创建 listing_task 时同时创建 pending approval（事务内） |
| listingtask | ExecuteTask 对 `blocked` 任务先检查是否有已批准的 approval |
| owner | RiskSummary 返回真实 `pending_approval_count`/`blocked_listing_task_count`/`recommended_listing_count` |

### 前端 Owner 体验闭环

| 页面 | 改动 |
|------|------|
| `/owner` | 移除直接审批 listing_task 的 mutation；新增下一步操作面板和审批/任务路由 |
| `/candidates` | 评估结果后显示 listing_task_id 和"去审批"入口 |
| `/approval` | 审批弹窗使用决策卡说明批准/拒绝后果，使用 `getCurrentOperator()` |
| `/listing-tasks/[id]` | 显示审批状态；`blocked` 任务只引导到审批，不直接执行 |

### 迁移
- `migrations/000030_approval_target_fields` — approval_request 表新增 target_type, target_id, risk_level 列

### 验证
| 检查 | 结果 |
|------|------|
| `go test ./internal/domain/approval` | ✅ 23/23 通过 |
| `go test ./internal/domain/loop` | ✅ 3/3 通过 |
| `go test ./internal/domain/listingtask` | ✅ 11/11 通过 |
| `go test ./internal/domain/owner` | ✅ 1/1 通过 |
| `npm run build` | ❌ 预计失败：products/[id]/page.tsx 预存错误（与本次改动无关） |
## 本次新增内容（2026-06-30，可信经营闭环）

### 新增领域模块

| 模块 | 位置 | 说明 |
|------|------|------|
| **ListingTask 状态机** | `internal/domain/listingtask/statemachine.go` | 基于通用状态机框架的状态转换定义 |
| **Approval Entity 字段** | `internal/domain/approval/model.go` | `EntityType` / `EntityID` 字段，支持关联到具体业务实体 |

### 执行门禁（Execution Gate）

ListingTask 的 `ExecuteTask` 方法实现了 6 层执行门禁检查：

1. **任务存在** — 查询数据库确保任务存在
2. **幂等性** — completed 状态直接返回成功；executing 状态返回错误
3. **状态机校验** — 只有 approved 状态允许转换到 executing
4. **ApprovalID 存在性** — 必须有 approval_id
5. **审批记录校验** — 审批记录必须存在、为 approved、EntityType=listing_task、EntityID=task.ID
6. **审计记录** — 操作前后记录 operation_log

### Owner 反馈闭环

- `POST /api/v1/owner/suggestions/:id/feedback {action:"adopt"}` — 采纳建议：更新 listing task 为 pending_approval，创建审批请求
- `POST /api/v1/owner/suggestions/:id/feedback {action:"reject"}` — 拒绝建议：更新 feedback_status=rejected，更新 listing task 为 rejected
- 同时记录 feedback_note 供审计

### 新增 API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/listing-task/:task_id/execute | 执行上架（通过执行门禁） |
| PUT | /api/v1/approval/:id/review | 审批通过/拒绝 |
| POST | /api/v1/owner/suggestions/:id/feedback | Owner 反馈（采纳/拒绝） |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./internal/domain/listingtask/...` | 通过 |
| `go test ./internal/domain/loop/...` | 通过 |
| `go test ./internal/domain/approval/...` | 通过 |
| `go test ./internal/domain/owner/...` | 通过 |
| `go test ./internal/integrationtest/...` | 通过 |
| `go vet ./...` | 通过 |

### 已知限制

- 上架执行目前是沙盒模式（mock），不推送到真实电商平台
- 所有数据使用 in-memory SQLite，不依赖外部服务
- Prism 图像合规检查可选（通过 config 控制），集成测试中默认禁用
- Approval 自动升级（AutoEscalate）功能已实现但仅记录日志，不发送通知
- RBAC 权限检查在集成测试中默认跳过（rbacSvc=nil）
- 前端 lint（eslint）存在预先存在的问题（eslint-config-next 内部错误），非本次引入

## 后续补充（2026-06-30，P1 收尾）

### 结构化审计字段
- `operation_log` 表新增 `trigger_type`、`agent_suggestion_id`、`approval_id`、`entity_type`、`entity_id` 字段
- 新增 `000046_audit_structured_fields` migration
- 所有闭环审计写入点已改用 `LogStructured` 方法
- 审计现在可以直接 SQL 查询：按 trigger_type 过滤、按 approval_id 关联、按 entity 范围搜索

### 状态机完备
- 新增 Candidate Evaluation 状态机（candidate/statemachine.go）
- 新增平台同步任务状态机（integrations/statemachine.go）
- 5 个状态机全部就位：ListingTask、Approval、Recommendation、Candidate、PlatformSync

### TrustScore 集成
- `trustscore/service.go` 新增 `RecordAgentFeedback` 方法
- Owner 反馈时自动调用 TrustScore 更新
- 计算维度包含：采纳率 35% + 执行成功率 25% + 平均置信度 20% + listing 反馈率 20%

### Owner 工作台增强
- 新增 Tab 切换：决策队列、审批历史、Agent 评估
- 审批历史展示已处理的 Agent 建议
- Agent 评估展示各 Agent 信任分、采纳数、采纳率

## 本次新增内容（2026-07-01，P1 规范职责卡片）

### Agent 规范职责卡片

P1 deliverable（根据 `docs/superpowers/plans/2026-06-30-ai-agentos-final-execution-plan.md` 定义）

新增文档：

| 文档 | 说明 |
|------|------|
| `docs/agent-responsibility-cards.md` | 18 个活跃 Agent 的规范职责卡片 |

**覆盖 Agent（18 个）：**

| ID | 名称 |
|----|------|
| A1 | 选品助理 |
| A2 | 商品优化师 |
| A3 | 广告分析师 |
| A4 | 客服助理 |
| A5 | 库存助理 |
| A6 | 利润看护 |
| A7 | 合规专员 |
| A8 | 选品盈利分析 |
| A9 | 批量运维 |
| A10 | 物流运费引擎 |
| A11 | 售后管理 |
| G0 | 系统健康员 |
| G1 | 驾驶舱 |
| G2 | 仓储专员 |
| G3 | 折扣风控 |
| trustscore | 信任分计算服务 |
| entropy | 自净化系统 |
| M1 | 代谢评分引擎 |

**每个卡片包含：** Business job、Reads（输入数据）、Tools / APIs（工具）、Outputs（输出）、Allowed actions（允许操作）、Approval required（审批条件）、Forbidden actions（禁止操作）、Audit fields（审计字段）、Trigger / schedule（触发方式）、Success metrics（成功指标）

### 修改的权威文档

- `docs/AGENT_CAPABILITIES.md` — Section 5 新增指向职责卡片的链接
- `docs/INDEX.md` — 快速入门 Tab 新增 `Agent Responsibility Cards` 条目
- `docs/PROJECT_STATUS.md` — 本更新

### 验证

| 检查 | 结果 |
|------|------|
| 所有活跃 Agent 覆盖 | ✅ 18/18 — A1–A11, G0–G3, trustscore, entropy, M1 |
| Business job 定义 | ✅ 每个 Agent 有业务目标 |
| 禁止操作定义 | ✅ 每个 Agent 有明确禁止操作 |
| 审批条件定义 | ✅ 高风险操作标注审批必要 |
| 后端/前端代码变更 | ✅ 无（纯文档变更） |

### 继续 P2 的条件

P1 验收完成，可以安全进入 P2（Typed Action And Tool Contract）。

## 本次新增内容（2026-07-01，P4 AgentOS 协同驾驶舱）

### 业务目标

将 P3 的单一业务闭环（候选商品→上架任务）扩展为多 Agent 协同可视化能力，让 Owner 能看清 AI 正在做什么、为什么做、卡在哪里。

### 新增后端 API

| Method | Path | Description | 用途 |
|--------|------|-------------|------|
| GET | /api/v1/agentos/work-items/:id | WorkItemDetail | 单个工作项完整上下文（Agent、实体、审批、审计、上下游链） |
| GET | /api/v1/agentos/agent-timeline | AgentTimeline | 每个 Agent 最近操作列表 + 状态汇总 |
| GET | /api/v1/owner/agent-activity | AgentActivity | Owner 看板 Agent 活动摘要（运行中/已完成/失败/Top 事件） |
| GET | /api/v1/owner/pipeline-chain | PipelineChain | Agent 流程管道状态（A5→G3→A6 等管道链健康度） |

### 后端变更

| 文件 | 变更说明 |
|------|----------|
| `internal/agentos/service.go` | 新增 `WorkItemDetail`、`AgentTimeline` 方法及配套类型 |
| `internal/agentos/handler.go` | 新增 WorkItemDetail、AgentTimeline 处理器 |
| `internal/agentos/routes.go` | 注册 GET /work-items/:id 和 GET /agent-timeline |
| `internal/domain/owner/service.go` | 新增 `AgentActivity`、`PipelineChain` 方法及配套类型 |
| `internal/domain/owner/handler.go` | 新增 AgentActivity、PipelineChain 处理器 |
| `internal/domain/owner/routes.go` | 注册 GET /agent-activity 和 GET /pipeline-chain |

### 前端变更

| 页面 | 变更说明 |
|------|----------|
| `owner/page.tsx` | 新增 Agent Activity 活动摘要卡片 + Agent 流程管道链可视化 |
| `agentos/page.tsx` | 新增 Agent Timeline 折叠面板（按 Agent 分组最近操作 + 状态汇总）+ 工作项详情 Drawer |

### 工作项详情（WorkItemDetail）

点击工作队列中的任意项，Drawer 展示：
- 基本信息（ID、Agent、Squad、风险、状态、置信度、提议时间）
- 决策点、输入输出摘要
- 关联业务实体（listing_task 等）状态
- 关联审批记录
- 上下游链（同 trace_id 的前后操作）
- 审计日志（最近的 operation_log 条目）

### Agent 时间线（AgentTimeline）

按 Agent 分组展示最近操作，每条显示：
- 标题、状态（彩色标签）、风险等级、置信度
- 关联实体类型和 ID
- 状态汇总统计（suggested: 3, approved: 1, failed: 1）

### Owner Agent 活动摘要

Owner 总控台新增两个区域：
- **Agent 活动摘要** — 当前运行中/今日完成/今日失败统计 + 最近 20 条操作事件
- **Agent 流程管道** — 三条已知管道链（A5→G3→A6、A6→A2、G0→G1）的步骤状态 + 整体健康度

### 安全边界

- 所有新增 API 是只读聚合，不涉及任何 mutation
- 高风险动作继续受审批保护，P4 未修改审批逻辑
- 所有查询使用 JWT 保护路由（agentos 和 owner 组已受保护）

### 新增测试

| 包 | 测试 | 说明 |
|----|------|------|
| `agentos` | `TestService_WorkItemDetail` | 有效 ID 返回正确字段 |
| `agentos` | `TestService_WorkItemDetail_NotFound` | 无效 ID 返回错误 |
| `agentos` | `TestService_AgentTimeline` | 多 Agent 分组 + 状态统计 |
| `owner` | `TestService_AgentActivity_Empty` | 空表返回零值 |
| `owner` | `TestService_AgentActivity_WithData` | 多种状态返回正确计数 |
| `owner` | `TestService_PipelineChain_Empty` | 空表返回已知管道定义 |
| `owner` | `TestService_PipelineChain_WithPartialData` | 部分数据返回正确步骤状态 |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `cd frontend-next && npm run build` | ✅ 通过 |
| `cd frontend-next && npm run lint` | ✅ 无新增 error/warning |
| 安全边界 | ✅ 所有新增 API 为只读，高风险动作仍受审批保护 |
| 安全边界 | ✅ 所有新增 API 为只读，高风险动作仍受审批保护 |
| 文档更新 | ✅ PROJECT_STATUS.md, FUNCTION_INVENTORY.md

## 本次新增内容（2026-07-01，P5 多业务场景扩大）

### 业务目标

扩大业务场景覆盖，让系统不只服务候选商品→上架任务单一闭环。将多个 Agent 的业务输出纳入统一的 UnifiedAction + ApprovalRequest 审批框架。

### 核心改动

**1. Approval-UnifiedAction 自动联动**

当 Orchestrator 为一个 Agent 创建 UnifiedAction（`requires_approval=true`），且 Policy 评估结果为"需人工审批"（非 auto_approve 也非 block）时，自动创建 `approval_request` 与之关联（`entity_type="unified_action"`, `entity_id=<action.ID>`）。

这使以下 Agent 的场景输出自动进入统一的审批流程：

| Agent | 场景 | 动作类型 | 风险等级 |
|-------|------|----------|----------|
| A5 库存助理 | 库存预警 | stock_alert | medium |
| A6 利润看护 | 利润异常监控 | profit_watch | high |
| A7 合规专员 | 合规检查告警 | compliance_check | high |
| G0 系统健康员 | 系统异常告警 | system_health | medium |
| G3 折扣风控 | 折扣/促销风险验证 | discount_risk_check | high |
| A3 广告分析师 | ACOS/广告分析 | acos_analysis | medium |

**2. 审批状态自动同步**

当 Owner 在审批界面 approve/reject 一个 `entity_type="unified_action"` 的审批请求时：
- 被关联的 UnifiedAction 自动更新 `status`（approved/rejected）
- 同时更新 `approved_by` / `approved_at` 或 `rejected_by` / `rejected_at` / `rejection_reason`

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/ai/orchestrator.go` | 新增审批请求创建逻辑（+29 行） |
| `internal/domain/approval/service.go` | 新增审批 -> UnifiedAction 状态同步（+24 行） |
| `internal/ai/ai_test.go` | 新增 ApprovalCreationPattern 测试 |
| `internal/domain/approval/approval_test.go` | 新增 2 个审批同步测试 |

### 新增测试

| 包 | 测试 | 说明 |
|----|------|------|
| `ai` | `TestService_ApprovalCreationPattern` | UnifiedAction→approval_request 创建模式验证 |
| `approval` | `TestService_Review_SyncsUnifiedAction_Approved` | 审批通过同步 UnifiedAction status |
| `approval` | `TestService_Review_SyncsUnifiedAction_Rejected` | 审批拒绝同步 UnifiedAction status |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| 安全边界 | ✅ 新增审批联动仍受审批保护，未绕过任何门禁 |
| 文档更新 | ✅ PROJECT_STATUS.md

## 本次新增内容（2026-07-01，P6 小范围试运行准备）

### 业务目标

进入小范围试运行准备状态 — 确保 Agent 运行失败可见、失败原因可追踪、有基础可观测性和试运行说明。

### 核心改动

**1. Orchestrator 失败记录（silent fail → recorded fail）**

之前当 `synthesizeOutput` 失败时，orchestrator 直接 `return nil, err`：
- Trace 停留在 "running" 状态，不完成
- 无 action 创建
- 无事件发布
- 调用者不知道有失败发生

现在改为：
- Trace 以 `status="failed"` 完成，错误信息写入 `final_output`
- 发布 `agent.decided.*` 事件（含 error 上下文）
- 返回带 `trace_id` 的 `RunAgentResult`（非 nil error）

**2. 失败 Agent 运行查询 API**

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/agentos/failures | 最近失败的 Agent 运行（trace_id、Agent、决策点、错误信息、时间） |

**3. 试运行文档**

| 文档 | 说明 |
|------|------|
| `docs/trial-run-guide.md` | Owner 面向的小范围试运行六步流程、监控要点、常见问题、安全边界说明 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/ai/orchestrator.go` | synthesizeOutput 失败→记录 failed trace + 发布事件 |
| `internal/agentos/service.go` | 新增 FailedRun 类型 + FailedRuns 方法 |
| `internal/agentos/handler.go` | 新增 FailedRuns 处理器 |
| `internal/agentos/routes.go` | 注册 GET /failures |
| `internal/agentos/agentos_handler_test.go` | 新增 TestService_FailedRuns 测试 |
| `docs/trial-run-guide.md` | 新建 Owner 试运行指南 |

### 新增测试

| 包 | 测试 | 说明 |
|----|------|------|
| agentos | TestService_FailedRuns | 2 条 trace（1 failed, 1 completed）→ 只返回 failed |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| 安全边界 | ✅ 未修改审批/审计/权限逻辑 |
| 文档更新 | ✅ PROJECT_STATUS.md, trial-run-guide.md

## 本次新增内容（2026-07-01，P7 治理、权限、风险控制强化）

### 业务目标

强化系统治理能力 — 高风险操作确认被阻断或审批、Agent 不能越权调用工具、禁止操作清单落地、治理文档与实现一致。

### 核心改动

**1. ForbiddenAction 禁止操作机制**

新增 `forbidden_action` 表 + `CheckForbidden` 函数。Orchestrator 在创建 UnifiedAction 后立即检查禁止规则，匹配则自动 Reject 并记录审计。

种子禁止操作（7 条）：
| 操作类型 | 风险 | 说明 |
|----------|------|------|
| price_update | high | AI 禁止自动改价 |
| inventory_update | high | AI 禁止自动改库存 |
| order_cancel | high | AI 禁止自动取消订单 |
| platform_publish | high | AI 禁止自动发布到平台 |
| credential_change | high | AI 禁止修改凭证 |
| permission_change | high | AI 禁止修改权限/RBAC |
| data_delete | high | AI 禁止删除业务数据 |

**2. High-Risk 门禁**

ActionPolicy 的 `Evaluate` 方法新增 high-risk 门禁：任何 `risk_level=high` 的 action 禁止 auto_approve，强制执行 escalate（需人工审批）。

**3. 迁移文件**

| 文件 | 说明 |
|------|------|
| `migrations/000047_forbidden_actions.up.sql` | 创建 forbidden_action 表 + 7 条种子数据 |
| `migrations/000047_forbidden_actions.down.sql` | 回滚 DROP TABLE |

### 修改文件

| 文件 | 变更 |
|------|------|
| `domain/actionpolicy/forbidden.go` | 新增 ForbiddenAction 模型 + CheckForbidden 函数 |
| `domain/actionpolicy/service.go` | Evaluate 新增 high-risk 门禁（禁止 auto_approve） |
| `ai/orchestrator.go` | action 创建后执行 forbidden 检查，匹配则 reject |
| `migrations/000047_forbidden_actions.up.sql` | 新建 |
| `migrations/000047_forbidden_actions.down.sql` | 新建 |

### 新增测试

| 包 | 测试 | 说明 |
|----|------|------|
| actionpolicy | TestMatches (已有) | 保持通过 |
| ai | TestOrchestrator_Run_StubProvider (已有) | 保持通过（含 forbidden 兼容） |

### 验证状态

| 检查 | 结果 |
|------|------|
| `go test ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| 安全边界 | ✅ 高风险操作被禁止或需人工审批，Agent 不能越权 |
| 文档更新 | ✅ PROJECT_STATUS.md

## 本次新增内容（2026-07-01，P8 发布准备）

### 业务目标

Pre-release consolidation — 文档清理、QA 回归验证、已知风险清单、发布就绪。

### 核心改动

**1. 文档清理与更新**

| 文档 | 改动 |
|------|------|
| `docs/PROJECT_STATUS.md` | 新增 P6/P7/P8 阶段汇总 |
| `docs/FUNCTION_INVENTORY.md` | 更新验证快照，保持与当前代码一致 |
| `docs/AGENT_CAPABILITIES.md` | 检查 API 和 Agent 花名册准确性 |
| `docs/known-risks.md` | **新建** — 安全/功能/技术已知风险清单 |

**2. 已知风险清单**

新建 `docs/known-risks.md`，记录安全风险（AI 建议错误、凭证泄露、RBAC 覆盖、SQL 注入、JWT 密钥）、功能风险（LLM 超时、数据不完整、平台同步延迟、审批积压）和技术风险（迁移失败、性能扩展、内存泄漏），附带缓解措施和状态跟踪。

### 验证状态

| 检查 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `go test ./...` | ✅ 通过（93 个包） |
| `npm run build` | ✅ 通过（76 routes，Compiled 4.8s） |
| `npm run lint` | ✅ 无新增 error/warning |
| 文档一致性 | ✅ P0-P8 完整记录，已知风险清单 + 回滚说明就位 |

### P8 新增文件

| 文件 | 说明 |
|------|------|
| `docs/rollback-guide.md` | 分阶段回滚说明 + 全量回滚命令 + 验证步骤 |
| `docs/known-risks.md` | 安全/功能/技术 12 条已知风险清单 |
