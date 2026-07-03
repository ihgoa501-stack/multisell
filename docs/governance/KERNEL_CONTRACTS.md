# Kernel Contracts

Last updated: 2026-07-03

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
- **Idempotency**: events that carry business-state mutations (inventory, order, financial)
  MUST set an `idempotency_key` via `eventbus.WithIdempotencyKey(ctx, key)` on the Publish
  context. The event bus uses an atomic claim/state model:

  1. A row is INSERTed into `event_processed` with `state='processing'` BEFORE handler dispatch
     (the INSERT is the atomic claim — only one worker wins per key).
  2. Concurrent duplicates (different `event_id`, existing `processing` row) are skipped.
  3. Retries (same `event_id`, existing `processing` row) pass through.
  4. On handler success, the row transitions to `state='succeeded'` with `processed_at` set.
  5. On final failure (DLQ), the row transitions to `state='failed'`.
  6. DLQ replay may reclaim a `failed` row back to `state='processing'` with `processed_at=NULL`.

  Key format:

  	```
  	{business_action}:{business_id}
  	Examples:
  	  purchase_order_received:PO-2024-001
  	  aftersale_processed:42
  	```

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
idempotency_key:  // set via WithIdempotencyKey(ctx, key) — bus-level dedup
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

## 4. Agent Action Contract

Agent Actions are the canonical typed envelope for everything an Agent proposes or executes. Every action must be representable as a structured record, not free-form model output.

Rules:

- Every action must have a name, version, risk level, approval requirement, and mode.
- Actions must distinguish dry_run (validate only), sandbox (execute on test data), and production (execute with guardrails).
- High-risk actions (RiskLevel ≥ high) require approval before production execution.
- Actions must carry audit context: agent identity, actor, target, risk level, and mode.
- Idempotency keys must be provided for mutation actions where duplicate execution is harmful.
- Rollback notes should be captured when an action is reversible.

### Canonical Action Shape

```text
action_type:       string               // e.g. "price_update", "stock_alert"
version:           string               // semantic version of the action schema
agent_id:          string               // who proposed the action
actor:             string               // system or user identity executing
tenant_id:         string               // optional tenant scope
target_type:       string               // "sku", "product", "order", "listing"
target_id:         string
risk_level:        low | medium | high  // business impact
approval_required: bool
approval_id:       int64 | nil          // set when an approval is obtained
mode:              dry_run | sandbox | production
idempotency_key:   string               // prevents duplicate execution
input:             map                  // action parameters
rollback_note:     string               // human guidance for reversing
```

### Risk Level Categories

| Risk Level | Business Examples                               | Approval Needed |
|------------|------------------------------------------------|-----------------|
| low        | stock_alert, dashboard_summary, read_data      | No              |
| medium     | listing_draft, compliance_flag, suggest_price  | No (suggestion) |
| high       | price_update, inventory_change, order_cancel, platform_publish, credential_change | Yes |

### Execution Mode Rules

| Mode       | Behavior                                                       |
|------------|---------------------------------------------------------------|
| dry_run    | Validate handler exists and inputs are parseable. Never mutate. |
| sandbox    | Execute against test/stub data. No external side effects.      |
| production | Full execution with guardrails: high-risk actions need approval. |

### Code Reference

- `internal/platform/command/action.go` — `AgentAction` struct, `RiskLevel` type, `ActionMode` type.
- `internal/platform/command/command.go` — `DispatchSafe` method enforces mode and approval rules.

## 5. Scheduler Contract

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

## 6. ToolBridge Contract

ToolBridge lets Agents use external tools through drivers or plugins.

Rules:

- Tools must declare capabilities, input schema, output schema, and side effects.
- Tools must distinguish dry-run, sandbox, and production modes when relevant.
- Credentials must come from approved configuration or secret storage.
- Tool calls must be logged with correlation IDs.
- External failures must degrade safely and return actionable errors.
- Tools must not bypass approval for critical actions.

### Tool Categories

| Category     | Side Effects                          | Example                     | Risk Level |
|-------------|---------------------------------------|-----------------------------|------------|
| read        | None. Search, fetch, inspect.         | search_product, fetch_page  | low        |
| suggestion  | None. Analyse and recommend.          | analyze_price_trend         | low        |
| mutation    | Create, update, delete, publish, sync.| publish_listing, sync_inventory | high   |

### Canonical Tool Call Shape

```text
tool_name:         string               // unique tool identifier
version:           string               // schema version
category:          read | suggestion | mutation
mode:              dry_run | sandbox | production
input:             map                  // tool-specific parameters
approval_id:       int64 | nil          // required in production for mutation tools
idempotency_key:   string               // prevents duplicate external calls
correlation_id:    string               // ties to agent workflow trace
```

### Validation Rules

| Category | Mode        | Approval Required | Notes                                   |
|----------|-------------|-------------------|-----------------------------------------|
| read     | any         | No                | Always allowed.                         |
| suggestion | any       | No                | Always allowed. Produces recommendations. |
| mutation | dry_run     | No                | Validated but not executed.             |
| mutation | sandbox     | No                | Executed against test/sandbox endpoints.|
| mutation | production  | Yes               | Requires a valid approval_id.           |

Mutation tools in production mode without a valid approval_id return `ErrMutationRequiresApproval`.

### Code Reference

- `internal/platform/toolbridge/tool.go` — `ToolCall` struct, `ToolCategory` type, `Validate()` method.
- `internal/platform/toolbridge/bridge.go` — existing `FetchPage` and `Route` for read-only tools.

## 7. Approval Contract

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

### UnifiedAction ↔ Approval Linkage (P5)

Agent UnifiedActions that require approval are automatically linked to `approval_request` records:

- When the Orchestrator creates a UnifiedAction with `requires_approval=true` and the policy outcome is "needs human review", an `approval_request` is created with `entity_type="unified_action"` and `entity_id=<action.ID>`.
- When the approval is reviewed (approved/rejected), the linked UnifiedAction's status, reviewer, and timestamp fields are synced automatically.
- This applies to all scheduled agents (A5 stock_alert, A6 profit_watch, A7 compliance_check, G3 discount_risk_check, etc.) that run through the Orchestrator with autonomy level `supervised` or `guided`.

## 8. Audit Contract

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

## 9. Auth and RBAC Contract

Auth verifies identity. RBAC determines allowed actions.

Rules:

- Non-public APIs require JWT.
- Mutation APIs require permission checks.
- Agent actions must run under an explicit actor identity or service identity.
- Permission checks must happen server-side.
- UI hiding is not a substitute for backend authorization.

Permission changes are high risk when they expand access.

## 10. Observability Contract

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

## 11. Migration Contract

Database changes are high risk when they affect existing data, critical tables, or production rollout.

Rules:

- Migrations must be deterministic.
- Destructive migrations require explicit Owner approval and rollback guidance.
- Data backfills must be idempotent or document safe retry behavior.
- Model changes must align with API and UI expectations.
- Tests should cover affected domain behavior.

## 12. Contract Change Rules

When changing Kernel contracts:

- State which contract changes.
- State backward compatibility impact.
- Update documentation.
- Add or update tests when behavior changes.
- Provide an Owner-readable risk statement for medium/high-risk changes.
