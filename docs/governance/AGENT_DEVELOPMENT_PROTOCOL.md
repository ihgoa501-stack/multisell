# Agent Development Protocol

Last updated: 2026-06-27

This protocol defines how multiple Agents work on LingMirror without losing system coherence. It applies to Codex, Claude Code, and any other coding, review, QA, documentation, or planning Agent.

## 1. Required Reading

Before non-trivial work, Agents must read:

- `docs/governance/OWNER_FIRST_PROTOCOL.md`
- `docs/governance/PLATFORM_CONSTITUTION.md`
- `docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md`
- `docs/governance/KERNEL_CONTRACTS.md`
- `AGENTS.md`
- `CLAUDE.md` when using Claude Code

For code understanding in this repository, use CodeGraph first when applicable. Skip CodeGraph for Markdown, JSON, TOML, YAML, lockfiles, and generated artifacts.

## 2. Agent Roles

### Lead Agent

Owns task interpretation and delivery.

Responsibilities:

- Translate Owner goals into engineering work.
- Classify risk.
- Choose the recommended approach.
- Keep work scoped.
- Coordinate sub-agents or tools.
- Deliver final results in Owner-readable language.

### Research Agent

Reads and reports. Does not modify files.

Responsibilities:

- Locate existing patterns.
- Identify affected modules.
- Find duplicate concepts or prior docs.
- Report risks and open questions.

### Planning Agent

Produces a plan. Does not implement unless explicitly assigned.

Responsibilities:

- Define steps.
- Identify tests and acceptance criteria.
- Identify docs to update.
- Split high-risk work into smaller changes.

### Implementation Agent

Changes files inside the agreed scope.

Responsibilities:

- Follow existing patterns.
- Keep changes minimal and reversible.
- Add focused tests when behavior changes.
- Avoid unrelated refactors.

### Review Agent

Finds risks before delivery.

Responsibilities:

- Check platform boundaries.
- Check duplicated concepts.
- Check approval, audit, auth, RBAC, and observability.
- Check tests and docs.
- Check whether the Owner can actually verify the result.

### QA Agent

Verifies behavior.

Responsibilities:

- Run relevant automated checks.
- Use browser or API checks when needed.
- Report exact pass/fail results and reproduction steps.

### Documentation Agent

Updates long-term memory.

Responsibilities:

- Update governance docs when rules change.
- Update feature docs when behavior changes.
- Update system maps or indexes when new modules are added.

## 3. Task Types

Every task must be classified as one of:

- Research only.
- Product clarification.
- Design / planning.
- Implementation.
- Bug fix.
- Refactor.
- Review.
- QA.
- Documentation.
- Release / deploy.

Do not mix roles unnecessarily. In high-risk work, research, planning, implementation, review, and QA should be separate steps.

## 4. Start Checklist

Before editing files, the active Agent must answer:

```text
Goal:
Layer touched:
Risk level:
Affected business areas:
Files or modules likely involved:
Approval / audit impact:
Testing plan:
Documentation plan:
```

For low-risk documentation changes, this may be brief. For high-risk work, it must be explicit.

## 5. Scope Control

Agents must:

- Modify only files needed for the current lake.
- Read a file before editing it.
- Preserve unrelated dirty work.
- Avoid broad formatting churn.
- Avoid opportunistic refactors.
- Complete the selected lake's necessary implementation, tests, primary error paths, safety, and recovery boundaries before calling it done.
- Stop when the lake's acceptance outcome is achieved, then record the next lake rather than silently expanding into it.

If an unrelated issue is discovered, report it as follow-up instead of fixing it silently.

## 6. High-Risk Work Rules

High-risk work includes:

- Prices.
- Inventory.
- Order state.
- Money, settlement, fees, profit.
- Auth, RBAC, account permissions.
- Approval and audit.
- Agent autonomy and execution.
- EventBus, Command Dispatcher, Scheduler, ToolBridge.
- External platform publishing or sync.
- Database migrations and destructive data changes.

High-risk work requires:

- Business-language impact statement.
- Recommended approach.
- Focused implementation scope.
- Tests or explicit verification.
- Review before delivery.
- Documentation update when contracts or workflows change.

## 7. Review Checklist

Review Agents must check:

- Does the change match the Owner goal?
- Does it stay in the correct system layer?
- Did it pollute Platform Kernel with business logic?
- Does it duplicate an existing concept?
- Does it bypass Auth, RBAC, Approval, or Audit?
- Does it create hidden external side effects?
- Are event and command contracts clear?
- Are tests proportional to risk?
- Can the Owner verify the outcome without reading code?
- Do governance or feature docs need updates?

Findings should lead with severity and file references when reviewing code.

## 8. QA Checklist

QA must verify the smallest useful surface:

- Backend: package tests for touched domain; broader tests for shared Kernel changes.
- Frontend: build, lint, unit tests, or browser QA depending on touched surface.
- E2E: required for critical user flows or high-risk UI workflow changes.
- Docs-only: verify links, file names, and internal consistency.

If a check is not run, explain why.

## 9. Final Report Checklist

Final reports must include:

- Summary in business language.
- Files changed.
- Verification performed.
- Risks or limitations.
- Where the Owner can read, try, or verify the result.

Do not claim completion if required verification failed or was not run.

## 10. Multi-Agent Handoff

When handing off to another Agent, include:

- Business goal.
- Current decision.
- Relevant docs.
- Files touched.
- Tests run.
- Remaining risk.
- Next recommended action.

The receiving Agent must not assume prior context that is not written down.
