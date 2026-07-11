# 凌镜 LingMirror — Agent Instructions

<!-- CODEGRAPH_START -->
## CodeGraph

This repository is indexed by CodeGraph (`.codegraph/` exists at the repo root). Use it before grep/find or opening source files when you need to understand or locate code:

- MCP tools: `codegraph_explore` answers most code questions in one call with relevant symbols, verbatim source, and call paths. `codegraph_node` reads one symbol or a whole file with line numbers.
- Shell fallback: `codegraph explore "<question or symbols>"` and `codegraph node <symbol-or-file>`.
- Skip CodeGraph only for files it does not index well, such as Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.
<!-- CODEGRAPH_END -->

## Project

### Owner Direction Must Be Confirmed

仓库目前没有一份已经由 Owner 在当前对话中确认、可以自动继承的最高产品方向。制定系统大目标、商业方向、资金预算、目标用户或经营主线前，必须先询问并记录 Owner 的当前意图；不得仅因历史文档日期较新或写有“当前”“生效”字样，就把它当作 Owner 现行决定。

`docs/SELF_USE_OPERATING_DIRECTION.md` 是 2026-07-11 的历史提案，状态为 `superseded / unconfirmed`。其中的 Owner 自用定位、跨市场实验主线、3,000 CNY 总预算、1,200 CNY 损失线及其他范围约束均不得自动用于规划、开发或结果判断。它只能作为历史背景，除非 Owner 再次逐项确认。

开始非平凡研究、规划、开发、审查、QA 或发布前，仍应阅读 `docs/research/project-truth-audit-2026-07-11.md`，但只能把它当作带日期的仓库证据快照，不能把其中的产品方向和行动顺序当成当前授权。审计中的证据等级规则仍适用：不得把模块存在、测试通过、页面可见或多个 Agent 意见一致写成真实市场、真实成交、生产可用或最终利润已经成立。代码或真实经营状态变化后，应重新核验并生成新的带日期审计，不能静默覆盖证据限制。

在 Owner 确认新方向前，不主动扩大产品范围或执行高成本、不可逆的外部动作。下文关于候选市场、经营实验和平台连接器的内容用于描述现有代码、证据边界与安全规则，不构成当前产品方向或继续建设授权。

生产服务器初始化、SSH、部署、恢复、测试和回滚只有一个可执行入口：`docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md`。

现有 Ozon 采集器只是平台连接器，不是当前市场方向。`GET /api/v1/candidates/collection-evidence/:id` 只读取已有采集证据；除非 Ozon 对应的国家、消费者和渠道已通过市场闸门，否则不得把该接口列为当前采集任务。所有平台采集线索都必须通过 `evidence_id` 引用不可变快照，并声明它服务的经营决策。

经营实验统一事实链位于 `internal/domain/experiment/`，API 根路径为 `/api/v1/experiments`，前端入口为 `/experiments`。每个案件以 `experiment_id` 关联现有业务对象；证据作用区分 `support / counter / conflict`，真实性区分 `actual / quoted / estimated / unknown / mock / inferred`。普通录入不能直接声明 `actual`，必须由 Owner 对来源和观察时间单独核验。最终利润必须关联可信且全部对账的结算、最终 `order_profit_record` 和同一订单；现金回收必须关联同一订单与结算的银行/现金账户交易。利润最终确认与现金回收是两个独立状态；模拟、未知或 AI 推断不得通过经营闸门。

候选市场比较的统一入口位于 `internal/domain/demandcase/`，API 根路径为 `/api/v1/demand-cases`。每个候选必须明确“国家/地区 × 目标消费者 × 需求场景 × 销售渠道”，覆盖需求、竞争、获客、履约、合规、收款、售后和利润可验证性八个维度，并包含来自不同 run 的独立反证。关键维度为 unknown、mock、inferred 或缺少来源/观察时间时，只能保持 `evidence_missing`，不得生成可实验结论。

候选市场 Owner 页面为 `/demand-cases`。研究输入限定为 `scout_result / falsifier_result / data_reality_result`，研究输出 payload 与 SHA-256 快照必须一致，重复 run 幂等。带日期研究批次只导入已完成并审阅的研究输出，不在按钮点击时联网。

渠道比较之前统一使用同一 `demandcase` 模块的 `ProblemCase`：API `/api/v1/problem-cases`，Owner 页面 `/problem-cases`。案件不得预设商品或渠道；只有受信研究运行写入的支持证据与不同采集者独立反证才能参与裁决。证据保存原始内容与可复算 SHA-256，审阅批次以 `owner_id + problem_key` 幂等导入；只有 `survives_falsification` 才能晋升为 `DemandCase`。

