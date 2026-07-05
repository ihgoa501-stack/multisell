> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Mabang ERP Benchmark Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define the route for LingMirror / MultiSell to grow from pre-listing decision support into a Mabang-style cross-border ERP with strong logistics, warehouse, listing, finance, and operational automation capabilities.

**Architecture:** Do not copy Mabang as one giant feature set. Build toward the same operational surface through bounded modules: Decision, Listing, Order, Inventory/WMS, Shipping/TMS, Procurement/SCM, Finance, BI, and Agent Orchestration. Each module must produce testable software independently and feed a shared profit/cost ledger.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL, Alembic, Pydantic, pytest, Vue 3, TypeScript, Vite, Naive UI, openpyxl, external platform/carrier APIs behind adapters.

---

## Why Mabang Is The Benchmark

Public Mabang ERP positioning shows these relevant capabilities:

- Full process coverage: product development, order processing, supply chain, product management, inventory planning, logistics management, overseas warehouse, financial reports, customer service, refined operations.
- Logistics strength: logistics API integration, real freight retrieval, lowest-cost logistics matching, logistics inquiry, package tracking, reconciliation.
- TMS/WMS depth: order tracking, in-transit tracking, billing, automatic reconciliation, multiple logistics pricing systems, warehouse barcode/PDA workflows.
- Financial depth: multi-platform profit reports, order-level analysis, Amazon-specialized profit reports, business-finance integrated data.

Reference:

- Mabang official site: `https://www.mabangerp.com/`

The product lesson is:

```text
Mabang is not "one ERP page".
It is an operations graph:
catalog -> listing -> order -> warehouse -> shipping -> reconciliation -> profit -> replenishment.
```

LingMirror should compete by having the same operations graph, but with stronger AI/Agent decision workflows.

## Current Position

Already available or planned in this repo:

- Catalog, SKU, price, inventory basics.
- Order creation and inventory lock/deduct/release.
- Shipping provider/channel/zone/rate rules.
- Shipping calculation and order shipping snapshot.
- Product logistics data completeness.
- Platform fee rules.
- Single and batch pre-listing decision.
- Excel batch pre-listing decision plan.
- Listing adapter boundary with mock adapter.
- RBAC and audit patterns.

Major gaps versus Mabang:

- No real carrier API or label purchasing.
- No actual freight bill import/reconciliation.
- No inbound shipment / first-leg / FBA allocation.
- No true warehouse bin/PDA/wave pick-pack flow.
- No purchase order and replenishment loop.
- No platform settlement import and true net-profit ledger.
- No real platform listing adapters.
- No BI dashboards comparable to financial/operations reports.
- No agent task queue, approval gate, or autonomous workflow audit.

## Strategic Principle

Build in this order:

```text
Decision accuracy -> listing task -> order execution -> actual cost reconciliation -> profit truth -> replenishment -> automation.
```

Do not build broad UI before cost truth is reliable.

Cost truth has four layers:

```text
estimated_cost     上架前/报价阶段预估
snapshot_cost      订单执行时快照
actual_cost        物流商/平台/广告/仓储账单
allocated_cost     头程/FBA/海外仓费用分摊
```

Every future finance/report feature must identify which layer its value comes from.

## Target Capability Map

### Product / PIM

Target:

- Central product library.
- Multi-platform title, description, image, attribute, variation mapping.
- Category mapping per platform.
- Listing readiness score.
- Bulk edit and validation.

Mabang-like value:

- Product data can be reused across platforms.
- Listing work becomes a controlled workflow instead of repeated manual editing.

### Listing / Publishing

Target:

- Draft listing tasks generated from approved decision results.
- Platform adapters for first real platform.
- Publish queue, retry, status sync, error normalization.
- Platform category and attribute mapping.

Mabang-like value:

- Approved products move from decision to platform publishing without copy-paste.

### OMS / Order

Target:

- Unified multi-platform order intake.
- Order risk checks and intercept rules.
- Merge/split shipment.
- Order status sync.
- Refund/return workflow.

Mabang-like value:

- Orders become centralized and automatable.

### Shipping / TMS

Target:

- Carrier account management.
- Rate cards with versions.
- Real-time carrier quote adapter.
- Label purchase adapter.
- Tracking sync.
- Freight bill import and reconciliation.
- Cheapest suitable carrier recommendation.

Mabang-like value:

- System can compare estimated freight, order snapshot freight, and actual billed freight.

### WMS / Inventory

Target:

- Multiple warehouses.
- Bin/location management.
- Inbound, outbound, transfer, adjustment.
- Pick-pack-ship workflow.
- Barcode scan events.
- Wave picking.
- Inventory cost layers.

Mabang-like value:

- Warehouse operations become measurable and auditable.

### Procurement / SCM

Target:

- Purchase orders.
- Supplier quotes and lead times.
- Inbound shipment planning.
- Replenishment suggestions.
- Supplier performance.

Mabang-like value:

- Sales and inventory convert into purchasing actions.

### Finance / Profit

Target:

- Cost ledger.
- Platform settlement import.
- Ad spend import.
- Freight bill import.
- First-leg/FBA/overseas warehouse allocation.
- Order true profit.
- SKU true profit.
- Store/platform/operator profit.

Mabang-like value:

- Profit report is no longer a rough estimate.

### BI / Reports

Target:

- Boss dashboard.
- Product profit dashboard.
- Order profit dashboard.
- Logistics cost variance dashboard.
- Inventory turnover dashboard.
- Listing success/failure dashboard.
- Replenishment and stockout risk dashboard.

Mabang-like value:

- Management can see where money is made or lost.

### Agent Orchestration

Target:

- Agent task queue.
- Human approval gate.
- Agent execution logs.
- Tool-level permission boundaries.
- Automated diagnosis for missing data and abnormal cost.

LingMirror differentiation:

- Mabang is workflow-heavy. LingMirror should be workflow + agent-heavy.

## Roadmap Overview

### Stage 0: Baseline Freeze And Release Discipline

Goal:

- Keep the current repo stable while multiple agents execute roadmap slices.

Deliverables:

- Clean `main`.
- Every feature branch uses one bounded plan.
- Every merge requires backend focused tests, backend full tests, frontend build.
- Dirty/untracked generated files are not mixed into feature commits.

Acceptance:

- `git status --short --branch` clean after every completed feature.
- Backend full suite passes.
- Frontend build passes.

### Stage 1: Decision To Listing Task

Goal:

- Convert approved pre-listing decision results into controlled listing draft tasks.

Why first:

- This connects the current core value, pre-listing decision, to real operation.

Major features:

- `listing_task` table.
- Create listing tasks from batch decision approve rows.
- Task status: `draft`, `ready`, `blocked`, `published`, `failed`, `cancelled`.
- Listing readiness checks.
- UI action: "生成上架任务".

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-decision-to-listing-task.md
```

### Stage 2: Shipping Reconciliation Layer

Goal:

- Build the actual freight cost layer.

Why second:

- Mabang's logistics value comes from real freight, lowest-cost matching, and reconciliation.

Major features:

- `shipping_bill` and `shipping_bill_item`.
- Import carrier bill `.xlsx`.
- Match by tracking number, order number, or label number.
- Compare estimated freight, snapshot freight, actual billed freight.
- Reconciliation statuses: `matched`, `unmatched`, `amount_mismatch`, `manual_resolved`.
- UI: freight reconciliation workbench.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-shipping-bill-reconciliation.md
```

### Stage 3: First-Leg / FBA / Overseas Warehouse Allocation

Goal:

- Allocate inbound logistics and warehouse costs onto SKU cost.

Why third:

- Without allocated first-leg/FBA cost, net profit is structurally wrong.

Major features:

- `inbound_shipment`.
- `inbound_shipment_item`.
- `inbound_cost`.
- Allocation methods: quantity, weight, volume, value.
- Inventory cost adjustment after allocation.
- UI: inbound shipment cost allocation.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-inbound-cost-allocation.md
```

### Stage 4: True Profit Ledger

Goal:

- Make profit reports traceable and recalculable.

Why fourth:

- Reports should read from one ledger, not recalculate inconsistently in different modules.

Major features:

- `profit_ledger_entry`.
- Source types:
  - `order_revenue`
  - `product_cost`
  - `shipping_snapshot`
  - `shipping_actual`
  - `platform_fee_estimated`
  - `platform_settlement`
  - `ad_spend`
  - `inbound_allocation`
  - `warehouse_fee`
  - `manual_adjustment`
- Rebuild ledger for an order.
- Order true profit API.
- SKU true profit API.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-profit-ledger.md
```

### Stage 5: WMS Core

Goal:

- Add warehouse operations beyond simple inventory quantity.

Why fifth:

- Mabang/WMS competitiveness comes from execution reliability, not just inventory numbers.

Major features:

- `warehouse`, `warehouse_zone`, `bin_location`.
- `inventory_movement`.
- Inbound receive.
- Outbound pick-pack-ship.
- Barcode scan event.
- Stock transfer.
- Inventory adjustment.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-wms-core.md
```

### Stage 6: Procurement And Replenishment

Goal:

- Close the sales -> stock -> purchase loop.

Why sixth:

- Mabang includes procurement and supply chain. LingMirror needs this before serious BI.

Major features:

- `purchase_order`.
- `purchase_order_item`.
- Supplier lead time.
- Reorder rule.
- Replenishment suggestion.
- PO receive into WMS inbound flow.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-procurement-replenishment.md
```

### Stage 7: Real Platform Listing Adapter

Goal:

- Publish to one real marketplace through the adapter boundary.

Why seventh:

- Listing automation is valuable only after data, cost, and approval are controlled.

Recommended first platform:

```text
Ozon or Shopee
```

Major features:

- Platform credential validation.
- Category mapping.
- Attribute mapping.
- Publish draft.
- Publish status sync.
- Error normalization.
- Retry queue.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-first-real-listing-adapter.md
```

### Stage 8: BI Dashboards

Goal:

- Build Mabang-like management reporting.

Why eighth:

- BI before true cost would be pretty but misleading.

Major dashboards:

- Boss dashboard.
- Product profit dashboard.
- Store/platform profit dashboard.
- Logistics variance dashboard.
- Inventory turnover dashboard.
- Listing performance dashboard.
- Procurement efficiency dashboard.

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-bi-dashboards.md
```

### Stage 9: Agent Orchestration

Goal:

- Turn LingMirror into an AI Agent协作运营平台, not just a traditional ERP.

Why ninth:

- Agent autonomy needs stable business tools first.

Major features:

- `agent_task`.
- `agent_task_step`.
- `approval_request`.
- Tool permission boundary.
- Agent run audit.
- Human approval for high-risk actions:
  - publish listing
  - change price
  - purchase inventory
  - change carrier route
  - apply manual financial adjustment

Independent implementation plan to write:

```text
docs/superpowers/plans/YYYY-MM-DD-agent-orchestration.md
```

## Recommended Execution Order

Execute in this exact order:

```text
1. decision-to-listing-task
2. shipping-bill-reconciliation
3. inbound-cost-allocation
4. profit-ledger
5. wms-core
6. procurement-replenishment
7. first-real-listing-adapter
8. bi-dashboards
9. agent-orchestration
```

Reason:

- Listing tasks create the operational bridge.
- Freight reconciliation and inbound allocation create cost truth.
- Profit ledger turns cost truth into reports.
- WMS and procurement close execution.
- Real listing adapter then has reliable inputs.
- BI reads mature data.
- Agents safely automate mature workflows.

## What Not To Do

Do not:

- Build a huge Mabang-like menu before core loops work.
- Build BI dashboards before true cost exists.
- Integrate five platforms before one adapter is production-grade.
- Treat freight as one `shipping_fee` field.
- Mix estimated cost and actual cost without source fields.
- Let agents directly publish, purchase, or adjust profit without approval.
- Add new pages without backend tests.

## Milestones

### Milestone A: Mabang-Style Decision Loop

Definition:

```text
Excel batch decision -> approved rows -> listing tasks -> readiness check.
```

Target stages:

- Stage 1.

Exit criteria:

- Approved batch rows can create listing tasks.
- Blocked tasks explain missing data.
- Frontend shows listing task queue.

### Milestone B: Mabang-Style Logistics Cost Loop

Definition:

```text
estimated freight -> order snapshot -> carrier bill -> reconciliation.
```

Target stages:

- Stage 2.

Exit criteria:

- Carrier bill import works.
- Orders show actual freight source.
- Variance dashboard can be computed from data.

### Milestone C: Mabang-Style True Profit Loop

Definition:

```text
order revenue - product cost - platform fees - actual freight - allocated inbound cost - ad cost = true profit.
```

Target stages:

- Stage 3.
- Stage 4.

Exit criteria:

- Order true profit is traceable by ledger entry.
- SKU profit includes allocated first-leg/FBA/warehouse costs.

### Milestone D: Mabang-Style Execution Loop

