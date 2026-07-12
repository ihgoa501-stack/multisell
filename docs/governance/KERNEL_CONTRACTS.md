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

- **Mutation Guard**: every EventBus subscriber that mutates business state MUST use
  `eventbus.MutationGuard.Guard()` to wrap the handler. The guard ensures:

  1. The mutation's `SystemAction` is registered in the ActionCatalog (mandatory convention — enforced during review, not code-enforced by the guard itself).
     (`actioncatalog.DefaultEntries`) as a `system.*` action type.
  2. A structured audit entry (`pending` -> `executed`|`failed`) is written to
     `operation_log` with `trigger_type='eventbus'` before and after handler execution.
  3. The mutation's domain, operator, and correlation ID are captured in the audit trail.

  Mutations that bypass `MutationGuard` but still write to business tables (inventory,
  order, finance, purchase, aftersales) are governance violations.

- **System Actions**: business-mutating EventBus handlers must have a corresponding
  `system.*` entry in `actioncatalog.DefaultEntries()`. These are L3 (production mutation)
  with `RequireApproval=false` because they are deterministic system-internal transitions,
  not agent-initiated decisions. The audit trail via MutationGuard replaces the
  DispatchSafe approval gate.

- **Registration Map**: new mutating event handlers must:

  1. Add a `system.*` entry to the ActionCatalog.
  2. Wrap the handler with `mutationGuard.Guard()`.
  3. The `MutationInfo.SystemAction` must match the catalog entry's `ActionType` exactly.

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
- Production Agent Actions that provide an `idempotency_key` are claimed in
  `command_execution` before handler execution. A succeeded command replays its
  stored result, an active claim rejects concurrent duplicates, a failed claim
  may retry, and an expired processing lease may be reclaimed after a crash.
- An idempotency key is globally bound to its first `action_type` and `agent_id`;
  callers must never reuse it for a different logical action.
