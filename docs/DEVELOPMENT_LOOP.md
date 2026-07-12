# LingMirror Development Loop

Last updated: 2026-06-28

This document defines the repeatable development loop for LingMirror / MultiSell. It turns the Owner-confirmed [ocean goal](OCEAN_GOAL.md) into complete, verified, auditable lakes. Use it for non-trivial product work, platform work, bug fixes, refactors, reviews, QA, and release work.

The loop complements the governance documents in `docs/governance/`. If there is a conflict, follow the governance documents unless the Owner explicitly overrides them.

## Purpose

LingMirror is now too large for ad hoc development. The system needs a loop that preserves business intent, platform boundaries, verification, review, and project memory.

The goal is not to make every task slower. The goal is to stop each task from starting from zero.

## Loop Overview

```text
Intake
-> Translate
-> Discover
-> Slice
-> Implement
-> Verify
-> Review
-> Record
-> Repeat
```

## 1. Intake

The Owner describes the need in business language.

Recommended format:

```text
I want:
User / scenario:
Current problem:
Ideal result:
Must not happen:
Priority:
```

The Owner does not need to name files, APIs, database tables, or technical architecture.

## 2. Translate

The Lead Agent translates the business goal into engineering scope before implementation.

Required translation:

```text
My understanding:
Expected user result:
Layer touched:
Risk level:
Affected business areas:
Likely modules:
Approval / audit impact:
Acceptance path:
Questions for Owner:
```

Risk must be stated in business terms:

- Low risk: wording, display, read-only views, docs, isolated fixes.
- Medium risk: business logic, API behavior, permissions, schedules, multi-module changes.
- High risk: prices, inventory, order state, money, external platform publishing, account permissions, approval, audit, Agent autonomy, migrations, destructive changes.

High-risk work must not proceed until approval, audit, and rollback expectations are clear.

## 3. Discover

Research current reality before changing files.

Discovery rules:

- Use CodeGraph first for indexed source code.
- Read governance docs for non-trivial work.
- Read current implementation and tests near the affected module.
- Check existing docs before inventing a new concept.
- Do not modify files during discovery.

Discovery output:

```text
Current behavior:
Existing patterns:
Relevant modules:
Known conflicts:
Recommended slice:
```

## 4. Define the Lake

Split the ocean into the smallest useful end-to-end lake that can be completed and verified. Small describes the boundary, not an excuse to omit required implementation, tests, error paths, safety, or recovery.

A good slice has:

- One business outcome.
- Clear affected layer.
- Clear files or modules.
- Clear acceptance criteria.
- A bounded test plan.
- A stop point.
- A clear relationship to the ocean goal.

When the remaining work is necessary to make the selected lake complete, finish it. Work that serves a different product, market, platform, or multi-quarter migration remains separate scope until confirmed.

Avoid slices like:

- "Improve AgentOS."
- "Refactor frontend."
- "Fix all lint."
- "Make the dashboard better."

Prefer slices like:

- "Show pending approval count on AgentOS dashboard."
- "Add read-only sourcing recommendation list to `/sourcing`."
- "Make feedback admin page pass lint without changing behavior."

## 5. Implement

Implementation rules:

- Modify only the agreed slice.
- Keep active stack only: `backend-go/` and `frontend-next/`.
- Do not touch legacy stacks unless the Owner explicitly asks.
- Follow existing module patterns.
- Do not put business-specific logic into Platform Kernel.
- Do not bypass Auth, RBAC, Approval, Audit, EventBus, Command, Scheduler, or ToolBridge contracts.
- Add focused tests when behavior changes.
- Preserve unrelated work in the git tree.

For Agent workflows:

- Read-only and suggestion actions are safer defaults.
- Critical actions require approval and audit unless written policy allows autonomy.
- External side effects must be explicit and logged.

## 6. Verify

Verification must match the touched surface.

Backend:

```bash
cd backend-go && go test ./...
cd backend-go && go vet ./...
```

Frontend:

```bash
cd frontend-next && npm test
cd frontend-next && npm run lint
cd frontend-next && npm run build
```

Focused checks are acceptable for narrow slices, but the final report must say what was and was not run.

High-risk user flows may also require:

- API smoke checks.
- Browser verification.
- E2E tests.
- Audit log inspection.
- Approval state inspection.

## 7. Review

Review checks whether the change is safe to accept, not whether the code merely compiles.

Review checklist:

```text
Does it match the Owner goal?
Does it stay in the right layer?
Does it duplicate an existing concept?
Does it pollute Platform Kernel with business logic?
Does it bypass Auth / RBAC / Approval / Audit?
Does it create hidden external side effects?
Are event and command contracts clear?
Are tests proportional to risk?
Can the Owner verify it without reading code?
Do docs need updates?
```

Findings should be concrete and tied to files, behavior, or business risk.

## 8. Record

Every non-trivial loop run must leave state behind.

Record in `.loop/dev-state.md`:

```text
Current goal:
Current slice:
Completed:
Verification:
Open risks:
Next recommended slice:
Blocked by:
```

Update long-term docs when behavior, contracts, pages, routes, or governance expectations change.

Examples:

- `docs/PROJECT_STATUS.md` for project state.
- `docs/FRONTEND_PAGES_AND_ROUTING.md` for route changes.
- `docs/AGENT_CAPABILITIES.md` for Agent capability changes.
- `docs/governance/*` for governance or Kernel contract changes.
- `docs/features/*` for feature specifications.

## 9. Repeat

The next run starts from recorded state, not from memory.

Before starting the next slice:

```text
Read `.loop/dev-state.md`.
Check git status.
Check whether previous verification passed.
Check whether unresolved risks affect the new slice.
Continue, revise, or stop.
```

## Minimum Viable Development Loop

Use this when the task is small but still non-trivial:

```text
Translate:
State goal, layer, risk, acceptance path.

Discover:
Read the nearest code/docs/tests.

Implement:
Make the smallest scoped change.

Verify:
Run the narrowest meaningful check.

Record:
Update `.loop/dev-state.md` with result and next step.
```

## Stop Conditions

Stop the loop and report when:

- The slice passes acceptance.
- Verification fails and the cause is not understood after a bounded attempt.
- The work requires Owner business judgment.
- The work touches high-risk behavior without defined approval/audit.
- Existing docs and code contradict the requested outcome.
- The change would require destructive migration or data deletion.
- The work is expanding beyond the agreed slice.

## Delivery Format

Final delivery should be readable by the Owner:

```text
What you can do now:
Where to try or read it:
How to verify success:
What changed:
What was tested:
Business areas affected:
Business areas not affected:
Remaining risks or limits:
Documents updated:
Next recommended slice:
```
