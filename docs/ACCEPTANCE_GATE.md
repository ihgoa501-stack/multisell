# LingMirror Acceptance Gate

Updated: 2026-07-06

This document defines when work may be called complete. A feature, iteration,
or release is not accepted by description alone. It needs reproducible evidence.

## Status Levels

| Status | Meaning |
|--------|---------|
| Dev Done | Code is implemented and locally runnable. |
| Test Green | Required automated checks pass. |
| Business Verified | Target business flow is proven end-to-end. |
| Beta Accepted | The system is allowed to enter controlled real or near-real operation. |

Do not collapse these into one word. If E2E or business verification has not
passed, report the exact level reached.

## Required Automated Checks

Run the same command locally and in CI:

```bash
scripts/verify_all.sh
```

Minimum required checks:

```bash
cd backend-go && go build ./...
cd backend-go && go vet ./...
cd backend-go && go test ./...

cd frontend-next && npm test
cd frontend-next && npm run lint
cd frontend-next && npm run build

cd frontend-next/e2e && npm run e2e
```

Any failed required check means the status is not `Test Green`.

## Business Verification

At minimum, Beta acceptance requires proof for these flows:

| Flow | Evidence Required |
|------|-------------------|
| Product loop | Candidate product -> completeness -> cost/logistics/platform fee/profit -> listing recommendation -> Owner approval -> controlled listing task -> result review |
| Order loop | Order -> inventory/logistics -> shipping cost snapshot -> settlement/profit check -> exception detection -> Agent recommendation -> Owner handling |
| High-risk action gate | Unapproved execution blocked; authenticated user bound as approver/executor; audit log written; sensitive fields redacted; dry-run does not write externally; production requires approval |

Evidence may be Playwright output, smoke-test logs, screenshots, or a manual
acceptance record, but it must be dated and reproducible.

## Release Lanes

Changes are categorised into one of four release lanes before merging or
deploying. Each lane has a minimum required acceptance status, an approval
count, and a set of allowed target environments.

| Lane | Typical Change | Min Acceptance | Required Approvals | Allowed Environments | Extra Gates |
|------|----------------|----------------|-------------------|---------------------|-------------|
| **Read-only (fast track)** | UI-only changes, documentation, refactors that do not alter behaviour, test additions, config comment changes. | Dev Done | 1 reviewer | local, preview | None beyond normal CI. |
| **Suggestion (standard)** | Agent recommendations, dashboard additions, read-only API additions, non-destructive reports. | Test Green | 1 reviewer + code owner | local, preview, staging | Changes that touch AI recommendation output must also include a before/after sample comparison in the PR. |
| **Approval-required (slow)** | Price/inventory/order changes, platform publishing, permission changes, AI action execution logic. | Business Verified | 2 reviewers + code owner + explicit Owner approval | local, preview, staging, sandbox | Dry-run evidence must be included. High-risk action matrix from ACCEPTANCE_MATRIX.md must be satisfied. |
| **Production-write (slowest + extra gates)** | External platform write-back, production data migrations, payment/refund flows, RBAC/Auth changes deployed to production. | Beta Accepted | 2 reviewers + code owner + Owner approval + sign-off from `docs/governance/OWNER_DECISION_LOG.md` | local, preview, staging, sandbox, production (gradual) | Owner must document the decision in `docs/governance/OWNER_DECISION_LOG.md`. Break-glass must be tested. Rollback plan must be attached and verified. Gradual rollout (canary / percentage) required before full production. |

**Lane escalation.** If a PR touches files from a higher-risk lane (e.g. a
"read-only" PR modifies a price-calculation function), the higher lane's rules
apply to the entire PR.

**Lane overrides.** The Owner may explicitly downgrade a lane for a specific
PR via `docs/governance/OWNER_DECISION_LOG.md`. The override must name the PR, the
reason, and the acceptance-status delta accepted.

### Choosing a Lane

- Always pick the **slowest lane** whose rules your change satisfies.
- When in doubt between two lanes, pick the slower one.
- CI must validate the lane tag in the PR template matches the actual diff
  (see `scripts/verify_lane.sh`, Phase 4).

## Report Status Vocabulary

Use only these result words in acceptance reports:

| Result | Meaning |
|--------|---------|
| PASS | Ran and passed. |
| FAIL | Ran and failed. |
| SKIPPED | Intentionally not run; include why. |
| BLOCKED | Could not run because a prerequisite was missing. |
| NOT RUN | Not attempted. |

Do not write `PASS with known issue`. A known failing required check is `FAIL`.

## Required Report Contents

`docs/BETA_ACCEPTANCE_REPORT.md` must include:

- date, branch, commit, and dirty worktree status
- exact commands run
- PASS/FAIL/SKIPPED/BLOCKED/NOT RUN result for each required check
- business-flow evidence
- high-risk action gate evidence
- known issues and owner-facing impact
- explicit decision: accepted for controlled trial or not accepted

## Owner Decision Log

All lane overrides, approval decisions, and risk acceptances for releases that
reach production must be recorded in `docs/governance/OWNER_DECISION_LOG.md`. Entries
must include:

- date and PR reference
- lane chosen and reason
- any acceptance-status deltas accepted
- risk acceptance statement
- Owner name or handle

## Lane Compliance Validation (Phase 4)

A CI check (`scripts/verify_lane.sh`) will validate lane declarations against
actual diff contents. It will reject a PR whose declared lane is lower than
the lane its file changes require. Until that script is implemented, lane
review is the Reviewer's responsibility.
