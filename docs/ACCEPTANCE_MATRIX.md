# LingMirror Acceptance Matrix

Updated: 2026-07-06

This matrix defines the minimum evidence required before LingMirror can be
called business-verified or beta-accepted.

## Environment Matrix

| Environment | Purpose | May Prove |
|-------------|---------|-----------|
| Unit / mock | Fast logic and UI feedback checks | Dev Done only |
| SQLite test DB | Isolated backend tests where SQL is portable | Partial Test Green |
| PostgreSQL test DB | Real database behavior and migrations | Test Green |
| Dry-run integration | External action contract without write-back | Business Verified for read/suggestion flows |
| Sandbox integration | External test environment write-back | Business Verified for controlled write flows |
| Production | Real store/account write-back | Beta Accepted only after explicit Owner approval |

Reports must name the environment. A result that passes in mock or dry-run does
not prove production behavior.

## Permission Regression Matrix

Every permission-level change or RBAC addition must be tested against the full
role matrix below. Regression means: a change must not accidentally grant an
action to a role that previously could not perform it.

| Role | CAN Do | CANNOT Do | Must Audit |
|------|--------|-----------|------------|
| Anonymous | Access public pages (login, landing, health endpoint). | Access any authenticated route, view any business data, trigger any Agent action or write operation. | Attempted access to protected routes (401/403 responses) via audit middleware. |
| Viewer | Read permitted dashboards and reports (dashboard overview, product read views, order read views, agent activity log). | Write any data (create/update/delete), approve any action, change any configuration, trigger any Agent write action. | Read access to non-permitted dashboards (logged and blocked). |
| Operator | Execute low/medium-risk business actions (prepare products, run reports, trigger Agent suggestions). Read most business data. | Bypass approval for high-risk actions (price changes, inventory mutations, refunds, platform publishing, AI action execution). Change RBAC or system settings. | Every write action (actor, resource, before/after, timestamp). High-risk action attempt that is blocked must log the blocked event. |
| Owner | Approve or reject high-risk business actions. Configure Agent rules within permitted scope. View all business data including sensitive fields redacted by default. | Delegate approval authority to non-Owner roles. Bypass audit. Delete audit logs. Modify system-level RBAC or admin accounts. | Every approve/reject decision (action ID, approver, decision, timestamp, reason). Sensitive-field access must be logged. |
| Admin | Manage system settings, RBAC, user accounts, and platform integrations. Read full audit logs. | Bypass the high-risk action approval flow for price/inventory/order/refund/publishing operations. Disable audit logging. Grant roles without documented Owner decision. | All RBAC mutations (who, what role, what user, timestamp). Admin-level read access to sensitive fields must be logged. System setting changes must be logged. |

### Permission Regression Test Requirements

A change that touches role checks or permission logic **must** include tests
that verify:

1. **Anonymous:** authenticated route returns 401/403, unauthenticated path does not leak business data.
2. **Operator:** cannot approve high-risk actions, cannot modify RBAC, every write is audited.
3. **Owner:** can approve high-risk actions, cannot bypass audit, sensitive-field access is logged.
4. **Admin:** can manage users/roles but cannot bypass high-risk approval flow.

High-risk action tests must include at least `anonymous`, `operator`, and
`owner/admin` coverage.

### Permission Regression Verification Command

```bash
cd backend-go && go test ./internal/rbac/... -run TestRoleRegression
```

If no such test exists for the modified area, the PR author must add one.

## Role Matrix

| Role | Must Verify |
|------|-------------|
| Anonymous | Cannot access protected business routes. |
| Viewer | Can read permitted dashboards only. |
| Operator | Can prepare low/medium-risk work but cannot bypass approval. |
| Owner | Can approve/reject high-risk business actions. |
| Admin | Can manage settings/RBAC, still cannot bypass audited high-risk flow. |

High-risk action tests must include at least `anonymous`, `operator`, and
`owner/admin` coverage.

## Business Flow Matrix

| Flow | Minimum Evidence |
|------|------------------|
| Product loop | Candidate product created or imported; completeness checked; cost/logistics/platform fee/profit calculated; recommendation created; Owner approval recorded; listing task created; result review visible. |
| Order loop | Order exists; inventory/logistics choice visible; shipping cost snapshot exists; settlement/profit check visible; exception detection creates recommendation; Owner handling recorded. |
| Agent action loop | Agent creates or returns an action; action cannot execute before approval when required; approval binds server-side user; execute path records status and audit log. |
| Platform write loop | Dry-run proves no external write; sandbox proves write contract; production requires approval, external reference ID, failure visibility, and recovery note. |

## High-Risk Action Matrix

| Action Area | Required Gate Evidence |
|-------------|------------------------|
| Price changes | before/after values, approval required, audit log |
| Inventory changes | quantity before/after, approval required, audit log |
| Order state changes | previous/new state, approval required for irreversible changes |
| Refunds / money | approval required, audit log, failure/recovery note |
| Platform publishing | dry-run/sandbox/production mode, external reference ID |
| Platform inventory sync | mode, target SKU/listing, failure visibility |
| RBAC / account permissions | authenticated admin identity, audit log |
| AI action execute | action catalog validation, approval policy, guardrails, audit log |

## Runtime Matrix

| Area | Evidence |
|------|----------|
| EventBus | worker alive, publish/receive test, queue/backpressure visibility |
| Scheduler | recent tick, recent success, consecutive failure count |
| Agent cost | token usage, budget limit, retry limit |
| Audit | searchable log with actor, resource, result, redaction |
| Backup/restore | backup command success and restore drill evidence |
| Migrations | up succeeds, latest down/up rollback smoke succeeds on test DB |

## Acceptance Decision

Beta acceptance requires:

- all required automated checks PASS
- no OPEN known issue blocking the target scope
- product loop or order loop Business Verified, depending on release goal
- high-risk actions in scope verified through the matrix above
- Owner explicitly accepts remaining documented risks
