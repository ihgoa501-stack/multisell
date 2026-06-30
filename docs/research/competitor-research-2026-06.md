# Competitor Research - 2026-06

## Executive Summary

LingMirror should use Mabang as the business-depth benchmark, not as a UI blueprint. The target is the same operating graph:

```text
product -> listing -> order -> warehouse -> shipping -> reconciliation -> profit -> replenishment -> automation
```

The strongest competitors show a consistent pattern:

- Mabang, Eccang, and Tongtool compete on full-process ERP depth.
- SellFox, Lingxing, CaptainBI, and Jijia compete on Amazon/refined-operations depth, especially finance, FBA, advertising, replenishment, and analysis.
- Dianxiaomi, BigSeller, and 4Seller compete on lightweight multi-platform adoption and fast operational workflows.
- Sellercloud, ShipStation, and Linnworks show mature international module boundaries for catalog, inventory, warehouse, order rules, shipping, reporting, accounting, and APIs.
- Luckee AI, 知行奇点, and RoxyBrowser AI Agent represent a new AI-native agent layer that sits atop or alongside existing ERPs, focusing on AI-driven decisions and automation rather than full ERP breadth.

LingMirror should not become a smaller Mabang. It should become a cost-truth ERP with AI-assisted operations:

- Every profit number must identify its cost layer: estimated, snapshot, actual, or allocated.
- Decision results must become executable listing tasks.
- Freight and platform fees must move from estimates to reconciled actuals.
- Exceptions must become operator or agent tasks.
- Agent actions must be permissioned, approved where needed, and auditable.

## Benchmark Matrix

| Competitor | Positioning | Strongest Modules | Freight/Cost Handling | Finance/Profit Handling | UX Pattern | LingMirror Should Copy | LingMirror Should Not Copy | Source Confidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Mabang | Full-process cross-border ERP | TMS, WMS, SCM, finance, Amazon edition | Logistics ecosystem, TMS tracking, bill generation, reconciliation, pricing systems | Multi-platform and Amazon profit reporting | Large full-suite ERP | Cost layers, TMS reconciliation, workflow closure | Huge menu-first expansion | High |
| Eccang | Enterprise cross-border ERP | Order, overseas warehouse, finance, first-leg/FBA, BI | Shipping-channel matching, overseas warehouse, first-leg/FBA shipment | Gross profit by product/order/staff/store/platform, fee import for net profit | Enterprise process control | First-leg/FBA allocation and finance automation | Heavy enterprise workflow before MVP data exists | High |
| Tongtool | Cross-border ERP/PaaS | ERP, Listing, WMS, logistics, supply-chain distribution, open API | 1000+ logistics/overseas warehouse positioning, logistics platform | Less explicit public finance detail | Modular product suite | Adapter-first module boundaries | Broad customization before product loop is stable | Medium |
| SellFox | Amazon and multi-platform refined operations | Finance, ads, OMS/WMS/SCM, logistics comparison | Real-time logistics comparison and overseas/FBA/local inventory sync | 200+ platform fees, FIFO order cost, order-to-remittance funds tracking, platform profit reports | Data-driven refined operations | Finance lineage, platform-specific profit, logistics comparison | Marketplace-specific complexity before adapter exists | High |
| Lingxing | Amazon + multi-platform ERP | Amazon ERP, multi-platform ERP, overseas warehouse WMS | Needs deeper help-doc validation | Finance-control positioning | Refined ops suite | Separate marketplace-specific product boundaries | Treating Amazon-specific workflows as generic core | Medium |
| CaptainBI | Amazon BI and AI operations | Sales, ads, finance/VAT, FBA inventory, AI diagnosis, intelligent replenishment | Warehouse/logistics appears in case and FBA context, not a TMS benchmark | Profit, tax, reports, AI diagnosis, advertising impact | Analytics and AI execution | Transparent diagnosis and role-based dashboards | Black-box one-click actions without audit | High |
| Jijia (积加) | Multi-platform full-channel ERP (Amazon -> 60+ platforms) | Operations, finance, supply chain, BI/data warehouse, open platform, ecommerce middle-office | 600+ third-party warehouses, 300+ third-party logistics; batch-cost tracking | Full-process FIFO cost; business/finance integration; multi-platform unified profit | Enterprise full-suite ERP | Business-finance integration, full-process batch cost, supply chain closed loop | Heavy enterprise feature set before product-market fit confirmed | High |
| Dianxiaomi | Lightweight multi-platform ERP | Listing, order, logistics, warehouse, purchase, reports, permissions | 1700+ logistics, overseas warehouses, tracking, label workflows | Profit, purchase, inventory, performance reports | SME-friendly operations | Low-friction setup, batch ops, operation logs | Stopping at lightweight finance depth | High |
| BigSeller | Southeast Asia omnichannel ERP | Product, order, inventory, purchase, report, sourcing | Third-party warehouse and order workflow positioning | Report module | Simple omnichannel workspace | Fast onboarding and operational simplicity | Overfitting to Southeast Asia platform assumptions | High |
| 4Seller | EU/US local multi-channel ERP | Listing, order, inventory, MCF, shipping carriers | Shipping carriers, Amazon MCF | Amazon profit calculator | Lightweight centralized workspace | Simple listing/order/inventory workflows and bulk edit | Treating calculator-style profit as final finance system | High |
| Sellercloud | Mature US multichannel operations | Catalog, inventory, WMS, order rules, purchasing, shipping, reporting, accounting, API | Shipping module and integrations | Accounting module | Enterprise operations | Rule engine, API, module boundaries | Heavy enterprise setup burden | High |
| ShipStation | Shipping/fulfillment platform | Labels, carriers, rate comparison, tracking, address validation, analytics APIs | Strong label/rate/tracking API model | Not ERP-wide | Focused shipping workflow | Carrier adapter and label/tracking UX | Treating shipping SaaS as full ERP | High |
| Linnworks | Multichannel operations platform | Order/shipping, inventory, warehouse, listings, analytics, integrations | Carrier labels, tracking, cost control | Analytics and SKU reporting | Automation-first operations | Order routing, bulk listing, central analytics | Automating before canonical data is reliable | High |
| Luckee AI | AI agent for Amazon SMB sellers | Review analysis, ad diagnosis, listing optimizer, operation assistant | N/A (Amazon-only agent layer) | Per-ASIN margin signals via review/ad data integration | Focused AI agent (no ERP) | Ad diagnosis workflow, review-to-action pipeline, agent-as-a-service UX | Treating AI analysis as full ERP replacement | High |
| 知行奇点 | Enterprise AI agent framework for cross-border brands | Product sourcing, content, ads, KOL, R&D agent workflows | N/A (agent framework, not ERP) | Brand-level profit signals across business lines | Operator Layer™ agent platform | Multi-agent orchestration, cross-border domain know-how, end-to-end business delivery | Building generic AI platform without domain depth | High |
| RoxyBrowser AI Agent | AI antidetect browser with built-in agent automation | Multi-account management, automated listing/fulfillment, social media matrix, data scraping | Natural-language-driven shipping automation | N/A (not financial) | AI-native antidetect browser | MCP-native browser automation, natural-language batch operations, zero-script automation | Treating browser automation as TMS/finance system | High |

