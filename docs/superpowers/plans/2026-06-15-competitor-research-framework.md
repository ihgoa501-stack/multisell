> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Competitor Research Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a sourced competitor research package that tells LingMirror / MultiSell which ERP capabilities to copy, adapt, delay, or reject before building toward a Mabang-class cross-border ERP.

**Architecture:** Treat competitor research as an evidence pipeline, not a feature wish list. Each agent studies one competitor cluster, extracts operational patterns, maps them to LingMirror's current modules, and outputs decisions that feed the roadmap. The final synthesis must separate public marketing claims, observed workflow design, implementation implications, and concrete product recommendations.

**Tech Stack:** Markdown research docs, official competitor websites, public help centers, public pricing/docs pages where available, repository plans under `docs/superpowers/plans/`, existing FastAPI + Vue 3 product architecture as the implementation target.

---

## Operating Principle

Do not benchmark competitors by counting menus.

Benchmark them by answering:

```text
Which business problem does this feature solve?
Which data does it require?
Which workflow does it shorten?
Which cost or risk does it reduce?
Can LingMirror implement a simpler version now?
Would an AI agent make this workflow materially better?
```

The final research must produce product choices:

- `must_build`: Required to become credible as a cross-border ERP.
- `build_later`: Valuable, but dependent on missing data or integrations.
- `agent_advantage`: Existing ERP workflow that LingMirror can improve through AI agents.
- `do_not_copy`: Feature shape that creates bloat, manual work, or weak differentiation.

## Current LingMirror Context

Use these existing planning files as context before starting:

- `docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md`
- `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`
- `docs/superpowers/plans/2026-06-15-platform-fee-rules.md`
- `docs/superpowers/plans/2026-06-15-batch-prelisting-decision.md`
- `docs/superpowers/plans/2026-06-15-excel-batch-prelisting-decision.md`

Assume LingMirror already has or is planning:

- Product/SKU basics.
- Logistics attributes.
- Shipping rule calculation.
- Platform fee rules.
- Pre-listing decision.
- Batch and Excel decision workflow.
- Listing adapter boundary.
- Order/inventory baseline.
- RBAC and audit conventions.

The competitor research should not re-propose these as generic ideas. It should identify what must happen next to turn these modules into a serious ERP.

## Source Baseline

Use official sources first. Public marketing claims are acceptable when labeled as claims.

Initial official sources already identified:

- Mabang ERP: `https://www.mabangerp.com/`
  - Relevant claims: full process modules covering product development, order processing, supply chain, product management, inventory planning, logistics management, overseas warehouse, financial reports, customer service, refined operations.
  - Relevant logistics/TMS claims: 1200+ logistics providers, logistics API integration, real freight retrieval, automatic lowest freight matching, package tracking, automatic reconciliation, multiple logistics pricing systems.
- Dianxiaomi: `https://www.dianxiaomi.com/`
  - Relevant claims: lightweight multi-platform cross-border ERP, large platform/logistics/overseas warehouse ecosystem, order/logistics/warehouse/finance operations.
- Eccang: `https://www.eccang.com/`
  - Relevant claims: 60+ platforms, 1000+ logistics providers, 600+ overseas warehouses, FBA/first-leg workflows, cost/profit/loss records, financial data management.
- Tongtool: `https://www.tongtool.com/`
  - Relevant claims: cross-border ERP/WMS/logistics platform, order and inventory operations.
- SellFox / Saihu ERP: `https://www.sellfox.com/`
  - Relevant claims: Amazon and multi-platform refined operations, data analysis, advertising, finance accounting, supply chain, logistics comparison, order profit, platform-specific profit reports, FIFO order cost accounting, funds tracking.
- Lingxing ERP: `https://www.lingxing.com/`
  - Relevant claims: Amazon ERP, multi-platform ERP, overseas warehouse WMS, 40+ mainstream cross-border platforms, finance control resources.
- BigSeller: `https://www.bigseller.com/`
  - Relevant claims: free omnichannel ecommerce solution, 20+ Southeast Asia platforms, product/order/inventory/purchase/marketing/report modules, automation to reduce manual operation.
- 4Seller: `https://www.4seller.com/`
  - Relevant claims: multi-platform listing, order management, inventory management, Amazon MCF, shipping carriers, Amazon profit calculator, bulk editing.
- Sellercloud: `https://sellercloud.com/`
  - Relevant claims: catalog, inventory, warehouse, order, rule engine, purchasing, shipping, reporting, accounting, web service API.
- ShipStation: `https://www.shipstation.com/`
  - Relevant claims: ecommerce shipping, label and carrier workflow, fulfillment operations.

## Research Output Files

Create these files if executing the research:

- Create: `docs/research/competitor-research-2026-06.md`
  - Purpose: final synthesis and benchmark matrix.
- Create: `docs/research/competitor-sources-2026-06.md`
  - Purpose: source log with URLs, retrieval dates, and claim categories.
- Create: `docs/research/lingmirror-capability-decisions-2026-06.md`
  - Purpose: must build / build later / agent advantage / do not copy decisions.
- Modify: `docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md`
  - Purpose: add a short "Competitor Research Follow-up" section only after synthesis is complete.

