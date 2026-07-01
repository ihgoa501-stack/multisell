# LingMirror Agent Responsibility Cards

> **P1 deliverable — Canonical Agent Responsibility Cards**
> Each card defines the Agent's business job, input data, tools/APIs, outputs,
> allowed actions, approval requirements, forbidden actions, audit fields,
> trigger/schedule, and success metrics.
>
> Last updated: 2026-07-01

## A1 Product Scout · 选品助理

| Field | Value |
|---|---|
| **Business job** | Discover profitable cross-border products by analyzing market trends, supplier data, and platform demand signals |
| **Reads** | Ozon/Shopee category trends, 1688 supplier catalog, historical sales data, competitor pricing |
| **Tools / APIs** | `product_scout`/`market_analysis` (decision points), sourcing APIs (`GET /api/v1/sourcing/recommendations`), 1688 CRUD, products CRUD, product-analysis API |
| **Outputs** | Candidate product list with estimated margin, demand score, competition level, and sourcing link |
| **Allowed actions** | Create candidate product record, propose market analysis report, update product scout rules |
| **Approval required** | Creating purchase orders or committing to supplier orders |
| **Forbidden actions** | Create purchase orders without approval, modify existing supplier contracts, change platform prices |
| **Audit fields** | `agent_id: A1`, `decision_point`, `product_id`, `recommendation`, `confidence`, `action_id` |
| **Trigger** | Scheduled (every 6 hr), manual via `POST /api/v1/agents/A1/actions`, on-demand via sourcing dashboard |
| **Success metric** | Candidate-to-listing conversion rate ≥5%, Owner adoption rate ≥40%, average estimated margin ≥15% |

## A2 Listing Optimizer · 商品优化师

| Field | Value |
|---|---|
| **Business job** | Improve listing titles, descriptions, keywords, images, and attributes to increase organic traffic and conversion rate |
| **Reads** | Product listing data, keyword performance, category attributes, platform listing status, platform fee config |
| **Tools / APIs** | `listing_optimize`/`keyword_research` (decision points), listing API (`GET/PUT /api/v1/listings/:id`), product search, platform integration APIs |
| **Outputs** | Optimized title/description draft, suggested keyword additions, image improvement recommendations |
| **Allowed actions** | Create listing task with optimized content, suggest keyword changes, propose image regeneration via `/api/v1/imagegen` |
| **Approval required** | Publishing optimized listing to external platform, applying price changes |
| **Forbidden actions** | Publish listings to external platforms without Owner approval, change prices directly |
| **Audit fields** | `agent_id: A2`, `listing_id`, `optimization_type`, `before/after_diff`, `action_id` |
| **Trigger** | Pipeline chain from A6 profit_watch (loss/threshold → A2), scheduled (every 2 hr), manual run |
| **Success metric** | Listing task acceptance rate ≥50%, organic traffic improvement tracked per listing |

## A3 Ad Analyst · 广告分析师

| Field | Value |
|---|---|
| **Business job** | Monitor advertising ACOS, recommend ad budget adjustments, identify under/over-performing campaigns |
| **Reads** | Advertising spend data, order data, profit calculations, platform fee config |
| **Tools / APIs** | `acos_analysis`/`ad_optimization` (decision points), finance profit APIs (`GET /api/v1/finance/profit/summary`), order summary API |
| **Outputs** | ACOS report, campaign-level budget suggestion, keyword bid recommendation, ad stop/go decision |
| **Allowed actions** | Propose budget changes, create ad adjustment recommendation, flag high-ACOS campaigns |
| **Approval required** | Changing ad budgets >10%, pausing campaigns with active orders, committing ad spend increases |
| **Forbidden actions** | Modify ad budgets directly without approval, pause campaigns without warning, change bids to loss-making levels |
| **Audit fields** | `agent_id: A3`, `campaign_id`, `current_acos`, `recommended_acos`, `budget_change`, `action_id` |
| **Trigger** | Scheduled (every 1 hr), manual run |
| **Success metric** | ACOS reduction on acted-upon campaigns ≥5%, false alert rate ≤20% |

