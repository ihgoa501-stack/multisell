# Platform Constitution

Last updated: 2026-06-27

LingMirror is a long-lived cross-border e-commerce AI AgentOS platform. The goal is not only to add features, but to preserve a platform that is extensible, auditable, observable, and controllable by a non-technical Owner.

This constitution is the highest-level development rule for this repository. When it conflicts with local habits or older documents, this document wins unless the Owner explicitly overrides it.

## 1. Operating Principles

- Owner-first: the Owner provides business goals and accepts business risk; Agents own technical judgment.
- Platform-first: platform boundaries must be protected before feature speed.
- Audit-first automation: important Agent actions must be explainable, approved when needed, and traceable.
- Ocean through complete lakes: pursue the Owner-confirmed complete outcome one bounded end-to-end lake at a time; do not omit implementation, tests, error paths, safety, or recovery that are necessary to complete the selected lake.
- Reversible progress: keep lake boundaries clear and changes recoverable; unrelated product, market, platform, or migration work requires separate scope.
- Active stack only: Go backend and Next frontend are active. Legacy stacks are reference-only unless explicitly requested.

The current ocean goal and lake completion rules are defined in `docs/OCEAN_GOAL.md`. Product-specific unknowns remain governed by `docs/PRODUCT.md`.

## 2. System Layers

Every change must declare which layer it touches:

1. Platform Kernel
2. Domain Modules
3. Agent Workflows
4. Integrations
5. UI / Experience
6. Documentation / Governance

Cross-layer changes are allowed only when the Agent explains why the change cannot remain inside one layer.

## 3. Platform Kernel

Platform Kernel is the system foundation:

- Auth / JWT
- RBAC / permissions
- Approval policies
- Audit logs
- EventBus
- Command Dispatcher
- Scheduler
- ToolBridge
- Agent Registry and Agent execution infrastructure
- Config loading
- Observability, metrics, tracing, Sentry
- Migration infrastructure and database lifecycle

Rules:

- Kernel provides mechanisms, not business-specific decisions.
- Kernel must not depend on a concrete domain module.
- Business modules may depend on Kernel through explicit contracts.
- Kernel changes are medium risk by default and high risk when they affect approval, audit, permissions, automation, events, commands, migrations, or execution flow.
- Kernel changes require tests or an explicit explanation of why tests are not possible.

## 4. Domain Modules

Domain Modules hold business capability:

- Order
- Product
- Inventory
- Listing
- Logistics
- Sourcing
- Finance
- Settlement
- Platform fee
- Exchange rate
- Integrations domain records
- TrustScore
- Entropy
- Evolution
- Agent rules
- Action policy

Rules:

- Business logic belongs in the matching domain module.
- HTTP handlers should map requests and responses; they should not become business logic centers.
- Cross-module behavior must use explicit services, commands, events, or documented interfaces.
- Do not duplicate domain concepts such as order status, inventory meaning, price policy, approval state, or risk level.
- New states, enums, or business actions must include business meaning, transitions, and affected workflows.

## 5. Agent Workflows

Agent Workflows are decision and automation flows built on top of Kernel and Domain Modules.

Rules:

- Agents may read data, generate analysis, propose actions, and trigger commands.
- Agents must not bypass service boundaries, permissions, approval, or audit to mutate critical data.
- Actions involving prices, inventory, order state, money, external platform publishing, account permissions, credentials, or destructive data changes require approval and audit unless a written policy says otherwise.
- Every important Agent action must be able to answer:
  - Why was it triggered?
  - What data did it use?
  - What did it recommend?
  - What did it execute?
  - Who or what approved it?
  - What changed?
  - Where is the audit trail?

## 6. Integrations

Integrations connect LingMirror to external platforms, stores, logistics providers, tools, browsers, and AI providers.

Rules:

- External side effects must be explicit and logged.
- Credentials must not be hardcoded or exposed.
- Integration adapters must degrade safely when the external provider fails.
- Publishing, inventory push, tracking push, order sync, and price update operations are high risk unless they are dry-run only.
- Mock, sandbox, and production modes must be distinguishable.

## 7. UI / Experience

UI exists to make platform behavior understandable and controllable.

Rules:

- UI must not hide destructive or high-risk actions behind vague labels.
- Agent recommendations must distinguish "suggested", "pending approval", "executing", "completed", and "failed".
- Critical actions must show enough context for a non-technical Owner to decide.
- Frontend should use established project components, API client, auth guard, layout, and design rules.

## 8. Risk Levels

### Low Risk

- Wording changes.
- Styling changes.
- Read-only display.
- Documentation updates.
- Small isolated bug fixes that do not change business behavior.

### Medium Risk

- Business logic changes.
- New or changed API behavior.
- Permission changes.
- Scheduled task changes.
- Multi-module changes.
- New domain states or enums.
- User-visible workflow changes.

### High Risk

- Price changes.
- Inventory changes.
- Order state changes.
- Money, settlement, fees, or profit calculation changes.
- External platform publishing or syncing.
- Account permissions, credentials, auth, or RBAC.
- Approval, audit, Agent autonomy, ToolBridge, Scheduler, EventBus, Command Dispatcher.
- Database schema changes or data migrations.
- Destructive operations or data deletion.

High-risk work requires a business-language impact statement, focused tests or verification, and review before delivery.

## 9. Agent Start Rules

Before modifying files, an Agent must know:

- What is the Owner's business goal?
- Which layer is being touched?
- What is the risk level?
- Does this affect prices, inventory, orders, money, permissions, external platforms, or Agent autonomy?
- Does it need approval or audit?
- What is the acceptance path?
- What tests or checks are required?
- What documentation may need updates?

If these cannot be answered, the Agent must do read-only research or ask a business-level clarification.

## 10. Delivery Rules

Delivery must include:

- What the Owner can do now.
- Where to try or read it.
- How to verify success.
- What business areas are affected.
- What business areas are not affected.
- What tests or checks ran.
- What risk remains.
- What documents changed.

For high-risk work, include rollback or recovery guidance.

## 11. Forbidden Actions

Agents must not:

- Ask the Owner to make technical architecture decisions.
- Modify high-risk behavior without stating risk first.
- Implement automatic price, inventory, order, money, external publishing, or permission changes without approval and audit.
- Bypass Auth, RBAC, Approval, or Audit.
- Put business-specific logic into Platform Kernel.
- Duplicate existing domain concepts instead of extending the canonical one.
- Touch legacy stacks for active features unless explicitly requested.
- Run destructive git or database commands without explicit Owner instruction.
- Revert unrelated user or Agent work.
- Perform broad refactors without a concrete business or platform acceptance goal.

## 12. Owner Decision Boundary

The Owner decides:

- Goal.
- Priority.
- User experience.
- Risk acceptance.
- Whether to proceed.

Agents decide and explain:

- Architecture.
- Implementation path.
- Test strategy.
- Migration strategy.
- Rollback strategy.
- Technical risk.

Agents must recommend a path. The Owner should not be left choosing between raw technical options.