带日期跨市场研究与独立反证分别位于 `deliverables/research/2026-07-11-live-market-permission-batch.md` 和 `2026-07-11-live-market-independent-falsification.md`。独立反证后本轮没有值得申请账号权限的候选：英国、美国保持 hold/evidence_missing，日本当前表述 reject。

凌镜 LingMirror (technical name: MultiSell) — cross-border e-commerce AI AgentOS.
Version `v0.3.0.0`.

## License And Ownership

This repository is proprietary and not open source. Do not add open-source
license language or publish/distribute project code unless the Owner explicitly
requests it. See `LICENSE`.

## Governance First

This repository uses an Owner-first, platform-first multi-Agent workflow. Before non-trivial development, refactor, review, QA, or release work, read the governance documents:

- `docs/governance/OWNER_FIRST_PROTOCOL.md` — the Owner describes business goals; Agents own technical translation and must report in business language.
- `docs/governance/PLATFORM_CONSTITUTION.md` — highest-level platform rules: system layers, risk levels, forbidden actions, and Owner decision boundaries.
- `docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md` — multi-Agent roles, start checklist, review checklist, QA checklist, and handoff rules.
- `docs/governance/KERNEL_CONTRACTS.md` — EventBus, Command, Scheduler, ToolBridge, Approval, Audit, RBAC, Observability, and Migration contracts.

When these governance docs conflict with older project docs, follow the governance docs unless the Owner explicitly overrides them.

| Stack | Dir | Entry |
|-------|-----|-------|
| Backend | `backend-go/` | `cmd/server/main.go` — Go 1.25, Gin, GORM, PostgreSQL 15 |
| Frontend | `frontend-next/` | `src/app/` — Next.js 16, React 19, TypeScript, Ant Design 6 |

API prefix: `/api/v1`. Health: `/api/health`. All non-auth endpoints require JWT.

## Commands

| Action | Command |
|---|---|
| Docker full stack | `docker compose up -d` |
| Docker DB only | `docker compose up -d db` |
| Backend dev | `cd backend-go && go run cmd/server/main.go` |
| Backend test all | `cd backend-go && go test ./...` |
| Backend test pkg | `cd backend-go && go test -v ./internal/domain/order/` |
| Backend vet | `cd backend-go && go vet ./...` |
| Backend build | `cd backend-go && go build -o bin/server cmd/server/main.go` |
| Frontend dev | `cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000` |
| Frontend build | `cd frontend-next && npm run build` |
| Frontend lint | `cd frontend-next && npm run lint` |
| Frontend test | `cd frontend-next && npm test` |
| E2E | `cd frontend-next/e2e && npx playwright test` |

New dev database: `multisell`. Migrations under `backend-go/migrations/`.

## Backend Architecture

### Module Pattern

Every domain module under `internal/domain/*/` follows a consistent layout: `routes.go` (route registration), `handler.go` (HTTP mapping), `service.go` (business logic), `model.go` (GORM models + request/response structs). Modules register in `internal/httpx/router.go`.

### Standard Response Envelope

```go
response.Success(c, data)                       // {"code":0, "message":"ok", "data":...}
response.Error(c, http.StatusBadRequest, msg)
response.Paginated(c, data, total, page, size)  // + pagination fields
response.InternalError(c, err)                  // 500, masked in release mode
```

Pagination: `common.ParsePagination(c)`, `common.ParseSort(c)`.

### Middleware Stack

`internal/httpx/middleware/`: CORS → RequestID → Metrics (opt-in) → RecoveryWithSentry → Audit (mutation logging). JWT `Auth` on the `/api/v1` protected group. Rate limiting via `ratelimit.go`.

### Platform Infrastructure (`internal/platform/`)

Four in-process coordination primitives for agent-to-agent and agent-to-system communication:

- **Event Bus** (`eventbus/bus.go`) — pub/sub with glob topic matching (`order.*`). Used for agent pipeline chains, scheduler ticks, cross-module async events. ~15 subscriptions in `router.go`.
- **Command Dispatcher** (`command/command.go`) — typed handler registry: `stock_alert`, `replenish`, `price_review`, `listing_optimize`, `compliance_check`.
- **Scheduler** (`scheduler/`) — periodic task runner (5 min to 6 hr intervals). Publishes `scheduler.tick.{agent_id}` events.
- **ToolBridge** (`toolbridge/bridge.go`) — plugin-driver-based tool execution bridge for agents to run external tools.

### Agent Pipeline Chain

Event bus subscriptions chain agent decisions automatically (defined in `router.go`):

```
A5 stock_alert (red)              → G3 discount_risk_check
G3 discount_risk_check (block)     → A6 profit_watch
A6 profit_watch (loss/threshold)   → A2 listing_optimize
G0 system_health (anomaly > 3)    → G1 dashboard_overview

Scheduled agents: G0/A4/G1/A5/G3/A6/A3/G2/A7/M1/trustscore/entropy
```