## A4 Customer Service Assistant · 客服助理

| Field | Value |
|---|---|
| **Business job** | Auto-reply to customer inquiries, classify intent, escalate complex issues to human support |
| **Reads** | Support conversation messages, customer order history, after-sales records, reply templates |
| **Tools / APIs** | `auto_reply`/`intent_classify` (decision points), support conversation APIs (`GET/POST /api/v1/support/conversations`), template API, after-sales API |
| **Outputs** | Draft reply, intent classification label, escalation recommendation |
| **Allowed actions** | Auto-reply to routine inquiries (shipping status, return policy), propose escalation, update conversation status |
| **Approval required** | Issuing refunds or replacements, modifying order status, sending compensation offers |
| **Forbidden actions** | Issue refunds without approval, modify order shipping address, promise delivery dates exceeding SLA |
| **Audit fields** | `agent_id: A4`, `conversation_id`, `intent`, `reply_template_id`, `auto_replied`, `action_id` |
| **Trigger** | Scheduled (every 5 min), real-time event on new support message |
| **Success metric** | Auto-resolution rate ≥60%, customer satisfaction on auto-replies ≥4.0/5.0, escalation accuracy ≥85% |

## A5 Stock Alert · 库存助理

| Field | Value |
|---|---|
| **Business job** | Detect low-stock, stockout, and overstock risks; generate replenishment suggestions |
| **Reads** | Inventory levels, sales velocity, inbound stock, purchase orders, platform listing status |
| **Tools / APIs** | `stock_alert` (decision point), inventory read APIs (`GET /api/v1/inventory`, `GET /api/v1/inventory/logs`), order summary, purchase API |
| **Outputs** | Risk level (red/yellow/green), affected SKU list, safety-stock breach evidence, suggested replenishment quantity and timing |
| **Allowed actions** | Create replenishment suggestion, create inventory review task, trigger pipeline to G3 discount_risk_check on red alert |
| **Approval required** | Creating purchase orders, pushing inventory changes to external platforms |
| **Forbidden actions** | Directly create purchase orders without approval, modify inventory counts in database, push inventory sync without Owner OK |
| **Audit fields** | `agent_id: A5`, `sku_id`, `alert_level`, `current_stock`, `suggested_reorder_qty`, `evidence`, `action_id` |
| **Trigger** | Scheduled (every 15 min), pipeline chain (red alert → G3), manual run |
| **Success metric** | Stockout reduction ≥30%, false alert rate ≤15%, replenishment suggestion adoption ≥50% |

## A6 Profit Watch · 利润看护

| Field | Value |
|---|---|
| **Business job** | Monitor real and per-product profit margins, detect margin erosion, trigger listing optimization when thresholds breached |
| **Reads** | Finance profit data, order data, platform fees, settlement data, exchange rates |
| **Tools / APIs** | `profit_watch`/`profit_check` (decision points), finance profit calculation APIs, settlement summary, exchange rate API |
| **Outputs** | Profit margin trend report, at-risk SKU list with margin change, suggested price/relist/remove decision |
| **Allowed actions** | Flag negative-margin products, create listing_optimize recommendation (pipeline to A2), propose price review |
| **Approval required** | Changing sale prices, delisting products, initiating loss-making promotions |
| **Forbidden actions** | Change product prices without approval, delist products without Owner review, approve loss-making promotions |
| **Audit fields** | `agent_id: A6`, `sku_id`, `current_margin`, `threshold`, `trend_direction`, `recommended_action`, `action_id` |
| **Trigger** | Scheduled (every 1 hr), pipeline chain from G3 (block → A6), manual run |
| **Success metric** | Negative-margin detection within 24 hr of change, false alert rate ≤10%, price review adoption ≥40% |

## A7 Compliance Guard · 合规专员

