# Owner Decision Log

**Version:** 1.0
**Last updated:** 2026-07-06
**Owner:** QA Architect
**Governance reference:** PLATFORM_CONSTITUTION.md (Section 12), OWNER_FIRST_PROTOCOL.md (Section 1)

---

## 1. Purpose

Track all Owner risk acceptance decisions with expiration dates. No decision is permanent. Every accepted risk has a defined expiry after which it lapses and must be re-evaluated. This log is the single source of truth for what risks the Owner has knowingly accepted and for how long.

## 2. Rules

1. **No permanent default acceptance.** Every acceptance has an expiry. The default expiry is 90 days. Explicitly state shorter or longer periods with justification.
2. **All acceptances expire.** Expired acceptances are treated as if they never existed. The system reverts to default behavior (blocked / require approval / alert).
3. **Renewal requires re-evaluation.** Expired decisions cannot be auto-renewed. The Owner must review current conditions and re-accept if still appropriate.
4. **Audit trail is mandatory.** Every decision entry is immutable once written. Corrections require a new entry superseding the old one.
5. **System must check expiry.** Automated processes that rely on an Owner acceptance must check whether the acceptance is still valid before acting.
6. **Acceptance scope must be narrow.** Decisions apply to a specific action, product, platform, or Agent. No wildcard acceptances ("accept all risk for all products").

## 3. Decision Log Entry Template

```yaml
decision_id:        "DEC-YYYYMMDD-NNN"
date:               YYYY-MM-DD
owner:              "[Owner name or identifier]"
risk_description:   "[One-line summary of the accepted risk]"
affected_scope:
  action_type:      "[price_change | inventory_change | order_cancel | refund | listing_publish | inventory_sync | permission_change | agent_autonomy | policy_override | data_migration]"
  target_type:      "[product | sku | order | listing | platform | role | agent | module | global]"
  target_ids:       "[list of specific IDs or 'all' and describe boundaries]"
acceptance_period:
  start_date:       YYYY-MM-DD
  expiry_date:      YYYY-MM-DD
  justification:    "[Why this period is appropriate]"
business_impact:    "[What happens if the accepted risk materializes]"
mitigation:         "[What measures are in place to reduce likelihood or impact]"
supersedes:         "[decision_id this replaces, if any]"
superseded_by:      "[filled when superseded]"
status:             "[active | expired | superseded | revoked]"
owner_signature:    "[Owner name or digital signature identifier]"
```

## 4. Common Decision Types

### 4.1 Price Change Threshold Override

Use when the Owner wishes to allow price changes exceeding the default 5% threshold for a specific product set or time window.

```yaml
decision_id:        "DEC-20260701-001"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Allow Agent A2 to adjust prices up to 15% without L2 approval for seasonal clearance products"
affected_scope:
  action_type:      "price_change"
  target_type:      "product"
  target_ids:       "all products in category 'seasonal_clearance' (category_id=42)"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-07-15
  justification:    "End-of-season clearance requires aggressive pricing. 5% threshold insufficient for 50%+ discount products. Manual approval creates unacceptable latency."
business_impact:    "Products may be sold below cost if Agent pricing logic is incorrect. Potential loss: category average $3/unit, estimated 200 units = $600 max exposure."
mitigation:         "Minimum profit floor set at -5%. Agent requires confidence >0.85. All changes still audited. Price alerts enabled for any change >30%."
supersedes:         null
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

### 4.2 Emergency Order Cancellation

Use when the Owner authorizes cancellation of orders that would normally require L3 approval.

```yaml
decision_id:        "DEC-20260701-002"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Authorize bulk cancellation of 23 orders from supplier X that failed quality inspection"
affected_scope:
  action_type:      "order_cancel"
  target_type:      "order"
  target_ids:       "ORDER-2026-07001 through ORDER-2026-07023 (supplier_id=7)"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-07-01
  justification:    "Supplier X QC failure detected. Orders must be cancelled within 24 hours to prevent shipment. Standard L3 approval per order would take >24h for 23 orders."
business_impact:    "Customer dissatisfaction from cancelled orders. Potential negative platform rating impact. Total order value: $3,450. Refunds required."
mitigation:         "All customers notified of cancellation with reason and compensation. Customer support manually handling escalations. Supplier relationship being reviewed."
supersedes:         null
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

### 4.3 Admin Permission Grant (Temporary)

Use when granting temporary elevated access for troubleshooting or emergency response.

```yaml
decision_id:        "DEC-20260701-003"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Grant temporary admin access to engineer for production database troubleshooting"
affected_scope:
  action_type:      "permission_change"
  target_type:      "role"
  target_ids:       "user_id=15 (engineer@example.com), grant role='emergency_dba'"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-07-02
  justification:    "Production incident: database replication lagging. Read-only access insufficient to diagnose. Grant expires in 24 hours automatically."
business_impact:    "Engineer has full DML access to production database. Potential for accidental data modification or data exfiltration."
mitigation:         "All queries logged. No DROP/TRUNCATE permissions granted. Read-replica available for SELECT. Changes reviewed after incident. Access auto-revoked by scheduled job."
supersedes:         null
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

### 4.4 Agent Autonomy Expansion

Use when the Owner allows a specific Agent to execute actions autonomously within defined boundaries.

```yaml
decision_id:        "DEC-20260701-004"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Allow Agent A5 (Stock Alert) to autonomously create replenishment purchase orders for low-stock SKUs under $500"
affected_scope:
  action_type:      "agent_autonomy"
  target_type:      "agent"
  target_ids:       "agent_id=A5"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-09-28
  justification:    "A5 has demonstrated 98% accuracy over 30-day trial period. Manual approval for routine low-stock replenishment adds 4-6 hour delay. $500 limit provides safety margin."
