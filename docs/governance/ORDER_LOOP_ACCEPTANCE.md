# Order Loop Acceptance

**Version:** 1.0
**Last updated:** 2026-07-06
**Owner:** QA Architect
**Governance reference:** PLATFORM_CONSTITUTION.md, KERNEL_CONTRACTS.md, statemachine.md

---

## 1. Purpose

Verify that the complete order lifecycle end-to-end operates correctly from import through fulfillment, settlement, exception detection, Agent recommendation, and Owner handling. The acceptance covers the domain modules, state machine transitions, cost snapshotting, profit calculation, exception detection, and closed-loop action.

## 2. Flow Overview

```
order import → inventory check / logistics assignment → shipping cost snapshot
→ settlement / profit check → exception detection → Agent recommendation → Owner handling
```

### Modules Involved

| Step | Module(s) | AI Agent(s) |
|------|-----------|-------------|
| Order import | `domain/order/`, `domain/integrations/` | — (platform sync) |
| Inventory check | `domain/inventory/` | A5 (Stock Alert) |
| Logistics assignment | `domain/logistics/` | A10 (Logistics Intelligence) |
| Shipping cost snapshot | `domain/logistics/`, `domain/finance/` | A10 |
| Settlement / Profit check | `domain/settlement/`, `domain/finance/`, `domain/exchangerate/` | A6 (Profit Watch), G3 (Discount Risk) |
| Exception detection | `domain/exceptions/`, `agentos/` | G0 (System Health) |
| Agent recommendation | `domain/decision/`, `domain/actionpolicy/` | A6, G3, A2 |
| Owner handling | `domain/approval/`, UI dashboard | Owner |

## 3. Order State Machine Transitions

### 3.1 Canonical Order States

```
pending → confirmed → shipped → delivered → completed
  ↓          ↓          ↓
cancelled  cancelled  cancelled
```

| Transition | Description | Guard Conditions |
|------------|-------------|------------------|
| pending → confirmed | Order verified, payment confirmed | TotalAmount > 0, payment received |
| pending → cancelled | Order cancelled before confirmation | No shipping cost incurred |
| confirmed → shipped | Goods dispatched | Tracking number present, inventory decremented |
| confirmed → cancelled | Order cancelled after confirmation | Restocking fee (if applicable), Owner approval required |
| shipped → delivered | Customer received goods | Delivery confirmation or tracking update |
| shipped → cancelled | In-transit cancellation | Carrier intercept success, full refund processing |
| delivered → completed | Settlement finalized | All fees calculated, no pending returns |
| completed → (terminal) | Final state | No transitions out |

### 3.2 Extended States (for exceptions)

```
              → refund_in_progress → refunded
delivered ---> return_requested ---> returned ---> restocked
              → dispute ---> dispute_resolved
```

| Extended Transition | Guard Conditions |
|--------------------|------------------|
| delivered → refund_in_progress | Owner approval required (money-impacting) |
| delivered → return_requested | Platform policy check, return window valid |
| refund_in_progress → refunded | Payment gateway processed |
| return_requested → returned | Carrier delivered to return warehouse |
| returned → restocked | Inventory system confirmed |
| delivered → dispute | Platform dispute filed |

### 3.3 State Machine Validation Acceptance

| Test Case | Transition | Expected Result |
|-----------|-----------|----------------|
| Normal flow | pending → confirmed → shipped → delivered → completed | All transitions accepted, timestamps recorded |
| Cancel before confirm | pending → cancelled | Accepted, no penalties |
| Cancel after confirm | confirmed → cancelled | Accepted, Owner approval enforced |
| Cancel after shipping | shipped → cancelled | Carrier intercept check run, approval required |
| Invalid transition | pending → delivered | Rejected with error message |
| Terminal state | completed → any | Rejected, terminal state |
| Double transition | confirmed → shipped → shipped | Rejected (idempotency check) |
| Exception flow | delivered → return_requested → returned → restocked | State machine accepts extended path |

**PASS criteria:** All valid transitions accepted. All invalid transitions rejected. Terminal states enforced. Edge cases (double transition, etc.) handled. State logs (`sales_order_status_log`) recorded for every transition.

---

## 4. Step-by-Step Acceptance

### 4.1 Order Import

