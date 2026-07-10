# Spec: AI-Native AgentOS Execution Path

## Assumptions

1. This spec guides development sequencing for LingMirror, not a single sprint implementation.
2. The active stack remains `backend-go/` + `frontend-next/`; Python is future cognition sidecar only.
3. Through Q4 2026, the product promise remains Copilot: AI recommends, Owner decides.
4. High-risk production actions remain approval-gated even if an Agent has high confidence.
5. The first commercial wedge is cross-border ecommerce profitability and listing decisions, not generic AgentOS or multi-industry automation.
6. Business Verified is currently blocked until browser E2E proves the main flows against a running backend and database.

If any assumption changes, update this spec before implementation.

## Objective

Build LingMirror from its current platform foundation into a trusted AI-Native AgentOS in a sequence that protects Owner control, business safety, and execution clarity.

The total path is:

```text
Trusted kernel
-> business-verified ecommerce Copilot
-> stateful sandbox and daily Owner cockpit
-> governed AIOS cognition sidecar
-> measured self-improvement
-> conditional multi-vertical expansion
```

The immediate objective is not to add more AI autonomy. It is to prove that an Owner can use LingMirror every day to answer:

- Can this product be sold profitably?
- Will this order or fulfillment path lose money?
- Which high-risk actions need approval, and what happens if I approve?

## Owner-Approved Defaults

These decisions are the default execution path until the Owner explicitly changes them:

| Question | Decision |
| --- | --- |
| First platform | Platform-neutral sandbox first; Ozon second; Shopee third. |
| First ICP | Small cross-border ecommerce Owner / operator, about 50-500 SKUs. |
| Next sprint | Product Loop E2E first. |
| MVP Business Verified threshold | 5 seeded realistic browser runs pass end to end. |
| Beta Business Verified threshold | 10 seeded realistic browser runs pass end to end. |
| CI/release hardening | Run in parallel with Product Loop E2E, but do not block first local business-flow proof. |

The current execution mainline is:

```text
Product Loop E2E
-> Action Gate closure
-> CI / E2E gate
-> Daily Owner Cockpit
-> Fulfillment Loop
-> Platform adapter sandbox
-> Python sidecar contract
```

## Tech Stack

| Layer | Current Stack | Notes |
| --- | --- | --- |
| Backend | Go / Gin / GORM / PostgreSQL | Canonical execution authority lives here. |
| Frontend | Next.js / React / TypeScript / Ant Design | Owner-facing workflows live here. |
| Kernel | Auth, RBAC, Approval, Audit, EventBus, Command, Scheduler, ToolBridge, Observability, Migrations | Must stay domain-agnostic. |
| AIOS future sidecar | Python / FastAPI or equivalent | Future read-only/suggestion sidecar only; no direct DB or platform writes. |
| Verification | Go test, vet, npm lint/type/test/build, Playwright E2E | E2E is required for Business Verified. |

## Commands

```bash
# Backend
cd backend-go && go test ./...
cd backend-go && go vet ./...
cd backend-go && go build -o bin/server cmd/server/main.go

# Frontend
cd frontend-next && npm run lint
cd frontend-next && npx tsc --noEmit --pretty false
cd frontend-next && npm test
cd frontend-next && npm run build

# E2E
cd frontend-next/e2e && npx playwright test

# Full stack
docker compose up -d
docker compose up -d db

# Repo-level verification, when available
./scripts/verify_all.sh
```

## Project Structure

```text
backend-go/internal/platform/       -> Platform Kernel contracts and mechanisms
backend-go/internal/ai/             -> AI action proposal, approval, execution orchestration
backend-go/internal/domain/*        -> Business domain modules
backend-go/internal/httpx/router.go -> Route composition and middleware
backend-go/migrations/              -> Database migrations

frontend-next/src/app/(main)/owner/         -> Owner cockpit
frontend-next/src/app/(main)/actions/       -> Action review / execution surface
frontend-next/src/app/(main)/listing-tasks/ -> Listing execution flow
frontend-next/src/app/(main)/agentos/       -> AgentOS operation views
frontend-next/src/components/actions/       -> Shared action confirmation UX
frontend-next/e2e/                          -> Browser flow verification

docs/governance/                   -> Non-negotiable system rules
docs/AI_NATIVE_DEVELOPMENT_PLAN.md -> Long-term vision
docs/specs/                        -> Executable planning specs
docs/PROJECT_STATUS.md             -> Current facts and verification status
docs/KNOWN_ISSUES.md               -> Open blockers
```

## Code Style

Backend business mutations should be explicit, typed, auditable, and idempotent.

