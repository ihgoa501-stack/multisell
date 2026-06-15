# LingMirror Capability Decisions - 2026-06

## Decision Rule

Build now when a capability:

1. Improves cost or profit truth.
2. Closes the decision-to-revenue loop.
3. Reduces repeated manual work with low integration dependency.
4. Creates an auditable AI-agent workflow.

Delay when a capability:

1. Requires platform/carrier APIs that are not selected.
2. Needs historical order, settlement, or bill volume that does not exist.
3. Produces dashboard value over unreliable data.
4. Expands menus without closing an operational loop.

Reject when a capability:

1. Copies competitor navigation without a LingMirror workflow.
2. Creates manual-heavy maintenance with no import or adapter path.
3. Shows precise profit while using estimated-only costs.
4. Generalizes marketplace-specific complexity before an adapter exists.

## Must Build

| Capability | Priority | Why | First Slice | Evidence Basis | Dependent Plan |
| --- | --- | --- | --- | --- | --- |
| Decision-to-listing task | P0 | Converts `approve` decisions into revenue-facing work | Create listing tasks from approved single/batch decision results | Mabang/Tongtool/4Seller listing workflows; inference from current LingMirror decision module | `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md` |
| Freight bill import and reconciliation | P0 | Moves shipping from estimate to actual cost | Import carrier bill CSV and match by tracking/order/channel | Mabang TMS reconciliation, Eccang finance automation, ShipStation shipping API model | New plan: `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md` |
| Cost-layer labeling | P0 | Prevents fake precision in margin and profit | Add `cost_layer` or equivalent field to decision/profit responses | Cross-competitor finance/profit claims; inference | New plan: `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md` |
| Platform settlement import | P0 | Required for true platform fees, refunds, adjustments, payouts | Import one platform settlement CSV into normalized rows | SellFox platform-specific finance, Eccang fee import/net profit | New plan: `docs/superpowers/plans/2026-06-15-platform-settlement-import.md` |
| Order true-profit ledger | P0 | Creates trusted profit by joining revenue and costs | Ledger rows for order revenue, product cost, platform fee, freight snapshot, freight actual, settlement adjustments | Mabang/SellFox/Eccang finance positioning | New plan: `docs/superpowers/plans/2026-06-15-order-profit-ledger.md` |
| Exception workbench | P1 | Turns failures into owned tasks | Show missing logistics data, failed listing, freight variance, negative profit, unmatched settlement | Eccang listing lifecycle exceptions, Dianxiaomi operation logs, CaptainBI diagnosis | New plan: `docs/superpowers/plans/2026-06-15-exception-workbench.md` |
| Agent action audit and approval | P1 | Makes AI execution safe in ERP operations | Add agent action log with source refs, before/after, permission, optional approval | CaptainBI AI diagnosis/execution pattern; LingMirror differentiation | New plan: `docs/superpowers/plans/2026-06-15-agent-action-audit.md` |
| First-leg/FBA allocation | P1 | Needed for SKU/order true profit | Allocate inbound/FBA shipment cost by quantity, weight, volume, or value | Eccang first-leg/FBA, Mabang Amazon FBA replenishment | New plan after ledger: `docs/superpowers/plans/2026-06-15-first-leg-fba-allocation.md` |

## Build Later

| Capability | Priority | Reason To Delay | Earliest Trigger |
| --- | --- | --- | --- |
| Full WMS/PDA | P2 | Needs warehouse/bin/inbound/outbound records first | Warehouse stock movement model is stable |
| Wave picking | P2 | Requires operational order volume | Daily warehouse order volume makes single-order picking inefficient |
| Supplier KPI | P2 | Requires purchase orders and receiving history | PO and inbound workflow are used in real operations |
| Advanced replenishment | P2 | Requires sales velocity, stock, lead time, and true profit | Order ledger and inventory history are reliable |
| Advertising automation | P2 | Requires ad data import and marketplace adapter | Amazon/TikTok/Walmart ad data imports reliably |
| Multi-language listing translation | P3 | Useful but not tied to cost truth | Listing task and platform attribute mapping are stable |
| Customer service automation | P3 | Outside current operational spine | Order/refund/customer message data exists |
| Multi-carrier live label purchase | P2 | Requires carrier selection and credentials | Mock label adapter and one carrier contract are ready |