If `docs/research/` does not exist, create it.

## Evidence Standard

Every competitor claim must be tagged:

```text
official_site        from the vendor website
official_help        from public help center/docs
official_pricing     from public pricing page
case_study           from vendor case/customer story
inference            derived from multiple sourced claims
unsourced            not accepted in final recommendations
```

Do not let `unsourced` claims influence roadmap priority.

If a source says "supports 1200+ logistics providers", the research may use it as evidence of ecosystem breadth. It must not claim verified integration quality unless a help doc or product workflow proves it.

## Benchmark Dimensions

Each competitor must be scored on these dimensions:

| Dimension | What To Study | Why It Matters To LingMirror |
| --- | --- | --- |
| Product/PIM | SKU library, variants, images, categories, attributes, bulk edit | Determines whether listing can scale |
| Listing | Draft, validation, publish, sync, errors, bulk operations | Connects decision results to revenue |
| OMS | Order intake, merge/split, intercept rules, fulfillment status | Core ERP operating spine |
| Shipping/TMS | Carrier quote, label, tracking, bill import, reconciliation | Turns shipping from estimate into controlled cost |
| WMS | Warehouse, bins, inbound, outbound, picking, packing, scanning | Required for stock accuracy and operational audit |
| Procurement/SCM | Suppliers, purchase orders, replenishment, lead time, supplier KPI | Converts sales signals into buying actions |
| Finance/Profit | Platform fees, settlement, freight actuals, ad cost, allocation, profit | Determines whether management trusts reports |
| BI | Dashboards, variance, drilldown, alerts | Converts raw data into decisions |
| Automation/AI | Rules, agents, anomaly detection, guided actions | LingMirror's differentiation area |
| Permissions/Audit | RBAC, operation logs, approval gates | Required for multi-user operations |

## Competitor Clusters

### Cluster A: Mabang / Eccang / Tongtool

Purpose:

Study full-process cross-border ERP depth.

Primary questions:

- How do they connect product development, listing, order, warehouse, shipping, finance, and reports?
- What logistics/TMS features are treated as core rather than optional?
- How do they present warehouse and overseas warehouse workflows?
- What finance/profit claims depend on actual bills, settlement, or cost allocation?

Expected output:

```markdown
## Cluster A Findings

### What LingMirror Must Learn
- ...

### What LingMirror Should Avoid
- ...

### Product Decisions
| Decision | Priority | Evidence | LingMirror Module | Implementation Implication |
| --- | --- | --- | --- | --- |
| ... | P0 | official_site | Shipping/TMS | ... |
```

### Cluster B: SellFox / Lingxing / CaptainBI / Jijia

Purpose:

Study Amazon and refined-operations depth.

Primary questions:

- How do they handle Amazon-specific finance, FBA, ad cost, profit, and replenishment?
- How do they present operational dashboards and drilldowns?
- Which functions are marketplace-specific and should not be generalized too early?
- Which workflows are repetitive enough for LingMirror agents to improve?

Expected output:

```markdown
## Cluster B Findings

### What LingMirror Must Learn
- ...

### What LingMirror Should Avoid
- ...

### Agent Advantage Candidates
| Workflow | Current ERP Pattern | Agent Upgrade | Required Data | Risk |
| --- | --- | --- | --- | --- |
| ... | ... | ... | ... | ... |
```

### Cluster C: Dianxiaomi / BigSeller / 4Seller

Purpose:

Study lightweight multi-platform user experience and SME adoption.

Primary questions:

- Which workflows are simple enough for small sellers?
- How do they reduce setup burden?
- What do they put on the first screen?
- Which batch operations and Excel flows matter most?
- Which features are intentionally shallow but useful?

Expected output:

```markdown
## Cluster C Findings

### What LingMirror Must Learn
- ...

### What LingMirror Should Avoid
- ...

### UX Decisions
| Workflow | Competitor Pattern | LingMirror Version | Reason |
| --- | --- | --- | --- |
| ... | ... | ... | ... |
```

### Cluster D: Sellercloud / ShipStation / Linnworks

Purpose:

Study mature international ecommerce operations, especially shipping, warehouse, order rules, purchasing, and APIs.

Primary questions:

- How do they structure catalog, order, warehouse, purchasing, shipping, accounting, and API boundaries?
- What can LingMirror learn from rule engines and integration ecosystems?
- Which international patterns differ from Chinese cross-border ERP patterns?
- Which features should be adapter-first rather than hardcoded?

Expected output:

```markdown
## Cluster D Findings

### What LingMirror Must Learn
- ...

### What LingMirror Should Avoid
- ...

### Architecture Decisions
| Capability | Competitor Pattern | LingMirror Boundary | First Implementable Slice |
| --- | --- | --- | --- |
| ... | ... | ... | ... |
```

## Final Matrix Format

The final synthesis in `docs/research/competitor-research-2026-06.md` must include this matrix:

| Competitor | Positioning | Strongest Modules | Freight/Cost Handling | Finance/Profit Handling | UX Pattern | LingMirror Should Copy | LingMirror Should Not Copy | Source Confidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Mabang | Full-process cross-border ERP | TMS/WMS/SCM/Finance | Quote, tracking, reconciliation, price systems | Multi-platform profit reports | Full operations suite | Cost layers, TMS reconciliation | Huge menu-first expansion | High |
| Eccang | Enterprise cross-border ERP | ERP/WMS/FBA/first-leg/finance | Logistics and overseas warehouse ecosystem | Cost/profit/loss records | Enterprise workflow | First-leg/FBA cost allocation | Enterprise bloat too early | Medium |
| Tongtool | Cross-border ERP/WMS/logistics | Order/inventory/logistics | Logistics platform emphasis | Needs deeper source validation | Traditional ERP | Module coverage map | Weakly sourced claims | Medium |
| SellFox | Amazon/multi-platform refined ops | Finance, ads, supply chain, logistics comparison | Logistics comparison and overseas warehouse integration | FIFO cost, platform fees, order profit | Data-driven dashboards | Profit drilldown and funds tracking | Marketplace-specific complexity too early | High |
| Lingxing | Amazon + multi-platform ERP | Amazon operations, WMS, finance control | Needs help-center validation | Finance control positioning | Refined ops | Amazon/FBA specialization when adapter exists | Copying Amazon-specific features before Amazon adapter | Medium |
| Dianxiaomi | Lightweight multi-platform ERP | Orders, logistics, warehouse, multi-platform | Large logistics ecosystem | Finance/report positioning | SME-friendly operations | Low-friction setup and batch ops | Treating free/light ERP as full target | Medium |
| BigSeller | Southeast Asia omnichannel ERP | Product/order/inventory/purchase/report | Platform and warehouse workflows | Report module | Simple multi-platform UX | Fast onboarding and operational simplicity | Overfitting to SEA marketplace assumptions | High |
| 4Seller | EU/US local multi-channel ERP | Listing/order/inventory/MCF/shipping | Shipping carriers and MCF | Amazon profit calculator | Lightweight central workspace | Listing + order + inventory simplicity | Feature-light profit model as final state | High |
| Sellercloud | Mature US multichannel ops | Catalog, inventory, WMS, order rules, purchasing, shipping, accounting, API | Shipping module and integrations | Accounting module | Enterprise operations | Rule engine and API boundaries | Heavy enterprise setup burden | High |
| ShipStation | Shipping/fulfillment platform | Labels, carriers, fulfillment | Shipping workflow depth | Not ERP-wide | Focused shipping UI | Label/tracking/reconciliation design cues | Treating shipping SaaS as full ERP | Medium |

Adjust `Source Confidence` only with evidence:

- `High`: official site plus help/docs or multiple product pages.
- `Medium`: official site only, with enough specific claims.
- `Low`: secondary sources or vague marketing.

## Decision Framework

After research, convert findings into product decisions using this rule:

```text
Build now if:
1. It improves cost/profit truth, OR
2. It closes the decision -> listing -> order loop, OR
3. It reduces repeated manual operations with low integration dependency, OR
4. It creates an agent workflow that existing ERP tools do not make easy.
```

Delay if:

```text
1. It requires real platform/carrier API access that is not selected yet.
2. It needs historical order/settlement volume we do not have.
3. It is mostly dashboard decoration over unreliable data.
4. It expands menus without closing an operational loop.
```

Reject if:

```text
1. It duplicates a competitor's menu without a clear LingMirror workflow.
2. It requires manual data entry that agents cannot reduce.
3. It makes profit reports look precise while using estimated-only costs.
4. It creates platform-specific complexity before the adapter exists.
```

## Recommended LingMirror Takeaways

These are the working hypotheses the research should test.

### Must Build

- Decision result to listing task queue.
- Freight bill import and reconciliation.
- Cost layers: estimated, snapshot, actual, allocated.
- Platform settlement import.
- Order true profit ledger.
- First-leg/FBA/overseas warehouse allocation.
- Shipping provider/channel versioning and effective dates.
- Listing publish queue with adapter-specific validation.
- Exception workbench for missing logistics data, failed listing, abnormal freight, negative profit.
- Permission and audit coverage for finance, shipping, listing, agent actions.

### Build Later

- Full PDA warehouse workflow.
- Wave picking.
- Supplier KPI.
- Advanced replenishment formulas.
- Advertising automation.
- Multi-language listing translation.
- Deep customer service automation.
- Full TMS label purchase across many carriers.
- Multi-country tax/VAT/EPR compliance.

### Agent Advantage

- Explain why a SKU is rejected before listing.
- Suggest missing logistics fields from similar SKUs.
- Choose between shipping channels with margin impact explanation.
- Convert approved batch decisions into listing tasks.
- Diagnose profit variance between estimate, snapshot, and actual bill.
- Summarize settlement anomalies.
- Recommend replenishment only after sales, inventory, lead time, and profit are aligned.
- Prepare operator task lists from exceptions.

### Do Not Copy

- Huge top-level navigation before operational loops work.
- Marketing dashboards that hide data quality.
- Platform-specific profit features before real adapter data exists.
- Manual-heavy workflows with no import, audit, or bulk path.
- "All platforms at once" adapter promises.
- Profit numbers without cost-layer labeling.

## Task 1: Create Research Directory And Source Log

**Files:**
- Create: `docs/research/competitor-sources-2026-06.md`

