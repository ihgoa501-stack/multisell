# Next Stage Execution Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Plan the next implementation stages after Stage 1 decision-to-listing tasks, so multiple agents can work in sequence without scope drift.

**Architecture:** Continue the cost-truth ERP spine: Listing Task -> Order -> Shipping Snapshot -> Carrier Bill -> Platform Settlement -> Profit Ledger -> Exception Workbench -> Agent Audit. Each stage must be independently shippable, tested, documented, permissioned, and auditable.

**Tech Stack:** FastAPI, SQLAlchemy async, PostgreSQL, Alembic, Pydantic, pytest, Vue 3, TypeScript, Vite, Naive UI, CSV/XLSX import patterns already present in the repo.

---

## Baseline Assumption

Start from branch:

```text
codex/decision-to-listing-task
```

Known Stage 1 baseline:

- `POST /api/listing-tasks/from-decisions`
- `GET /api/listing-tasks`
- `POST /api/listing-tasks/{id}/recheck`
- `POST /api/listing-tasks/{id}/cancel`
- `POST /api/listing-tasks/{id}/publish`
- Backend validation reported by previous agent: `259 passed`
- Frontend validation reported by previous agent: `npm run build` passed

Before starting any new stage, the implementing agent must run:

```bash
git status --short
```

Expected:

```text
No unrelated modified files in the working tree for that agent's stage.
```

If there are unrelated files, the agent must not stage or revert them.

## Implementation Order

Do the next stages in this order:

1. Stage 2: Shipping Bill Reconciliation
2. Stage 3: Cost Layer Labeling
3. Stage 4: Platform Settlement Import
4. Stage 5: Order True Profit Ledger
5. Stage 6: Exception Workbench
6. Stage 7: Agent Action Audit And Approval

Do not parallelize stages that depend on earlier data models. Safe parallelism is limited to research, UI mock review, or isolated docs.

## Stage 2: Shipping Bill Reconciliation

### Goal

Answer:

```text
This order had a shipping estimate/snapshot. What did the carrier actually bill, and what is the variance?
```

### Why This Comes Next

Stage 1 makes decisions executable. Stage 2 makes one major cost source truthful. Competitors like Mabang, Eccang, SellFox, ShipStation, and Linnworks all point toward shipping cost control, reconciliation, or carrier-backed shipping workflows.

### Scope

Build:

- Carrier bill batch model.
- Carrier bill row model.
- Reconciliation status.
- CSV import endpoint.
- Bill-row matching by `tracking_number`, `order_no`, and provider/channel.
- Reconciliation list endpoint.
- Frontend page for import, summary, and abnormal rows.
- Permissions and audit.

Do not build:

- Real carrier API.
- Label purchase.
- Automatic payment.
- Profit ledger.
- Platform settlement.
- AI diagnosis.

### Backend Files

- Modify: `backend/app/models.py`
- Create: `backend/app/shipping/bill_schemas.py`
- Create: `backend/app/shipping/bill_service.py`
- Modify: `backend/app/shipping/router.py`
- Create: `backend/tests/test_shipping_bill_reconciliation.py`
- Modify: `backend/seed.py`
- Create: `backend/alembic/versions/<revision>_add_shipping_bill_reconciliation.py`

### Frontend Files

- Modify: `frontend/src/api/modules/shipping.ts`
- Create: `frontend/src/views/shipping/ShippingBillReconciliation.vue`
- Modify: `frontend/src/router/modules/shipping.ts`

### Data Model

Create `shipping_bill_batch`:

| Column | Type | Meaning |
| --- | --- | --- |
| `id` | BigInteger PK | Batch id |
| `provider_id` | BigInteger nullable FK | Provider if known |
| `source_filename` | String(255) | Uploaded file name |
| `currency` | String(10) | Default bill currency |
| `row_count` | Integer | Imported row count |
| `matched_count` | Integer | Matched rows |
| `mismatch_count` | Integer | Amount/currency mismatch rows |
| `unmatched_count` | Integer | Bill rows without order match |
| `status` | String(30) | `imported`, `reconciled`, `failed` |
| `created_by` | String(100) | Operator |
| `created_at` | DateTime | Import time |

