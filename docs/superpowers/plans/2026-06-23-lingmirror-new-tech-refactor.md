# LingMirror Complete New-Tech AI Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fully rebuild LingMirror as a new AI-native AgentOS on Go + Next.js in 7 calendar days, including all existing backend modules, all existing frontend pages, all AgentOS/Agent functions, and the new AI Command Center, Copilot, realtime reasoning, trace replay, and action approval experience.

**Architecture:** Build `backend-go/` and `frontend-next/` as full replacements for the current Python/FastAPI backend and Vue frontend. PostgreSQL remains the system of record during migration, but the new stack owns all `/api/*` routes by final cutover. The old stack remains hot only as rollback, not as a planned feature fallback.

**Tech Stack:** Go + Gin + GORM + golang-migrate + PostgreSQL + Redis/asynq + gorilla/websocket + SSE; Next.js + React + TypeScript + Ant Design + shadcn/ui + TanStack Query + Zustand + cmdk + reconnecting-websocket; Playwright + k6 + Go tests.

---

## Non-Negotiable Scope

This is a complete full-stack refactor. Do not reduce this to “core only.”

The 7-day delivery must include:

- All existing backend API modules.
- All existing frontend pages and routes.
- All Agent and AgentOS workflows.
- All existing auth/RBAC behavior.
- All catalog, platform, listing, listing task, shipping, fee, order, settlement, finance, decision, allocation, import, image generation, notification, exception, dashboard, search, audit, aftersales, sourcing, and integration functionality.
- New AI pages and AI runtime.
- Data migration and validation.
- Full cutover and rollback plan.

No planned “legacy fallback” is allowed for normal operation after cutover. Legacy Python/Vue is rollback only.

## New AI Product Surface

The refactor must add these new AI-native pages on top of full legacy parity:

### `/ai` AI Command Center

- Natural-language command bar.
- Agent roster grouped by squad.
- Live decision stream.
- Evidence chips.
- Confidence and risk indicators.
- Proposed action cards.
- Realtime event ticker.
- Persistent Copilot.

### `/agentos` AgentOS Cockpit

- Squad health map.
- Agent lanes.
- Pending approvals by risk.
- Autonomy/trust controls.
- Entropy and rule-health warnings.
- Work queue and SLA status.

### `/agents/[id]/trace/[traceId]` Trace Replay

- Reasoning timeline.
- Prompt/model metadata.
- Tool-call inputs and outputs.
- Evidence references.
- Rule hits and vetoes.
- Final recommendation and linked action.

### `/actions/[id]` Action Review Room

- Before/after comparison.
- Evidence drawer.
- Risk explanation.
- Modify payload.
- Approve/reject/execute/review.
- Audit timeline.

### Copilot Everywhere

- Explain current object/page/action.
- Answer why an Agent recommended something.
- Draft approval/rejection notes.
- Generate filters.
- Launch Agent runs.
- Surface urgent work.

## Complete Backend Module Coverage

Every current Python module must have a Go equivalent.

| Domain | Go Package | Required Coverage |
|---|---|---|
| auth | `internal/auth` | login, refresh, current user, JWT middleware |
| rbac | `internal/rbac` | role, permission, checks, admin bypass |
| core/common | `internal/httpx`, `internal/common` | response envelope, pagination, upload/static helpers |
| category | `internal/domain/category` | full CRUD |
| brand | `internal/domain/brand` | full CRUD |
| sku | `internal/domain/sku` | full CRUD, generation, lookup |
| price | `internal/domain/price` | pricing rules and updates |
| inventory | `internal/domain/inventory` | stock, safety stock, lock/unlock, allocation visibility |
| supplier | `internal/domain/supplier` | full CRUD |
| platform | `internal/domain/platform` | platform config |
| listing | `internal/domain/listing` | publish flow, adapters, status sync |
| listing_task | `internal/domain/listingtask` | queue, retry, AI listing workbench data |
| shipping | `internal/domain/shipping` | channels, fee calculation |
| platform_fee | `internal/domain/platformfee` | fee rules and calculation |
| order | `internal/domain/order` | order list/detail/status |
| order_import | `internal/domain/orderimport` | import batches, parsing, sync status |
| settlement | `internal/domain/settlement` | settlement list/detail/reconcile |
| finance | `internal/domain/finance` | ledger, accounts, profit summaries |
| decision | `internal/domain/decision` | pre-listing profitability decision |
| allocation | `internal/domain/allocation` | warehouses, rules, inventory allocation, cost allocation |
| agent | `internal/agent` | 10 Agents, registry, decision execution, rules, evolution |
| agent_actions | `internal/agent/actions` | legacy-compatible action APIs |
| agentos | `internal/agentos` | cockpit, squads, work items, action center |
| exceptions | `internal/domain/exceptions` | exception workbench |
| notification | `internal/domain/notification` | notifications and alert rules |
| dashboard | `internal/domain/dashboard` | all dashboard summaries |
| search | `internal/domain/search` | global search |
| image_gen | `internal/domain/imagegen` | image generation jobs and status |
| import_batch | `internal/domain/importbatch` | batch imports |
| operation_log | `internal/domain/operationlog` | mutation audit logs |
| platform_integrations | `internal/domain/integrations` | connection config and status |
| aftersales | `internal/domain/aftersales` | returns/after-sales flows |
| sourcing1688 | `internal/domain/sourcing1688` | candidate list/detail/import/reject |
| report | `internal/domain/report` | report pages and summaries |
| AI runtime | `internal/ai` | orchestrator, tools, traces, prompts, streaming |
| realtime | `internal/realtime` | WebSocket hub and event broadcast |