- [ ] **Step 1: Create `docs/research/competitor-sources-2026-06.md`**

Use this content:

```markdown
# Competitor Sources - 2026-06

## Rules

- Retrieval date: 2026-06-15.
- Prefer official vendor pages and public help docs.
- Tag every claim as `official_site`, `official_help`, `official_pricing`, `case_study`, or `inference`.
- Do not use `unsourced` claims in roadmap decisions.

## Sources

| Competitor | URL | Source Type | Claims To Verify |
| --- | --- | --- | --- |
| Mabang | https://www.mabangerp.com/ | official_site | Full-process ERP, logistics provider ecosystem, freight quote, tracking, reconciliation, TMS/WMS/SCM/finance modules |
| Dianxiaomi | https://www.dianxiaomi.com/ | official_site | Lightweight multi-platform ERP, platform/logistics/warehouse ecosystem, order/logistics/warehouse/finance workflows |
| Eccang | https://www.eccang.com/ | official_site | Multi-platform ERP, logistics/overseas warehouse ecosystem, FBA/first-leg workflow, cost/profit/loss records |
| Tongtool | https://www.tongtool.com/ | official_site | Cross-border ERP/WMS/logistics platform, order and inventory workflow |
| SellFox | https://www.sellfox.com/ | official_site | Amazon/multi-platform refined operations, finance accounting, logistics comparison, platform profit reports, FIFO order cost |
| Lingxing | https://www.lingxing.com/ | official_site | Amazon ERP, multi-platform ERP, overseas warehouse WMS, finance control positioning |
| BigSeller | https://www.bigseller.com/ | official_site | Omnichannel ecommerce solution, 20+ Southeast Asia platforms, product/order/inventory/purchase/report modules |
| 4Seller | https://www.4seller.com/ | official_site | Multi-platform listing, order management, inventory management, Amazon MCF, shipping carriers, Amazon profit calculator |
| Sellercloud | https://sellercloud.com/ | official_site | Catalog, inventory, warehouse, order rules, purchasing, shipping, reporting, accounting, API |
| ShipStation | https://www.shipstation.com/ | official_site | Shipping labels, carrier workflow, ecommerce fulfillment |

## Claim Log

| Competitor | Claim | Evidence Type | URL | Product Implication |
| --- | --- | --- | --- | --- |
| Mabang | Supports logistics API integration, real freight retrieval, lowest-cost matching, package tracking, and reconciliation. | official_site | https://www.mabangerp.com/ | LingMirror should build TMS around quote -> snapshot -> actual bill -> reconciliation. |
| SellFox | Presents finance accounting with platform fee details, FIFO cost accounting, order cost, funds tracking, and multi-platform profit reports. | official_site | https://www.sellfox.com/ | LingMirror should label profit by cost layer and build order true-profit ledger after settlement import. |
| Eccang | Presents first-leg/FBA shipment workflow and cost/profit/loss records. | official_site | https://www.eccang.com/ | LingMirror should add first-leg/FBA allocation after freight reconciliation. |
| BigSeller | Positions itself as a low-friction omnichannel solution with product, order, inventory, purchase, report modules. | official_site | https://www.bigseller.com/ | LingMirror should keep SME workflows simple and batch-friendly. |
| Sellercloud | Exposes catalog, inventory, warehouse, order rule engine, purchasing, shipping, reporting, accounting, and API as separate product capabilities. | official_site | https://sellercloud.com/ | LingMirror should preserve module boundaries and adapter-first architecture. |
```

- [ ] **Step 2: Commit source log**

Run:

```bash
git add docs/research/competitor-sources-2026-06.md
git commit -m "docs: add competitor source log"
```

Expected result:

```text
[branch ...] docs: add competitor source log
```

## Task 2: Assign Cluster Research

**Files:**
- Create: `docs/research/competitor-research-2026-06.md`

- [ ] **Step 1: Create the research synthesis shell**

Use this content:

