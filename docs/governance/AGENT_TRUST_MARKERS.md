# Agent Trust Markers

Last updated: 2026-07-06

Every agent output, recommendation, decision, or structured message must include a `trust_level` field. This field tells the platform and downstream consumers how much weight to give the output without needing to trace its origin manually.

The system defines five trust levels, inheriting upward:

```
STUB < DETERMINISTIC_RULE < REAL_LLM < HUMAN_APPROVED < PRODUCTION_EXECUTED
```

Higher levels include all guarantees of lower levels plus their own.

## 1. Marker Definitions

### STUB

| Attribute | Value |
|-----------|-------|
| Label | Stub / Placeholder |
| Meaning | No real logic. A hardcoded return, mocked response, or empty branch. |
| When to use | Feature not implemented, demo shell, test fixture, fallback path before real logic is wired. |
| Downstream rule | Consumers MUST NOT act on STUB output. Agents receiving STUB must report "not available" at their boundary. |
| Audit evidence | Code reference (file + line). No runtime evidence required — the marker IS the signal. |
| Persistence | STUB outputs SHOULD NOT be persisted in business tables. If persisted (e.g. during development), the record MUST be tagged with `is_stub=true`. |

### DETERMINISTIC_RULE

| Attribute | Value |
|-----------|-------|
| Label | Rule-based |
| Meaning | Computed by hardcoded logic: formulas, conditional branches, database queries, math. No LLM involved. |
| When to use | Price calculations, fee computations, aggregation, status transitions, threshold checks, validation. |
| Downstream rule | Trust the output as the correct execution of the programmed rule. Bugs are logic bugs, not trust-level issues. Downstream consumers MAY act without additional review. |
| Audit evidence | Input parameters used, rule version or function hash, timestamp, and output value. For money-related rules: also log the formula or rule reference. |

### REAL_LLM

| Attribute | Value |
|-----------|-------|
| Label | LLM-generated |
| Meaning | Produced by an actual LLM call (OpenAI, Anthropic, etc.) with the given prompt and context. |
| When to use | Recommendations, analysis, natural language responses, classification, extraction, summarization, decision reasoning. |
| Downstream rule | Output is probabilistic. Downstream consumers MUST treat it as advisory, not authoritative. REAL_LLM output MUST NOT directly trigger L3 actions (production mutations) without HUMAN_APPROVED. |
| Audit evidence | Prompt template ID or full prompt snapshot, model name + version used, temperature / top-p, response, token count, latency. The audit log MUST allow reproducing the generation context. |

### HUMAN_APPROVED

| Attribute | Value |
|-----------|-------|
| Label | Human-approved |
| Meaning | A human Owner or operator reviewed the output and explicitly approved it. |
| When to use | After a human approves a REAL_LLM or DETERMINISTIC_RULE recommendation that triggers a business mutation (publish listing, change price, modify inventory, approve order). |
| Downstream rule | The output is authoritative for the approved action. The system MAY execute the action automatically after approval is recorded. HUMAN_APPROVED does not mean the output is correct — it means a human accepted responsibility. |
| Audit evidence | Approval record: who approved (actor ID + name), when (ISO timestamp), what they saw at approval time (snapshot of the output and its context), approval action taken (Approve / Reject / Approve-with-changes). If the human provided modifications, those MUST be recorded separately. |

### PRODUCTION_EXECUTED

| Attribute | Value |
|-----------|-------|
| Label | Executed in production |
| Meaning | The output was written back to a production system (database write, API call to external platform, state transition that affects live data). |
| When to use | After a mutation is applied: order created, listing published, price updated, inventory decremented, fee charged. |
| Downstream rule | The output is a historical fact. Downstream consumers read it as the record of what happened. No further action on this specific output. |
| Audit evidence | Execution timestamp, request/response pair for the write, target system + resource ID, status (success/failure), idempotency key, error detail if failed. Must be connectable to the HUMAN_APPROVED or REAL_LLM that preceded it via a correlation ID. |

## 2. Inheritance Rules

A marker inherits all guarantees of the levels below it:

- `PRODUCTION_EXECUTED` implies `HUMAN_APPROVED` (or a bypass documented in the approving policy), which implies the output was real, which implies reasoning was applied.
- `HUMAN_APPROVED` implies `REAL_LLM` or `DETERMINISTIC_RULE` (the human approved a specific output).
- `REAL_LLM` implies the output exists and is generated, not mocked.

The field is always the highest applicable level:

```json
{
  "trust_level": "HUMAN_APPROVED",
  "inherited_from": "REAL_LLM",
  "original_trust_level": "REAL_LLM",
  "content": "Recommendation to raise price of SKU-X to $29.99"
}
```

## 3. Cross-Cutting Rules

1. **Every agent message MUST carry a `trust_level`.** The platform EventBus, Command Dispatcher, and ToolBridge MUST reject messages without a defined trust level (configurable per environment — enforced in production, warn-only in development).

2. **STUB must never reach production consumers.** CI/CD pipelines that detect STUB trust_level in a deployed artifact's output MUST fail the deployment gate. See `ACCEPTANCE_GATE.md`.

3. **L3 actions (production mutations) require at minimum HUMAN_APPROVED.** REAL_LLM alone is insufficient. The ONLY exception is system-initiated deterministic actions defined in `actioncatalog.DefaultEntries()` with `RequireApproval=false` — these are DETERMINISTIC_RULE by nature and follow their own audit path via MutationGuard.

4. **Trust level must be logged in the audit trail.** The `operation_log` entry for any action MUST include the trust_level at the time of the action and any escalation path (e.g., "started as REAL_LLM, escalated to HUMAN_APPROVED at 2026-07-06T14:30:00Z by user admin@example.com").

5. **Downsampling is not allowed.** An output produced by REAL_LLM must not be re-classified as DETERMINISTIC_RULE. The marker reflects the actual production path, not the desired one.

## 4. Example Usage

| Scenario | trust_level |
|----------|-------------|
| Mock agent returning canned response during development | STUB |
| Shipping cost calculated by formula engine | DETERMINISTIC_RULE |
| Listing title generated by GPT-4o | REAL_LLM |
| Price change reviewed and confirmed by Owner via approval dialog | HUMAN_APPROVED |
| Listing published to Shopee via PlatformAdapter.Publish() | PRODUCTION_EXECUTED |

## 5. Schema

Agents SHOULD use the following type in messages and audit records:

```go
type TrustLevel string

const (
    TrustStub              TrustLevel = "STUB"
    TrustDeterministicRule TrustLevel = "DETERMINISTIC_RULE"
    TrustRealLLM           TrustLevel = "REAL_LLM"
    TrustHumanApproved     TrustLevel = "HUMAN_APPROVED"
    TrustProductionExecuted TrustLevel = "PRODUCTION_EXECUTED"
)

type TrustMarker struct {
    Level        TrustLevel `json:"trust_level"`
    InheritedFrom TrustLevel  `json:"inherited_from,omitempty"`
    OriginalLevel TrustLevel  `json:"original_trust_level,omitempty"`
    EvidenceRef  string     `json:"evidence_ref,omitempty"` // link to audit log entry
}
```