business_impact:    "Erroneous PO creates excess inventory or wrong items. Max exposure: $500 per PO, estimated 2-3 PO per week = $1,000-$1,500 weekly max."
mitigation:         "A5 reviews its POs in a daily digest sent to Owner. Any PO > $200 requires confirmation email. Owner can revoke autonomy instantly via dashboard. Auto-pause if A5 confidence < 0.9."
supersedes:         "DEC-20260601-005"
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

### 4.5 Data Migration Acceptance

Use when the Owner accepts risks associated with a database migration or data transformation.

```yaml
decision_id:        "DEC-20260701-005"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Accept risks of migration 000068 (add execution_mode column with default value)"
affected_scope:
  action_type:      "data_migration"
  target_type:      "module"
  target_ids:       "migration 000068, table: agent_action, column: execution_mode"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-07-08
  justification:    "Migration adds NOT NULL column with default 'production'. Existing rows get default. Rollback requires reverse migration. Standard migration risk: temporary table lock on large table (estimated < 1000 rows)."
business_impact:    "If migration fails mid-way, DB state is inconsistent. Rollback requires reverse script. Existing rows get default value which may not match their actual intent."
mitigation:         "Migration tested in staging. Transactional migration (single tx). Reverse migration prepared and tested. Business hours migration with on-call engineer."
supersedes:         null
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

### 4.6 Platform Policy Override

Use when the Owner overrides a platform policy that would normally block an action.

```yaml
decision_id:        "DEC-20260701-006"
date:               2026-07-01
owner:              "Platform Owner"
risk_description:   "Override product compliance policy to allow listing of product SKU-P789 (battery-powered device) without UN38.3 certificate"
affected_scope:
  action_type:      "policy_override"
  target_type:      "product"
  target_ids:       "sku_id=789, product_id=456 (portable charger)"
acceptance_period:
  start_date:       2026-07-01
  expiry_date:      2026-07-15
  justification:    "Certificate application in progress (ETA 14 days). Need listing live for pre-order campaign. Manual review confirms product meets UN38.3 requirements despite missing certificate."
business_impact:    "Listing without certificate violates platform policy. Potential delisting, warning, or account suspension. Customer notification if issues arise."
mitigation:         "Certificate ETA confirmed. Listing includes note 'certificate pending' in product description. Auto-delist if certificate not uploaded by expiry date. Single product only, not blanket override."
supersedes:         null
superseded_by:      null
status:             "active"
owner_signature:    "______"
```

## 5. Decision Log Entry Form (Blank)

```yaml
decision_id:        "DEC-YYYYMMDD-NNN"
date:               YYYY-MM-DD
owner:              ""
risk_description:   ""
affected_scope:
  action_type:      ""
  target_type:      ""
  target_ids:       ""
acceptance_period:
  start_date:       YYYY-MM-DD
  expiry_date:      YYYY-MM-DD (default: +90 days)
  justification:    ""
business_impact:    ""
mitigation:         ""
supersedes:         ""
superseded_by:      ""
status:             "active"
owner_signature:    ""
```

## 6. Decision Log Index

| Decision ID | Date | Summary | Type | Expiry | Status |
|------------|------|---------|------|--------|--------|
| | | | | | |
| | | | | | |
| | | | | | |

## 7. Expiry Calendar (Next 90 Days)

| Date | Decisions Expiring | Action Required |
|------|-------------------|-----------------|
| YYYY-MM-DD | DEC-*, DEC-* | Review and re-accept, let expire, or update |
| YYYY-MM-DD | DEC-* | Review and re-accept, let expire, or update |

## 8. System Integration

The decision log should be:

1. **Stored:** In the database (`owner_decision_log` table) for programmatic expiry checking. Automated systems must query this table before relying on an Owner acceptance.
2. **Displayed:** In the Owner dashboard as an "Active Risk Acceptances" section showing all active decisions with expiry countdown.
3. **Enforced:** A scheduled job runs daily to expire decisions past their expiry_date. Expired decisions trigger:
   - Notification to Owner: "Decision DEC-XXX has expired. Default policy is now in effect."
   - System reverts to default behavior for the affected scope.
4. **Audited:** Every decision create, supersede, and expiry is recorded in `operation_log`.

---

## 9. References

- PLATFORM_CONSTITUTION.md — Section 8 (Risk Levels), Section 12 (Owner Decision Boundary)
- OWNER_FIRST_PROTOCOL.md — Section 1 (Owner Role), Section 3 (Risk Language), Section 7 (Escalation Triggers)
- HIGH_RISK_ACTION_ACCEPTANCE.md — approval levels and action types
- ACCEPTANCE_MATRIX.md — traceability for accepted risks vs. acceptance tests
- RELEASE_READINESS_CHECKLIST.md — pre-release review of outstanding Owner decisions
- PRODUCTION_SERVER_INFO.md — production environment where decisions apply
- `backend-go/internal/domain/approval/` — approval system integration
- `backend-go/internal/domain/actionpolicy/` — policy engine that checks decision validity