### WebSocket

`internal/realtime/` — hub for AI streaming and live updates. Endpoint: `GET /ws`.

### Configuration

`backend-go/configs/config.yaml`, overridden by env vars:

| Env | Config Path |
|-----|-------------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `database.*` |
| `JWT_SECRET` | `jwt.secret` |
| `SERVER_PORT` | `server.port` |
| `REDIS_ADDR` / `REDIS_PASSWORD` | `redis.*` |
| `SENTRY_DSN` | `sentry.dsn` |
| `CORS_ALLOWED_ORIGINS` | `cors.allowed_origins` |
| `METRICS_ENABLED` | `metrics.enabled` |

### Platform Integrations (`domain/integrations/`)

E-commerce platforms implement the `PlatformAdapter` interface (Publish, SyncStatus, SyncInventory, FetchOrders, etc.). Register via `RegisterAdapter("{code}", &Adapter{})` in `init()`. Current: `ozon`, `shopee`.

### Auth & RBAC

- `internal/auth/` — JWT login/register/refresh (public routes).
- `internal/rbac/` — role-based permissions on protected routes.

### Monitoring

- Prometheus metrics (opt-in): `/metrics` endpoint + request tracking middleware.
- Sentry in Go (`middleware.RecoveryWithSentry`) and frontend (`@sentry/nextjs`).
- Audit middleware logs mutations to `operationlog` table.