```markdown
# Competitor Research - 2026-06

## Executive Summary

LingMirror should benchmark Mabang for business coverage, SellFox and Lingxing for refined Amazon finance/operations, Dianxiaomi/BigSeller/4Seller for low-friction multi-platform experience, and Sellercloud/ShipStation for mature international architecture patterns.

The product direction is not to clone any one ERP. LingMirror should build the same operating graph as mature ERPs while differentiating through AI-assisted decisions, exception diagnosis, and auditable agent execution.

## Cluster A: Mabang / Eccang / Tongtool

### Findings

| Finding | Evidence Type | Source | LingMirror Interpretation |
| --- | --- | --- | --- |
| Full-process ERP is framed as product -> order -> supply chain -> inventory -> logistics -> finance. | official_site | https://www.mabangerp.com/ | LingMirror roadmap should close operational loops instead of adding isolated pages. |
| TMS strength is freight quote, tracking, bill generation, and reconciliation. | official_site | https://www.mabangerp.com/ | Shipping estimate alone is not enough; actual bill reconciliation is a P0 path. |
| Enterprise ERP highlights FBA/first-leg and overseas warehouse resources. | official_site | https://www.eccang.com/ | First-leg/FBA allocation belongs after cost ledger foundations. |

### Product Decisions

| Decision | Priority | LingMirror Module | Implementation Implication |
| --- | --- | --- | --- |
| Add freight bill import and reconciliation. | P0 | Shipping/TMS + Finance | Compare carrier bill rows to order shipping snapshots and produce variance records. |
| Add cost layer labels everywhere profit appears. | P0 | Finance + Decision + BI | Every margin must say estimated, snapshot, actual, or allocated. |
| Delay full WMS/PDA. | P2 | WMS | Start with warehouse/bin/inbound/outbound models before scanning workflows. |

## Cluster B: SellFox / Lingxing / CaptainBI / Jijia

### Findings

| Finding | Evidence Type | Source | LingMirror Interpretation |
| --- | --- | --- | --- |
| Refined Amazon ERP emphasizes finance accounting, platform fees, FIFO cost, order profit, funds tracking, and advertising. | official_site | https://www.sellfox.com/ | LingMirror should build true-profit accounting before complex ad automation. |
| Multi-platform refined operations include platform-specific profit reports. | official_site | https://www.sellfox.com/ | Do not generalize marketplace finance too early; build adapter-specific settlement import. |
| Lingxing positions Amazon ERP, multi-platform ERP, and WMS as separate products. | official_site | https://www.lingxing.com/ | LingMirror should keep Amazon-specific features behind adapter boundaries. |

### Agent Advantage Candidates

| Workflow | Current ERP Pattern | Agent Upgrade | Required Data | Risk |
| --- | --- | --- | --- | --- |
| SKU rejection before listing | User checks fields and reports manually | Agent explains missing data, margin driver, and next action | SKU logistics, price, platform fee, shipping rate | Bad advice if costs are estimated-only |
| Profit variance | Finance compares reports manually | Agent explains estimate vs actual freight/platform/ad variance | Order snapshot, carrier bill, platform settlement | Needs strict source labels |
| Replenishment | ERP computes formula outputs | Agent summarizes why to buy or not buy | Sales velocity, stock, lead time, purchase cost, profit | Premature without real sales history |

## Cluster C: Dianxiaomi / BigSeller / 4Seller

### Findings

| Finding | Evidence Type | Source | LingMirror Interpretation |
| --- | --- | --- | --- |
| Lightweight ERPs sell simple multi-platform setup, order handling, inventory, purchase, and reports. | official_site | https://www.bigseller.com/ | LingMirror should keep MVP workflows importable, batchable, and easy to explain. |
| 4Seller centers listing, orders, inventory, Amazon MCF, shipping carriers, and bulk editing. | official_site | https://www.4seller.com/ | Listing task queue should support bulk actions and clear publish errors. |
| Dianxiaomi is useful as a low-friction benchmark, not the final ERP depth benchmark. | inference | https://www.dianxiaomi.com/ | Copy simplicity, not shallow cost accounting. |

### UX Decisions

| Workflow | Competitor Pattern | LingMirror Version | Reason |
| --- | --- | --- | --- |
| Batch pre-listing | Spreadsheet-friendly workflow | Excel import -> preview -> approve/reject/needs_data -> export | Cross-border users already operate in spreadsheets. |
| Listing operations | Bulk listing and sync | Approved decision -> listing task queue -> adapter validation | Reduces copy-paste while preserving control. |
| Errors | Operational exception list | Exception workbench with owner, reason, next action | Better than hiding failures in reports. |

## Cluster D: Sellercloud / ShipStation / Linnworks

### Findings

| Finding | Evidence Type | Source | LingMirror Interpretation |
| --- | --- | --- | --- |
| Mature international systems separate catalog, inventory, warehouse, order, purchasing, shipping, reporting, accounting, and API. | official_site | https://sellercloud.com/ | LingMirror module boundaries are directionally right. |
| Rule engines are core to order workflow automation. | official_site | https://sellercloud.com/ | LingMirror should add rules after stable order and shipping data exist. |
| Shipping-focused SaaS should inform label/tracking UX, not the whole ERP model. | official_site | https://www.shipstation.com/ | Build carrier adapters behind TMS, not as scattered shipping UI. |

### Architecture Decisions

| Capability | Competitor Pattern | LingMirror Boundary | First Implementable Slice |
| --- | --- | --- | --- |
| Order rules | Custom order automation | `order_rules` service later | Intercept abnormal destination/country/channel before shipment |
| Shipping labels | Carrier label workflow | `shipping.adapters` | Mock label adapter, then one real carrier |
| Accounting | Financial module | `finance` ledger | Settlement import and freight bill reconciliation |

## Benchmark Matrix

| Competitor | Positioning | Strongest Modules | Freight/Cost Handling | Finance/Profit Handling | UX Pattern | LingMirror Should Copy | LingMirror Should Not Copy | Source Confidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Mabang | Full-process cross-border ERP | TMS/WMS/SCM/Finance | Quote, tracking, reconciliation, price systems | Multi-platform profit reports | Full operations suite | Cost layers, TMS reconciliation | Huge menu-first expansion | High |
| Eccang | Enterprise cross-border ERP | ERP/WMS/FBA/first-leg/finance | Logistics and overseas warehouse ecosystem | Cost/profit/loss records | Enterprise workflow | First-leg/FBA cost allocation | Enterprise bloat too early | Medium |
| Tongtool | Cross-border ERP/WMS/logistics | Order/inventory/logistics | Logistics platform emphasis | Needs deeper source validation | Traditional ERP | Module coverage map | Weakly sourced claims | Medium |
| SellFox | Amazon/multi-platform refined ops | Finance, ads, supply chain, logistics comparison | Logistics comparison and overseas warehouse integration | FIFO cost, platform fees, order profit | Data-driven dashboards | Profit drilldown and funds tracking | Marketplace-specific complexity too early | High |
| Lingxing | Amazon + multi-platform ERP | Amazon operations, WMS, finance control | Needs help-center validation | Finance control positioning | Refined ops | Amazon/FBA specialization when adapter exists | Copying Amazon-specific features before Amazon adapter | Medium |
| Dianxiaomi | Lightweight multi-platform ERP | Orders, logistics, warehouse, multi-platform | Large logistics ecosystem | Finance/report positioning | SME-friendly operations | Low-friction setup and batch ops | Treating free/light ERP as full target | Medium |
| BigSeller | Southeast Asia omnichannel ERP | Product/order/inventory/purchase/report | Platform and warehouse workflows | Report module | Simple multi-platform UX | Fast onboarding and operational simplicity | Overfitting to SEA marketplace assumptions | High |
| 4Seller | EU/US local multi-channel ERP | Listing/order/inventory/MCF/shipping | Shipping carriers and MCF | Amazon profit calculator | Lightweight central workspace | Listing + order + inventory simplicity | Feature-light profit model as final state | High |
| Sellercloud | Mature US multichannel ops | Catalog, inventory, WMS, order rules, purchasing, shipping, accounting, API | Shipping module and integrations | Accounting module | Enterprise operations | Rule engine and API boundaries | Heavy enterprise setup burden | High |
| ShipStation | Shipping/fulfillment platform | Labels, carriers, fulfillment | Shipping workflow depth | Not ERP-wide | Focused shipping UI | Label/tracking/reconciliation design cues | Treating shipping SaaS as full ERP | Medium |
```

