# Changelog

## v0.4.1.0 (2026-07-06) — 限制链路加固

### Added
- **MutationGuard 全覆盖** — 8+ EventBus 变更处理器现在被 MutationGuard 包裹（trustscore/entropy/ozon sync/sourcing/aftersale/stock/listing/agentos/metabolism），审计日志覆盖 pending→executed/failed 全周期。
- **系统动作注册** — 10 个 `system.*` 内部动作已注册到 ActionCatalog（信任分升级/熵防御/审批创建/集成同步/供应链流/售后/库存/上架/SLA/代谢），满足架构合规要求。
- **DispatchSafe 架构文档** — KERNEL_CONTRACTS.md 明确 raw Dispatch vs DispatchSafe 选择策略。

### Fixed
- **Compilation 修复** — 解决 internal/ai 中 8 个文件的 Git 冲突标记，UserIDFromCtx 去重，所有调用点参数顺序统一。
- **Scheduler JWT** — `GET /api/v1/aios/scheduler/tasks` 从未认证的 root engine 移至 JWT 保护的 protected 组。
- **动作命名统一** — `inventory_update→inventory_change`, `platform_publish→listing_publish`, `data_delete→destructive_data_change`，涉及 6 个文件（migration seed + guardrails test + 3 doc + engine ponytail）。
- **Dead code 清理** — 删除 handler.go 中未使用的局部 `userIDFromCtx` 函数。
- **KERNEL_CONTRACTS 措辞修正** — "guard enforces" → "guard ensures"，明确目录注册为强制性约定而非代码执行。

### Governance
- **限制清单文档** — `docs/RESTRICTION_LIST.md` 面向 Owner 的完整限制说明（自动放行/审批/禁止/审计查询/变更方法）。
- **路由安全审计报告** — `deliverables/reports/route-security-audit.md` 含 70+ 路由的 JWT/RBAC/审计流量灯表。
- **限制链路验证报告** — `deliverables/reports/restriction-chain-verify.md` 覆盖 ActionCatalog/DispatchSafe/forbidden_action/审批/ToolBridge/MutationGuard 6 项。

## v0.4.0 (2026-07-05) — 可信 AgentOS 执行门禁

### Added
- **Execution mode safety** — ExecuteAction now checks execution_mode before proceeding. `dry_run` validates through all gates without side effects, `sandbox` is rejected when no executor is configured, unknown modes return errors instead of falling through to production.
- **Idempotency key support** — UnifiedAction.idempotency_key with atomic conditional UPDATE (`WHERE status IN ?` + `RowsAffected` check) prevents concurrent double-execution of the same action. If already completed, returns existing result.
- **JWT identity binding** — approval (approve/reject) and execution actions now persist `approved_by_user_id`, `rejected_by_user_id`, and `executed_by_user_id` from the server JWT context, not client-supplied values.
- **Closed-loop approval events** — approval Review now publishes lifecycle events (`approval.{status}.{request_type}`) to the event bus, triggering downstream workflows like listing task auto-creation on `approval.approved.listing_task`.
- **Guardrails integration** — L4 ExecutionGuard runs on all production ExecuteAction calls, validating payloads through the AIOS guardrails chain before dispatching.
- **Database migration 000064** — adds execution gate columns (`execution_mode`, `idempotency_key`, `approved_by_user_id`, `executed_by_user_id`, `rejected_by_user_id`, `requester_user_id`, `reviewer_user_id`) with partial and full indices.

### Fixed
- **Frontend risk level handling** — ActionConfirmModal now normalizes risk level case and validates high-risk confirmations before proceeding.

### Tests
- **ai** — 6 new execution gate tests: execution mode persistence, dry_run side-effect prevention, unknown/sandbox mode rejection, user ID persistence on approval and execution
- **approval** — 1 new event publication test: verifies Review() publishes `approval.approved.listing_task` with correct payload

## v0.2.3 (2026-06-30) — 测试 & 管线 bugfix

### Bugfixes
- **Pipeline agent triggering** — triggerAgent sends correct decision point names matching agent registry. Adds content_ai and scheduler to DefaultRegistry.
- **Approval.Stats()** — removed PostgreSQL-specific EXTRACT for SQLite test compatibility.
- **PipelineOrchestrator** — nil-safe guard for aiOrch field.