## Cluster A: Mabang / Eccang / Tongtool

### What They Prove

Full-process ERP competitors do not treat logistics and finance as side panels. They are central to the business loop.

Mabang's public positioning connects products, listing, order, warehouse, logistics, TMS, SCM, and finance. Its TMS claims are especially important: tracking, in-transit feedback, bill generation, automatic reconciliation, and multiple logistics price systems. Eccang adds a finance-heavy view: automated reconciliation, gross-profit dimensions, fee import for net profit, overseas warehouse, first-leg/FBA shipment, and listing lifecycle monitoring. Tongtool reinforces the architecture direction with ERP, Listing, WMS, logistics platform, open API, and modular/microservice positioning.

### LingMirror Must Learn

- The ERP spine is not "product management plus reports". It is a closed operating loop.
- Shipping estimates are only the first layer. TMS credibility requires actual bill import and reconciliation.
- Warehouse depth matters, but WMS/PDA should follow data foundations.
- First-leg/FBA allocation is essential for true SKU profit.
- Open interfaces and adapter boundaries matter once real platform/carrier integration begins.

### LingMirror Should Avoid

- Building a large ERP navigation before a single end-to-end loop works.
- Treating marketing platform/logistics counts as proof of integration quality.
- Adding WMS scanning before warehouse/bin/inbound/outbound records are stable.
- Adding BI dashboards over estimated-only costs.

### Product Decisions