Create `shipping_bill_row`:

| Column | Type | Meaning |
| --- | --- | --- |
| `id` | BigInteger PK | Row id |
| `batch_id` | BigInteger FK | Import batch |
| `row_number` | Integer | Source row number |
| `order_no` | String(100), nullable | Source order number |
| `tracking_number` | String(100), nullable | Carrier tracking number |
| `provider_name` | String(100), nullable | Source provider name |
| `channel_name` | String(100), nullable | Source channel name |
| `currency` | String(10) | Row currency |
| `actual_shipping_fee` | Numeric(10, 2) | Base billed shipping fee |
| `surcharge_fee` | Numeric(10, 2) | Additional carrier fees |
| `total_actual_fee` | Numeric(10, 2) | Actual plus surcharge |
| `billed_at` | DateTime, nullable | Bill date |
| `matched_order_id` | BigInteger nullable FK | Matched order |
| `matched_snapshot_id` | BigInteger nullable FK | Matched shipping snapshot |
| `snapshot_shipping_fee` | Numeric(10, 2), nullable | Snapshot fee at shipment/order time |
| `variance_amount` | Numeric(10, 2), nullable | Actual minus snapshot |
| `reconciliation_status` | String(30) | `matched`, `unmatched_bill`, `missing_snapshot`, `amount_mismatch`, `currency_mismatch` |
| `raw_payload` | JSON | Original normalized row |
| `created_at` | DateTime | Import time |

### CSV Format

Required headers:

```csv
order_no,tracking_number,provider_name,channel_name,currency,actual_shipping_fee,surcharge_fee,billed_at
```

Example:

```csv
order_no,tracking_number,provider_name,channel_name,currency,actual_shipping_fee,surcharge_fee,billed_at
SO202606150001,TRACK001,DHL,DHL-US,CNY,82.00,3.00,2026-06-15
SO202606150002,TRACK002,YunExpress,YunExpress-US,CNY,45.00,0.00,2026-06-15
```

### API

Create:

```text
POST /api/shipping/bills/import
GET /api/shipping/bills
GET /api/shipping/bills/{batch_id}
GET /api/shipping/bills/{batch_id}/rows
GET /api/shipping/reconciliation/summary
```

### Permissions

Add:

```text
shipping:bill:import
shipping:bill:view
shipping:reconcile
```

### Tests

Required backend tests:

- Import valid CSV creates one batch and rows.
- Row with matching `order_no` and shipping snapshot becomes `matched`.
- Row with same order but different amount becomes `amount_mismatch`.
- Row with different currency becomes `currency_mismatch`.
- Row with no order becomes `unmatched_bill`.
- Row with order but no snapshot becomes `missing_snapshot`.
- Import writes operation log.
- Import requires `shipping:bill:import`.
- View requires `shipping:bill:view`.

### Acceptance

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
cd frontend && npm run build
```

Expected:

```text
Backend tests pass.
Frontend build passes.
```

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目，当前基线是 codex/decision-to-listing-task。

任务：实现 Stage 2 Shipping Bill Reconciliation。

必须阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- docs/research/lingmirror-capability-decisions-2026-06.md
- backend/app/models.py
- backend/app/shipping/
- backend/app/order/
- frontend/src/api/modules/shipping.ts
- frontend/src/views/shipping/
- frontend/src/router/modules/shipping.ts

只做：
- 物流账单 CSV 导入
- 账单批次与账单行
- 与订单 shipping snapshot 对账
- 对账状态和异常列表
- 前端导入/查看页面
- 权限、审计、测试、文档

不要做：
- 真实物流 API
- 自动支付
- 平台结算
- 订单利润账本
- BI 大屏
- AI 自动处理

完成后运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build
```

## Stage 3: Cost Layer Labeling

### Goal

Make every decision/profit/shipping cost explicitly identify its source layer:

```text
estimated
snapshot
actual
allocated
```

### Why This Comes After Stage 2

Before carrier bills exist, most shipping/profit values are estimates or snapshots. After Stage 2, actual freight begins to exist. The UI and API must stop showing margin/profit without source context.