| Field | Detail |
|-------|--------|
| Test data needed | 5 sample orders from 2 platforms (Shopee, Ozon): 1 with standard fields, 1 with missing tracking number, 1 with zero-amount item, 1 multi-item order (3 SKUs), 1 with international shipping address. Platform sync mocks configured with known order IDs. |
| Operation steps | 1. Platform sync adapter imports orders via scheduled task. 2. Manually create order via `POST /api/v1/order`. 3. Import order with duplicate order_no (same as step 2). 4. Verify PlatformID maps correctly. |
| API/UI/DB verification points | API: `GET /api/v1/order` lists imported orders. `GET /api/v1/order/{id}` returns full detail with items and status logs. DB: `sales_order` row has order_no unique, correct platform_id, status=`pending`. `sales_order_item` rows for each SKU. |
| Expected results | 5 orders imported. Duplicate order_no correctly detected (idempotency: 409 or upsert behavior documented). Missing tracking_number accepted (nullable). Multi-item order has 3 rows in order_items. |
| Audit evidence | `operation_log` records import with source (platform sync, manual). Platform sync log shows external platform response. |
| Failure impact | Order import failure means lost sales data. Duplicate import causes double fulfillment attempt. Wrong platform_id breaks settlement and fee calculations. |

**PASS criteria:** All orders import with correct fields. Duplicate detection works. Multi-item order creates correct item count. Missing optional fields accepted.

---

### 4.2 Inventory / Logistics Check

| Field | Detail |
|-------|--------|
| Test data needed | 5 imported orders from step 4.1. Inventory setup: 2 SKUs with sufficient stock, 1 SKU low stock (below reorder threshold), 1 SKU out of stock, 1 SKU not in inventory system. Logistics quotes pre-configured for destination countries. |
| Operation steps | 1. Order import triggers inventory availability check. 2. A5 Agent runs stock alert analysis on low-stock and out-of-stock SKUs. 3. A10 Agent assigns optimal logistics route. 4. System generates shipping cost estimate. 5. Test manual inventory adjustment: decrement a SKU's quantity and verify order status updated. |
| API/UI/DB verification points | API: `GET /api/v1/inventory/{sku_id}` returns current stock. `GET /api/v1/logistics/quote` returns routes. DB: `inventory` table reflects changes. Logistics quote records for each order. |
| Expected results | Sufficient stock orders: status stays `confirmed` or advances to `shipped`. Low stock: A5 generates stock alert. Out of stock: order flagged `pending` with blocking reason. Missing SKU: order flagged for admin review. |
| Audit evidence | Inventory decrement recorded in operation_log. Stock alert recommendation stored in agent_trace. Logistics quote selection logged. |
| Failure impact | Out-of-stock order ships without product → customer complaint. Incorrect inventory counts cause over-selling or dead stock. Wrong logistics route increases cost or delivery time. |

**PASS criteria:** Inventory check runs on order confirmation. Low stock triggers alert. Out of stock blocks fulfillment. Missing SKU flagged for admin. Logistics route assigned with cost estimate.

---

### 4.3 Shipping Cost Snapshot

| Field | Detail |
|-------|--------|
| Test data needed | 3 orders from step 4.2 with logistics routes assigned. Different carriers with different rates. |
| Operation steps | 1. When order transitions to `shipped`, verify shipping cost snapshot is created. 2. Compare snapshot cost with estimated cost (should be close, actual cost may differ). 3. Verify shipping_cost is stored in `sales_order` row. 4. Test partial fulfillment: split order ships in 2 batches, verify cumulative shipping cost. |
| API/UI/DB verification points | API: `GET /api/v1/order/{id}` includes shipping_fee and shipped_at. DB: `sales_order.shipping_fee` reflects actual cost. |
| Expected results | Shipping cost snapshotted at ship time. Actual cost within expected range of estimate. Partial fulfillment captures combined shipping cost. |
| Audit evidence | Shipping cost snapshot event (`logistics.shipping.confirmed`) logged. Carrier tracking number stored. |
| Failure impact | No shipping cost snapshot → profit calculation uses estimate or zero. Wrong cost skews settlement and profit reporting. Partial fulfillment missing combined cost → each batch appears as full shipping fee. |

**PASS criteria:** Shipping_fee populated at ship time. Actual cost differs from estimate by less than 20% (acceptable tolerance). Partial fulfillment aggregates costs correctly.

---

### 4.4 Settlement / Profit Check

| Field | Detail |
|-------|--------|
| Test data needed | Completed orders (status=completed or delivered) with all cost components: product_cost, shipping_fee, platform_fee, payment_fee, other_fee, total_amount, pay_amount. 1 order where platform_fee was incorrectly calculated (overcharge scenario). 1 order in multi-currency (CNY purchase cost, THB revenue). |
| Operation steps | 1. When order status reaches `completed`, verify profit calculation runs: Profit = pay_amount - product_cost - shipping_fee - platform_fee - payment_fee - other_fee. 2. Verify profit_margin = profit_amount / pay_amount. 3. For platform fee overcharge scenario: Agent A6 flags anomaly. 4. For multi-currency: verify exchange rate applied to convert product_cost to revenue currency. 5. Verify settlement records created for platform payout. |
| API/UI/DB verification points | API: `GET /api/v1/order/{id}` includes profit_amount and profit_margin. `POST /api/v1/decision` or equivalent Agent decision endpoint shows profit analysis. DB: `sales_order.profit_amount` and `sales_order.profit_margin` populated. `settlement` table (or equivalent) has records. |
| Expected results | Standard order: profit_amount matches manual calculation. Overcharge scenario: flagged with "unexpectedly low profit" alert. Multi-currency: exchange rate conversion matches source-of-truth rate at order time. |
| Audit evidence | Profit calculation event (`order.profit.calculated`) published. Settlement record shows platform payout details. Exchange rate snapshot at order time preserved. |
| Failure impact | Wrong profit calculation → incorrect business decisions. Unflagged platform fee overcharge → ongoing revenue leakage. Missing exchange rate conversion → currency mismatch in financials. |