```go
action := command.AgentAction{
    ActionType:       "listing.publish",
    Version:          "v1",
    AgentID:          agentID,
    Actor:            userID,
    TargetType:       "listing_task",
    TargetID:         taskID,
    RiskLevel:        command.RiskHigh,
    ApprovalRequired: true,
    ApprovalID:       approvalID,
    Mode:             command.ModeProduction,
    IdempotencyKey:   idempotencyKey,
    CorrelationID:    correlationID,
    Input:            input,
}
```

Frontend high-risk actions should present Owner-readable business context, not raw internal state:

```tsx
<HighRiskConfirmDialog
  riskLevel="high"
  targetLabel="Ozon listing publish"
  mode="sandbox"
  beforeValues={before}
  afterValues={after}
  auditDestination="operation_log"
  rollbackNote="The listing can be paused manually if publish succeeds with bad data."
/>
```

## Bottom Layer Plan

The bottom layer is the trust substrate. It must be complete before adding more autonomy.

### B0. Contract Closure

Goal: prove high-risk actions cannot bypass the canonical execution path.

Required work:

- Audit Owner, listing-task, integration, workflow, scheduler, ToolBridge, and Agent paths for mutation bypasses.
- Route high-risk actions through AgentAction / ActionCatalog / Approval / RBAC / Audit / Command.
- Document any intentional exception as deterministic system action with MutationGuard and audit.

Acceptance:

- Price, inventory, order, money, platform publish, credential, and destructive actions have approval/audit policy.
- Action creation, approval, rejection, execution, failure, and idempotency are covered by tests.
- There is no hidden production mutation path from UI or Agent workflow.

### B1. Runtime Reliability

Goal: make EventBus, Scheduler, Command, and ToolBridge observable and stable.

Required work:

- Verify EventBus/Scheduler lifecycle under server startup and shutdown.
- Ensure scheduled Agent ticks and event handlers carry correlation IDs.
- Add or enforce DLQ/failure visibility for event-driven business mutations.
- Ensure ToolBridge mutation tools cannot execute without Go Kernel gates.

Acceptance:

- EventBus/Scheduler lifecycle tests pass.
- Mutating event handlers are wrapped by MutationGuard or explicitly documented as non-mutating.
- Failed event/tool/action execution has Owner-visible status.

### B2. Integration Safety

Goal: make external writes impossible to confuse with dry-run or sandbox.

Required work:

- Define adapter capability metadata: supported operations, side effects, idempotency, rate limits, rollback/recovery, sandbox gaps.
- Enforce dry-run default for platform publishing and sync.
- Add static or CI checks that prevent raw provider write clients from bypassing safe constructors.

Acceptance:

- Ozon/Shopee publishing can run dry-run/sandbox with no external side effects.
- Production mode requires approval, RBAC, audit, idempotency, external reference ID, and visible failure state.

### B3. Verification Infrastructure

Goal: make Business Verified reproducible.

Required work:

- Fix E2E seed and startup path.
- Make product loop, fulfillment loop, and high-risk action gate runnable in Playwright.
- Archive E2E artifacts for current acceptance reports.
- Treat migration failure as deploy-blocking unless explicitly known as already-applied.

Acceptance:

- `scripts/verify_all.sh` or equivalent runs with E2E enabled.
- `docs/KNOWN_ISSUES.md` no longer has Business Verified blockers for main browser flows.

## Top Layer Plan

The top layer is the Owner product. It should reduce decisions, not expose platform complexity.

### T0. Daily Owner Cockpit

Goal: the first screen tells the Owner what matters today.

Required views:

- Top product opportunities.
- Loss-risk orders or fulfillment exceptions.
- Pending high-risk approvals.
- Failed or blocked actions with next steps.
- Recent outcomes and audit trail.

Acceptance:

- Owner can start from cockpit and complete the top decision without knowing internal module names.
- Every recommendation answers what, why, risk, expected outcome, approve consequence, reject/wait consequence, mode, and audit location.

### T1. Product Profitability Copilot

Goal: make the first business loop complete.

Flow:

```text
Candidate product
-> completeness check
-> cost / logistics / platform fee / profit calculation
-> listing recommendation
-> Owner approval
-> controlled listing task
-> result review
```

Acceptance:

- Browser E2E passes with seeded realistic data.
- Owner can see assumptions, missing data, expected margin, risk level, and action mode.
- Reject/edit feedback is captured as structured data.

### T2. Fulfillment Risk Copilot

Goal: make the second business loop complete.

Flow:

```text
Order
-> inventory and logistics choice
-> shipping cost snapshot
-> settlement and profit check
-> exception detection
-> Agent recommendation
-> Owner approval or manual handling
```

Acceptance:

- Browser E2E proves an order-risk scenario.
- Owner sees expected loss/profit, cause, recommended next step, and fallback if they wait.

### T3. Action Center

Goal: all high-risk actions share one visible lifecycle.

