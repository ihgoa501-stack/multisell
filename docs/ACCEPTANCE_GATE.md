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
