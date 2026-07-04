# LingMirror Current Direction and Priorities

> Updated: 2026-07-04
> Status: current execution guidance
> Scope: product direction, documentation alignment, AgentOS safety priorities

## Purpose

This document is the current decision record for where LingMirror should go next.
It does not replace detailed feature specs, module catalogs, or governance
contracts. It exists to stop development from drifting across too many product
ideas at once.

Some information in this repository is still incomplete or mixed with historical
plans. Treat this document as a decision guide based on the latest repository
review, not as a claim that every implementation detail has been fully verified.

## Current Product Direction

LingMirror should be developed as a cross-border e-commerce AI AgentOS:

```text
Trusted commerce data foundation
-> Owner-facing decision cockpit
-> controlled Agent recommendations
-> approval-gated execution
-> audit and review loop
```

The near-term product promise should not be "fully automatic company operation".
The safer and more useful promise is:

```text
Help the Owner see what matters, understand why it matters, approve the right
actions, and avoid unsafe automatic changes.
```

## What To Focus On

### 1. Business Loop First

The first useful loop remains:

```text
Candidate product
-> completeness check
-> cost, logistics, platform fee, and profit calculation
-> listing recommendation
-> Owner approval
-> controlled listing task
-> result review
```

The second loop should build from fulfillment:

```text
Order
-> inventory and logistics choice
-> shipping cost snapshot
-> settlement and profit check
-> exception detection
-> Agent recommendation
-> Owner approval or manual handling
```

New work should explain which of these loops it makes more complete.

### 2. Owner Control Before Autonomy

Owner should not need to read logs, trace code paths, or understand module
internals to make a decision.

Every important Agent recommendation should answer:

- What happened?
- Why is this important?
- What does the Agent recommend?
- What happens if the Owner approves?
- What happens if the Owner rejects or waits?
- Is this dry-run, sandbox, read-only, or production?
- Where is the audit trail?

### 3. Safety Gates Before Production Execution

Prices, inventory, order state, money, refunds, platform publishing,
credential changes, and account permissions are high-risk business areas.

Until the safety gates are fully unified, these actions should remain in:

```text
read-only -> suggestion -> approval required
```

Do not promote them to autonomous production execution just because an Agent has
a high trust score.

## What Not To Prioritize Now

Avoid spending the next phase on:

- More standalone CRUD pages that do not improve the two business loops.
- A generic no-code Agent builder product.
- Fully automatic production execution.
- More Agent names without clearer inputs, outputs, risk boundaries, and review
  metrics.
- Real external write-back before sandbox/read-only behavior is observable and
  auditable.

These ideas may still be useful later, but they should not drive the current
development queue.

## Current High-Risk Gaps To Resolve First

The following gaps were identified during repository review and should be
verified and addressed before expanding autonomous behavior.

### P0. Platform Runtime Lifecycle

`backend-go/internal/httpx/router.go` starts EventBus and Scheduler-related
infrastructure during router construction. The EventBus context is currently
created with a cancel function deferred inside router setup. Verify and fix the
lifecycle so EventBus workers and scheduled Agent ticks remain alive for the
server lifetime.

Business impact: scheduled Agent checks and event-driven workflows may appear
configured but not actually run reliably.

### P0. Unified Action Execution Gate

`backend-go/internal/ai/service.go` executes actions through raw command
dispatch. The production path should go through the canonical action contract:

```text
UnifiedAction / AgentAction
-> ActionCatalog
-> approval policy
-> DispatchSafe
-> audit
-> result and failure state
```

Business impact: high-risk actions may bypass the strongest approval and mode
checks.

### P0. Approval Identity And RBAC

Approval, rejection, and execution should bind to the authenticated server-side
user identity. The client should not be trusted to declare the operator.

Business impact: audit and accountability are weak if the approver can be
spoofed in request payloads.

### P1. External Platform Write Safety

External publishing, inventory sync, tracking push, and platform write-back
must distinguish dry-run, sandbox, and production. Production writes require
approval, audit, external reference IDs, and failure visibility.

Business impact: wrong platform actions can affect real listings, stock,
orders, and money.

### P1. Sensitive Audit Redaction

Mutation audit currently records request body snippets. Ensure credentials,
tokens, API keys, secrets, and other sensitive fields are redacted before they
enter operation logs.

Business impact: audit logs should improve trust, not become a credential leak.

### P1. Frontend High-Risk Action UX

All publish, approve, execute, price, inventory, refund, order-state, and
autonomy-upgrade actions should use one shared Owner-facing confirmation
pattern.

The confirmation must show:

- target object
- risk level
- before and after values where applicable
- environment mode
- approval requirement
- expected consequence
- audit destination
- rollback or recovery note when available

Business impact: Owner can make informed decisions without reading technical
details.

## Documentation Cleanup Rules

### Current Fact Sources

Use these documents first:

- `docs/governance/*`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `docs/PROJECT_STATUS.md`
- `docs/ACTIVE_STACK_POLICY.md`
- `docs/reference-module-catalog.md`

When these sources conflict, the governance documents win unless the Owner
explicitly overrides them.

### Historical Or Research Sources

The following types of documents are useful for context but should not override
current direction:

- old FastAPI/Vue plans
- broad no-code Agent platform PRDs
- market research deliverables
- archived multi-agent execution plans
- older roadmap phases that conflict with current governance docs

If a document says to use `backend/`, `frontend/`, or legacy `/api/*` paths,
treat it as historical unless a current fact source explicitly says otherwise.

## Recommended Next Iteration

The next development iteration should be:

```text
Trusted AgentOS execution gate cleanup
```

Acceptance path:

1. EventBus and Scheduler lifecycle is verified by test or runtime check.
2. `/api/v1/ai/actions/:id/execute` uses the safe action execution path.
3. Approval and execution use authenticated identity and RBAC.
4. High-risk external platform actions are blocked, dry-run, or approval-gated.
5. Operation logs redact sensitive fields.
6. Owner-facing UI uses a consistent high-risk action confirmation pattern.
7. Project status documents clearly mark what is verified, what is known risk,
   and what remains unverified.

## Business-Level Success Definition

The Owner should be able to open LingMirror and trust these statements:

- The system will not silently change prices, inventory, orders, money, or
  external listings.
- Agent recommendations are visible, explainable, and reviewable.
- High-risk actions require explicit approval.
- Approved actions are traceable to a user, reason, target, and audit record.
- Failures are visible with next steps.
- Mock, sandbox, read-only, and production modes are clearly separated.
