# High-Risk Action Gate Acceptance

**Version:** 1.0
**Last updated:** 2026-07-06
**Owner:** QA Architect
**Governance reference:** PLATFORM_CONSTITUTION.md (Sections 5, 8), KERNEL_CONTRACTS.md (Sections 3, 4, 7, 8)

---

## 1. Purpose

Verify that every high-risk action type passes through the correct approval gate, audit trail, execution mode validation, and rollback/ recovery procedure before reaching production. High-risk actions are those that change prices, inventory, order state, money flows, platform publishing, permissions, or autonomous Agent execution.

## 2. High-Risk Action Catalog

| ID | Action Type | Risk Level | Approval Required | System Module |
|----|------------|------------|-------------------|---------------|
| HR-01 | Price change | High | Yes | `domain/decision/`, `domain/integrations/` |
| HR-02 | Inventory change | High | Yes | `domain/inventory/` |
| HR-03 | Order state change (cancel, refund) | High | Yes | `domain/order/`, `domain/aftersales/` |
| HR-04 | Refund / Money change | High | Yes | `domain/finance/`, `domain/settlement/` |
| HR-05 | Platform listing publish | High | Yes | `domain/listingtask/`, `domain/integrations/` |
| HR-06 | Platform inventory sync | High | Yes | `domain/integrations/` |
| HR-07 | RBAC / Permission change | High | Yes | `rbac/`, `auth/` |
| HR-08 | AI Agent action execute | High | Per policy | `ai/`, `agent/`, `command/` |

## 3. Cross-Cutting Acceptance Criteria

Every high-risk action must pass these 5 gates:

### 3.1 Approval Level

| Level | Who Approves | Turnaround | When Required |
|-------|-------------|------------|---------------|
| L1: System auto-approve | Policy engine | Instant | Actions within pre-set policy thresholds (e.g., price change < 5%, refund < $50, quantity < 10) |
| L2: Single Owner | Platform Owner | < 4 hours | Standard high-risk actions within policy boundaries |
| L3: Owner + confirmation | Platform Owner with second factor | < 24 hours | Actions exceeding policy boundaries, first-time actions, destructive operations |
| L4: Owner + documented justification | Platform Owner with written business case | < 48 hours | Platform-wide changes, credential resets, data deletion, bulk operations |

**Acceptance verification:** For each HR action, confirm the correct level is enforced. Confirm override paths are documented and auditable.

### 3.2 Audit Evidence Format

Every high-risk action execution must produce:

```
action_type:     string    // e.g., "price_update"
target_type:     string    // "sku", "order", "product", "listing", "user", "role"
target_id:       string    // entity identifier
before_state:    JSON     // snapshot before mutation
after_state:     JSON     // snapshot after mutation (or expected for rejected/dry-run)
actor:           string   // user_id or agent_id
approval_id:     int64    // link to approval_request
correlation_id:  string   // workflow trace
request_id:      string   // HTTP request ID
ip_address:      string   // for user-initiated actions
mode:            string   // "dry_run", "sandbox", "production"
status:          string   // "pending", "approved", "rejected", "executing", "completed", "failed", "rolled_back"
timestamp:       DateTime
```

**Acceptance verification:** For each HR action, confirm all audit fields are populated. Verify `before_state` and `after_state` accurately reflect the change.

### 3.3 Mode Validation

| Mode | Behavior | Acceptance Test |
|------|----------|-----------------|
| dry_run | Validate inputs and permissions. Check approval requirement. Never mutate. Return expected before/after comparison. | Run dry_run → verify no DB mutations, no external API calls. Verify response shows "would have changed X from Y to Z" |
| sandbox | Execute against test data or sandbox platform account. No production side effects. | Run sandbox → verify mutations only in test schema. Verify external platform receives sandbox credentials. |
| production | Full execution with guardrails. High-risk actions need approval. | Run production → verify approval checked before execution. Verify mutating calls go to production platform. |

### 3.4 Rollback Procedure

Each HR action must document:

- Is the action reversible? (Yes/No/Conditional)
- What is the rollback command or procedure?
- What is the rollback timeout (SLA)?
- Who can trigger the rollback?
- What audit record does the rollback produce?