Lifecycle:

```text
suggested -> pending_approval -> approved/rejected -> executing -> completed/failed/blocked
```

Acceptance:

- Owner can filter actions by risk, mode, status, domain, and audit state.
- Failed actions have recovery notes and no silent terminal state.

### T4. Feedback And Learning Data

Goal: collect the data required before DSPy/Memory/self-improvement.

Required data:

- Approval / rejection reason.
- Owner correction notes.
- Outcome after execution or simulated execution.
- Recommendation source data and assumptions.
- Business result label: useful, wrong, incomplete, too risky, missing context.

Acceptance:

- Feedback is structured enough to build offline evaluation datasets later.
- Feedback does not automatically change production behavior.

## AIOS Future Layer Plan

AIOS cognition starts only after bottom and top layers pass Business Verified.

### A0. ProposedAgentAction Contract

Define the sidecar-to-Go envelope before building Python services.

Minimum fields:

- service identity
- tenant
- actor
- correlation ID
- schema version
- prompt/model/policy version
- source citations
- data sensitivity
- requested mode
- target type and ID
- proposed action input
- timeout/retry/failure semantics

### A1. Memory Contract

Memory is advisory context only. It must never grant permissions, waive approvals, alter risk level, override compliance, or select production mode.

Each memory item must include:

- source trace/action ID
- source table/document reference
- extraction job ID
- embedding model/version
- freshness
- tenant scope
- deletion policy
- training/evolution eligibility

### A2. Prompt And Model Registry

DSPy or any prompt optimizer may produce candidates only.

Registry fields:

- prompt version
- model version
- dataset version
- evaluator version
- approval record
- rollout mode
- shadow metrics
- rollback target
- Owner-readable diff

### A3. Shadow Mode Cognition

Python cognition may run in read-only/shadow mode first.

Acceptance:

- It produces recommendations without changing production behavior.
- Its output can be compared against current recommendations.
- Promotion requires offline evaluation, Owner approval, and rollback.

## Development Path

| Phase | Name | Primary Goal | Do Not Start Until |
| --- | --- | --- | --- |
| 0 | Product Loop E2E | Candidate -> profit -> recommendation -> approval -> controlled execution -> result review passes locally | Current dirty work is understood and scoped |
| 1 | Action Gate Closure | No high-risk mutation bypasses remain in Owner/listing/integration paths | Product loop path is visible enough to audit |
| 2 | CI / E2E Gate | Product Loop E2E and seed path are reproducible in verification scripts/CI | Local Product Loop E2E passes |
| 3 | Daily Owner Cockpit | Owner sees top decisions, pending approvals, failures, and recent outcomes | Product and action loops produce real statuses |
| 4 | Fulfillment Copilot | Order/fulfillment risk loop is usable | Product loop UX is stable |
| 5 | Platform Adapter Sandbox | Platform-neutral sandbox first, then Ozon/Shopee sandbox adapters | Product loop and action gate are stable |
| 6 | AIOS Contracts | Sidecar, memory, prompt, observability contracts exist | Business Verified is achieved |
| 7 | Python Shadow Sidecar | Read-only cognition and evaluation | AIOS contracts are accepted |
| 8 | Controlled Self-Improvement | Offline/shadow DSPy with approval | Enough labeled feedback data exists |
| 9 | Conditional Verticals | Finance/trade exploration | Ecommerce ROI and compliance gates pass |

## Expected Risks And Blockers

These blockers are expected. Future Agents should address them inside the Product Loop mainline instead of expanding scope.

| Blocker | How It Will Show Up | Mitigation |
| --- | --- | --- |
| Product Loop data is incomplete | Pages open, but recommendations are not credible because cost, logistics, platform fee, exchange rate, inventory, or listing task data is missing. | Define the minimum required Product Loop data contract before E2E; show missing fields as Owner-visible blocked state. |
| Frontend/backend API mismatch | Browser flow fails with 404/400, wrong path, wrong parameter name, or status enum mismatch. | Run the Product Loop through the real API client and update `reference-module-catalog.md` when route facts change. |
| Action Gate bypasses remain | The feature works but executes through old approval/listing-task/integration paths outside canonical AgentAction flow. | Audit all mutation paths touched by Product Loop; route high-risk execution through Action Gate or document deterministic system-action exceptions. |
| E2E seed data is unstable | Test passes locally once but fails in CI, or one run pollutes the next. | Build idempotent seed/reset scripts with fixed users, roles, products, fees, logistics, approvals, and task states. |
| Sandbox is too shallow | Mock execution always returns success and does not change state, produce fees, fail, or expose reference IDs. | Require platform-neutral sandbox to maintain state, simulate failure modes, and produce result snapshots before Ozon/Shopee adapters. |
| Daily Cockpit becomes a dashboard dump | Many cards appear, but Owner still does not know what to do next. | Cockpit must show the top decisions, pending approvals, failures, and outcomes generated by real Product Loop state. |
| Feedback is only audit text | Approve/reject history exists, but cannot train or evaluate future DSPy/Memory because reasons and outcomes are unstructured. | Capture structured approval reason, rejection reason, correction, outcome, and usefulness labels from the start. |
| CI/release hardening consumes the sprint | Work shifts from business flow to infrastructure before local Product Loop proof exists. | Run CI hardening in parallel; first prove local business flow, then promote E2E and seed path into CI/release gates. |
| Dirty work and parallel edits conflict | Changes touch files already modified by other Agents, causing large diffs or unstable tests. | Check `git status` before each slice; only touch files required for the slice; do not revert unrelated work. |
| Long-term AIOS work distracts from Owner value | Python/DSPy/multi-industry work grows while Product Loop remains unverified. | Keep AIOS work limited to contracts until Business Verified; no Python runtime implementation before gate. |