### Test DB Helper

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})  // in-memory SQLite, isolated per call
    svc := NewService(db, logger)
}
```

Safe for `t.Parallel()`. No PostgreSQL needed.

## Frontend

```
src/
├── app/(auth)/login/       # public login
├── app/(main)/{mod}/page.tsx  # one per domain module
├── components/
│   ├── crud/CrudListPage.tsx   # reusable CRUD table + search
│   ├── layout/                 # AntdProvider, AppHeader, AppSidebar
│   └── ui/                     # PageContainer, FilterBar, ConfirmDialog, etc.
├── lib/api-client.ts        # fetch wrapper with JWT refresh + dedup
├── stores/                  # Zustand (app, auth, permission)
├── config/menu.ts           # sidebar items
└── types/api.ts             # Result / PageResult types
```

Alias `@` → `src/`. E2E: `frontend-next/e2e/` (Playwright).

## Conventions

- Non-public endpoints must use JWT auth. Mutation routes should be auditable.
- Use GORM transactions for multi-step state changes.
- Add focused Go tests near touched behavior.
- Frontend: keep `npm run build` and `npm run lint` green (lint has known issues).
- Do not touch `.kilo/worktrees/` — managed by external tooling.
- **Documentation must stay in sync with code.** Any PR that changes module names, API paths, directory layouts, or removes/adds packages must update `CLAUDE.md`, `AGENTS.md`, and `docs/INDEX.md` as needed. PRs with stale doc references will be rejected by CI (`doc-links` job).

## AI & AgentOS

| Package | Purpose |
|---------|---------|
| `internal/ai/` | LLM orchestration, chat, streaming, traces, provider abstraction |
| `internal/agent/` | Agent registry + execution |
| `internal/agentos/` | Cockpit dashboard, work items, autonomy |
| `domain/agentrule/` | Agent behavior rules |
| `domain/entropy/` | Self-cleansing: SPC control, health scoring |
| `domain/evolution/` | Agent evolution nudges |
| `domain/logistics/` | Cross-border shipping rate engine (A10) |
| `domain/sourcing/` | Sourcing profit formula engine (A8) |
| `domain/trustscore/` | Trust score + autonomy gating |
| `domain/actionpolicy/` | Action approval policy |

## Documentation

- `CLAUDE.md` — Claude Code guidance (keep consistent).
- `docs/PRODUCT.md` — Owner 已确认产品方向的唯一入口；未确认字段必须保持 `unknown`。
- `docs/CURRENT.md` / `docs/BACKLOG.md` — 当前管理状态和任务顺序；`NOW` 最多一项。
- `docs/governance/` — Owner-first and platform-first multi-Agent governance rules.
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` — `superseded / unconfirmed` 历史方向提案。
- `docs/INDEX.md` — full doc index.
- `docs/PROJECT_STATUS.md` — current new-stack status.
- `docs/ACTIVE_STACK_POLICY.md` — active/legacy policy.
- `docs/CODEBASE_ANALYSIS.md` — codebase analysis snapshot, knowledge graph usage, and regeneration guidance.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` — page map.

## Project-Level Codex Agents

Project-scoped Codex agent definitions live in `.codex/agents/`:

- `researcher` — read-only discovery and evidence gathering.
- `implementer` — scoped implementation after the task card is approved.
- `reviewer` — read-only correctness, risk, and scope review.
- `qa` — automated and user-path verification; does not fix product code.

The root Agent coordinates them. Prefer parallel Agents for independent read-heavy work; do not let multiple implementation Agents edit the same surface concurrently.

## Project Medical Record

> Last updated: 2026-07-06. Read this before any work. It prevents repeating mistakes.
> For the latest verification status, run: `cd backend-go && go test ./...`

### ✅ What Works (verified this session)

- `go build ./...` — passes
- `go vet ./...` — passes
- `go test ./...` — 96 packages green, 11 pkgs no-test (107 total), 0 failures
- Frontend: `npm run dev` — starts on port 3001 (but dev server can exit unexpectedly)
- Login: the historical `admin / admin123456` credential is no longer valid in the current local database; use an existing valid Owner account or the approved credential-reset procedure.
- All 30+ frontend pages render (product hub, categories, brands, SKU, inventory, orders, agents, AI command center, etc.)
- Seed data in DB: 5 categories, 3 brands, 2 platforms (Ozon + Shopee), product + SKU + inventory

### 🐛 Known Issues (unfixed)

| Priority | Issue | Location |
|----------|-------|----------|
| P0 | Agent output is stub (fake data, not real LLM) | `orchestrator.go:172` — `synthesizeOutput()` |
| P1 | MoA aggregation is structured but still deterministic, not LLM-synthesized | `moa.go` — `synthesize()` returns structured findings/conflicts/recommendation |
| P1 | Owner dashboard /owner is Mock | `frontend-next/src/app/(main)/owner/` |
| P2 | Only 3 platform adapters (Ozon + Shopee + Shopify), still thinly tested | `domain/integrations/` |
| P2 | Frontend dev server has no watchdog — exits silently | `npm run dev` process |
| P3 | No real CI trigger yet (doc-links job added but not tested) | `.github/workflows/ci.yml` |

### 🛠️ What Was Fixed (2026-07-06)

- **工程可信度恢复**: `go build ./...` / `go vet ./...` / `go test ./...` 全绿
  - 修复 `internal/common/types.go` 中 `UserIDFromCtx` 重复定义（删除第二个副本）
  - 清理 `internal/ai/` 6 个文件共 19 处 merge conflict（handler.go, service.go, orchestrator.go, routes.go, model.go, ai_test.go）
  - 冲突来自 merge commit `964e0624`（合并远程 main v0.4.0），保留 HEAD 版本
- Merge conflicts in `routes.go` + `router.go` (HEAD won over worktree-wf)
- Duplicate `UserIDFromCtx` in `types.go` (kept first, deleted second)
- AuthGuard SSR crash (`useState` reading localStorage during server render → `useEffect` + `mounted`)
- RBAC endpoint 404 (frontend called `/v1/rbac/current/permissions` but route was unregistered)
- Inventory + product-hub 403 (operator users not linked to `ops` RBAC role → migration `000064`)
- Supplier test failure (handler read `c.Query()` but route used path param → fixed to `c.Param()`)
- Owner test failure (test CREATE TABLE missing `requester_user_id` + `reviewer_user_id` columns)
- Doc dead links removed from CLAUDE.md, INDEX.md, KERNEL_CONTRACTS.md

### 🏛️ Project Rules (Do Not Violate)

1. **Doc sync is mandatory.** Changing module names, API paths, or package layout requires updating AGENTS.md, CLAUDE.md, and docs/INDEX.md. CI `doc-links` job rejects stale references.
2. **Do not touch `.kilo/worktrees/`** — managed by external tooling.
3. **Do not rewrite history.** No `git rebase` on shared branches (main, feat/*, codex/*).
4. **Test before commit.** Minimum: `go build ./...` + `go vet ./...` + `go test ./...` for touched packages.
5. **No unrequested refactors.** Match existing patterns. Drive-by style changes are rejected.
6. **Old-stack docs (superpowers/plans/ etc.) are marked deprecated** — do not treat as executable instructions.
7. **Keep frontend API path format consistent:** `/api/v1/*` prefix, apiClient with `/v1/*` paths.

### 🔔 Cron Jobs

| Name | Schedule | What it does |
|------|----------|-------------|
| 文档链接审计 | Mon 9:00 | Checks AGENTS.md/CLAUDE.md/INDEX.md for dead links |
| 依赖安全检查 | Mon 10:00 | go mod verify + npm audit |
| 每周健康检查 | Mon 9:00 | Full test suite + git status + service check |
- `docs/features/` — feature specs; use `TEMPLATE.md`.