- [ ] **Step 2: Commit research shell**

Run:

```bash
git add docs/research/competitor-research-2026-06.md
git commit -m "docs: add competitor benchmark research"
```

Expected result:

```text
[branch ...] docs: add competitor benchmark research
```

## Task 3: Create Capability Decision Document

**Files:**
- Create: `docs/research/lingmirror-capability-decisions-2026-06.md`

- [ ] **Step 1: Create the decision document**

Use this content:

```markdown
# LingMirror Capability Decisions - 2026-06

## Decision Rule

Build now when a capability improves cost/profit truth, closes the decision-to-revenue loop, reduces repeated manual work, or creates an auditable agent advantage.

Delay when a capability requires unavailable platform/carrier data, historical volume, or settled operational foundations.

Reject when a capability copies competitor navigation without improving a LingMirror workflow.

## Must Build

| Capability | Why | First Slice | Dependent Plan |
| --- | --- | --- | --- |
| Decision to listing task | Converts approve decisions into action | Generate listing task from approved batch decision row | `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md` |
| Freight bill reconciliation | Moves shipping from estimate to truth | Import carrier bill CSV and compare to order shipping snapshot | New plan: `2026-06-15-shipping-bill-reconciliation.md` |
| Cost layer labels | Prevents fake precision | Add `cost_source_layer` to decision/profit responses | New plan: `2026-06-15-cost-layer-labeling.md` |
| Platform settlement import | Required for true profit | Import settlement CSV with fees/refunds/adjustments | New plan: `2026-06-15-platform-settlement-import.md` |
| Order true profit ledger | Management needs trusted profit | Join order revenue, product cost, platform fee, freight snapshot, actual bill | New plan: `2026-06-15-order-profit-ledger.md` |
| Exception workbench | Turns failures into operator tasks | Show missing logistics data, failed listing, freight variance, negative profit | New plan: `2026-06-15-exception-workbench.md` |

## Build Later

| Capability | Reason To Delay | Earliest Trigger |
| --- | --- | --- |
| Full PDA warehouse workflow | Needs warehouse model and physical process first | Warehouse/bin/inbound/outbound models are stable |
| Wave picking | Requires enough order volume | Daily warehouse order volume makes single-order picking inefficient |
| Advertising automation | Needs marketplace adapter and ad data | Amazon or TikTok adapter imports ad spend reliably |
| Supplier KPI | Needs purchase orders and receiving history | Purchase order workflow is used for real inbound stock |
| Advanced replenishment | Needs sales velocity and lead time history | True profit and inventory history exist for at least one marketplace |

## Agent Advantage

| Agent Workflow | Human Value | Required Data | Guardrail |
| --- | --- | --- | --- |
| Explain pre-listing rejection | Operator sees exact fix path | SKU, logistics, shipping quote, platform fee | Must show source cost layer |
| Suggest missing logistics fields | Reduces setup friction | Similar SKU dimensions/weight/category | Must require human confirmation |
| Diagnose freight variance | Finance sees why cost changed | Shipping snapshot, carrier bill, rate rule version | Must preserve original bill row |
| Create listing tasks from approved decisions | Removes copy-paste | Decision batch result, product/SKU, platform | Must require permission `listing:task:create` |
| Summarize settlement anomalies | Speeds finance review | Settlement rows, order records, fees | Must link to source row |

## Do Not Copy

| Pattern | Why Not | LingMirror Alternative |
| --- | --- | --- |
| Huge menu-first ERP expansion | Creates surface area before workflow truth | Build vertical loops with tests and audit |
| Profit dashboards on estimates only | Misleads users | Label estimated/snapshot/actual/allocated cost |
| All-platform adapter promise | Too broad and brittle | Pick one real platform after mock adapter proves boundary |
| Manual-only freight maintenance | Does not scale | CSV import first, API adapter later |
| Platform-specific finance before adapter | Creates fake generic abstractions | Implement settlement import per selected platform |

## Next Plan Sequence

1. `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`
2. `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md`
3. `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md`
4. `docs/superpowers/plans/2026-06-15-platform-settlement-import.md`
5. `docs/superpowers/plans/2026-06-15-order-profit-ledger.md`
6. `docs/superpowers/plans/2026-06-15-exception-workbench.md`
```