Definition:

```text
listing -> order -> pick/pack/ship -> stock movement -> purchase replenishment.
```

Target stages:

- Stage 5.
- Stage 6.
- Stage 7.

Exit criteria:

- Warehouse movements are auditable.
- Purchase suggestions are explainable.
- One real platform adapter can publish and sync status.

### Milestone E: LingMirror Differentiation

Definition:

```text
Agents diagnose, recommend, prepare, and execute approved operational tasks.
```

Target stages:

- Stage 8.
- Stage 9.

Exit criteria:

- Agents can create task proposals.
- Humans approve high-risk actions.
- Every agent step is auditable.

## Agent Assignment Model

Use separate agents by domain:

```text
Agent A: Listing task and platform adapter
Agent B: Shipping/TMS reconciliation
Agent C: Inbound/FBA cost allocation
Agent D: Profit ledger and BI
Agent E: WMS and procurement
Agent F: Agent orchestration and approval
```

Rules:

- One agent owns one bounded plan.
- No agent edits another module without reading its plan and tests.
- All agents must run focused tests.
- The integrator agent runs full backend and frontend verification before merge.

## Immediate Next Plan To Write

Write this implementation plan next:

```text
docs/superpowers/plans/2026-06-15-decision-to-listing-task.md
```

It should implement Stage 1 only.

Required prompt:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md
- docs/superpowers/plans/2026-06-15-excel-batch-prelisting-decision.md
- backend/app/decision/
- backend/app/listing/
- backend/app/models.py
- frontend/src/views/decision/
- frontend/src/views/listing/

请只为 Stage 1 写或执行计划：从批量上架决策 approve 结果生成 listing task。
不要做真实平台 API，不要做物流对账，不要做 WMS，不要做 BI。

完成后必须运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build

交付时说明：
- 新增了哪些数据模型和 API
- listing task 状态流如何设计
- 如何从决策结果生成任务
- 如何阻止缺数据任务进入发布
- 测试命令和结果
- 剩余限制
```

## Final Acceptance Criteria For This Roadmap

This roadmap is considered usable when:

- It clearly identifies Mabang-like target capabilities.
- It separates implementation into bounded stages.
- It defines why each stage comes before the next.
- It avoids broad UI-first implementation.
- It protects cost-truth architecture with estimated/snapshot/actual/allocated layers.
- It identifies the immediate next implementation plan.
- It gives agent assignment boundaries.

## Competitor Research Follow-up

Research package:

- `docs/research/competitor-sources-2026-06.md`
- `docs/research/competitor-research-2026-06.md`
- `docs/research/lingmirror-capability-decisions-2026-06.md`

Research conclusion:

- Mabang is the business-depth benchmark, not the UI blueprint.
- Eccang validates that first-leg/FBA, automated reconciliation, fee import, and net-profit accounting are key enterprise ERP capabilities.
- SellFox validates that platform-specific finance, FIFO order cost, funds tracking, and logistics comparison are refined-operations priorities.
- CaptainBI validates transparent AI diagnosis and AI-assisted operations as a differentiation direction, but LingMirror must keep approval and audit controls.
- Dianxiaomi, BigSeller, and 4Seller validate low-friction multi-platform UX, batch operations, and SME-friendly workflow design.
- Sellercloud, ShipStation, and Linnworks validate clear module boundaries, order/shipping automation, carrier adapters, accounting, reporting, and API-first integration.

Roadmap adjustment:

1. Keep Stage 1 as decision-to-listing task.
2. Move freight bill reconciliation immediately after Stage 1.
3. Add cost-layer labeling before any profit dashboard expansion.
4. Add platform settlement import before claiming true platform profit.
5. Build order true-profit ledger before BI.
6. Build exception workbench before broad agent automation.

Updated next implementation sequence:

1. Execute `docs/superpowers/plans/2026-06-15-decision-to-listing-task.md`.
2. Write and execute `docs/superpowers/plans/2026-06-15-shipping-bill-reconciliation.md`.
3. Write and execute `docs/superpowers/plans/2026-06-15-cost-layer-labeling.md`.
4. Write and execute `docs/superpowers/plans/2026-06-15-platform-settlement-import.md`.
5. Write and execute `docs/superpowers/plans/2026-06-15-order-profit-ledger.md`.
6. Write and execute `docs/superpowers/plans/2026-06-15-exception-workbench.md`.
