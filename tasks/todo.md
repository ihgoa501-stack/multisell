# Todo: AI-Native AgentOS Execution Path

> [!CAUTION]
> **2026-07-11 已冻结。** 当前任务唯一事实源是根目录 `TODOS.md`。本清单不得驱动 AgentOS、外部 SaaS 或多产品扩张。

Source plan: `tasks/plan.md`
Source spec: `docs/specs/2026-07-09-ai-native-agentos-execution-path.md`

## Phase 0: Product Loop E2E

- [x] Task: Map the current Product Loop route and API path.
  - Acceptance: Document the actual path from candidate page to recommendation, approval, listing task, and result review.
  - Verify: Route/API notes are added to the task PR or follow-up doc.
  - Files: read-only first; likely `frontend-next/src/app/(main)/candidates`, `frontend-next/src/app/(main)/owner`, `frontend-next/src/app/(main)/listing-tasks`, backend candidate/listing/approval routes.
  - Note 2026-07-10: Agent 1 completed mapping. Found 3 critical gaps (decision-queue 404, missing feedback route, listing task missing ApprovalID) and 2 high gaps (duplicate approval in OwnerPage, execute precondition block). See `docs/superpowers/plans/2026-07-10-owner-cockpit-audit.md`.

- [x] Task: Define minimum Product Loop seed data.
  - Acceptance: Five scenarios are defined: profitable listing, loss-making listing, missing logistics fee, missing platform/category fee, approval-to-sandbox listing task.
  - Verify: Seed data can be reset and re-run without manual cleanup.
  - Files: `scripts/e2e_seed.sh`, E2E fixtures or backend seed helpers.
  - Note 2026-07-10: Agent 2 added 5 scenarios to `backend-go/scripts/demo_seed.go` (~490 lines, idempotent inserts). No scripts/e2e_seed.sh changes needed. See `docs/superpowers/plans/e2e-seed-design-2026-07-10.md`.

- [x] Task: Fix Product Loop API mismatches found by the mapping.
  - Acceptance: No 404/400/path/enum mismatch blocks the main Product Loop.
  - Verify: Manual browser run reaches approval or controlled execution.
  - Files: focused frontend API calls and matching backend handlers only.
  - Note 2026-07-10: Fixed 3 critical blockers:
    1. Frontend: changed `/v1/owner/decision-queue` (404) to `/v1/owner/suggestions` with response mapping
    2. Backend: added `POST /listing-task/:task_id/feedback` route + handler (was 404)
    3. Backend: set `ApprovalID` on listing task in `loop/service.go` Evaluate (was breaking execute precondition)

- [ ] Task: Add Product Loop Playwright E2E.
  - Acceptance: One test covers candidate -> recommendation -> approval -> sandbox listing task -> result review.
  - Verify: `cd frontend-next/e2e && npx playwright test`.
  - Files: `frontend-next/e2e/tests/*`, seed fixtures/scripts.

## Phase 1: Action Gate Closure

- [x] Task: Audit Product Loop mutation paths.
  - Acceptance: Identify whether Owner/listing/integration execution bypasses `AgentAction`.
  - Verify: List each mutation path as gated, exception, or needs fix.
  - Files: backend AI, listing task, approval, integrations routes/services.
  - Note 2026-07-10: Agent 3 completed full 13-path audit. Found 4 must-fix bypasses: (M1) `listing.read` RBAC gates all listing task CRUD; (M2) approval review directly syncs unified_action table skipping ai.action gate; (M3) approval routes have no RBAC; (M4) owner RecordFeedback has no RBAC. See `docs/superpowers/plans/action-gate-audit-2026-07-10.md`.

- [ ] Task: Close high-risk bypasses in Product Loop.
  - Acceptance: Product publish/listing high-risk mutations are approval-gated and audited.
  - Verify: Focused backend tests plus Product Loop E2E.
  - Files: no more than one vertical slice at a time.

- [ ] Task: Add failure and blocked states to Product Loop actions.
  - Acceptance: Owner can see failed/blocked status and recovery note.
  - Verify: E2E includes one blocked or missing-data scenario.
  - Files: action/listing task backend and relevant UI.

## Phase 2: CI / E2E Gate

- [ ] Task: Make seed path reproducible.
  - Acceptance: Seed/reset command works from repo root and E2E working directory.
  - Verify: Run seed twice; second run is idempotent.
  - Files: `scripts/e2e_seed.sh`, `frontend-next/e2e` config.
  - Note 2026-07-10: Agent 6 audit found `scripts/e2e_seed.sh` already works from any CWD and is idempotent. No changes needed.
  - Critical finding: `scripts/verify_all.sh` defaults to `RUN_E2E=1` but runs E2E without seeding first — order must be seed → verify. Recommended fix: change default to `RUN_E2E="${RUN_E2E:-0}"` and add health check before E2E.