**Acceptance verification:** For each HR action, perform the action, then execute the rollback procedure. Verify rollback restores original state. Verify rollback is audited.

### 3.5 Failure Recovery

| Failure Scenario | Expected Behavior |
|-----------------|------------------|
| Action blocked by policy | Clear error message with policy name and reason. Suggestion for resolution path. |
| Action approved but execution fails | Status → "failed". Approval remains valid for retry within expiry. Error message logged with failure detail. |
| Partial execution failure (batch) | Succeeded items committed. Failed items reported individually. No auto-retry without Owner confirmation. |
| External platform timeout | Action marked "executing" with timeout. Retry logic with exponential backoff (max 3 attempts). |
| Duplicate execution (idempotency miss) | Idempotency key check. If already executed, return existing result. |

---

## 4. Action-Specific Acceptance

### 4.1 HR-01: Price Changes

| Field | Detail |
|-------|--------|
| Examples | SKU sale price update, listing price change, bulk price adjustment, promotional pricing |
| Required approval | L1 (auto-approve) if change ≤ 5% of current price AND confidence ≥ 0.9. L2 otherwise. L3 for first-time price change on a product. |
| Dry-run test | Supply new_price=9.99, old_price=12.99. Verify dry_run response shows "would change price from 12.99 to 9.99 (-23%)". Verify no DB change and no platform API call. |
| Sandbox test | Execute against Shopee sandbox store. Verify test listing price reflects new price. Verify no production listing affected. |
| Production test | Execute with approved request. Verify DB price updated. Verify platform API confirms price change. Verify operation_log contains before/after. |
| Rollback | Reversible: execute reverse price change to original price. Rollback must also go through approval unless within grace period (< 1 hour auto-rollback). Rollback must be audited. |
| Failure recovery | If platform rejects price (e.g., below minimum allowed), action → "failed" with platform error message. Retry with adjusted price requires new approval. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Single SKU small change (+3%) | price: $10 → $10.30 | Auto-approved, executed, audited |
| Single SKU large change (-20%) | price: $10 → $8.00 | Approval required, executed after approve |
| Bulk 50 SKU price change | CSV with 50 rows | Each item individually audited, partial failure allowed |
| Price below platform minimum | price: $10 → $1.00 | Platform rejects, action failed, approval consumed |
| Price change on unapproved listing | Listing status = blocked | Blocked by status precondition, clear error message |
| Rollback within grace period | Undo price change within 1 hour | Auto-approve rollback, audit shows "rolled_back" |

---

### 4.2 HR-02: Inventory Changes

| Field | Detail |
|-------|--------|
| Examples | Stock level adjustment, bulk inventory import, warehouse transfer, stock correction |
| Required approval | L1 if decrease < 10 units AND absolute value < $500. L2 otherwise. L3 for bulk > 1000 units. |
| Dry-run test | Supply inventory adjustment (-50 units). Verify dry_run shows "would decrement SKU-123 from 200 to 150". Verify no DB mutation. |
| Sandbox test | Execute against test warehouse. Verify sandbox inventory reflects change. |
| Production test | Execute approved change. Verify inventory table updated. Verify platform inventory sync triggered if integrated. |
| Rollback | Reversible: execute inverse adjustment (+50 units). Rollback must be audited. If platform already synced, platform inventory must also be re-synced. |
| Failure recovery | If adjustment would make inventory negative, block with error. If platform sync fails, log failure, retry with backoff, notify Owner if sync fails after 3 retries. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Correct stock decrease | Decrease by 50 from 200 | Inventory → 150, audited |
| Negative inventory prevention | Decrease by 999 from 5 | Blocked: "insufficient stock (5 < 999)" |
| Bulk import with mixed results | 200 SKUs: 198 success, 2 fail | Partial success, failed items listed |
| Zero adjustment | delta = 0 | Accepted or rejected? (document behavior) |
| Platform sync after adjustment | Inventory change → auto-sync | Platform API called, sync result recorded |

---

### 4.3 HR-03: Order State Changes