- Consumers with their own comprehensive gate chain (e.g. ai.Service.ExecuteAction) may use raw Dispatch() after passing their gates. DispatchSafe() enforces mode, approval, and audit at the AgentAction envelope level for consumers without their own gate chain.

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
status:            suggested | pending_approval | approved | rejected | executing | completed | failed | blocked
idempotency_key:   string               // prevents duplicate execution
correlation_id:    string               // ties to agent workflow trace
audit_id:          string               // audit record reference
input:             map                  // action parameters
rollback_note:     string               // human guidance for reversing
```

### Risk Level Categories

| Risk Level | Business Examples                               | Approval Needed |
|------------|------------------------------------------------|-----------------|
| low        | stock_alert, dashboard_summary, read_data      | No              |
| medium     | listing_draft, compliance_flag, suggest_price  | No (suggestion) |
| high       | price_update, inventory_change, order_cancel, listing_publish, credential_change | Yes |

### Execution Mode Rules

| Mode       | Behavior                                                       |
|------------|---------------------------------------------------------------|
| dry_run    | Validate handler exists and inputs are parseable. Never mutate. |
| sandbox    | Execute against test/stub data. No external side effects.      |
| production | Full execution with guardrails: high-risk actions need approval. |

### Code Reference

- `backend-go/internal/platform/command/action.go` — `AgentAction` struct, `RiskLevel` type, `ActionMode` type.
- `backend-go/internal/platform/command/command.go` — `DispatchSafe` method enforces mode and approval rules.
- `backend-go/internal/platform/command/idempotency.go` — durable command claim,
  result replay, failure retry, and expired-lease recovery.

## 5. Scheduler Contract

Scheduler triggers periodic or delayed work. It must not hide business decisions.

Failed ticks are persisted in `scheduler_retry`. Startup must recover the retry queue before reporting running; an unreadable recovery store keeps readiness false. EventBus follows the same rule for pending outbox events. Bus and Scheduler start only after all subscriptions and tasks are registered.

Only the backend instance holding the PostgreSQL advisory leader lease may report Scheduler running. Standby instances retry acquisition while remaining not-ready. EventBus shutdown rejects new publications, drains queued and in-flight handlers within a deadline, and only then closes workers.

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

Production mutation calls must have all three guarantees: target-bound approval, a durable `tool_execution` idempotency claim, and a driver that forwards the same idempotency key to the external provider. Missing any guarantee must fail closed. Persisting only a local result does not prove an external side effect is exactly-once.

External calls use bounded, context-cancellable retries and a closed/open/half-open circuit breaker. Production mutation retries must reuse the original provider idempotency key. Circuit state, attempts, failures and duration are observable metrics.

Rules:

- Tools must declare capabilities, input schema, output schema, and side effects.
- Tools must distinguish dry-run, sandbox, and production modes when relevant.
- Credentials must come from approved configuration or secret storage.
- Tool calls must be logged with correlation IDs.
- External failures must degrade safely and return actionable errors.
- Tools must not bypass approval for critical actions.
- The registered driver's category is authoritative; callers cannot relabel a
  mutation tool as read-only.
- Production mutation tools use the typed `ToolCall` path, require a live
  target-bound approval and an idempotency key, and cannot use opaque legacy
  approval strings.

Tool categories:

- Read-only tools: search, fetch, inspect, summarize.
- Suggestion tools: analyze and recommend.
- Mutation tools: create, update, delete, publish, sync, push.

Mutation tools are high risk unless explicitly scoped to local test data.

## 7. Approval Contract

Approval controls whether a proposed action may execute.

Execution consumption rules:

- A production side effect consumes exactly one approved Owner decision through
  `approval_execution`; approval ID and idempotency key are globally unique.
- The execution binding (approval, idempotency key, action and target) is
  immutable after insertion.
- PostgreSQL permits only `processing -> succeeded|failed` and
  `failed -> processing` for a retry with the same binding. `succeeded` is
  terminal and execution rows cannot be deleted.
- HTTP, Command and ToolBridge must authorize, claim durable idempotency, then
  consume approval before invoking a side effect. Approval status alone never
  authorizes execution.

HTTP mutation classification rules:

- Every POST/PUT/PATCH/DELETE route must appear exactly once in
  `internal/platform/routecatalog/mutation_policy.tsv` as `public`, `standard`
  or `high`; additions, removals, path changes and source moves are CI failures
  until explicitly reviewed.
- `public` is a closed allowlist for login/register/refresh and signature-
  verified platform webhooks. `standard` requires authentication at the route
  or protected group and is synchronously audited. `high` additionally maps to
  an action type and uses one-time approval consumption plus idempotency.
- Runtime middleware fails closed when an authenticated mutation has no policy.
  All DELETE routes are high risk. Emergency kill-switch activation/deactivation
  remains admin-RBAC + synchronous-audit protected so the switch cannot lock its
  own recovery path behind the write gate it disables.

Rules:

- Approval must be required for critical mutations unless policy explicitly grants autonomy.
- Approval records must contain action summary, actor, target, risk, proposed changes, and expiry when relevant.
- Approval decision must be recorded as approved, rejected, expired, canceled, or superseded.
- Execution must verify approval status at execution time.
- Execution must bind approval to the exact action type and target; approval
  for one mutation cannot authorize another mutation.
- Expired approvals cannot be reviewed as approved or discovered as executable.
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

## 8. Audit Contract

`operation_log` is append-only at the PostgreSQL trigger layer and forms a serialized SHA-256 predecessor chain. Integrity verification must recompute the complete chain. This detects ordinary row tampering and blocks the application database role from UPDATE/DELETE; it does not protect against a PostgreSQL superuser disabling the trigger, so external immutable checkpoints remain a separate production control.

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

JWTs carry a `kid`. Only the current key signs new tokens; explicitly configured previous keys may verify tokens during a bounded rotation window. Removing a previous key ends that compatibility window. Unknown key IDs fail closed.

Auth verifies identity. RBAC determines allowed actions.

Rules:

- Non-public APIs require JWT.
- Access JWTs use the configured HS256 contract and require a positive numeric
  `user_id`; incomplete identities fail closed.
- Refresh JWTs are backed by persisted sessions, rotate once, and revoke the
  active token family when a rotated predecessor is replayed.
- Mutation APIs require permission checks.
- Agent actions must run under an explicit actor identity or service identity.
- Permission checks must happen server-side.
- UI hiding is not a substitute for backend authorization.

Permission changes are high risk when they expand access.
Disabled roles grant no permissions even when historical assignments remain.

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