## Complete Frontend Page Coverage

Every Vue route must have a Next.js replacement page.

Required app structure:

- `/ai`
- `/dashboard`
- `/products`
- `/products/[id]`
- `/products/create`
- `/categories`
- `/brands`
- `/sku`
- `/inventory`
- `/suppliers`
- `/platforms`
- `/platform-integrations`
- `/listings`
- `/listings/create`
- `/listing-tasks`
- `/listing-tasks/workbench`
- `/orders`
- `/order-import`
- `/shipping`
- `/platform-fees`
- `/finance`
- `/settlement`
- `/decision`
- `/allocation`
- `/allocation/cost`
- `/agents`
- `/agents/actions`
- `/agents/evolution`
- `/agents/entropy`
- `/agents/[id]`
- `/agents/[id]/trace/[traceId]`
- `/agentos`
- `/agentos/work-items`
- `/actions/[id]`
- `/exceptions`
- `/notifications`
- `/image-gen`
- `/image-gen/canvas`
- `/import-batches`
- `/operation-logs`
- `/search`
- `/reports`
- `/aftersales`
- `/sourcing1688`
- `/settings`
- `/settings/llm`
- `/settings/rbac`

Page parity rules:

- Every old route must be reachable.
- Every old list page must support search/filter/pagination.
- Every old create/edit form must submit to Go.
- Every old detail page must display equivalent fields.
- Every mutation must show success/failure state and write audit.
- New AI pages must not replace old detail pages; they augment them.

## AI Backend Data Model

Add AI-native tables and unified action center. Do not drop old tables until after production validation.

```sql
CREATE TABLE IF NOT EXISTS ai_trace (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT,
    agent_id VARCHAR(20) NOT NULL,
    decision_point VARCHAR(80) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'running',
    model_provider VARCHAR(80),
    model_name VARCHAR(120),
    prompt_version VARCHAR(80),
    input_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    final_output JSONB,
    confidence NUMERIC(5,4),
    risk_level VARCHAR(20),
    token_count INTEGER,
    latency_ms INTEGER,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_trace_event (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL REFERENCES ai_trace(trace_id),
    event_type VARCHAR(64) NOT NULL,
    seq INTEGER NOT NULL,
    content TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (trace_id, seq)
);

CREATE TABLE IF NOT EXISTS ai_evidence_ref (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL REFERENCES ai_trace(trace_id),
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS unified_action (
    id BIGSERIAL PRIMARY KEY,
    source_table VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    agent_id VARCHAR(20),
    squad_id VARCHAR(50),
    user_id BIGINT,
    action_type VARCHAR(100) NOT NULL,
    business_object_type VARCHAR(64),
    business_object_id VARCHAR(128),
    title TEXT NOT NULL,
    description TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    before_snapshot JSONB,
    after_snapshot JSONB,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',
    requires_approval BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(32) NOT NULL DEFAULT 'suggested',
    confidence NUMERIC(5,4),
    proposed_by VARCHAR(100),
    approved_by VARCHAR(100),
    rejected_by VARCHAR(100),
    executed_by VARCHAR(100),
    rejection_reason TEXT,
    proposed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    executing_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_table, source_id)
);
```

## Team Structure

This 7-day plan assumes a large team and strict lane ownership.

| Lane | Headcount | Ownership |
|---|---:|---|
| Backend Platform | 3 | scaffold, DB, auth, RBAC, audit, migrations, release |
| Backend Commerce A | 4 | product/category/brand/SKU/price/inventory/supplier/allocation |
| Backend Commerce B | 4 | platform/listing/listing_task/shipping/fees/order/import/settlement/finance |
| Backend Ops | 3 | notification/exception/dashboard/search/image/import_batch/operation_log/integrations/aftersales/sourcing/report |
| AI Runtime | 5 | Agent registry, tools, traces, streaming, action center, entropy, autonomy |
| Frontend Shell | 3 | Next shell, routing, auth, layout, design system, API client |
| Frontend Commerce A | 4 | catalog/SKU/inventory/supplier/allocation pages |
| Frontend Commerce B | 4 | platform/listing/order/shipping/finance/settlement pages |
| Frontend Ops | 3 | notification/exception/image/search/report/settings/rbac pages |
| Frontend AI | 5 | `/ai`, `/agentos`, trace replay, action review, Copilot, command palette |
| QA/Data/Release | 4 | migration validation, E2E, k6, parity, cutover |
| Tech Leads | 2 | API contract, architecture, blockers, final cutover |