### Tests
- **agentlearning** — 8 tests: pure helpers + RecalculateAccuracy CRUD
- **approval** — 17 tests: service CRUD, handler endpoints, event bus subscriber, Stats/AutoEscalate
- **orchestration** — 15 tests: pipeline lifecycle, failure paths, config CRUD
- **ai** — Registry default agent count updated (14→16)

## v0.3.0 (2026-06-29) — July gap-fill P2 final

### New Modules
- **Prism image compliance** (`backend-go/internal/prismadapter/`) — HTTP client + PrismService interface for generating platform-compliant product images. Integrated into listing task execution flow.
- **Supply chain orchestrator** (`internal/domain/supplychain/`) — bridges A8 sourcing with A10 logistics quoting, handles reverse logistics via return events.
- **Tariff engine** (`internal/domain/tariff/`) — customs duty and tax calculation for cross-border shipping.
- **Content AI** (`internal/domain/content/`) — LLM-powered content generation for product listings.

### New Agents (M1, A10, A11)
- M1 (代谢排泄评分), A10 (物流运费引擎), A11 (售后管理) — total agent roster now 15+3.

### Platform Evolution
- **Agent autonomy upgrade** — A4 (Customer Service) promoted from guided to autonomous; A2 (Listing Optimizer) from advisory to guided; G0 → supervised, G1/G3 → supervised.

### Platform Infrastructure
- **EventBus metrics** — new metrics: topic latency histogram, publish error counter, subscriber duration histogram
- **EventBus DLQ** — dead-letter queue with storage-backed retry for failed events (max 3 deliveries)
- **EventBus context propagation** — context-aware publish/subscribe with cancellation and timeout
- **Scheduler observability** — task metrics: execution duration, error count, success/failure tracking

## v0.2.2 (2026-06-29) — Compliance Risk Engine

### New Modules
- **Compliance Risk Engine** (`internal/domain/compliance/`) — product compliance checking with HS code validation, banned terms, and certificate requirements per platform

### Performance
- **Connection pool** — GORM max open connections reduced from 100 to 25; per-operation timeout added

### Documentation
- **Architecture decision records** — ADR-0001: GORM with In-Memory SQLite for Tests; first ADR in docs/adr/
- **Onboarding docs** — docs/AGENT_CAPABILITIES.md now fully enumerates all MCP servers, API endpoints and CLI tools

## v0.2.1 (2026-06-26) — July gap-fill P1

### New Modules
- **Import Batch** (`internal/domain/importbatch/`) — bulk import management with file upload and status tracking
- **Sourcing 1688** (`internal/domain/sourcing1688/`) — 1688 platform adapter for sourcing and supplier discovery
- **Purchase Order** (`internal/domain/purchase/`) — purchase order creation and management workflow
- **Operation Log viewer** (`internal/domain/operationlog/`) — full-text search and filtering for audit logs
- **Routing panel** (`frontend-next/src/app/(main)/routing/`) — logistics routing UI with carrier comparison

### Platform Governance
- **Action Policy** (`internal/domain/actionpolicy/`) — approval rules for agent actions, with rule evaluation engine and risk-based routing
- **Orchestration** (`internal/domain/orchestration/`) — product lifecycle pipeline (sourcing → enrichment → compliance → pricing → listing → monitoring → delisting)

### Ecosystem
- **UX review** — P2 gap-fill UX audit reduced high-severity issues by 40%: new product onboarding, first-run state, model P0 indicators, action hint positioning
- **Landed cost** — profit formula now includes landed cost (platform fee + tariff + logistics) per A8 sourcing flow
- **Stock page UI** — table-only stock page replaced with Ant Design multi-tab layout (all / low / out-of-stock)
- **Multi-Agent protocol** — 4 new cross-Agent handoff rules added to AGENTS.md

## v0.2.0 (2026-06-25)

### Added
- Initial release with core modules: product hub, listing, order, profit, sourcing, logistics, finance, compliance, content, ai
- Agent OS runtime with 11 agents (A1-A8, G0-G3)
- Cross-border e-commerce platform adapters (Shopee, Ozon, Lazada)