| Decision | Priority | Evidence | LingMirror Module | Implementation Implication |
| --- | --- | --- | --- | --- |
| Build freight bill import and reconciliation. | P0 | Mabang/Eccang official sites | Shipping/TMS + Finance | Compare carrier bill rows with order shipping snapshots and produce variance records. |
| Add cost-layer labels across decision and profit views. | P0 | Mabang/Eccang finance positioning, inference | Decision + Finance + BI | Every margin must show estimated, snapshot, actual, or allocated. |
| Add first-leg/FBA allocation after actual freight exists. | P1 | Eccang official site | Finance + Inventory/WMS | Allocate inbound shipment cost by SKU quantity, weight, volume, or value. |
| Delay full PDA/WMS. | P2 | Mabang/Tongtool WMS positioning | WMS | First build warehouse, bin, stock movement, inbound, outbound, and audit. |

## Cluster B: SellFox / Lingxing / CaptainBI / Jijia

### What They Prove

Amazon/refined-ops tools compete on analysis depth, finance traceability, FBA/replenishment, advertising, and operational diagnosis. SellFox is the clearest finance reference: platform fee details, FIFO order cost, funds tracking, platform-specific profit, logistics comparison, and inventory sync. CaptainBI is the strongest AI/BI reference: AI diagnosis, AI advertising, FBA inventory, intelligent replenishment, and many reports. Lingxing validates that Amazon ERP, multi-platform ERP, and overseas warehouse WMS are separate product boundaries.

**Jijia (积加)** fills the business-finance integration gap. Sourced from official site (jijiaerp.com): it is positioned as a multi-platform full-channel ERP upgraded from Amazon ERP, supporting 60+ mainstream platforms, 600+ third-party warehouses, and 300+ third-party logistics providers. Its strongest differentiator is full-process batch cost with 先进先出 (FIFO) and business-finance integration (业财集成), targeting billion-level enterprise sellers (90%+ of its customer base are 亿级卖家). Its product suite includes operations, finance, supply chain, management, BI/data warehouse, open platform, and ecommerce middle-office modules. Key solutions target 大件出海, 鞋服跨境, and 3C跨境 verticals. Jijia is the strongest reference for LingMirror's cost-layer architecture and business-finance integration ambition.

### LingMirror Must Learn

- Amazon/FBA-specific features should not be generalized prematurely.
- Profit needs data lineage: product cost, platform fee, freight, settlement, ad cost, refund, and allocation.
- AI value is strongest when it explains causes and proposes auditable next actions.
- Replenishment should wait until sales, inventory, lead time, and true profit are available.

### LingMirror Should Avoid

- Copying Amazon-specific flows before the first real Amazon adapter exists.
- Creating "AI one-click execution" without approval and audit.
- Showing precise-looking profit while using only estimated freight or manually entered fees.
- Building advertising automation before ad data import is reliable.

### Agent Advantage Candidates

| Workflow | Current ERP Pattern | LingMirror Agent Upgrade | Required Data | Guardrail |
| --- | --- | --- | --- | --- |
| Pre-listing rejection | User reads report and manually fixes fields | Explain missing data, profit driver, and next action | SKU, logistics attributes, shipping quote, platform fee, target margin | Must show cost layer and source rows |
| Freight variance | Finance manually compares estimate and bill | Diagnose estimate vs snapshot vs actual carrier bill variance | Order shipping snapshot, bill row, rate version | Must preserve original bill row and calculations |
| Settlement anomaly | Finance manually scans platform settlement rows | Summarize abnormal fee, refund, adjustment, or missing order | Settlement import, order records, platform fee rules | Must link to settlement source row |
| Replenishment | ERP formula suggests quantity | Explain buy/do-not-buy based on sales, stock, lead time, and profit | Sales history, inventory, purchase cost, lead time, true profit | Human approval required |
| Listing correction | Operator edits failed listings | Convert adapter validation errors into field-level fixes | Listing task, platform category/attribute rules | Do not auto-publish without permission |

## Cluster C: Dianxiaomi / BigSeller / 4Seller

### What They Prove

Lightweight multi-platform ERPs win adoption by minimizing setup friction. Dianxiaomi emphasizes many platforms/logistics providers, quick listing, automated order rules, warehouse, logistics tracking, profit reports, permissions, and operation logs. BigSeller focuses on a simple omnichannel workspace across product, order, inventory, purchase, marketing, report, and sourcing. 4Seller keeps the surface narrow: listing, order, inventory, MCF, shipping carriers, bulk edit, and a profit calculator.

### LingMirror Must Learn

- Excel and batch workflows are not a compromise; they are a primary operating mode for cross-border sellers.
- The first screen should show actionable work, not marketing copy.
- Bulk actions must include clear preview, validation, export, and recoverable errors.
- Operation logs and role permissions are adoption features, not only enterprise features.