Minimum credible staffing: 44 engineers plus 2 leads. If staffing is much lower, keep the full scope but move the date, not the scope.

## 7-Day Execution Plan

### Day 0: Full Scope Lock

- [ ] Assign every module and page to a named owner.
- [ ] Freeze legacy Python/Vue changes except P0.
- [ ] Export current OpenAPI route inventory.
- [ ] Export frontend route inventory.
- [ ] Export DB table inventory and row counts.
- [ ] Lock response envelope and pagination format.
- [ ] Lock AI event protocol.
- [ ] Lock unified action status flow.
- [ ] Create rollback DB backup and test restore.

Acceptance:

- Every route and page has an owner.
- No unowned module remains.
- Old stack still runs.

### Day 1: Platform And Skeleton Parity

Backend Platform:

- [ ] Scaffold `backend-go`.
- [ ] Add config, DB, logging, CORS, recovery, request ID.
- [ ] Add `Result` and `PageResult`.
- [ ] Add auth middleware skeleton.
- [ ] Add route registration structure for every module.
- [ ] Add migration runner.
- [ ] Add `/api/health`.

Frontend Shell:

- [ ] Scaffold `frontend-next`.
- [ ] Add app shell, auth shell, route guard, sidebar, header, breadcrumbs.
- [ ] Add all target routes as pages with real empty states.
- [ ] Add API client and error boundary.
- [ ] Add design tokens and component wrappers.

AI Runtime:

- [ ] Add AI migrations.
- [ ] Add trace writer.
- [ ] Add SSE writer.
- [ ] Add WebSocket hub skeleton.

QA/Data:

- [ ] Add route parity checklist.
- [ ] Add page parity checklist.
- [ ] Add seed-data verification command.

Acceptance:

- Every target route exists in Go and Next, even if some handlers return temporary structured empty data.
- `/api/health` and `/ai` work.
- Migrations apply.

### Day 2: Commerce Backend And Catalog Frontend

Backend Commerce A:

- [ ] Implement category, brand, product, SKU, price, inventory, supplier.
- [ ] Implement allocation warehouse/rule/inventory/cost allocation.
- [ ] Add operation logs for mutations.

Frontend Commerce A:

- [ ] Implement category, brand, product, SKU, inventory, supplier, allocation pages.
- [ ] Add list/detail/create/edit flows.
- [ ] Add table filters, pagination, loading, empty, error states.

AI Runtime:

- [ ] Implement AI tools for SKU lookup, inventory lookup, profit snapshot.
- [ ] Add evidence references for catalog/inventory data.

QA:

- [ ] Add E2E for product CRUD, SKU lookup, inventory update, allocation rule.

Acceptance:

- Catalog/inventory/allocation parity works end to end.
- AI can pull evidence from SKU/inventory.

### Day 3: Commerce Backend B And Transaction Frontend

Backend Commerce B:

- [ ] Implement platform, platform integrations, listing, listing tasks.
- [ ] Implement shipping, platform fee.
- [ ] Implement order, order import.
- [ ] Implement settlement and finance.
- [ ] Implement marketplace adapter compatibility boundaries.

Frontend Commerce B:

- [ ] Implement platform, integrations, listing, listing task, AI listing workbench.
- [ ] Implement order, order import.
- [ ] Implement shipping, platform fee, settlement, finance.

AI Runtime:

- [ ] Implement listing diagnosis tool.
- [ ] Implement order anomaly lookup tool.
- [ ] Implement finance evidence tool.

QA:

- [ ] Add E2E for listing flow, order import, order detail, settlement, finance summary.

Acceptance:

- Transaction and finance pages work through Go APIs.
- AI evidence can include listing/order/finance data.

### Day 4: Ops Modules, Agent Runtime, Unified Action

Backend Ops:

- [ ] Implement notifications and alert rules.
- [ ] Implement exceptions workbench.
- [ ] Implement dashboard summaries.
- [ ] Implement global search.
- [ ] Implement image generation job APIs.
- [ ] Implement import batches.
- [ ] Implement operation logs.
- [ ] Implement aftersales, sourcing1688, reports, settings.

AI Runtime:

- [ ] Implement Agent registry for A1-A7 and G1-G3.
- [ ] Implement Agent orchestrator.
- [ ] Implement rule engine.
- [ ] Implement autonomy/trust endpoints.
- [ ] Implement entropy endpoints.
- [ ] Implement unified action repository.
- [ ] Backfill from old action tables.
- [ ] Implement action lifecycle.
- [ ] Link new Agent runs to `ai_trace` and `unified_action`.

Frontend Ops:

- [ ] Implement notifications, exceptions, dashboard, search, image gen, import batches, operation logs.
- [ ] Implement aftersales, sourcing1688, reports, settings, RBAC.

Frontend AI:

- [ ] Implement `/actions/[id]` Action Review Room.
- [ ] Implement trace timeline skeleton.

QA:

- [ ] Add E2E for notification, exception, search, image job, operation logs.
- [ ] Add action backfill validation.

Acceptance:

- All non-commerce modules have Go APIs and Next pages.
- Agent can create unified action.
- Action approval/reject/execute works.

### Day 5: AI Pages And Realtime

AI Runtime:

- [ ] Implement WebSocket event publishing.
- [ ] Implement SSE streaming for Agent reasoning.
- [ ] Implement `/api/ai/chat`.
- [ ] Implement tool-call trace events.
- [ ] Implement evidence persistence.
- [ ] Implement prompt/policy version metadata.

Frontend AI:

- [ ] Implement `/ai` AI Command Center.
- [ ] Implement `/agentos` AgentOS Cockpit.
- [ ] Implement `/agents/[id]/trace/[traceId]`.
- [ ] Implement Copilot panel.
- [ ] Implement Cmd+K command palette.
- [ ] Implement evidence drawer.
- [ ] Implement live Agent lanes.
- [ ] Implement streaming reasoning renderer.

All Frontend Lanes:

- [ ] Add Copilot context hooks to every page.
- [ ] Add “open in AI” affordances from every major object page.

QA:

- [ ] Add E2E: run Agent -> stream -> evidence -> action -> approve -> realtime update -> trace replay.
- [ ] Test 100 WebSocket clients.

Acceptance:

- AI Command Center is fully usable.
- AgentOS Cockpit is fully usable.
- Copilot works across all major pages.
- Realtime event state stays consistent.

### Day 6: Full Parity Closure

All lanes:

- [ ] Complete route parity checklist.
- [ ] Complete page parity checklist.
- [ ] Complete permission parity checklist.
- [ ] Complete data parity checklist.
- [ ] Complete audit parity checklist.
- [ ] Complete AI trace/action checklist.
- [ ] Freeze API changes by noon.
- [ ] Fix P0/P1 only after freeze.
- [ ] Run all Go tests.
- [ ] Run Next build.
- [ ] Run full Playwright suite.
- [ ] Run k6:
  - 100 concurrent dashboard users
  - 100 concurrent `/ai` users
  - 50 concurrent action approvers
  - 100 WebSocket clients
  - 1000 SKU Agent dry-run batch

Acceptance:

- No unimplemented route.
- No unimplemented page.
- No P0/P1 open.
- AI trace events are complete and ordered.
- Data validation passes.

### Day 7: Full Cutover

Release:

- [ ] Take final DB backup.
- [ ] Run migrations.
- [ ] Run full backfill.
- [ ] Run row-count validation.
- [ ] Run sample checksum validation.
- [ ] Run smoke suite.
- [ ] Route internal users to Go/Next.
- [ ] Monitor auth, API 5xx, p95 latency, WebSocket disconnects, action failures, trace gaps, data mismatches.
- [ ] Route all users to Go/Next after clean internal window.
- [ ] Keep Python/Vue hot for rollback for 72 hours.

Acceptance:

- Go owns `/api/*`.
- Next owns the app UI.
- All old routes are represented.
- All new AI pages are live.
- Legacy stack is rollback only.

## Hard Gates

No cutover if any fail:

- Any old route lacks a Go equivalent.
- Any old frontend route lacks a Next page.
- Auth/RBAC bypass found.
- Data migration loses source rows.
- Operation logs missing for mutations.
- Action lifecycle mutates the wrong row.
- Completed Agent run has missing/out-of-order trace events.
- Evidence cannot be traced to source data.
- WebSocket duplication creates inconsistent action status.
- Full Playwright suite fails.
- Full data validation fails.

## Validation Commands

```bash
cd backend-go && go test ./...
```

```bash
cd frontend-next && npm run build
```

```bash
cd frontend-next && npx playwright test
```

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest -q
```

```bash
cd frontend && npm run build
```

## Final Recommendation

Execute this only as a full-team rewrite. The scope is complete parity plus new AI-native product surfaces. The schedule can stay at 7 days only if staffing and ownership match the plan; otherwise the scope stays fixed and the date moves.