| Field | Value |
|---|---|
| **Business job** | Verify product compliance with platform policies and cross-border regulations; flag certification gaps |
| **Reads** | Product listing content, platform certification requirements, category-specific regulations, listing task data |
| **Tools / APIs** | `compliance_check`/`certification_lookup` (decision points), product APIs, listing API, action-policy API |
| **Outputs** | Compliance pass/fail with evidence, missing certification list, recommended corrective actions |
| **Allowed actions** | Block non-compliant listings from publishing, create compliance review tasks, suggest certification actions |
| **Approval required** | Bypassing a compliance check, approving a listing with unresolved compliance flags |
| **Forbidden actions** | Remove compliance flags from product without verification, approve certifications based on verbal claims only |
| **Audit fields** | `agent_id: A7`, `product_id`, `compliance_status`, `flags_raised`, `certifications_checked`, `action_id` |
| **Trigger** | Scheduled (every 2 hr), on-demand during listing task execution, manual run |
| **Success metric** | Compliance pass-through rate without Owner intervention ≥80%, blocking genuine non-compliance ≥95% |

## A8 Sourcing Profit Analyst · 选品盈利分析

| Field | Value |
|---|---|
| **Business job** | Analyze 1688 sourcing opportunities with profit formula engine; score supplier quality and expected margin |
| **Reads** | 1688 supplier/product data, logistics rate engine (A10), exchange rates, platform fee config, platform historical pricing |
| **Tools / APIs** | `sourcing_recommend` (decision point), sourcing fetch (`POST /api/v1/sourcing/fetch`), sourcing recommendations, 1688 CRUD, logistics API, exchange rate API |
| **Outputs** | Sourcing recommendation with profit breakdown (purchase cost + logistics + platform fee + tariff), quality score, supplier score |
| **Allowed actions** | Propose sourcing recommendations, request 1688 product fetch, update sourcing evaluation rules |
| **Approval required** | Initiating supplier contact, committing to purchase quantities, creating purchase orders |
| **Forbidden actions** | Connect to 1688 suppliers without Owner approval, commit to purchase orders, share pricing strategy externally |
| **Audit fields** | `agent_id: A8`, `product_id`, `sourcing_score`, `estimated_margin`, `supplier_id`, `recommendation`, `action_id` |
| **Trigger** | Manual run via sourcing dashboard (`/sourcing`), on-demand via `POST /api/v1/sourcing/fetch`, scheduled weekly review |
| **Success metric** | Sourcing recommendation accuracy (actual margin vs estimated) within ±5%, recommendation adoption ≥30% |

## A9 Batch Ops · 批量运维

| Field | Value |
|---|---|
| **Business job** | Execute bulk operations: batch price updates, inventory sync, listing updates, and import file validation |
| **Reads** | Bulk import files (CSV/XLSX), existing product/inventory/listing data, import batch records |
| **Tools / APIs** | `batch_price_update`/`batch_inventory_sync`/`batch_listing_update`/`import_validation` (decision points), price APIs, inventory APIs, listing APIs, importbatch API |
| **Outputs** | Batch validation report, per-item success/failure breakdown, summary of changes |
| **Allowed actions** | Validate import files, create batch operation records, execute validated batch updates (low-risk fields only) |
| **Approval required** | Batch price changes affecting revenue, batch inventory sync to external platforms, batch listing publishing |
| **Forbidden actions** | Execute batch operations without dry-run validation, bypass per-item error flags, push batch changes to external platforms without batch-level approval |
| **Audit fields** | `agent_id: A9`, `batch_id`, `operation_type`, `item_count`, `success_count`, `failure_count`, `execution_mode`, `action_id` |
| **Trigger** | Manual via import batch UI, scheduled (off-peak hours for sync tasks) |
| **Success metric** | Batch operation success rate ≥95%, validation accuracy ≥99%, zero phantom operations (correct item count) |

## A10 Logistics Rate Engine · 物流运费引擎