**PASS criteria:** Profit_amount and profit_margin match manual verification. Overcharge detected. Multi-currency conversion correct. Settlement records created.

---

### 4.5 Exception Detection

| Field | Detail |
|-------|--------|
| Test data needed | 1 order with prolonged `pending` status (>72 hours). 1 order where inventory was never decremented. 1 order with carrier delivery delay (>48 hours after expected delivery). 1 order with platform fee >50% of pay_amount anomaly. 1 order with multiple failed fulfillment attempts. |
| Operation steps | 1. G0 system health Agent scans for order anomalies. 2. For prolonged pending: Agent generates "stuck order" alert. 3. For inventory not decremented: Agent generates "fulfillment gap" alert. 4. For delivery delay: Agent generates "at-risk customer satisfaction" alert. 5. For fee anomaly: Agent A6 generates "profit leak" alert. 6. For failed attempts: Agent generates "order exception" alert with retry recommendation. |
| API/UI/DB verification points | API: `GET /api/v1/exceptions` lists active exceptions with order_id, type, severity, recommendation. Agent dashboard shows exception count. DB: `exceptions` table (or equivalent) has rows with status=open or active. |
| Expected results | All 5 exceptions detected and classified. Severity classification correct (e.g., fee anomaly = high, prolonged pending = medium, delivery delay = low). Each exception has an actionable recommendation. |
| Audit evidence | Agent trace shows detection rules evaluated. Exception record contains order_id, order_no, exception type, severity, detected_at, and Agent recommendation. |
| Failure impact | Undetected stuck orders → missed fulfillment SLA. Undetected inventory gaps → customer receives no product but system shows shipped. Undetected fee anomaly → ongoing financial loss. |

**PASS criteria:** All 5 exception scenarios detected. Severity classification correct. Each exception has recommendation. Exception resolution tracked.

---

### 4.6 Agent Recommendation

| Field | Detail |
|-------|--------|
| Test data needed | Active exceptions from step 4.5. |
| Operation steps | 1. For stuck order (72h+ pending): Agent recommends "review vendor fulfillment or cancel order". 2. For inventory not decremented: Agent recommends "manual inventory adjustment, contact platform support". 3. For delivery delay: Agent recommends "contact carrier, offer customer compensation or replacement". 4. For fee anomaly: Agent recommends "audit platform fee calculation, file dispute if confirmed". 5. For failed fulfillment: Agent recommends "switch carrier, update tracking, notify customer". |
| API/UI/DB verification points | API: Exception detail includes recommendation field with action type, priority, expected outcome. UI: Dashboard shows recommendations with status: suggested/pending_approval/executing/completed. |
| Expected results | All recommendations actionable: "do X → results in Y". Priority ordering correct (revenue-impacting > customer-impacting > operational). High-risk recommendations flagged for Owner approval. |
| Audit evidence | Agent recommendation stored in exception record. Decision trace shows reasoning. Risk level assessment documented. |
| Failure impact | Vague recommendations waste Owner time. Wrong priority leads to costly delays. Missing approval flag allows autonomous execution of high-risk actions. |

**PASS criteria:** All exceptions have recommendations. Recommendations are specific, actionable, and include expected outcome. Priority scoring is correct. High-risk actions flagged for approval.

---

### 4.7 Owner Handling

| Field | Detail |
|-------|--------|
| Test data needed | Exception set from step 4.6. |
| Operation steps | 1. Owner views exception dashboard showing all active exceptions grouped by severity. 2. Owner reviews a low-severity exception: accepts Agent recommendation → exception resolves automatically. 3. Owner reviews a high-severity exception (e.g., fee anomaly > $1000): must approve action → approval request created → action executes on approval. 4. Owner dismisses a false-positive exception with reason → status changes to "dismissed". 5. Owner manually resolves an exception via direct action (e.g., contacts carrier). |
| API/UI/DB verification points | API: `POST /api/v1/exceptions/{id}/resolve` with action_taken field. `POST /api/v1/approval` for high-severity actions. UI: Exception status updates after Owner action. Dashboard sorting/grouping works. |
| Expected results | Accepted recommendation: exception status → resolved, timestamp recorded. Dismissed: exception status → dismissed with reason. High-severity: approval request created, action blocked until approved. Manual: status → resolved with note. |
| Audit evidence | `operation_log` records every Owner interaction with exception. Approval trail for high-severity actions. Exception resolution linked to action taken. |
| Failure impact | Owner cannot see exceptions → actions slip. Cannot accept recommendation → wasted automation benefit. Approval for high-risk actions bypassed → unauthorized financial action. |