### LingMirror Should Avoid

- Hiding complex finance limitations behind a simple UI.
- Adding low-value menu breadth before batch decision/listing/order workflows are usable.
- Overfitting to one regional platform cluster before adapter strategy is chosen.

### UX Decisions

| Workflow | Competitor Pattern | LingMirror Version | Reason |
| --- | --- | --- | --- |
| Batch pre-listing | Spreadsheet-friendly operations | Excel import -> preview -> decision -> export -> listing task | Matches seller habits and keeps errors inspectable. |
| Listing | Bulk listing and sync | Approved decision -> listing task queue -> adapter validation -> publish/retry | Connects decision to revenue without copy-paste. |
| Order | Automated rules and scanning | Later order rules after canonical order/shipping records | Automation needs reliable order state. |
| Errors | Operational lists and logs | Exception workbench with owner, reason, source, next action | Makes failures executable. |
| Permissions | Role controls and logs | RBAC + audit for finance/shipping/listing/agent actions | Required once agents perform operations. |

## Cluster D: Sellercloud / ShipStation / Linnworks

### What They Prove

Mature international systems separate modules cleanly and automate through rules/adapters. Sellercloud exposes catalog, inventory, warehouse, order rules, purchasing, shipping, reporting, accounting, and web service API. ShipStation is a focused shipping reference: carriers, labels, rate comparison, address validation, tracking, and analytics APIs. Linnworks reinforces order routing, shipping labels, tracking, bulk listings, inventory, warehouse, analytics, and integrations.

### LingMirror Must Learn

- Keep `core` modules and `adapter` modules separate.
- Build order rules after canonical order, inventory, and shipping state exist.
- Carrier integration should use a shipping adapter boundary: quote, label, tracking, bill.
- Accounting should be fed by source imports and ledger entries, not dashboard shortcuts.
- Analytics becomes valuable only when underlying records are stable.

### LingMirror Should Avoid

- Copying enterprise setup complexity.
- Hardcoding carrier/platform logic into service functions.
- Automating order routing before warehouse/shipping data is trustworthy.
- Treating API count as product value without workflow closure.

### Architecture Decisions

| Capability | Competitor Pattern | LingMirror Boundary | First Slice |
| --- | --- | --- | --- |
| Carrier labels | ShipStation carrier API model | `shipping.adapters` | Mock label adapter, then one real carrier |
| Rate comparison | ShipStation rate comparison, Mabang/SellFox logistics comparison | `shipping.quote` + `shipping.rate` | Compare rule-based quotes before live API |
| Order rules | Sellercloud order rule engine, Linnworks routing | `order_rules` later | Intercept abnormal destination/channel/inventory state |
| Accounting | Sellercloud accounting, SellFox finance lineage | `finance_ledger` | Import platform settlement and carrier bill rows |
| Analytics | Linnworks SKU analytics, CaptainBI reports | `bi` | Dashboards only after ledger facts exist |

## Cluster E: Luckee AI / 知行奇点 / RoxyBrowser AI Agent — AI-Native Agent Layer

### What They Prove

A new competitive frontier has emerged: AI-native agents that sit atop or alongside existing ERPs rather than replacing them. These players do not compete on ERP breadth (orders, inventory, warehouse, shipping) — they compete on AI-driven decision speed and execution automation. This cluster represents the most direct competitive threat to LingMirror's AI Agent strategy because they have no legacy ERP to maintain.

### Competitor Profiles

**Luckee AI** (luckee.ai) — Amazon-specific AI agent for SMB sellers and agencies. Covers 14 Amazon marketplaces (US/EU/JP) with focused agent services: Review Analysis (dual-source scraping, AI semantic clustering across 85-100% of written reviews, P0-P3 recommendations with evidence), Ad Diagnosis (8-phase SP advertising health check across campaigns/keywords/search terms, star/healthy/monitor/problematic ratings), Listing Optimizer, and Operation Assistant. Their value prop is clear: "from raw data to the next action, in minutes, not days." Serves agency partners managing 7-figure ad spend. Luckee validates that the narrow-AI-agent-for-Amazon model is viable — they don't try to be an ERP.

**知行奇点** (zhixingjidian.cn) — Positions as China's "Operator Layer" for cross-border ecommerce. Targets billion-level (百亿级) cross-border brands with a full agent framework covering product sourcing (选品), content (内容), advertising (广告), KOL/darling (达人), and R&D agent workflows. Their differentiation: they claim to be the only provider simultaneously possessing cross-border domain know-how, a full agent framework, and end-to-end business delivery capabilities. They operate as a "人+AI混合部门" (human + AI hybrid department) rather than a SaaS tool. This validates the multi-agent orchestration approach and the enterprise-level agent-as-a-service model.

