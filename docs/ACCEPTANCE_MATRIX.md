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