### Scope

Build:

- Shared enum/constant for cost layers.
- Add `cost_layer` to decision response shipping fee.
- Add `shipping_cost_layer` to order profit response.
- Add `platform_fee_cost_layer` where platform fees are still rules/estimates.
- Add UI badges for estimated/snapshot/actual/allocated.
- Update docs.

Do not build:

- New finance ledger.
- Platform settlement import.
- First-leg allocation.

### Backend Files

- Create: `backend/app/finance/cost_layers.py`
- Modify: `backend/app/decision/schemas.py`
- Modify: `backend/app/decision/service.py`
- Modify: `backend/app/order/schemas.py`
- Modify: `backend/app/order/service.py`
- Modify: `backend/app/shipping/bill_schemas.py` if Stage 2 exists
- Create: `backend/tests/test_cost_layer_labeling.py`

### Frontend Files

- Create: `frontend/src/components/CostLayerTag.vue`
- Modify: `frontend/src/views/decision/PreListingDecision.vue`
- Modify: `frontend/src/views/decision/BatchPreListingDecision.vue`
- Modify: `frontend/src/views/order/OrderDetail.vue`
- Modify: `frontend/src/views/shipping/ShippingBillReconciliation.vue` if Stage 2 exists

### API Contract

Decision result should include:

```json
{
  "shipping_fee": 45.0,
  "shipping_cost_layer": "estimated",
  "platform_fee": 12.5,
  "platform_fee_cost_layer": "estimated"
}
```

Order profit should include:

```json
{
  "shipping_fee": 45.0,
  "shipping_cost_layer": "snapshot",
  "platform_fee": 12.5,
  "platform_fee_cost_layer": "estimated",
  "profit_amount": 31.2,
  "profit_cost_layer": "mixed"
}
```

After Stage 5, `profit_cost_layer` may become `actual` or `mixed`. Stage 3 only labels current truth.

### Tests

Required backend tests:

- Pre-listing decision returns `shipping_cost_layer=estimated`.
- Pre-listing decision returns `platform_fee_cost_layer=estimated`.
- Order without shipping snapshot returns `shipping_cost_layer=estimated` when using order shipping fee.
- Order with bound shipping snapshot returns `shipping_cost_layer=snapshot`.
- Reconciled shipping bill row returns `actual` in bill row schema if Stage 2 exists.

### Acceptance

Run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
cd frontend && npm run build
```

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目。

任务：实现 Stage 3 Cost Layer Labeling。

目标：所有决策、订单利润、运费相关金额都要标识成本来源层：estimated / snapshot / actual / allocated / mixed。

必须先阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- docs/research/lingmirror-capability-decisions-2026-06.md
- backend/app/decision/
- backend/app/order/
- backend/app/shipping/
- frontend/src/views/decision/
- frontend/src/views/order/

只做成本层标识和 UI 标签，不做新账本、不做平台结算、不做分摊。
```

## Stage 4: Platform Settlement Import

### Goal

Import platform settlement rows so platform fees, refunds, adjustments, and payouts can become actual source data.

### Why This Comes Before Profit Ledger

True order profit requires actual platform fee and refund data. A ledger without settlement import would still depend on estimates.

### Scope

Build:

- Settlement batch model.
- Settlement row model.
- CSV import endpoint.
- Matching to order by `order_no` or platform order number.
- Normalized transaction types.
- Frontend import and row review page.
- Permissions and audit.

Do not build:

- Automatic platform API.
- Full payout accounting.
- Tax compliance.
- Advertising import.

### Backend Files

- Modify: `backend/app/models.py`
- Create: `backend/app/settlement/__init__.py`
- Create: `backend/app/settlement/schemas.py`
- Create: `backend/app/settlement/service.py`
- Create: `backend/app/settlement/router.py`
- Create: `backend/tests/test_platform_settlement_import.py`
- Modify: `backend/seed.py`
- Create: `backend/alembic/versions/<revision>_add_platform_settlement_import.py`

### Frontend Files

- Create: `frontend/src/api/modules/settlement.ts`
- Create: `frontend/src/views/settlement/SettlementImport.vue`
- Create: `frontend/src/router/modules/settlement.ts`