- [ ] **Step 2: Commit decisions**

Run:

```bash
git add docs/research/lingmirror-capability-decisions-2026-06.md
git commit -m "docs: capture lingmirror competitor decisions"
```

Expected result:

```text
[branch ...] docs: capture lingmirror competitor decisions
```

## Task 4: Update Mabang Roadmap With Research Follow-up

**Files:**
- Modify: `docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md`

- [ ] **Step 1: Append this section near the end of the roadmap**

Add this markdown:

```markdown
## Competitor Research Follow-up

Research package:

- `docs/research/competitor-sources-2026-06.md`
- `docs/research/competitor-research-2026-06.md`
- `docs/research/lingmirror-capability-decisions-2026-06.md`

Roadmap changes from the research:

1. Treat Mabang as the full-process benchmark, not the UI blueprint.
2. Treat SellFox and Lingxing as refined-operations references for Amazon finance, FBA, advertising, and replenishment.
3. Treat Dianxiaomi, BigSeller, and 4Seller as low-friction UX references for small and medium sellers.
4. Treat Sellercloud and ShipStation as architecture references for order rules, shipping, warehouse, purchasing, accounting, and APIs.
5. Prioritize cost truth and operational loops over broad menu coverage.

Immediate next implementation sequence:

1. Decision to listing task.
2. Freight bill reconciliation.
3. Cost layer labeling.
4. Platform settlement import.
5. Order true profit ledger.
6. Exception workbench.
```

- [ ] **Step 2: Commit roadmap update**

Run:

```bash
git add docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md
git commit -m "docs: connect competitor research to roadmap"
```

Expected result:

```text
[branch ...] docs: connect competitor research to roadmap
```

## Agent Prompts

Use only these agents if delegating locally: `claude`, `opencode`, `copilot`, `gemini`, `reasonix`.

### Prompt For Cluster A Agent

```text
你在 /Users/lc/multisell 工作。只做竞品调研，不改业务代码。

目标：调研马帮、易仓、通途，输出“全流程跨境 ERP 应该学什么、不学什么、凌镜下一步怎么做”。

必须阅读：
- docs/superpowers/plans/2026-06-15-competitor-research-framework.md
- docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md

必须使用官方来源优先：
- https://www.mabangerp.com/
- https://www.eccang.com/
- https://www.tongtool.com/

输出到：
- docs/research/cluster-a-full-process-erp.md

输出结构：
1. 证据表：竞品、URL、证据类型、原始主张、对凌镜的意义。
2. 能力矩阵：PIM、Listing、OMS、Shipping/TMS、WMS、SCM、Finance、BI、Audit。
3. must_build / build_later / do_not_copy。
4. 对 docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md 的修改建议，只写建议，不直接改。

约束：
- 不要写无来源结论。
- 不要把营销数字当成事实质量证明。
- 不要提出“全部平台一起做”。
- 最后给出 5 个最该落地的功能，按优先级排序。
```

### Prompt For Cluster B Agent

```text
你在 /Users/lc/multisell 工作。只做竞品调研，不改业务代码。

目标：调研赛狐、领星、船长BI、积加，输出“亚马逊精细化运营和财务利润模块应该怎么借鉴”。

必须阅读：
- docs/superpowers/plans/2026-06-15-competitor-research-framework.md
- docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md

优先使用官方来源：
- https://www.sellfox.com/
- https://www.lingxing.com/
- https://www.captainbi.com/
- https://www.jijia.com/

输出到：
- docs/research/cluster-b-amazon-refined-ops.md

输出结构：
1. 证据表：竞品、URL、证据类型、原始主张、对凌镜的意义。
2. 财务能力拆解：平台费、FBA、头程、广告费、退款、结算、订单利润、SKU利润。
3. Agent advantage：哪些人工分析适合让 AI agent 解释、诊断、生成任务。
4. must_build / build_later / do_not_copy。
5. 对凌镜“先做哪个真实平台 adapter”的建议。

约束：
- 亚马逊专属能力不得伪装成通用能力。
- 利润功能必须区分 estimated / snapshot / actual / allocated。
- 没有 settlement 或 bill 数据来源的利润报表只能标为估算。
```

