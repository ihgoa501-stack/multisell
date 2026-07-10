# LingMirror Known Issues

Updated: 2026-07-09

This file tracks accepted failures and unresolved risks. It is not a place to
hide red checks. Any known issue that blocks a required acceptance gate keeps
the affected status below `Test Green`, `Business Verified`, or `Beta Accepted`.

## Rules

Every known issue must have:

- owner
- date opened
- target fix date
- affected acceptance level
- owner-facing impact
- current workaround, if any
- evidence link or command output

Expired issues must be reviewed before new work in the same area is marked
complete.

## Status Values

| Status | Meaning |
|--------|---------|
| OPEN | Still failing or unverified. |
| MITIGATED | Impact has a workaround, but the root cause remains. |
| FIXED | Fix merged and verified by the required command. |
| ACCEPTED | Owner explicitly accepts the remaining risk for a named trial scope. |

## Current Issues

| ID | Status | Owner | Opened | Target Fix | Acceptance Impact | Owner-Facing Impact | Evidence |
|----|--------|-------|--------|------------|-------------------|---------------------|----------|
| KI-2026-07-06-001 | FIXED | Codex | 2026-07-06 | 2026-07-09 | None | Supplier detail/update/delete/comparison API behavior is covered by passing backend regression tests. | `cd backend-go && go test ./internal/domain/supplier` and `cd backend-go && go test ./...` pass on 2026-07-09. |
| KI-2026-07-06-002 | FIXED | Codex | 2026-07-06 | 2026-07-09 | None | Frontend lint/type/test/build gates are clean. | `cd frontend-next && npm run lint`, `npx tsc --noEmit --pretty false`, `npm test`, and `npm run build` pass on 2026-07-09. |
| KI-2026-07-06-003 | OPEN | Unassigned | 2026-07-06 | TBD | Business Verified blocked | Main browser flows are not proven end-to-end against a running backend and database. | `cd frontend-next/e2e && npm run e2e` failed locally; main-chain was previously skipped when backend was unavailable. |

## Template

```text
ID:
Status:
Owner:
Opened:
Target Fix:
Affected Acceptance Level:
Owner-Facing Impact:
Technical Summary:
Workaround:
Evidence:
Next Verification Command:
```