### CSV Format

Required headers:

```csv
platform,store_name,platform_order_no,order_no,transaction_type,currency,amount,settled_at,description
```

Allowed `transaction_type` values:

```text
sale
platform_fee
payment_fee
refund
adjustment
payout
tax
other
```

### API

Create:

```text
POST /api/settlements/import
GET /api/settlements
GET /api/settlements/{batch_id}
GET /api/settlements/{batch_id}/rows
GET /api/settlements/unmatched
```

### Permissions

Add:

```text
settlement:import
settlement:view
settlement:match
```

### Tests

Required backend tests:

- Import valid settlement CSV creates batch and rows.
- `platform_fee` row matches order by `order_no`.
- Row with unknown order becomes unmatched.
- Refund row is stored as negative effect if CSV amount is negative.
- Import writes operation log.
- Import requires `settlement:import`.
- View requires `settlement:view`.

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目。

任务：实现 Stage 4 Platform Settlement Import。

目标：导入平台结算 CSV，沉淀平台费、退款、调整、付款等实际来源数据。

必须阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- docs/research/competitor-research-2026-06.md
- backend/app/order/
- backend/app/models.py
- frontend/src/api/modules/order.ts

只做 CSV 导入、批次/行、匹配、权限、审计、前端查看。
不要做真实平台 API，不要做利润账本，不要做广告数据。
```

## Stage 5: Order True Profit Ledger

### Goal

Create a ledger that can explain order profit from source facts:

```text
order revenue
product cost
shipping snapshot
actual carrier bill
platform settlement fee
refund
adjustment
other fee
```

### Why This Comes After Stage 2 And Stage 4

The ledger needs actual freight and settlement rows. Without those, it is only an estimate.

### Scope

Build:

- `finance_ledger_entry` model.
- Ledger rebuild service for one order.
- Ledger rebuild endpoint.
- Order profit endpoint based on ledger.
- UI panel in order detail.
- Cost layer summary.

Do not build:

- BI dashboard.
- First-leg/FBA allocation.
- Advertising import.
- Tax/VAT.

### Backend Files

- Modify: `backend/app/models.py`
- Create: `backend/app/finance/__init__.py`
- Create: `backend/app/finance/schemas.py`
- Create: `backend/app/finance/ledger_service.py`
- Create: `backend/app/finance/router.py`
- Create: `backend/tests/test_order_profit_ledger.py`
- Modify: `backend/seed.py`
- Create: `backend/alembic/versions/<revision>_add_finance_ledger.py`

### Frontend Files

- Create: `frontend/src/api/modules/finance.ts`
- Create: `frontend/src/components/OrderProfitLedger.vue`
- Modify: `frontend/src/views/order/OrderDetail.vue`
- Create: `frontend/src/router/modules/finance.ts` only if adding finance list pages

### Ledger Entry Fields

| Field | Meaning |
| --- | --- |
| `order_id` | Order |
| `entry_type` | `revenue`, `product_cost`, `shipping_cost`, `platform_fee`, `payment_fee`, `refund`, `adjustment`, `other_fee` |
| `amount` | Positive revenue, negative cost, or signed adjustment |
| `currency` | Currency |
| `cost_layer` | `estimated`, `snapshot`, `actual`, `allocated` |
| `source_type` | `order`, `shipping_snapshot`, `shipping_bill_row`, `settlement_row`, `manual` |
| `source_id` | Source row id |
| `description` | Human-readable reason |

### API

Create:

```text
POST /api/finance/orders/{order_id}/ledger/rebuild
GET /api/finance/orders/{order_id}/ledger
GET /api/finance/orders/{order_id}/profit
```

### Permissions

Add:

```text
finance:ledger:view
finance:ledger:rebuild
```

### Tests

Required backend tests:

- Rebuild creates revenue and product cost entries.
- Rebuild uses shipping snapshot when no carrier bill exists.
- Rebuild prefers actual carrier bill over snapshot when matched.
- Rebuild includes settlement platform fee.
- Refund reduces profit.
- Profit response reports `profit_cost_layer=mixed` when sources differ.
- Rebuild is idempotent for the same order.
- Rebuild requires `finance:ledger:rebuild`.

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目。

任务：实现 Stage 5 Order True Profit Ledger。

目标：为订单生成可解释的真实利润账本，来源包括订单、商品成本、运费快照、实际物流账单、平台结算。

必须阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- backend/app/order/
- backend/app/shipping/
- backend/app/settlement/
- backend/app/models.py
- frontend/src/views/order/OrderDetail.vue

不要做 BI，不要做头程/FBA分摊，不要做广告费，不要做税务。
```

