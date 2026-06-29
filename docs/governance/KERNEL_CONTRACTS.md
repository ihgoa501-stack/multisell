# Kernel Contracts

Last updated: 2026-06-27

This document defines the expected contracts for LingMirror Platform Kernel. It is intentionally higher level than code. It exists so Agents can extend the platform without inventing new incompatible patterns.

## 1. Kernel Scope

Platform Kernel includes:

- Auth / JWT
- RBAC / permissions
- Approval policies
- Audit logs
- EventBus
- Command Dispatcher
- Scheduler
- ToolBridge
- Agent Registry and execution infrastructure
- Config
- Observability
- Migration infrastructure

Kernel code provides reusable mechanisms. Domain code provides business meaning.

## 2. Event Contract

Events are for facts that happened or signals that a workflow should observe.

Rules:

- Event names must be stable and namespaced.
- Prefer lowercase dot-separated topics, such as `order.created`, `inventory.low`, `scheduler.tick.agent_id`.
- Payloads must be structured and versionable.
- Events should include enough identity to trace source and affected entity.
- Events must not silently perform business mutations by themselves.
- Subscribers must be safe to retry or must document non-idempotent behavior.

Recommended event payload fields:

```text
id:
topic:
version:
source:
occurred_at:
actor:
tenant_id:
entity_type:
entity_id:
correlation_id:
payload:
```

Adding or changing an event that affects Agent workflows is medium risk by default and high risk when it can trigger external side effects or critical business mutations.

## 3. Command Contract

Commands are explicit requests to perform an action.

Rules:

- Command names must be stable and action-oriented.
- Commands must validate inputs before side effects.
- Commands that mutate critical data must enforce permission, approval, and audit requirements.
- Command results must distinguish success, validation failure, policy block, external failure, and internal error.
- Commands should be idempotent where possible.

Recommended command shape:

```text
name:
version:
actor:
tenant_id:
correlation_id:
approval_id:
input:
```

Recommended result shape:

```text
status:
message:
data:
audit_id:
external_reference:
error_code:
```

Commands involving price, inventory, order state, money, external publishing, account permissions, credentials, or destructive changes require approval and audit unless a written policy explicitly allows otherwise.

## 4. Scheduler Contract

Scheduler triggers periodic or delayed work. It must not hide business decisions.

Rules:

- Scheduled jobs must have a name, owner, interval, and purpose.
- Scheduled jobs should publish events or dispatch commands through documented contracts.
- Scheduled jobs must be safe against duplicate execution.
- Scheduled jobs that can cause side effects must record trace and audit context.
- New scheduled jobs are medium risk by default and high risk when they can trigger critical actions.

Every scheduled Agent should document:

```text
agent_id:
schedule:
input data:
possible recommendations:
possible actions:
approval requirement:
audit trail:
failure behavior:
```

## 5. ToolBridge Contract

ToolBridge lets Agents use external tools through drivers or plugins.

Rules:

- Tools must declare capabilities, input schema, output schema, and side effects.
- Tools must distinguish dry-run, sandbox, and production modes when relevant.
- Credentials must come from approved configuration or secret storage.
- Tool calls must be logged with correlation IDs.
- External failures must degrade safely and return actionable errors.
- Tools must not bypass approval for critical actions.

Tool categories:

- Read-only tools: search, fetch, inspect, summarize.
- Suggestion tools: analyze and recommend.
- Mutation tools: create, update, delete, publish, sync, push.

Mutation tools are high risk unless explicitly scoped to local test data.

## 6. Approval Contract

Approval controls whether a proposed action may execute.

Rules:

- Approval must be required for critical mutations unless policy explicitly grants autonomy.
- Approval records must contain action summary, actor, target, risk, proposed changes, and expiry when relevant.
- Approval decision must be recorded as approved, rejected, expired, canceled, or superseded.
- Execution must verify approval status at execution time.
- Approval bypasses must be explicit and auditable.

Approval is required by default for:

- Price changes.
- Inventory changes.
- Order state changes.
- Refunds, settlement, fee, or money-impacting changes.
- External platform publishing.
- Credential or permission changes.
- Destructive data changes.
- Autonomous Agent execution of business mutations.

## 7. Audit Contract

Audit records what changed and why.

Rules:

- Mutations to important business data must produce audit records.
- Audit records must be immutable from normal application flows.
- Audit should include actor, action, target, before/after where practical, request ID, correlation ID, approval ID, and result.
- Agent actions must include Agent identity and decision context.
- External calls should include external reference IDs when available.

Recommended audit fields:

```text
actor_type:
actor_id:
action:
target_type:
target_id:
risk_level:
approval_id:
request_id:
correlation_id:
before:
after:
result:
created_at:
```

## 8. Auth and RBAC Contract

Auth verifies identity. RBAC determines allowed actions.

Rules:

- Non-public APIs require JWT.
- Mutation APIs require permission checks.
- Agent actions must run under an explicit actor identity or service identity.
- Permission checks must happen server-side.
- UI hiding is not a substitute for backend authorization.

Permission changes are high risk when they expand access.

## 9. Observability Contract

Platform behavior must be observable enough to diagnose and trust.

Rules:

- Requests should carry request IDs.
- Agent workflows should carry correlation IDs.
- Errors should be logged with enough context to debug without exposing secrets.
- High-risk workflows should expose status: pending, approved, executing, succeeded, failed, blocked.
- External calls should record provider, operation, latency, result, and safe error summary.

For Agent workflows, the system should be able to answer:

- What triggered this?
- What data was used?
- What decision was made?
- What action was proposed?
- Was it approved?
- What executed?
- What changed?
- What failed?

## 10. Migration Contract

Database changes are high risk when they affect existing data, critical tables, or production rollout.

Rules:

- Migrations must be deterministic.
- Destructive migrations require explicit Owner approval and rollback guidance.
- Data backfills must be idempotent or document safe retry behavior.
- Model changes must align with API and UI expectations.
- Tests should cover affected domain behavior.

## 11. Contract Change Rules

When changing Kernel contracts:

- State which contract changes.
- State backward compatibility impact.
- Update documentation.
- Add or update tests when behavior changes.
- Provide an Owner-readable risk statement for medium/high-risk changes.