### Prompt For Cluster C Agent

```text
你在 /Users/lc/multisell 工作。只做竞品调研，不改业务代码。

目标：调研店小秘、BigSeller、4Seller，输出“轻量多平台 ERP 的上手体验、批量操作、Excel/导入导出、订单和刊登体验应该怎么借鉴”。

必须阅读：
- docs/superpowers/plans/2026-06-15-competitor-research-framework.md
- docs/superpowers/plans/2026-06-15-excel-batch-prelisting-decision.md
- docs/superpowers/plans/2026-06-15-decision-to-listing-task.md

优先使用官方来源：
- https://www.dianxiaomi.com/
- https://www.bigseller.com/
- https://www.4seller.com/

输出到：
- docs/research/cluster-c-lightweight-multiplatform-ux.md

输出结构：
1. 证据表：竞品、URL、证据类型、原始主张、对凌镜的意义。
2. UX矩阵：首屏、批量操作、导入导出、异常处理、刊登、订单、库存、报表。
3. 凌镜当前 batch/excel decision 流程如何改得更像可用工具。
4. must_build / build_later / do_not_copy。
5. 三个可以一周内落地的小体验改进。

约束：
- 不要为了“像 ERP”增加复杂菜单。
- 优先给低成本、高频、可测试的体验建议。
- 每条建议都要能落到具体页面或 API。
```

### Prompt For Cluster D Agent

```text
你在 /Users/lc/multisell 工作。只做竞品调研，不改业务代码。

目标：调研 Sellercloud、ShipStation、Linnworks，输出“国际成熟电商运营系统在模块边界、规则引擎、物流、仓库、采购、会计、API 方面有什么值得凌镜借鉴”。

必须阅读：
- docs/superpowers/plans/2026-06-15-competitor-research-framework.md
- docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md

优先使用官方来源：
- https://sellercloud.com/
- https://www.shipstation.com/
- https://www.linnworks.com/

输出到：
- docs/research/cluster-d-international-architecture.md

输出结构：
1. 证据表：竞品、URL、证据类型、原始主张、对凌镜的意义。
2. 架构矩阵：catalog、inventory、warehouse、order rules、purchasing、shipping、reporting、accounting、API。
3. 哪些能力应该 adapter-first，哪些应该 core-first。
4. must_build / build_later / do_not_copy。
5. 对凌镜后端模块边界的建议。

约束：
- 不要把国际系统的企业复杂度直接搬进当前 MVP。
- 优先找可被 FastAPI service/router/schema 清晰承载的边界。
- 明确哪些建议依赖真实平台或 carrier API。
```

### Prompt For Synthesis Agent

```text
你在 /Users/lc/multisell 工作。只做文档综合，不改业务代码。

目标：把各 cluster 文档综合成最终竞品研究和凌镜能力决策。

必须阅读：
- docs/superpowers/plans/2026-06-15-competitor-research-framework.md
- docs/research/cluster-a-full-process-erp.md
- docs/research/cluster-b-amazon-refined-ops.md
- docs/research/cluster-c-lightweight-multiplatform-ux.md
- docs/research/cluster-d-international-architecture.md

输出：
- docs/research/competitor-research-2026-06.md
- docs/research/lingmirror-capability-decisions-2026-06.md

必须包含：
1. Executive summary。
2. Benchmark matrix。
3. must_build / build_later / agent_advantage / do_not_copy。
4. 下一阶段 6 个实施计划的顺序。
5. 每个结论至少关联一个来源或标记为 inference。

最终判断口径：
- 马帮是业务覆盖基准，不是 UI 蓝图。
- 赛狐/领星是亚马逊精细化参考，不是当前通用 ERP 蓝图。
- 店小秘/BigSeller/4Seller 是轻量体验参考，不是利润深度基准。
- Sellercloud/ShipStation 是架构和物流边界参考，不是当前 MVP 范围。
- 凌镜差异化必须落在 AI agent + 审批 + 审计 + 异常诊断。
```

## Verification

After executing all research tasks, run a placeholder scan across `docs/research` and this plan. The scan must not find empty planning markers, deferred-detail markers, or vague future-work markers.

`unsourced` may appear only in the evidence-standard explanation. It must not appear in final recommendation rows.

Run:

```bash
rg -n "https://www.mabangerp.com|https://www.sellfox.com|https://www.eccang.com|https://sellercloud.com" docs/research
```

Expected result:

```text
Each final research file contains official source URLs for its claims.
```

## Final Handoff

When research is complete, hand the next implementation agent this ordered build sequence:

1. Execute `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`.
2. Write and execute `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md`.
3. Write and execute `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md`.
4. Write and execute `docs/superpowers/plans/2026-06-15-platform-settlement-import.md`.
5. Write and execute `docs/superpowers/plans/2026-06-15-order-profit-ledger.md`.
6. Write and execute `docs/superpowers/plans/2026-06-15-exception-workbench.md`.

The strategic message to keep:

```text
LingMirror should not become a smaller Mabang.
It should become a cost-truth ERP with AI-assisted operations.
```