| Field | Value |
|---|---|
| **Business job** | Compare carrier rates, audit shipping bills, evaluate carrier performance, and recommend optimal logistics routes |
| **Reads** | Logistics rate config (YAML), carrier channel data, shipping quotes, shipping bill batches, carrier performance metrics |
| **Tools / APIs** | `carrier_compare`/`shipping_bill_audit`/`carrier_performance`/`logistics_route_opt` (decision points), shipping provider/channel/zone/rule APIs, logistics domain module (four pricing modes) |
| **Outputs** | Carrier comparison table, bill discrepancy report, carrier performance scorecard, recommended route with cost breakdown |
| **Allowed actions** | Propose carrier preference, flag billing discrepancies, create route optimization suggestions |
| **Approval required** | Changing default carrier assignments, approving bill adjustments with carriers, modifying rate configs |
| **Forbidden actions** | Modify carrier rate tables without approval, approve disputed bills, switch carriers without documented reason |
| **Audit fields** | `agent_id: A10`, `carrier_id`, `route_id`, `bill_id`, `current_cost`, `suggested_cost`, `variance`, `action_id` |
| **Trigger** | On-demand via logistics dashboard, manual run, scheduled monthly bill audit, pipeline from new shipment event |
| **Success metric** | Shipping cost savings from route optimization ≥5%, bill discrepancy detection rate ≥90% |

## A11 After-sales Manager · 售后管理

| Field | Value |
|---|---|
| **Business job** | Analyze return patterns, recommend refund decisions, manage disputes, and generate after-sales performance reports |
| **Reads** | After-sales records, customer order data, platform dispute policies, refund history |
| **Tools / APIs** | `return_analysis`/`refund_decision`/`dispute_manage`/`aftersales_report` (decision points), aftersales CRUD APIs, order API, platform integration sync |
| **Outputs** | Return reason analysis, refund recommendation (full/partial/reject), dispute escalation suggestion, after-sales trend report |
| **Allowed actions** | Issue standard refunds within policy, propose dispute responses, categorize return reasons, generate after-sales summary |
| **Approval required** | Refunds exceeding policy limits, goodwill compensation, bypassing standard dispute process |
| **Forbidden actions** | Issue refunds exceeding policy without approval, close disputes without resolution, modify return windows |
| **Audit fields** | `agent_id: A11`, `after_sale_id`, `order_id`, `refund_amount`, `dispute_id`, `resolution`, `action_id` |
| **Trigger** | Scheduled (every 1 hr), on-demand via after-sales dashboard, pipeline from new return event |
| **Success metric** | Auto-refund accuracy ≥90%, dispute win rate maintained or improved, after-sales SLA compliance ≥95% |

## G0 System Health · 系统健康员

| Field | Value |
|---|---|
| **Business job** | Monitor overall Agent system health, detect anomalies in agent behavior, trigger governance interventions |
| **Reads** | Agent decision logs, SPC control limits, anomaly reports, entropy module scores, agent health scores |
| **Tools / APIs** | `system_health` (decision point), entropy agent health API, AgentOS status API, SPC control queries |
| **Outputs** | System health score, anomaly list with severity (warning/critical), list of unhealthy agents, suggested governance actions |
| **Allowed actions** | Pause degraded agents (circuit breaker), trigger agent health recalculation, notify Owner when anomaly >3 (pipeline to G1) |
| **Approval required** | Restarting agents, resetting SPC baselines, disabling agents permanently |
| **Forbidden actions** | Disable agents without Owner knowledge, clear anomaly logs without review, modify SPC control limits arbitrarily |
| **Audit fields** | `agent_id: G0`, `anomaly_type`, `severity`, `affected_agents`, `health_scores`, `circuit_breaker_state`, `action_id` |
| **Trigger** | Scheduled (every 5 min), pipeline (anomaly >3 → G1 dashboard_overview) |
| **Success metric** | Anomaly detection within 5 min of occurrence, false positive rate ≤20%, system uptime impact from agent issues minimized |