## Stage 6: Exception Workbench

### Goal

Turn operational failures into owned, filterable work items.

### Why This Comes After Ledger

The useful exceptions now come from multiple modules: listing tasks, shipping reconciliation, settlement import, and profit ledger.

### Scope

Build:

- Exception item model.
- Exception generation service.
- Sources: listing blocked/failed, shipping bill unmatched/mismatch, settlement unmatched, negative profit.
- Workbench list/detail/update endpoints.
- Frontend exception workbench.
- Assignment, status, and audit.

Do not build:

- AI auto-resolution.
- Notification system.
- SLA escalation.

### Backend Files

- Modify: `backend/app/models.py`
- Create: `backend/app/exceptions/__init__.py`
- Create: `backend/app/exceptions/schemas.py`
- Create: `backend/app/exceptions/service.py`
- Create: `backend/app/exceptions/router.py`
- Create: `backend/tests/test_exception_workbench.py`
- Modify: `backend/seed.py`
- Create: `backend/alembic/versions/<revision>_add_exception_workbench.py`

### Frontend Files

- Create: `frontend/src/api/modules/exceptions.ts`
- Create: `frontend/src/views/exceptions/ExceptionWorkbench.vue`
- Create: `frontend/src/router/modules/exceptions.ts`

### Exception Fields

| Field | Meaning |
| --- | --- |
| `source_module` | `listing`, `shipping`, `settlement`, `finance` |
| `source_type` | Source row/entity type |
| `source_id` | Source row/entity id |
| `severity` | `low`, `medium`, `high`, `critical` |
| `status` | `open`, `assigned`, `resolved`, `ignored` |
| `title` | Short title |
| `description` | Explanation |
| `recommended_action` | Human next step |
| `assigned_to` | Username |
| `resolved_at` | Resolution time |
| `resolved_by` | Resolver |

### API

Create:

```text
POST /api/exceptions/generate
GET /api/exceptions
GET /api/exceptions/{exception_id}
POST /api/exceptions/{exception_id}/assign
POST /api/exceptions/{exception_id}/resolve
POST /api/exceptions/{exception_id}/ignore
```

### Permissions

Add:

```text
exception:view
exception:manage
exception:generate
```

### Tests

Required backend tests:

- Generate creates exception for failed listing task.
- Generate creates exception for unmatched shipping bill row.
- Generate creates exception for amount mismatch.
- Generate creates exception for unmatched settlement row.
- Generate creates exception for negative order profit after ledger rebuild.
- Generate is idempotent for the same source.
- Assign changes status to `assigned` and writes audit.
- Resolve changes status to `resolved` and writes audit.
- View requires `exception:view`.

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目。

任务：实现 Stage 6 Exception Workbench。

目标：把上架失败、物流账单异常、结算未匹配、负利润等问题集中成异常工作台。

必须阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- backend/app/listing/
- backend/app/shipping/
- backend/app/settlement/
- backend/app/finance/
- frontend/src/router/

