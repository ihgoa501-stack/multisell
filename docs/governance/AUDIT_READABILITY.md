# Audit Readability

Last updated: 2026-07-06

Audit logs exist so the non-technical Owner can verify what happened, who did it, and why. If the Owner cannot understand an audit entry, the audit system has failed. This document defines the mandatory structure, format, and searchability requirements for all audit entries across the platform.

## 1. Required Fields per Audit Entry

Every audit entry MUST contain the following fields. The system MUST reject entries that omit a required field.

| # | Field | Name | Requirement | Example |
|---|-------|------|-------------|---------|
| 1 | `who` | Actor identity | Actor's unique ID, display name, and role label | `{"id": "usr_a1b2c3", "name": "Alice Chen", "role": "Owner"}` |
| 2 | `when` | Timestamp | ISO 8601 with timezone and millisecond precision | `"2026-07-06T14:30:00.000+08:00"` |
| 3 | `what` | Resource + action + before/after | Resource type and identifier, action verb, state change description. Before/after MUST be included for any mutation. | `{"resource": "listing", "id": "lst_9876", "action": "price_update", "before": {"price": 19.99}, "after": {"price": 24.99}}` |
| 4 | `why` | Reason / trigger context | Free-text explaining why the action happened. MUST reference the trigger source (scheduler tick, agent decision, human action, API call, EventBus event). | `"Agent A2 recommended price increase based on competitor analysis. Approved by Owner via approval dialog."` |
| 5 | `result` | Outcome | `"success"` or `"failure"`. On failure: include error code and human-readable error detail. | `{"status": "success"}` or `{"status": "failure", "error_code": "PLATFORM_TIMEOUT", "error_detail": "Shopee API returned 503 after 3 retries"}` |
| 6 | `ref_id` | External reference ID | Optional string linking to a correlated external process, support ticket, order ID, or trace identifier. | `"PO-2024-001"`, `"trace:7a9b3c"` |

Full example entry:

```json
{
  "who":    { "id": "usr_a1b2c3", "name": "Alice Chen", "role": "Owner" },
  "when":   "2026-07-06T14:30:00.000+08:00",
  "what":   { "resource": "listing", "id": "lst_9876", "action": "price_update",
              "before": { "price": 19.99, "currency": "USD" },
              "after":  { "price": 24.99, "currency": "USD" },
              "summary": "Changed listing price from $19.99 to $24.99" },
  "why":    "Agent A2 recommended price increase based on competitor analysis. Approved by Owner via approval dialog (approval_id: apprv_456).",
  "result": { "status": "success" },
  "ref_id": "trace:7a9b3c"
}
```

## 2. Human-Readable Summary Requirement

Every audit entry MUST include a human-readable summary field. This field exists so the Owner can scan the audit log without parsing structured JSON.

- The summary MUST be a plain-text sentence.
- It MUST be understandable by a non-technical person who knows the business domain.
- It MUST NOT contain JSON, raw IDs without context, stack traces, or technical jargon.

**Good summaries:**

- "Alice Chen approved price change for listing 'Summer Dress S/M/L' from $19.99 to $24.99."
- "System A5 stock_alert agent triggered automatic replenishment for SKU-1234 (threshold 20, current 5)."
- "Failed to sync inventory for Shopee listing #9876: API returned 503 timeout. 3 retries attempted."
- "Admin user 'bob@example.com' changed user role for 'charlie@example.com' from 'viewer' to 'operator'."

**Bad summaries (rejected by review):**

- "Updated listing" — lacks before/after detail and human-who.
- "audit:price_update:lst_9876:success" — opaque, no human meaning.
- "err:503" — incomprehensible without context.
- "System processed event order.updated" — which order, what changed, why.
- `{"what": {"action": "update", "resource": "inventory"}}` — JSON blob in a summary field.

## 3. Anti-Patterns

### No opaque JSON blobs

The `what.before` and `what.after` fields MAY be structured (they are consumed by the UI for diff rendering), but the `what.summary` field MUST be plain text. A JSON blob placed in the summary field in place of a proper summary is a violation.

### No implicit context

Every entry must be self-standing. Do not write entries that require reading the previous entry to understand. If an action is part of a sequence, each entry repeats the relevant context (or includes a sequence ID in `ref_id`).

### No silent failures

If `result.status` is `"failure"`, the `error_detail` field MUST contain a human-readable explanation of the failure, its impact, and any automatic recovery attempted. "System error" is insufficient.

## 4. Searchability Requirements

The audit system MUST support the following filters via the API and the Owner dashboard:

| Filter | Type | Requirements |
|--------|------|-------------|
| Actor | String search | Match by actor name, email, or ID. Partial match allowed. |
| Resource | String search | Match by resource type, resource ID, or both. |
| Date range | Start + End (ISO 8601) | Required. Default to last 7 days. MUST support custom ranges. |
| Action type | Enum filter | Drop-down of known action types. MUST support multiple selection. |
| Result status | Enum filter | `"success"`, `"failure"`, or `"all"`. |
| Free text | String search | MUST search through the `summary`, `why`, and `error_detail` fields. |

