# Release Readiness Checklist

Last updated: 2026-07-06

All releases must pass the pre-release checks and follow the risk-gated release lane appropriate to the change scope.

## 1. Pre-Release Checks

Every release, regardless of lane, must pass these checks before deployment:

| # | Check | Status | Verified By |
|---|-------|--------|-------------|
| 1.1 | `cd backend-go && go test ./...` passes | | |
| 1.2 | `cd backend-go && go vet ./...` passes | | |
| 1.3 | `cd frontend-next && npm test` passes | | |
| 1.4 | `cd frontend-next && npm run build` succeeds | | |
| 1.5 | `cd frontend-next/e2e && npx playwright test` passes (at least critical paths) | | |
| 1.6 | Database migration files (up + down) included and reviewed | | |
| 1.7 | Full DB backup taken and verified restorable | | |
| 1.8 | Feature flags configured (all new gated capabilities default to OFF) | | |
| 1.9 | Config changes reviewed against production config.yaml | | |
| 1.10 | CHANGELOG.md updated with release notes | | |
| 1.11 | Version bumped in VERSION file | | |
| 1.12 | Swagger annotations updated if API changed | | |
| 1.13 | docs/INDEX.md updated if routes/modules changed | | |

## 2. Risk-Gated Release Lanes

Changes are assigned to a lane based on their risk level. The lane determines required checks, approval count, deployment environments, and rollback plan.

| Lane | Risk Level | Examples | CI Required | Approvals | Environments | Rollback |
|------|-----------|----------|-------------|-----------|-------------|---------|
| **Read-only Fast Track** | Very low | UI copy change, non-functional frontend fix, documentation update, passive monitoring | build + lint (both) | 1 (any team member) | production directly | revert PR, re-deploy |
| **Suggestion Standard** | Low | Read-only Agent suggestion, dashboard report, new listing display field, existing test coverage | build + lint + unit test + e2e critical | 1 (peer review) | staging → production | revert + deploy |
| **Approval-required Slow** | Medium | Agent write-back to platform, price change, inventory adjustment, new scheduled agent, migration | full suite | 2 (peer + Owner) | staging → canary (1h) → production | down migration + restore |
| **Production-write Slowest** | High | Platform adapter changes, production launch, billing, credentials, auth, kill switch, budget system | full suite + smoke test | 3 (peer + tech lead + Owner) | staging → canary (4h) → production (gradual 10%-50%-100%) | full db restore + config revert |

### Lane Assignment Rules

- If the change touches a **Kernel** module (auth, RBAC, EventBus, Scheduler, Command, ToolBridge, Audit, Approval), default to the next higher lane.
- If the change writes to an **external platform** (Ozon, Shopee, Lazada), minimum **Approval-required Slow**.
- If the change affects **financial calculations** (order, settlement, platform fee, exchange rate), minimum **Approval-required Slow**.
- If the change is **purely frontend read-only**, default to **Read-only Fast Track**.
- The Owner can override lane assignment, but the override must be documented in the release notes.

## 3. Release Sign-Off Table

Every release must be signed off by each role before proceeding to the next environment.

| Role | Sign-off | Date | Notes |
|------|----------|------|-------|
| **Owner** | [sign] | | Business risk acceptance, feature correctness, user experience |
| **Tech Lead** | [sign] | | Technical correctness, test coverage, migration safety, rollback plan |
| **QA** | [sign] | | Test results reviewed, regression confirmed clear, edge cases covered |

### Sign-Off Criteria

- **Owner** signs when: the business goal is met, the priority is correct, the business risk is acceptable.
- **Tech Lead** signs when: all tests pass, migration reviewed, rollback plan exists and is tested, observability confirmed, alerting configured.
- **QA** signs when: critical path e2e passes, no P0/P1 regression, integration tests pass, trial boundary confirmed.

## 4. Post-Release Monitoring Checklist

After deployment, monitor for at least the duration specified per lane:

| # | Check | Fast Track | Standard | Slow | Slowest |
|---|-------|-----------|----------|------|---------|
| 4.1 | No 5xx errors above baseline | 15 min | 30 min | 1 h | 4 h |
| 4.2 | No event processing backlog | — | 30 min | 1 h | 4 h |
| 4.3 | Agent heartbeat maintained | — | — | 1 h | 4 h |
| 4.4 | No platform write-back errors | — | — | 1 h | 4 h |
| 4.5 | LLM cost within expected range | — | — | 1 h | 4 h |
| 4.6 | Migration completed without error | — | 30 min | 1 h | 4 h |
| 4.7 | Feature flag toggle verified (enabled/disabled both work) | — | 30 min | 1 h | 4 h |
| 4.8 | Backup retention confirmed | — | — | — | post-release |

### Rollback Trigger Signals

- P0 incident (see INCIDENT_DRILL_CHECKLIST.md) — immediate rollback, suppress lane discretion.
- Error rate > 2x baseline for > 5 minutes — initiate rollback unless root cause identified and fix-forward approved.
- Financial calculation discrepancy detected — immediate rollback and freeze.
- Platform write-back producing incorrect data — immediate rollback and freeze that platform adapter.
- Migration failure — down-migrate and roll back release.

## 5. Release Communication

Before deployment:

- [ ] Release notes posted to team Slack channel
- [ ] Scheduled maintenance window confirmed (if applicable)
- [ ] On-call engineer notified of incoming release
- [ ] Owner notified of expected impact (if medium/high risk lane)

After deployment:

- [ ] Release complete message sent to team channel
- [ ] Monitoring dashboard checked and confirmed green
- [ ] Post-release checklist completed (Section 4)
- [ ] Rollback decision timestamp noted (if rollback triggered)