不要做 AI 自动处理，不要做通知系统，不要做 SLA。
```

## Stage 7: Agent Action Audit And Approval

### Goal

Prepare the system for AI agents to suggest and execute controlled actions with approval and audit.

### Why This Comes After Exception Workbench

Agents need a task surface. The exception workbench provides that surface.

### Scope

Build:

- Agent action proposal model.
- Approval status.
- Source references.
- Before/after summary.
- Endpoints to create proposal, approve, reject, mark executed.
- UI panel attached to exception detail.

Do not build:

- LLM provider integration.
- Autonomous background workers.
- External API execution.

### Backend Files

- Modify: `backend/app/models.py`
- Create: `backend/app/agent_actions/__init__.py`
- Create: `backend/app/agent_actions/schemas.py`
- Create: `backend/app/agent_actions/service.py`
- Create: `backend/app/agent_actions/router.py`
- Create: `backend/tests/test_agent_action_audit.py`
- Modify: `backend/seed.py`
- Create: `backend/alembic/versions/<revision>_add_agent_action_audit.py`

### Frontend Files

- Create: `frontend/src/api/modules/agentActions.ts`
- Create: `frontend/src/components/AgentActionPanel.vue`
- Modify: `frontend/src/views/exceptions/ExceptionWorkbench.vue`

### API

Create:

```text
POST /api/agent-actions
GET /api/agent-actions
GET /api/agent-actions/{action_id}
POST /api/agent-actions/{action_id}/approve
POST /api/agent-actions/{action_id}/reject
POST /api/agent-actions/{action_id}/mark-executed
```

### Permissions

Add:

```text
agent_action:view
agent_action:propose
agent_action:approve
agent_action:execute
```

### Tests

Required backend tests:

- Create action proposal for an exception.
- Proposal stores source module/type/id.
- Approve changes status to `approved`.
- Reject changes status to `rejected`.
- Mark executed only works after approval.
- Approval writes operation log.
- Execute writes operation log.
- Approval requires `agent_action:approve`.

### Agent Prompt

```text
你接手 /Users/lc/multisell 项目。

任务：实现 Stage 7 Agent Action Audit And Approval。

目标：为未来 AI agent 准备动作提案、审批、执行标记和审计，不接入真实 LLM。

必须阅读：
- docs/superpowers/plans/2026-06-15-next-stage-execution-roadmap.md
- backend/app/exceptions/
- backend/app/operation_log/
- frontend/src/views/exceptions/ExceptionWorkbench.vue

不要做 LLM API，不要做后台自动执行，不要调用真实平台或物流 API。
```

## Integration Rules For All Stages

- Each stage gets its own branch:

```text
codex/shipping-bill-reconciliation
codex/cost-layer-labeling
codex/platform-settlement-import
codex/order-profit-ledger
codex/exception-workbench
codex/agent-action-audit
```

- Each stage must commit independently.
- Each stage must update:

```text
docs/PROJECT_STATUS.md
docs/ROADMAP.md
docs/PERMISSIONS_AND_AUDIT.md
```

- Each stage must run:

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
cd frontend && npm run build
```

- Each stage handoff must report:

```text
1. Branch and commit hash
2. New models
3. New API endpoints
4. Frontend pages
5. Permissions
6. Audit behavior
7. Test results
8. Known limits
```

## Suggested Agent Allocation

Use only the agents approved by the user:

```text
claude
opencode
copilot
gemini
reasonix
```

Recommended allocation:

| Stage | Agent | Reason |
| --- | --- | --- |
| Stage 2 Shipping Bill Reconciliation | claude or reasonix | Backend-heavy matching and CSV logic |
| Stage 3 Cost Layer Labeling | copilot or opencode | Cross-cutting API/UI field propagation |
| Stage 4 Platform Settlement Import | claude or reasonix | Finance CSV import, order matching, and settlement transaction rules |
| Stage 5 Order True Profit Ledger | reasonix | Requires careful model and accounting logic |
| Stage 6 Exception Workbench | opencode or copilot | Full-stack CRUD/workflow page |
| Stage 7 Agent Action Audit | gemini or opencode | Workflow/audit surface, no LLM integration |

## Stop Conditions

Stop and ask for human decision if:

- A stage requires real platform credentials.
- A stage requires real carrier API credentials.
- Existing migrations conflict with current database state.
- A model name conflicts with already merged work.
- Tests fail due to an unrelated dirty baseline.

## Strategic Reminder

The goal is not to build more ERP menus. The goal is to make this statement true:

```text
For any order, LingMirror can explain decision, listing state, shipping estimate, shipping actual, platform settlement, true profit, exception status, and agent/human actions.
```
