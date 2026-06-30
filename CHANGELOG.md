# Changelog

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
