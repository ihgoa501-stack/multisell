# Changelog

## v0.3.0 (2026-06-29) — July gap-fill P2 final

### New Modules
- **Prism image compliance** (`backend-go/internal/prismadapter/`) — HTTP client + PrismService interface for generating platform-compliant product images. Integrated into listing task execution flow.
- **Supply chain orchestrator** (`internal/domain/supplychain/`) — bridges A8 sourcing with A10 logistics quoting, handles reverse logistics via return events.
- **Tariff engine** (`internal/domain/tariff/`) — customs duty and tax calculation for cross-border shipping.
- **Content AI** (`internal/domain/content/`) — LLM-powered content generation for product listings.

### New Agents (M1, A10, A11)
- M1 (代谢排泄评分), A10 (物流运费引擎), A11 (售后管理) — total agent roster now 15+3.

### New Frontend Pages
- `/candidates` — candidate product management with CRUD table, seed data button, completeness check modal
- `/image-gen` — AI image generation history, canvas management, prompt templates

### Extended Modules
- **Sourcing 1688** (`internal/domain/sourcing1688/`) — direct 1688 data integration
- **Personal rule engine** (`internal/domain/personalrule/`) — user-defined listing rules with score-based matching
- **Metabolism** — M1 excretion scoring with semantic analysis
- **Evolution** — agent evolution nudges and upgrade paths
- **Entropy** — self-cleansing SPC control, health scoring, defenses
- **EventBus** — extracted Backend interface, InProcessBackend struct, WithBackend option
- **Scheduler** — added sync.Map guards with TryLock to prevent overlapping ticks
- **Listing task** — ExecuteTask now calls Prism for compliance check before platform publish

### Infrastructure
- **Caddyfile** — dev and prod Caddy configurations for reverse proxy + TLS
- **Monitoring** — Prometheus + Grafana dashboards, k6 load testing suite
- **ToolBridge** — plugin-driver-based tool execution bridge for agent tool use
- **Chrome extension** — browser extension with sidebar panel and WebSocket protocol

### Documentation
- `docs/prism-integration.md` — how to enable, configure, and operate Prism in the publish flow
- `docs/AGENT_CAPABILITIES.md` expanded — all 15+ agents documented
- `docs/governance/` — Owner-first development protocol, platform constitution, agent protocol, kernel contracts
- `docs/7_DAY_AGENT_COMPANY_MVP_PLAN.md` + battle board — 7-day MVP sprint plan

### Fixed
- P1 T2–T7: EventBus async worker buffer, error info leak on API responses, AgentOS error handling in UI, N+1 queries in listing/product queries, entropy defense timer precision, agent pipeline timeout on sync path
- P2 T8–T9: Dead code removal in legacy Python stack, JWT refresh audit logging

## v0.2.1 (2026-06-26) — July gap-fill P1

### New Modules
- **Sourcing domain** (`internal/domain/sourcing/`) — A8 agent-backed profit formula engine that aggregates 1688 price, exchange rate, logistics rate, platform fee, weight estimate, and target sale price into a single margin recommendation. Eval engine with confidence scoring. Routes defined (`POST /api/v1/sourcing/fetch`, `GET /api/v1/sourcing/recommendations`) but **not yet wired** in `router.go`.
- **Logistics rate engine** (`internal/domain/logistics/`) — clean-slate rate calculator with four pricing modes (first_additional, tiered, fixed, per_kg), YAML rate table loading, fuel surcharge support. Independent of the shipping domain package.
- **ToolBridge** (`internal/platform/toolbridge/`) — plugin-driver-based tool execution bridge so agents can run external tools through registered plugins.
- **Chrome extension** (`chrome-extension/`) — browser extension with content script injection, sidebar panel, and real-time WebSocket communication. Protocol defined in `shared/protocol.ts`.

### New Agents (A8–A11)
- A8 (选品盈利分析), A9 (批量运维), A10 (物流运费引擎), A11 (售后管理) — bringing the total agent roster from 11 to 15.

### New Frontend Pages
- `/sourcing` — AI sourcing panel powered by the sourcing domain API
- `/metabolism` — M1 metabolism scoring engine UI (waste detection, event outbox scoring)

### Extended Modules
- **Aftersales** — platform after-sales order sync (`sync.go`)
- **Allocation** — extended cost allocation dimensions
- **Import batch** — new YAML/JSON parser (`parser.go`) and async processor (`processor.go`)
- **Inventory** — stock field extensions and alert rule tuning
- **Realtime hub** — extension handler for WebSocket connections
- **Platform integrations** — Shopee/Ozon adapter improvements

### Documentation
- `docs/AGENT_CAPABILITIES.md` — comprehensive reference of all MCP servers, API endpoints, agents, frontend pages, deployment info, and development commands. Now the standard onboarding document.
- `CLAUDE.md` — onboarding pointer to AGENT_CAPABILITIES.md

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>

## v0.2.0 (2026-06-25)

### Major
- New Go backend (`backend-go/`) with Gin/GORM/PostgreSQL — full replacement of Python/FastAPI stack
- New Next.js 15 frontend (`frontend-next/`) with Ant Design — replaces Vue 3/Naive UI
- AI AgentOS: 10 agents (A1-A7, G0-G3) with LLM-powered decision analysis, trust score, entropy defense
- Event-driven architecture: in-memory event bus with outbox persistence, scheduler, command dispatcher
- Platform integrations: Shopee, Ozon, 1688 API adapters
- Full CI/CD pipeline: GitHub Actions, Docker Compose (dev/prod/monitoring), deploy/rollback scripts
- Production-readiness: CORS, JWT auth, rate limiter, Sentry, Prometheus metrics, structured logging

### Infrastructure
- Q3 production readiness: middleware stack, graceful shutdown, auth guard on all API routes
- Prometheus + Grafana monitoring stack (`deploy/prometheus/`, `deploy/grafana/`)
- k6 load testing suite (`backend-go/loadtest/`)
- Pre-commit hooks (pre-commit-config.yaml)
- Versioned SQL migrations (000001-000011)

### Under Review
- [P1] T2-T7: EventBus async worker, error info leak, AgentOS error handling, N+1 queries, entropy defense timer, agent pipeline timeout
- [P2] T8-T9: Dead code removal, JWT refresh audit
- [P3] Frontend design review, distributed tracing, horizontal scaling deferred

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
