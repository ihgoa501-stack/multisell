# LingMirror Known Issues

Updated: 2026-07-06

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

## Expired Issue Auto-Check (Phase 4)

A CI check (`scripts/check_known_issues.sh`, Phase 4) will enforce these rules
automatically:

1. **Every CI run** scans `docs/KNOWN_ISSUES.md` for issues where `Target Fix`
   is a date in the past (according to `date +%F`) and `Status` is still `OPEN`
   or `MITIGATED`.
2. **Blocking behaviour** depends on the lane of the PR (see ACCEPTANCE_GATE.md):
   - Read-only lane: expired issues produce a non-blocking warning.
   - Suggestion lane: expired issues produce a non-blocking warning.
   - Approval-required lane: any expired OPEN `OPEN` issue **blocks** merge.
   - Production-write lane: any expired `OPEN` or `MITIGATED` issue **blocks**
     merge.
3. **New work blocking.** A PR that touches code in the same area as an expired
   OPEN issue will be flagged for mandatory reviewer attention. The reviewer
   must either:
   - mark the issue `FIXED` or `ACCEPTED`, or
   - set a new realistic `Target Fix` date, or
   - note in the PR that the expired issue is unrelated.

Until `scripts/check_known_issues.sh` is implemented, these checks are manual
and the Reviewer must enforce them.

## Deadline Escalation Policy

When a known issue's `Target Fix` date passes without resolution:

1. **Day 0** (target fix date). The issue becomes `EXPIRED`. The assignee
   should either fix, re-estimate, or escalate.
2. **Day +3**. If still EXPIRED, a comment must be added explaining the delay
   and the new target. The issue is surfaced in the weekly triage.
3. **Day +7**. If still EXPIRED, the issue is escalated to the Owner, who
   decides: allocate resources, convert to `ACCEPTED` (risk acceptance), or
   downgrade the acceptance level of the affected area.

Escalation does not mean the feature is blocked — it means the risk is
explicitly surfaced and owned.

## Status Values

| Status | Meaning |
|--------|---------|
| OPEN | Still failing or unverified. |
| EXPIRED | Target fix date passed without resolution. Subject to auto-check rules and escalation policy. |
| MITIGATED | Impact has a workaround, but the root cause remains. |
| FIXED | Fix merged and verified by the required command. |
| ACCEPTED | Owner explicitly accepts the remaining risk for a named trial scope. |

## Current Issues

| ID | Status | Owner | Opened | Target Fix | Acceptance Impact | Owner-Facing Impact | Evidence |
|----|--------|-------|--------|------------|-------------------|---------------------|----------|
| KI-2026-07-06-001 | OPEN | Unassigned | 2026-07-06 | TBD | Test Green blocked | Supplier detail/update/delete/comparison API behavior is not proven by backend regression tests. | `cd backend-go && go test ./...` fails in `internal/domain/supplier`. |
| KI-2026-07-06-002 | OPEN | Unassigned | 2026-07-06 | TBD | Test Green blocked | Frontend code quality gate is not clean; build passes but lint still reports correctness risks. | `cd frontend-next && npm run lint` fails. |
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