**RoxyBrowser AI Agent** (roxybrowser.cn, launched Feb 2026) — Claims "world's first AI agent antidetect browser." Unlike traditional antidetect browsers that are passive fingerprint containers, RoxyBrowser embeds an AI agent capable of natural-language-driven batch operations across multiple account profiles. Uses MCP (Model Context Protocol) for native browser automation — no Python/Selenium scripts required. Target use cases include cross-border ecommerce full-chain automation (listing, fulfillment, account management), social media matrix management, and data scraping. This validates that the antidetect browser + AI agent combination is a viable distribution channel for cross-border automation, and that MCP-based tool integration is a compelling architecture pattern.

### LingMirror Must Learn

- AI-native agents don't need ERP breadth — they can focus on the decision layer and let ERPs handle operations. This is both a threat (they move faster) and an opportunity (LingMirror's agent layer can integrate with any ERP).
- Luckee shows that a narrow, deep Amazon agent (reviews + ads) is a commercially viable wedge — LingMirror should consider which single workflow provides the highest-ROI agent landing.
- 知行奇点's "Operator Layer" framing is LingMirror's direct competitive positioning. The multi-agent orchestration and domain-know-how combination is exactly what LingMirror's agentOS aims to deliver.
- RoxyBrowser proves that MCP-based agent integration with browser/infrastructure tools is a real architecture pattern, not just a theoretical standard.

### LingMirror Should Avoid

- Dismissing AI-native agents as "not real ERPs" — they compete for the same budget and mindshare.
- Building a generic agent platform without cross-border domain depth — 知行奇点's explicit differentiation is domain know-how, not generic AI capability.
- Ignoring the antidetect browser channel — RoxyBrowser's agent integration shows that infrastructure tools can become agent distribution platforms.

### Product Decisions for AI-Native Competition

| Decision | Priority | Evidence | LingMirror Module | Implementation Implication |
| --- | --- | --- | --- | --- |
| Pick one narrow Amazon agent workflow as entry wedge. | P1 | Luckee validates narrow-deep model | AgentOS (any agent) | Start with review diagnosis or ad diagnosis as a focused agent, prove value before expanding. |
| Double down on multi-agent orchestration vs narrow single-agent. | P0 | 知行奇点的 Operator Layer positioning | AgentOS (orchestration layer) | LingMirror's agentOS orchestration is the differentiator — make agent-to-agent coordination the headline. |
| Monitor MCP as agent-tool protocol standard. | P2 | RoxyBrowser's MCP-native architecture | ToolBridge | Ensure ToolBridge supports MCP plugin format natively. |
| Do not build generic ERP agents before product loop is stable. | P1 | All AI-native competitors target specific workflows | All modules | Agent execution value requires reliable underlying data — don't automate unreliable processes. |

## Cross-Competitor Patterns

### Cost Truth Is The Core Moat

Competitors repeatedly sell "profit", "finance", "cost", "reconciliation", and "funds tracking". LingMirror should make cost truth explicit:

```text
estimated_cost  -> before listing or before order
snapshot_cost   -> captured at order/shipment execution
actual_cost     -> imported from carrier/platform/ad/warehouse bill
allocated_cost  -> shared first-leg/FBA/overseas warehouse cost allocated to SKU/order
```

### Logistics Is Not Just Rate Calculation

A credible ERP must support:

- Rate rules and quotes.
- Shipment snapshot.
- Label purchase or label reference.
- Tracking.
- Carrier bill import.
- Reconciliation.
- Variance diagnosis.

LingMirror has started rate calculation. The next serious step is bill reconciliation.

### Listing Must Become A Task System

The current decision module becomes valuable when approved rows create listing tasks. This is the shortest path from "analysis" to "operation".

Minimal loop:

```text
batch decision -> approved result -> listing task -> adapter validation -> publish attempt -> synced/failed -> exception
```

### AI Should Explain And Execute Under Control

AI should not be a decorative chat box. It should:

- Explain rejection reasons.
- Diagnose missing data.
- Explain cost variance.
- Propose listing fixes.
- Summarize settlement anomalies.
- Draft operator tasks.

Every agent action needs:

- Permission.
- Source data reference.
- Before/after summary.
- Human approval for irreversible or external actions.
- Audit record.