| Field | Detail |
|-------|--------|
| Examples | Cancel order, force ship, mark delivered, return processing, refund |
| Required approval | L2 for cancel. L3 for cancel after shipped (carrier intercept). L3 for refund. L4 for bulk cancellation (> 10 orders). |
| Dry-run test | Supply order_id and target status. Verify dry_run shows "would transition Order-123 from confirmed to cancelled". Verify state machine validation passes. Verify no DB mutation. |
| Sandbox test | Execute against test order. Verify state transition works. |
| Production test | Execute approved transition. Verify DB status updated. Verify status_log records from→to. Verify platform notified of cancellation. Verify inventory restored (if cancelled after inventory decrement). |
| Rollback | Cancel: not reversible (order state terminal). Refund: not reversible. Other transitions may be reversible if next state is not terminal. Document each. |
| Failure recovery | If platform rejects cancellation (e.g., order already shipped), action → "failed". Owner must contact platform support directly. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Standard cancel | pending → cancelled | Accepted, inventory restored, status_log recorded |
| Cancel after ship | shipped → cancelled | Owner L3 approval, carrier intercept checked |
| Refund full | delivered → refund_in_progress → refunded | Financial tx recorded, both states logged |
| State machine violation | pending → delivered | Blocked: "invalid transition" |
| Bulk cancel 15 orders | 15 order IDs | L4 approval required, each cancelled independently |

---

### 4.4 HR-04: Refunds / Money Changes

| Field | Detail |
|-------|--------|
| Examples | Full refund, partial refund, manual fee adjustment, settlement correction |
| Required approval | L2 for refund ≤ $100. L3 for refund > $100. L3 for fee adjustment. L4 for batch refunds. |
| Dry-run test | Supply refund amount. Verify dry_run shows totals. Verify no financial transaction. |
| Sandbox test | Execute against test payment gateway. Verify sandbox transaction recorded. |
| Production test | Execute approved refund. Verify payment gateway transaction. Verify DB: order profit recalculated, settlement record updated. Verify platform notified. |
| Rollback | Refund is not reversible. Correction requires separate transaction (recharge). Reversal must be documented as separate action. |
| Failure recovery | Payment gateway timeout → retry with backoff. Max 3 retries, then → "failed" with manual intervention required. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Partial refund | Order $100, refund $25 | Payment processed, profit recalculated, order status updated |
| Full refund | Order $50, refund $50 | Full reversal, profit → $0, status → refunded |
| Refund > pay_amount | Pay $100, request refund $150 | Blocked: "refund exceeds payment amount" |
| Batch refund 5 orders | 5 order IDs, total $500 | L4 approval, each refund processed individually |
| Fee adjustment correct | Platform fee $10 → $8 | Financial record updated, profit recalculated |

---

### 4.5 HR-05: Platform Publishing

| Field | Detail |
|-------|--------|
| Examples | Publish new listing, update listing content, activate/pause/end listing, bulk publish |
| Required approval | L2 for single listing publish. L3 for bulk publish (> 10 listings). L3 for first publish to new platform. |
| Dry-run test | Supply listing data. Verify dry_run sends validation-only to platform API. Verify no listing goes live. Verify response shows validation results. |
| Sandbox test | Execute against platform sandbox store. Verify listing created in sandbox. Verify no production listing affected. |
| Production test | Execute approved publish. Verify platform API returns success. Verify DB listing status → "active". Verify audit record with external listing ID. |
| Rollback | Reversible: platform unpublish/delist action. Rollback requires separate approval if outside grace period (< 1 hour). |
| Failure recovery | Platform validation error → action "failed" with specific error. Platform timeout → retry with backoff. Listing data mismatch → detailed diff in error message. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Publish single listing | Valid listing data → Shopee | Listing live on platform, status=active, external_id recorded |
| Publish duplicate listing | Same product same platform | Blocked: "listing already exists" or return existing listing |
| Bulk publish 15 listings | 15 products → Shopee | L3 approval, each published individually, partial failure allowed |
| Publish with invalid data | Missing required platform field | Platform rejection → "failed" with field-level error |
| Unpublish listing | Active listing → end | Listing removed from platform, status=ended |

---

### 4.6 HR-06: Platform Inventory Sync