- [ ] Task: Add Product Loop E2E to verification command.
  - Acceptance: `scripts/verify_all.sh` or equivalent can run Product Loop E2E with backend/frontend dependencies.
  - Verify: Verification command output is archived or copied into status docs.
  - Files: `scripts/verify_all.sh`, E2E config.

- [ ] Task: Update known issue status only after evidence.
  - Acceptance: `KI-2026-07-06-003` remains OPEN until Product Loop browser proof exists.
  - Verify: `docs/KNOWN_ISSUES.md` evidence includes command output/date.
  - Files: `docs/KNOWN_ISSUES.md`, `docs/PROJECT_STATUS.md`.

## Phase 3: Daily Owner Cockpit

- [ ] Task: Define cockpit cards from real Product Loop state.
  - Acceptance: Cards are limited to top decisions, pending approvals, failed/blocked actions, and recent outcomes.
  - Verify: No card depends on mock-only data.
  - Files: `frontend-next/src/app/(main)/owner/page.tsx` and supporting API.
  - Note 2026-07-10: Agent 4 audit found 7-question coverage is best on Owner page (uses HighRiskConfirmDialog correctly). Weaknesses: listing tasks list is generic CRUD with no mode/decision context; action detail has no confirmation dialog before approve/execute. 4 gaps are backend-dependent (profit impact, audit page link, agent context, reject endpoint).

- [x] Task: Add execution_mode display to actions and listing tasks.
  - Acceptance: Each action/listing task shows dry-run/sandbox/production mode.
  - Verify: TypeScript compile and lint pass.
  - Files: `frontend-next/src/app/(main)/actions/page.tsx`, `actions/[id]/page.tsx`, `listing-tasks/page.tsx`, `listing-tasks/[id]/page.tsx`.
  - Note 2026-07-10: Agent 4 added mode badge and helper across 4 frontend files. All pass lint/typecheck/build.

- [ ] Task: Add Owner-readable decision copy.
  - Acceptance: Each card answers what, why, risk, approve consequence, wait consequence, mode, and audit location.
  - Verify: Browser review with seeded data.
  - Files: Owner page/components.

## Phase 4: Fulfillment Copilot

- [ ] Task: Define minimum order-risk seed scenario.
  - Acceptance: One order can show inventory/logistics/settlement/profit risk.
  - Verify: Seed scenario appears in UI.
  - Files: seed scripts and order/logistics/settlement modules.

- [ ] Task: Add fulfillment-risk E2E.
  - Acceptance: Owner can review a loss-risk order and choose approve/manual handling.
  - Verify: Playwright E2E passes against backend/database.
  - Files: E2E test plus minimal UI/API fixes.

## Phase 5: Platform Adapter Sandbox

- [x] Task: Implement platform-neutral stateful sandbox contract.
  - Acceptance: Sandbox maintains listing task state and supports success/failure/missing-data scenarios.
  - Verify: Backend tests and Product Loop E2E.
  - Files: integration adapter sandbox slice.
  - Note 2026-07-10: Agent 5 created `sandbox_adapter.go` (full PlatformAdapter interface implementation with 5 deterministic scenarios: success/blocked_missing_data/blocked_loss_making/failed_platform_validation/dry-run) and `sandbox_adapter_test.go` (16 tests, all pass). Also `sandbox-contract-2026-07-10.md` with state machine diagram and integration points table.

- [ ] Task: Add Ozon/Shopee sandbox only after neutral sandbox proves loop.
  - Acceptance: Real platform adapters remain dry-run/sandbox by default.
  - Verify: No production write in tests.
  - Files: integration adapters.

## Phase 6: AIOS Contracts

- [ ] Task: Draft `ProposedAgentAction` sidecar contract.
  - Acceptance: Contract includes identity, tenant, actor, correlation, schema version, source citations, sensitivity, mode, timeout/retry/failure semantics.
  - Verify: Reviewed against `docs/governance/KERNEL_CONTRACTS.md`.
  - Files: docs/specs or governance docs only.

- [ ] Task: Draft memory and prompt/model registry contracts.
  - Acceptance: Memory is advisory only; prompt/model changes require versioning, approval, shadow metrics, and rollback.
  - Verify: Reviewed before any Python implementation.
  - Files: docs/specs or governance docs only.