**PASS criteria:** Exceptions display correctly grouped by severity. Accept recommendation resolves exception. High-severity actions require approval. Dismiss with reason updates status. Audit trail complete.

---

## 5. End-to-End Acceptance Test Matrix

| Test Case | Steps | Expected Outcome | PASS/FAIL |
|-----------|-------|-----------------|-----------|
| Happy path: single item | Import → Confirm → Ship → Deliver → Complete | State transitions valid, costs correct, profit matches, no exceptions | |
| Happy path: multi-item | Import with 3 SKUs → Ship complete → Deliver → Complete | All items processed, shipping cost aggregated, profit per SKU trackable | |
| Cancel before ship | Import → Confirm → Cancel | Order cancelled, inventory restored, no shipping cost incurred | |
| Out of stock block | Import → Stock unavailable → Blocked → Alert Agent | Order stays pending, stock alert triggered, Owner notified | |
| Platform fee anomaly | Import → Complete → Fee >50% | Exception detected, Agent recommends audit, Owner handles | |
| Multi-currency order | Import CNY cost + THB revenue → Complete | Exchange rate applied, profit correct | |
| Stuck order | Import → Pending 72h+ → Exception detected | Agent alert generated, Owner notified | |
| Duplicate order import | Import same order_no twice | 409 Conflict or idempotent merge with audit trail | |
| Delivery delay | Ship → 48h late → Exception detected | At-risk customer alert, compensation recommendation | |
| Partial fulfillment | Import → Ship batch 1 → Ship batch 2 → Deliver → Complete | Shipping cost aggregated, both batches recorded | |

---

## 6. Order Financial Accuracy Verification

| Cost Component | Source | Verification Method |
|---------------|--------|-------------------|
| product_cost | Purchase order or product cost table | Match against latest landed cost calculation |
| shipping_fee | Logistics quote or carrier API | Match against carrier invoice or tracking system |
| platform_fee | Platform settlement report | Match against platform payout statement |
| payment_fee | Payment gateway rate table | Match against payment gateway fees |
| other_fee | Manual input or customs | Documentation attached to order |
| pay_amount | Platform order sync | Match against platform order total minus discounts |
| profit_amount | Calculated: pay - all costs | Recalculate manually from above components |
| profit_margin | Calculated: profit / pay * 100 | Recalculate manually |

**Acceptance criteria:** Each financial field independently verifiable against its source. Discrepancy > 0.5% in any field requires investigation. All calculations double-entry: stored values match recomputed values from raw components.

---

## 7. Failure Impact Analysis

| Failure Point | Detection | Impact | Mitigation |
|--------------|-----------|--------|------------|
| Order import fails | Order count mismatch | Lost orders, missing data | Manual platform export fallback |
| State machine bypass | Direct DB update of status | State integrity loss | Enforce status update only through service layer |
| Wrong profit calc | Profit not matching manual calc | Financial reporting wrong | Lock profit fields after settlement |
| Exception detection dropped | Alert not triggered | Issues undiagnosed | Scheduled re-scan of open orders |
| Approval bypass for money actions | Unauthorized refund or fee | Financial loss | Double-check policy on every money mutation |

---

## 8. References

- PLATFORM_CONSTITUTION.md — risk levels for order, finance, exception handling
- KERNEL_CONTRACTS.md — EventBus (order.created, order.status.changed, order.profit.calculated), Command Dispatch, Audit
- OWNER_FIRST_PROTOCOL.md — Owner handling of order exceptions
- statemachine.md — order state machine definition and validation
- KNOWN_ISSUES.md — known defects affecting order processing
- ACCEPTANCE_MATRIX.md — cross-module order traceability
- `backend-go/internal/domain/order/model.go` — Order, OrderItem, OrderStatusLog schemas
- `backend-go/internal/domain/order/service.go` — order service with state machine validation
- `backend-go/internal/domain/inventory/` — inventory management
- `backend-go/internal/domain/logistics/` — logistics rate engine
- `backend-go/internal/domain/finance/` — profit calculation
- `backend-go/internal/domain/settlement/` — settlement records
- `backend-go/internal/domain/exceptions/` — exception detection
- `backend-go/internal/domain/approval/` — approval gates for high-risk order actions
- `backend-go/internal/domain/actionpolicy/` — policy evaluation for order actions