| Field | Detail |
|-------|--------|
| Examples | Push inventory count to platform, sync platform stock changes to system, batch stock update |
| Required approval | L2 for single SKU sync. L3 for bulk sync > 100 SKUs. L3 for sync to multiple platforms. |
| Dry-run test | Supply SKU and new quantity. Verify dry_run shows "would sync inventory from 100 to 80 on Shopee". Verify no platform mutation. |
| Sandbox test | Execute against platform sandbox. Verify sandbox listing stock updated. |
| Production test | Execute approved sync. Verify platform API response. Verify DB sync_log records before/after. |
| Rollback | Reversible: sync original quantity back. Rollback may need approval if time-sensitive. |
| Failure recovery | Rate limit → retry after delay. Authentication failure → block, notify Owner. Mismatch between local and platform stock → warn if delta > 10%. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Single SKU sync | Stock 100 → 75 | Platform stock refreshed, log created |
| Bulk sync 200 SKUs | CSV, 200 entries | L3 approval, 200 individual calls, partial failure handled |
| Platform API rate limited | API returns 429 | Backoff + retry, max 3 attempts |
| Stock mismatch warning | Local 100, Platform 95, Push 85 | Execute sync, log includes "platform had 95, synced to 85" warning |
| Deactivated listing sync | Listing ended, attempt stock sync | Blocked: "listing not active on platform" |

---

### 4.7 HR-07: RBAC / Permission Changes

| Field | Detail |
|-------|--------|
| Examples | Add role, modify role permissions, grant admin access, revoke permissions, create user with elevated access |
| Required approval | L3 for any RBAC change. L4 for granting admin, owner, or super-admin roles. L4 for changes to approval policies. |
| Dry-run test | Supply user/role and permission delta. Verify dry_run shows "would grant 'edit_prices' to role 'merchant'". Verify no permission change. |
| Sandbox test | Execute in sandbox environment. Verify sandbox user gains permissions. |
| Production test | Execute approved change. Verify DB roles/permissions updated. RBAC middleware enforces new permissions immediately. |
| Rollback | Reversible: revert permission change. Rollback requires separate L3 approval. Permissions should have expiry where possible. |
| Failure recovery | Conflicting permission definition → blocked with explanation. Invalid role name → blocked. User not found → blocked. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Grant view-only access | User → role "viewer" | User can read but not mutate |
| Grant admin access | User → role "admin" | L3 approval, full permissions after approval |
| Revoke price permission | Role "merchant" → no "edit_prices" | Immediate enforcement, existing sessions affected? (document) |
| Create new role | Name "custom_role", 3 permissions | Role created, permissions validated |
| Modify approval policy | AutoApprove threshold $50 → $100 | L4 approval, policy rule updated |

---

### 4.8 HR-08: AI Agent Action Execute

| Field | Detail |
|-------|--------|
| Examples | Agent autonomously adjusting price, Agent canceling order, Agent publishing listing, Agent triggering inventory sync |
| Required approval | Per action policy rules (action_policy service evaluates). Default: L2 for any mutation action. L3 for first-time action type from an Agent. |
| Dry-run test | Agent generates action with mode=dry_run. Verify action passes through Gate: policy check → approval check → dry-run mode → validation result returned. Verify no execution. |
| Sandbox test | Agent generates action with mode=sandbox. Verify execution against sandbox data. |
| Production test | Agent action approved. DispatchSafe() or ExecuteAction() enforces mode, approval, audit. Verify side effect. |
| Rollback | Per action type (refer to HR-01 through HR-07 rollback procedures). |
| Failure recovery | Agent action fails → status "failed". Agent notified via event bus. Owner dashboard shows failed action. Retry requires new approval. |

**Test scenarios:**

| Scenario | Input | Expected Result |
|----------|-------|----------------|
| Agent recommends price drop (within threshold) | price $10-$9.50, discount 5% | Policy auto-approves (L1), action executes, audited |
| Agent recommends cancel order | order shipped → cancel | Policy requires L2, Owner approves, action executes, platform notified |
| Agent action confidence too low | confidence=0.6, threshold=0.85 | Action blocked: "confidence below threshold" |
| Agent action with expired approval | approval_id from 7 days ago, approval expired | Blocked: "approval expired, resubmit" |
| Agent attempts autonomous mutate without approval | no approval_id, mode=production | Blocked by DispatchSafe, action status "blocked" |
| First-time Agent mutation type | Agent A2 first time "unpublish" | L3 required, Owner explicitly approves new action type |