Additional requirements:

- Results MUST be sorted by `when` descending by default.
- Pagination MUST be supported (page + size or cursor-based).
- Export to CSV MUST be available for any filtered result set.
- The total count matching the filter MUST be displayed (even before pagination).
- Filter parameters MUST be persisted in the URL so the Owner can share a link to a specific view.

## 5. Approval Audit Requirements

When an audit entry records an approval action, it MUST include enough context for the Owner to verify the decision was correct without cross-referencing other systems.

Mandatory fields for approval entries (in addition to the standard 6):

| Field | Description | Example |
|-------|-------------|---------|
| `approval.decision` | Approve, Reject, or Approve-with-changes | `"approve"` |
| `approval.decision_context` | What the approving person saw at decision time (snapshot, not reference) | `{"recommended_price": 24.99, "current_price": 19.99, "agent_reasoning": "Competitor X raised to $24.99; market avg is $25.50"}` |
| `approval.modifications` | If approved-with-changes: what the approver modified | `{"price": 22.99, "reason": "Compromise between current and recommended"}` |
| `approval.policy_violations` | Any policy rules that were overridden or waived by this approval | `["price_increase_exceeds_20pct_threshold"]` |
| `approval.expires_at` | If the approval is time-bound | `"2026-07-13T14:30:00.000+08:00"` |

Full approval example:

```json
{
  "who":    { "id": "usr_a1b2c3", "name": "Alice Chen", "role": "Owner" },
  "when":   "2026-07-06T14:30:00.000+08:00",
  "what":   { "resource": "listing", "id": "lst_9876", "action": "price_change_approval",
              "before": { "price": 19.99 }, "after": { "price": 24.99 },
              "summary": "Alice Chen approved price change for listing 'Summer Dress S/M/L' from $19.99 to $24.99." },
  "why":    "Agent A2 recommended increase. Approval requested via approval queue.",
  "result": { "status": "success" },
  "ref_id": "apprv_456",
  "approval": {
    "decision": "approve",
    "decision_context": {
      "recommended_price": 24.99,
      "current_price": 19.99,
      "agent_reasoning": "Competitor X raised listing price to $24.99 yesterday. Market average across 3 platforms is $25.50. Product cost basis unchanged at $11.50. Projected margin at $24.99: 54%.",
      "agent_confidence": "HIGH",
      "cost_data_completeness": { "score": 95, "threshold": 70, "passed": true }
    },
    "modifications": null,
    "policy_violations": [],
    "expires_at": null
  }
}
```

## 6. Enforcement

1. **All mutations** (database writes, external API calls, state transitions) MUST produce an audit entry conforming to this standard. The platform MUST NOT execute a mutation without a valid audit payload.

2. **Audit entries missing required fields MUST be rejected** at write time. The writing component receives an error.

3. **The `summary` field MUST be populated with a human-readable sentence.** Automated generation is acceptable (e.g., template-based: `"{actor} {action} {resource} from {before} to {after}."`), but the result MUST be a plain-text sentence.

4. **Downstream dashboards and notification systems MUST display the summary field** as the primary text for each entry. Raw JSON must be viewable on expand but not the default view.

5. **The Owner MUST be able to produce a plain-language report** of any audit-filtered result set without additional processing. CSV export fulfills this requirement.

6. **Approval audit entries** are subject to additional review during code review and system audit. They MUST NOT have missing fields and MUST include the `decision_context` snapshot.

## 7. Schema Reference

```go
type AuditEntry struct {
    ID       string          `json:"id"`
    Who      ActorRef        `json:"who"`
    When     string          `json:"when"`       // ISO 8601
    What     ResourceAction  `json:"what"`
    Why      string          `json:"why"`
    Result   ActionResult   `json:"result"`
    RefID    string          `json:"ref_id,omitempty"`
    Approval *ApprovalInfo  `json:"approval,omitempty"`
}

type ActorRef struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Role string `json:"role"`
}

type ResourceAction struct {
    Resource string      `json:"resource"`
    ID       string      `json:"id"`
    Action   string      `json:"action"`
    Before   interface{} `json:"before,omitempty"`
    After    interface{} `json:"after,omitempty"`
    Summary  string      `json:"summary"`
}

type ActionResult struct {
    Status      string `json:"status"`       // "success" | "failure"
    ErrorCode   string `json:"error_code,omitempty"`
    ErrorDetail string `json:"error_detail,omitempty"`
}

type ApprovalInfo struct {
    Decision          string      `json:"decision"`           // "approve" | "reject" | "approve_with_changes"
    DecisionContext   interface{} `json:"decision_context"`
    Modifications     interface{} `json:"modifications,omitempty"`
    PolicyViolations  []string    `json:"policy_violations,omitempty"`
    ExpiresAt         string      `json:"expires_at,omitempty"`
}
```
