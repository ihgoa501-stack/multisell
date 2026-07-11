# Plan: AI-Native AgentOS Execution Path

> [!CAUTION]
> **2026-07-11 已冻结。** 当前执行计划已改为 `docs/ROADMAP.md` 的 Owner 自用商品实验 Sprint 0–5。本文件仅供历史参考，不得继续执行 AgentOS 扩张。

Source spec: `docs/specs/2026-07-09-ai-native-agentos-execution-path.md`

## Goal

Turn the long-term AI-Native AgentOS vision into an executable sequence without losing the current Copilot constraint.

The next development line is:

```text
Product Loop E2E
-> Action Gate closure
-> CI / E2E gate
-> Daily Owner Cockpit
-> Fulfillment Copilot
-> Platform Adapter Sandbox
-> AIOS Contracts
-> Python Shadow Sidecar
```

Do not start Python runtime, DSPy, finance, foreign trade, or generic Agent builder work until Business Verified is achieved.

## Current Baseline

Already present or partially present:

- Go Platform Kernel foundations: Auth, RBAC, Approval, Audit, EventBus, Command, Scheduler, ToolBridge.
- `AgentAction` contract with risk, mode, approval, audit, idempotency, and correlation fields.
- AI action execute route and approval/reject/execute RBAC route protection.
- Audit redaction and dry-run/sandbox execution mode support.
- Owner, Actions, Candidates, Listing Tasks, AgentOS pages.
- High-risk confirmation UI components.

Known blocker:

- `KI-2026-07-06-003`: main browser flows are not proven end-to-end against running backend and database.

## Implementation Strategy

### Phase 0: Product Loop E2E

Prove one end-to-end product decision loop locally before adding more architecture.

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

Key decisions:

- First platform is platform-neutral sandbox.
- First ICP is small cross-border ecommerce Owner/operator with about 50-500 SKUs.
- MVP Business Verified requires 5 seeded realistic browser runs.

Risks:

- Missing cost/logistics/platform fee fields.
- Frontend/backend API mismatch.
- Seed data pollution between runs.

### Phase 1: Action Gate Closure

Ensure Product Loop high-risk actions cannot execute through legacy paths.

Scope:

- Owner approval paths.
- Listing task execute paths.
- Integration publish paths.
- AI action execute paths.

Requirement:

High-risk mutations must be represented as `AgentAction` or documented deterministic system actions with audit.

### Phase 2: CI / E2E Gate

Make the Product Loop reproducible outside one local session.

Scope:

- Seed/reset scripts.
- Playwright path.
- Backend/frontend startup assumptions.
- Verification artifacts.

Goal:

Promote local Product Loop E2E into a repeatable gate after first local proof.

### Phase 3: Daily Owner Cockpit

Build the cockpit only from real loop state.

Cockpit must show:

- Top product opportunities.
- Pending high-risk approvals.
- Failed/blocked actions.
- Recent result reviews.

Do not build a dashboard wall of metrics.

### Phase 4: Fulfillment Copilot

Add the second business loop after Product Loop is stable.

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

### Phase 5: Platform Adapter Sandbox

Expand from platform-neutral sandbox to Ozon/Shopee sandbox adapters.

Sandbox must maintain state and simulate:

- Successful listing draft.
- Loss-making listing block.
- Missing data block.
- Platform publish failure.
- External reference ID and failure recovery.

### Phase 6: AIOS Contracts

Write contracts before Python implementation.

Required contracts:

- `ProposedAgentAction` IPC envelope.
- Sidecar identity/auth.
- Memory lineage and deletion.
- Prompt/model registry.
- Shadow mode rollout and rollback.
- Sandbox/outbound enforcement.

### Later Phases

Only after Business Verified and enough structured feedback:

- Python shadow sidecar.
- Offline/shadow DSPy.
- Controlled self-improvement.
- Finance/trade exploration.

## Verification Checkpoints

| Checkpoint | Required Evidence |
| --- | --- |
| Local Product Loop proof | Browser run against backend/database completes candidate -> result review. |
| MVP Business Verified | 5 seeded realistic browser runs pass, silent failures = 0. |
| Beta Business Verified | 10 seeded realistic browser runs pass, silent failures = 0. |
| Action gate closure | High-risk Product Loop actions are approval-gated and audited. |
| CI/E2E gate | Seed + Playwright + backend/frontend startup reproducible in verification command. |
| Trusted sandbox | Sandbox state changes and failure modes are visible to Owner. |

## Non-Goals

- No production platform writes.
- No Python runtime implementation.
- No DSPy automation.
- No finance/trade schemas.
- No generic Agent builder.
- No dashboard-only cockpit.