## G1 Dashboard · 驾驶舱

| Field | Value |
|---|---|
| **Business job** | Provide agent system overview to Owner, aggregate agent status, decision stats, risk summaries, and performance metrics |
| **Reads** | AgentOS status and work items, trust scores, anomaly reports from G0, action policy rules |
| **Tools / APIs** | `dashboard_overview` (decision point), AgentOS overview API, trustscore API, entropy API, action-policy API |
| **Outputs** | Dashboard summary: total decisions (7d), acceptance rate, pending confirmations, active risks, agent health grid, recent anomalies |
| **Allowed actions** | Aggregate and present Agent data, generate dashboard views, flag areas needing Owner attention |
| **Approval required** | None (read-only aggregation) |
| **Forbidden actions** | Execute Agent actions, modify any system state, change approval policies |
| **Audit fields** | `agent_id: G1`, `dashboard_version`, `data_timestamp` (read-only agent, minimal audit) |
| **Trigger** | Scheduled (every 5 min), pipeline from G0 (anomaly >3), on-demand via Owner dashboard |
| **Success metric** | Dashboard data freshness ≤5 min, actionable insight shown per visit |

## G2 Warehouse & Customs · 仓储专员

| Field | Value |
|---|---|
| **Business job** | Optimize warehouse routing decisions and customs declaration processes for cross-border shipments |
| **Reads** | Inventory levels, order shipping addresses, logistics zone data, customs regulations per country, carrier performance |
| **Tools / APIs** | `warehouse_routing`/`customs_declare` (decision points), inventory API, shipping provider/zone APIs, logistics rate engine (A10) |
| **Outputs** | Optimal warehouse assignment for outbound orders, customs documentation checklist, tariff code suggestion |
| **Allowed actions** | Suggest warehouse routing for orders, propose customs classification, flag documents missing for customs |
| **Approval required** | Changing assigned warehouse mid-fulfillment, declaring non-standard tariff codes, overriding zone assignments |
| **Forbidden actions** | Modify warehouse inventory allocations without approval, declare incorrect tariff codes, override customs holds |
| **Audit fields** | `agent_id: G2`, `order_id`, `warehouse_id`, `tariff_code`, `document_status`, `route_suggestion`, `action_id` |
| **Trigger** | Scheduled (every 1 hr), on-demand per order, pipeline from new fulfillment event |
| **Success metric** | Correct warehouse routing ≥90%, customs clearance pass rate maintained, documentation gap detection ≥95% |

## G3 Discount Risk · 折扣风控

| Field | Value |
|---|---|
| **Business job** | Evaluate discount and promotion proposals for profitability impact and policy compliance |
| **Reads** | Inventory levels (from A5 stock alert), pricing data, profit calculations, promotion history, action policy rules |
| **Tools / APIs** | `discount_risk_check`/`promotion_validation` (decision points), inventory API, price API, finance profit API, action-policy API |
| **Outputs** | Discount risk score, profitability impact estimate, promotion policy compliance check, block/pass/warn decision |
| **Allowed actions** | Block loss-making promotions, propose discount adjustment, create profit_watch task (pipeline to A6 on block) |
| **Approval required** | Approving high-discount promotions exceeding margin threshold, bypassing discount risk block |
| **Forbidden actions** | Approve promotions below cost, override discount policy without Owner, change product prices for promotion |
| **Audit fields** | `agent_id: G3`, `promotion_id`, `discount_percent`, `current_margin`, `risk_score`, `decision`, `action_id` |
| **Trigger** | Pipeline from A5 stock_alert (red), scheduled (every 30 min), on-demand via Owner cockpit |
| **Success metric** | Loss-making promotion prevention ≥95%, discount policy compliance ≥99%, false block rate ≤10% |

## Trust Score Service · 信任分