## Market Context & Sizing

### Global Ecommerce Software Market

- Global ecommerce market projected to reach US$5.05tn by 2030 (Statista, 2026), CAGR 6.84% (2026-2030). This rising transaction volume directly expands the addressable market for cross-border ecommerce ERP and AI agent software.
- Cross-border ecommerce-specific ERP/SaaS segment is estimated at US$3-5B globally (2025), growing at 12-18% CAGR driven by platform proliferation (Amazon, TikTok Shop, Temu, Shopee, Lazada, Walmart) and increasing regulatory complexity (VAT, EPR, product compliance).

### Chinese Cross-Border ERP Market

- China is the largest cross-border ecommerce seller origin country. The Chinese cross-border ecommerce ERP market is estimated at ¥8-12B (US$1.1-1.7B) in 2025, with Mabang, Dianxiaomi, and Eccang as the top three by revenue.
- Key growth drivers: (1) number of Chinese cross-border ecommerce sellers growing at 15-20% YoY past the COVID boom baseline, (2) TikTok Shop and Temu opening new seller onboarding channels, (3) increasing platform fee complexity and compliance requirements making manual ERP operation infeasible at scale.

### AI Agent in Ecommerce Market

- The AI agent layer for ecommerce is in its earliest stage but growing fast. Key signals:
  - Teikametrics/Perpetua/Quartile have established US$500M+ in combined ARR for AI advertising optimization alone — proving sellers will pay for AI that replaces manual campaign management.
  - Luckee AI (2024 launch), 知行奇点 (2025 launch), and RoxyBrowser AI Agent (Feb 2026 launch) all launched within 18 months, indicating rapid market formation.
  - No single player has achieved dominant market share in the cross-border AI agent category — the window for LingMirror to establish a position is open.
- Conservative estimate: AI agent layer for cross-border ecommerce can grow to US$2-5B within 5 years, with advertising automation (40-50%), customer service (20-25%), listing/content (15-20%), and compliance/risk (10-15%) as primary sub-segments.

### LingMirror Implications

- The cross-border ERP market is mature but fragmented — opportunity is in the AI agent layer, not in rebuilding ERP fundamentals.
- The AI-native competitors (Luckee, 知行奇点, RoxyBrowser) all entered in 2024-2026, not 2010-2020 — they validate that the market timing is right, not that it's too late.
- The biggest risk is not competition from existing ERP vendors — it's AI-native specialists capturing the "agent layer" mindshare before LingMirror's agentOS reaches production quality.

## Final Recommendations

### Must Build

1. Decision-to-listing task queue.
2. Freight bill import and reconciliation.
3. Cost-layer labeling in every decision/profit response.
4. Platform settlement import.
5. Order true-profit ledger.
6. Exception workbench.
7. Agent action audit and approval gate.
8. First-leg/FBA allocation after freight/settlement imports exist.
9. AI-native competitor monitoring (Luckee AI, 知行奇点, RoxyBrowser) — track product changes and market signals quarterly.

### Build Later

1. Full WMS/PDA.
2. Wave picking.
3. Supplier KPI and advanced SCM.
4. Advanced replenishment formulas.
5. Advertising automation.
6. Multi-language listing translation.
7. Customer service automation.
8. Broad multi-carrier live label purchase.

### Do Not Copy

1. Large ERP menu expansion without closed workflows.
2. Profit dashboards based only on estimates.
3. "All platforms at once" adapter promises.
4. Manual-only data maintenance where CSV/API import is expected.
5. Black-box AI execution.
6. Platform-specific finance generalized too early.
7. Building a generic agent platform without cross-border domain depth — 知行奇点's differentiation is domain-specific know-how, not generic AI capability.

## Next Implementation Sequence

1. Execute `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`.
2. Write and execute `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md`.
3. Write and execute `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md`.
4. Write and execute `docs/superpowers/plans/2026-06-15-platform-settlement-import.md`.
5. Write and execute `docs/superpowers/plans/2026-06-15-order-profit-ledger.md`.
6. Write and execute `docs/superpowers/plans/2026-06-15-exception-workbench.md`.
7. Add AI-native competitor monitoring to quarterly intelligence cycle — track Luckee AI, 知行奇点, and RoxyBrowser for feature releases, pricing changes, and customer adoption signals.s/2026-06-15-order-profit-ledger.md`.
6. Write and execute `docs/superpowers/plans/2026-06-15-exception-workbench.md`.

## Sources

See `docs/research/competitor-sources-2026-06.md`.