---

## 5. Execution Mode Acceptance Matrix

| Action Type | Dry-Run Support | Sandbox Support | Production Guardrails |
|-------------|----------------|----------------|----------------------|
| HR-01 Price change | Required | Recommended (if platform has sandbox) | Approval + audit + rollback procedure |
| HR-02 Inventory change | Required | Recommended | Approval + audit + platform sync guard |
| HR-03 Order state change | Required | Recommended (test order) | Approval + state machine validation + audit |
| HR-04 Refund / Money | Required | Required (test gateway) | Approval + financial transaction logging + audit |
| HR-05 Platform publish | Required | Required (platform sandbox) | Approval + audit + retry logic |
| HR-06 Inventory sync | Required | Required (platform sandbox) | Approval + audit + rate limiting |
| HR-07 RBAC / Permission | Required | Required (test environment) | Approval + audit + session re-evaluation |
| HR-08 AI Agent execute | Required | Required | Policy evaluation + approval + audit + termination capability |

---

## 6. Rollback Procedure Summary

| Action Type | Reversible? | Rollback Method | Rollback Approval | Rollback Timeout |
|-------------|------------|----------------|-------------------|------------------|
| Price change | Yes | Set price to original | Auto (within 1h grace) | < 10 minutes |
| Inventory change | Yes | Apply inverse delta | L1 (auto within delta) | < 5 minutes |
| Order cancel | No | N/A (create new order) | N/A | N/A |
| Order refund | No | N/A (issue recharge) | L2 | N/A |
| Platform publish | Yes | Delist/end listing | L2 (after 1h grace) | < 30 minutes |
| Inventory sync | Yes | Sync original quantity | L1 (auto within delta) | < 5 minutes |
| RBAC change | Yes | Revert permission set | L3 | < 15 minutes |
| AI Agent action | Per type | Per type rollback | Per type policy | Per type |

---

## 7. Acceptance Gate Sign-off

| Action Type | Dry-Run | Sandbox | Production | Rollback | Audit | Overall |
|-------------|---------|---------|------------|----------|-------|---------|
| HR-01 Price change | | | | | | |
| HR-02 Inventory change | | | | | | |
| HR-03 Order state change | | | | | | |
| HR-04 Refund / Money | | | | | | |
| HR-05 Platform publish | | | | | | |
| HR-06 Inventory sync | | | | | | |
| HR-07 RBAC / Permission | | | | | | |
| HR-08 AI Agent execute | | | | | | |

Date: ______________  Sign-off: ______________

---

## 8. References

- PLATFORM_CONSTITUTION.md — Section 5 (Agent Workflows), Section 8 (Risk Levels)
- KERNEL_CONTRACTS.md — Section 3 (Event Contract), Section 4 (Agent Action Contract), Section 7 (Approval Contract), Section 8 (Audit Contract)
- OWNER_FIRST_PROTOCOL.md — Section 4 (Default Safety Modes), Section 7 (Escalation Triggers)
- `backend-go/internal/domain/approval/model.go` — ApprovalRequest schema and status transitions
- `backend-go/internal/domain/actionpolicy/model.go` — PolicyRule, ActionContext, PolicyEvaluationResult
- `backend-go/internal/platform/command/action.go` — AgentAction struct, RiskLevel, ActionMode
- `backend-go/internal/platform/command/command.go` — DispatchSafe method
- `backend-go/internal/domain/integrations/` — PlatformAdapter with mode-aware execution
- `backend-go/internal/domain/order/service.go` — order state machine with guards
- `backend-go/internal/domain/inventory/` — inventory management with rollback support
- `backend-go/internal/rbac/` — role-based access control
- `backend-go/internal/ai/` — Agent action execution pipeline
- `docs/statemachine.md` — state machine validation framework
- KNOWN_ISSUES.md — known defects affecting high-risk actions
- ACCEPTANCE_MATRIX.md — cross-module acceptance traceability for high-risk paths