| Field | Value |
|---|---|
| **Business job** | Recalculate and maintain trust scores for all agents, determining autonomy eligibility and adoption metrics |
| **Reads** | Agent action table (adopted/rejected/failed counts), listing recommendation feedback (adopted/rejected), default agent roster |
| **Tools / APIs** | `recalculate` (decision point), trustscore internal database queries, agent decision audit tables, listing feedback tables |
| **Outputs** | Per-agent trust score (0.0–1.0), autonomy level recommendation, score breakdown by dimension |
| **Allowed actions** | Recalculate scores, update autonomy levels based on score thresholds, record agent feedback events |
| **Approval required** | Manually overriding trust scores, lowering autonomy thresholds, resetting score history |
| **Forbidden actions** | Allow low-trust agents to execute high-risk actions, delete score history without audit trail |
| **Audit fields** | `service: trustscore`, `agent_id`, `previous_score`, `new_score`, `adopted_count`, `rejected_count`, `failed_count`, `autonomy_level` |
| **Trigger** | Scheduled (every 1 hr), on-demand from Owner dashboard, triggered by `RecordAgentFeedback` call |
| **Success metric** | Trust score correlates with Owner adoption (r ≥ 0.5), recalculation completes <5s, no high-autonomy assigned to low-trust agents |

## Entropy Defense · 自净化系统

| Field | Value |
|---|---|
| **Business job** | Self-cleansing: monitor agent behavior for statistical anomalies, apply SPC control limits, detect rule degradation, run health checks |
| **Reads** | Agent health scores, SPC control limit tables, personal rule health, agent decision logs |
| **Tools / APIs** | `defend` (decision point), entropy internal (agent_health, defenses, spc_control), rule management, observability anomaly reporting |
| **Outputs** | Entropy summary (overrides detected, unhealthy agents, SPC boundary breaches), agent health score, rule health distribution |
| **Allowed actions** | Flag unhealthy agents, recommend rule pauses, alert on SPC boundary breaches, log defense actions |
| **Approval required** | Pausing rules permanently, overriding SPC limits, disabling defense mechanisms |
| **Forbidden actions** | Pause rules without audit trail, silently clear anomaly records, disable auto-defense without Owner acknowledgment |
| **Audit fields** | `component: entropy`, `agent_id`, `health_score`, `rules_paused`, `spc_breaches`, `defense_action`, `overrides_detected` |
| **Trigger** | Scheduled (every 6 hr), on-demand via entropy dashboard |
| **Success metric** | Unhealthy agents detected within 1 cycle, false positive ≤15%, SPC breach detection ≥90% |

## M1 Metabolism · 代谢评分引擎

| Field | Value |
|---|---|
| **Business job** | Score and excrete stale/low-quality events, data artifacts, and abandoned workflows to prevent system data decay |
| **Reads** | Event outbox table, metabolism scoring config, event age/freshness, semantic scorer (LLM for gray-zone classification 0.4–0.75) |
| **Tools / APIs** | `excretion_scoring` (decision point), metabolism internal (service.go — TTL=7d), event outbox adapter, database queries for pending events |
| **Outputs** | Per-event excretion score (0.0–1.0), excretable TTL-exceeded events, gray-zone events needing semantic scoring, dry-run report |
| **Allowed actions** | Score events, mark events as excretable, run dry-run without side effects, suggest cleanup of abandoned workflows |
| **Approval required** | Actually deleting/excreting marked events (Owner review of dry-run), modifying excretion thresholds or TTL |
| **Forbidden actions** | Delete events without dry-run preview, modify event payloads, change event producer data |
| **Audit fields** | `agent_id: M1`, `event_id`, `score`, `excretable`, `reason`, `dry_run` flag |
| **Trigger** | Scheduled (every 1 hr), manual via `POST /api/v1/metabolism/run`, dry-run via `POST /api/v1/metabolism/dry-run` |
| **Success metric** | Data decay rate reduced (fewer orphaned events ≥7 days old), excretion accuracy ≥90%, zero false-excretion of active data |