Most likely first blockers:

1. E2E seed data and environment instability.
2. Listing-task / approval / action-gate paths not fully unified.
3. Product profitability data missing fields or inconsistent business meaning.

## Testing Strategy

| Concern | Verification |
| --- | --- |
| Kernel action gates | Backend unit/integration tests around AgentAction, approvals, RBAC, audit, idempotency |
| Event and scheduler reliability | Lifecycle tests and failure/DLQ tests |
| Platform writes | Dry-run/sandbox/production contract tests, no external side effects in dry-run |
| Owner product loop | Playwright E2E with seeded realistic product data |
| Fulfillment loop | Playwright E2E with seeded realistic order/logistics data |
| Frontend correctness | `npm run lint`, `npx tsc --noEmit --pretty false`, `npm test`, `npm run build` |
| Backend correctness | `go test ./...`, `go vet ./...`, focused package tests |
| Release safety | CI requires E2E artifacts, migration checks, backup/restore evidence before beta claims |

## Boundaries

### Always

- Tie work to either the product loop, fulfillment loop, action gate, or Owner cockpit.
- Keep Go Kernel as the execution authority.
- Make high-risk actions approval-gated and auditable.
- Preserve unrelated dirty work.
- Update docs when contracts or acceptance gates change.

### Ask First

- Adding Python runtime services.
- Adding dependencies or provider SDKs.
- Changing database schemas.
- Changing CI/release/deploy behavior.
- Introducing production external writes.
- Moving finance or trade from future direction into implementation.

### Never

- Do not implement production autonomy for price, inventory, orders, money, platform publishing, credentials, or destructive actions without written policy.
- Do not let sidecars access production DB or external write adapters directly.
- Do not advertise guaranteed ROI, autonomous compliance, or full auto-operation.
- Do not build generic Agent builder features before ecommerce loops are Business Verified.
- Do not treat mocked UI-only E2E as Business Verified.

## Success Criteria

### Current Milestone: Business Verified Copilot

- Main browser E2E runs against backend and database.
- Product loop works end to end.
- At least one fulfillment-risk scenario works end to end.
- High-risk action approval and execution state is Owner-visible.
- `docs/KNOWN_ISSUES.md` has no open Business Verified blocker for main flows.
- Owner can verify success without reading code.
- MVP threshold: 5 seeded realistic browser runs pass end to end.
- Beta threshold: 10 seeded realistic browser runs pass end to end.

### Next Milestone: Trusted Sandbox

- Platform-neutral mock adapter maintains state first.
- Ozon/Shopee mock adapters are added only after the platform-neutral sandbox proves the product loop.
- Dry-run and sandbox produce different, explicit states.
- Production writes cannot happen without approval and audit.
- Failed platform actions produce visible recovery instructions.

### Later Milestone: Governed AIOS Cognition

- Sidecar contract exists and is reviewed.
- Memory and prompt/model registry contracts exist.
- Python cognition runs read-only/shadow mode.
- No self-improvement affects production until offline evaluation and approval pass.

## Resolved Decisions

- First platform: platform-neutral sandbox.
- First ICP: small cross-border ecommerce Owner / operator with about 50-500 SKUs.
- Next sprint: Product Loop E2E.
- MVP Business Verified: 5 seeded realistic browser runs.
- Beta Business Verified: 10 seeded realistic browser runs.
- CI/release hardening: parallel with Product Loop E2E, then promoted into the gate after first local proof.

## Remaining Questions

1. Which exact seeded SKU/category scenarios should define the first 5 browser runs?
2. Which screen should be the first Product Loop entry point: `/owner`, `/candidates`, or `/listing-tasks`?
3. Who owns closing `KI-2026-07-06-003`, and what is the target date?