## Agent Advantage

| Agent Workflow | Business Value | Required Data | Guardrail | First UI Surface |
| --- | --- | --- | --- | --- |
| Explain pre-listing rejection | Operator can fix SKU data faster | SKU, logistics fields, shipping quote, platform fee, target margin | Show source cost layer and formula | Decision result panel |
| Suggest missing logistics fields | Reduces SKU setup friction | Similar SKUs, category, dimensions, weight | Human confirmation required | Product logistics form |
| Convert approved decisions to listing tasks | Removes copy-paste | Decision result, product/SKU, target platform | Permission `listing:task:create` | Batch decision page |
| Diagnose freight variance | Finance sees why actual freight changed | Shipping snapshot, carrier bill row, rate rule version | Preserve source bill row and calculation | Shipping reconciliation page |
| Summarize settlement anomalies | Speeds finance review | Settlement rows, orders, fees, refunds | Link every claim to source row | Settlement import page |
| Recommend exception owner | Reduces operational drift | Exception type, source module, user roles | Audit assignment changes | Exception workbench |
| Draft replenishment recommendation | Converts data into buying action | Sales, stock, lead time, purchase cost, true profit | Approval required before PO | Future procurement page |

## Do Not Copy

| Pattern | Why Not | LingMirror Alternative |
| --- | --- | --- |
| Huge menu-first ERP expansion | Creates surface area before workflow truth | Build vertical loops with tests, audit, and source data |
| Profit dashboards on estimates only | Misleads users | Label estimated/snapshot/actual/allocated everywhere |
| All-platform adapter promise | Too broad and brittle | Pick one real platform after mock adapter proves boundary |
| Manual-only freight maintenance | Does not scale | CSV import first, API adapter later |
| Platform-specific finance before adapter | Creates fake generic abstractions | Implement settlement import for one selected platform |
| Black-box AI execution | Unsafe in finance, listing, and shipping workflows | Agent suggestions with approval, source refs, and audit |
| Full WMS before inventory truth | Expensive and hard to test | Warehouse/bin/stock movement foundations first |

## Recommended Product Spine

```text
Decision
  -> Listing Task
  -> Platform Adapter
  -> Order
  -> Shipping Snapshot
  -> Carrier Bill
  -> Platform Settlement
  -> Profit Ledger
  -> Exception Workbench
  -> Agent Diagnosis / Agent Task
```

This spine should drive implementation order until the first real platform and first real shipping reconciliation loop are working.

## Next Six Plans To Write Or Execute

1. Execute existing plan: `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`.
2. Write and execute: `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md`.
3. Write and execute: `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md`.
4. Write and execute: `docs/superpowers/plans/2026-06-15-platform-settlement-import.md`.
5. Write and execute: `docs/superpowers/plans/2026-06-15-order-profit-ledger.md`.
6. Write and execute: `docs/superpowers/plans/2026-06-15-exception-workbench.md`.

## Selected Platform Recommendation

For the first real platform adapter, choose based on data availability, not competitor prestige.

Recommended decision order:

1. If the team can provide Amazon settlement/order/listing exports soon, choose Amazon because competitor depth and profit/FBA patterns are strongest.
2. If the team has Temu/TikTok/Walmart data first, choose that platform and keep the adapter boundary platform-specific.
3. If no real platform credentials or exports are ready, continue with mock adapter plus CSV import workflows and do not build platform-specific UI.

## One-Sentence Strategy

LingMirror should benchmark Mabang for business coverage, SellFox/Eccang for finance depth, Dianxiaomi/BigSeller/4Seller for low-friction operations, Sellercloud/ShipStation/Linnworks for architecture, and CaptainBI for transparent AI diagnosis.
