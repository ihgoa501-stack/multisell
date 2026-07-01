# LingMirror Agent Responsibility Cards

> Created: 2026-06-30 | Phase: P1
> Each card defines a LingMirror Agent as a **business role**, not a code name.
> High-risk actions marked **approval-required**.

---

## A1 — 选品助理 (Product Scout)

| Field | Value |
|---|---|
| **Business job** | 为 Owner 发现值得上架的新商品，多维度评分排序 |
| **Input data** | category, marketplace, candidate product list, search volume, trend data |
| **Tools / APIs** | `product_scout` (scoring ranking), `market_analysis` (market summary); ToolRegistry |
| **Outputs** | scored & ranked candidate list (Top-20), market analysis summary |
| **Allowed actions** | create candidate recommendation, update product scout score |
| **Approval required** | 创建采购订单、上架发布 |
| **Forbidden actions** | 直接创建采购订单或上架商品 |
| **Audit fields** | agent_id, trigger, category, marketplace, top_candidates, scoring_formula |
| **Schedule / Trigger** | manual, event‑driven (candidate.proposed) |
| **Success metrics** | Owner adoption rate of scout recommendations, lower false positive rate |

---

## A2 — 商品优化师 (Listing Optimizer)

| Field | Value |
|---|---|
| **Business job** | 优化商品标题、卖点、搜索关键词，提升曝光和转化率 |
| **Input data** | product info (title, description, specs), competitor data, seed keywords |
| **Tools / APIs** | `listing_optimize` (title/bullets/search terms), `keyword_research` (keyword expansion); ToolRegistry |
| **Outputs** | optimized listing (title, bullets, search terms, keyword research), actionable suggestions |
| **Allowed actions** | generate optimized listing draft, suggest keyword additions |
| **Approval required** | 执行上架、修改已上架商品标题/关键词 |
| **Forbidden actions** | 直接发布或修改已上架的 Listing |
| **Audit fields** | agent_id, product_id, original_content, optimized_content, suggestion_id |
| **Schedule / Trigger** | manual, triggered by A6 profit_watch (loss/threshold) |
| **Success metrics** | listing optimization adoption rate, conversion improvement |

---

## A3 — 广告分析师 (Ad Analyst)

| Field | Value |
|---|---|
| **Business job** | 分析广告投放效果，提出 ACOS 优化和投放策略建议 |
| **Input data** | ad spend, sales, impressions, clicks, ACOS, campaign structure |
| **Tools / APIs** | `acos_analysis` (cost analysis), `ad_optimization` (strategy); internal API for ad data |
| **Outputs** | ACOS breakdown, ad efficiency analysis, optimization suggestions |
| **Allowed actions** | produce analysis report, suggest ad budget adjustment |
| **Approval required** | 调整广告预算、暂停/启动广告活动 |
| **Forbidden actions** | 直接修改广告投放设置、操作广告账户资金 |
| **Audit fields** | agent_id, campaign_id, original_budget, suggested_budget, analysis_id |
| **Schedule / Trigger** | scheduled (1 hr), manual |
| **Success metrics** | ACOS reduction, ad recommendation adoption rate |

---

## A4 — 客服助理 (Customer Service)

| Field | Value |
|---|---|
| **Business job** | 自动回复买家常见问题、识别意图并分派给人工客服 |
| **Input data** | incoming message text, conversation history, platform, buyer profile |
| **Tools / APIs** | `auto_reply` (auto response), `intent_classify` (intent routing); Support API |
| **Outputs** | auto‑reply draft, intent classification, escalation recommendation |
| **Allowed actions** | reply to low‑risk inquiries, tag conversation intent, suggest escalation |
| **Approval required** | 发送退款、改价、补发等涉及资金/订单的回复 |
| **Forbidden actions** | 发送涉及退款/改价/补发的内容 |
| **Audit fields** | agent_id, conversation_id, intent, reply_draft, action_taken |
| **Schedule / Trigger** | scheduled (5 min), message.incoming event |
| **Success metrics** | auto‑reply accuracy, first‑response time reduction, escalation rate |

---

## A5 — 库存助理 (Inventory Alert)

| Field | Value |
|---|---|
| **Business job** | 监控库存健康度，红/黄/绿三级预警，建议补货数量和物流渠道 |
| **Input data** | SKU code, sellable/locked/in‑transit stock, multi‑period sales, lead time, MOQ, safety stock days |
| **Tools / APIs** | `stock_alert` (status + replenish), `replenishment_plan` (quantity), `logistics_choice` (channel); Inventory API |
| **Outputs** | stock status (red/yellow/green), days of cover, replenish suggestion, logistics recommendation |
| **Allowed actions** | create replenishment suggestion, create inventory review task |
| **Approval required** | 创建采购订单、调整库存数据、同步至外部平台 |
| **Forbidden actions** | 直接创建采购订单或修改库存数据 |
| **Audit fields** | agent_id, trigger, sku_id, current_stock, recommendation, action_id |
| **Schedule / Trigger** | scheduled (15 min), inventory.low event; triggers G3 on red alert |
| **Success metrics** | fewer stockouts, approval adoption rate, lower false alert rate |

---

## A6 — 利润看护 (Profit Watch)

| Field | Value |
|---|---|
| **Business job** | 监控 SKU 级利润，检测亏损风险，提出定价和成本优化建议 |
| **Input data** | SKU code, selling price, cost price, platform, country, fees (shipping/platform/ad), refund rate |
| **Tools / APIs** | `profit_check` / `profit_watch` (margin analysis), `cost_optimization` (cost structure); PlatformFee API, SKU API |
| **Outputs** | per‑unit profit, gross margin, fee breakdown, loss risk, price/cost suggestions |
| **Allowed actions** | flag loss‑risk SKUs, suggest price adjustment, alert for cost review |
| **Approval required** | 调整售价、修改成本数据、执行定价变更 |
| **Forbidden actions** | 直接修改商品价格或成本 |
| **Audit fields** | agent_id, sku_id, selling_price, cost_price, margin, suggestion, action_id |
| **Schedule / Trigger** | scheduled (1 hr), triggered by G3 discount_risk_check (block); triggers A2 on threshold |
| **Success metrics** | early loss detection rate, price suggestion adoption rate, margin improvement |

---

## A7 — 合规专员 (Compliance Guard)

| Field | Value |
|---|---|
| **Business job** | 检测商品合规风险，检查资质认证，标记不合规商品 |
| **Input data** | product_id, category, platform, country, certifications, product attributes |
| **Tools / APIs** | `compliance_check` (risk scan), `certification_lookup` (cert requirements); Compliance API |
| **Outputs** | compliance status (pass/warning/fail), risk level, evidence summary |
| **Allowed actions** | flag compliance risk, suggest certification action, suppress reviewed findings |
| **Approval required** | 下架不合规商品、修改商品违禁属性 |
| **Forbidden actions** | 直接下架商品或修改合规相关属性 |
| **Audit fields** | agent_id, product_id, compliance_status, risk_level, evidence, action_id |
| **Schedule / Trigger** | scheduled (2 hr), manual |
| **Success metrics** | compliance check coverage, false positive rate, regulatory risk reduction |

---

## A8 — 选品盈利分析 (Sourcing Profit Engine)

| Field | Value |
|---|---|
| **Business job** | 评估候选商品的全链路利润前景，给出上架推荐或阻止理由 |
| **Input data** | product candidate info, procurement cost, logistics cost, platform fee, tariff, sales estimate |
| **Tools / APIs** | `sourcing_recommend` (profit forecast); `/api/v1/sourcing/fetch`, `/api/v1/sourcing/recommendations` |
| **Outputs** | profit forecast, sourcing recommendation (list/block/review), risk note |
| **Allowed actions** | produce sourcing evaluation, recommend or block candidate |
| **Approval required** | 创建上架任务、启动采购流程 |
| **Forbidden actions** | 直接创建采购订单或上架任务 |
| **Audit fields** | agent_id, candidate_id, profit_forecast, recommendation, evidence |
| **Schedule / Trigger** | manual, on‑demand via `/api/v1/sourcing/fetch` |
| **Success metrics** | recommendation → listing conversion, forecast accuracy vs actual |

---

## A9 — 批量运维 (Batch Ops)

| Field | Value |
|---|---|
| **Business job** | 执行批量商品/库存/订单更新操作，导入数据验证 |
| **Input data** | CSV/XLSX upload(s), operation type (product/order/inventory), validation rules |
| **Tools / APIs** | `batch_price_update`, `batch_inventory_sync`, `batch_listing_update`, `import_validation` |
| **Outputs** | batch operation result (success/fail rows), validation report |
| **Allowed actions** | validate imported data, execute bulk updates on internal state |
| **Approval required** | 批量修改价格、批量同步库存到外部平台 |
| **Forbidden actions** | 批量修改价格/库存/上架状态不经过审批 |
| **Audit fields** | agent_id, batch_id, operation_type, success_count, fail_count, summary |
| **Schedule / Trigger** | manual, importbatch.created event |
| **Success metrics** | batch success rate, validation accuracy, processing speed |

---

## A10 — 物流运费引擎 (Logistics Rate Engine)

| Field | Value |
|---|---|
| **Business job** | 比较物流商费率，审核运费账单，优化物流线路选择 |
| **Input data** | package dimensions/weight, origin/destination, carrier rates, shipment history |
| **Tools / APIs** | `carrier_compare` (rate comparison), `shipping_bill_audit` (bill review), `carrier_performance`, `logistics_route_opt` |
| **Outputs** | carrier recommendation, bill audit findings, route optimization suggestion |
| **Allowed actions** | recommend carrier, flag billing anomalies, suggest route change |
| **Approval required** | 切换物流商、确认运费账单、修改物流配置 |
| **Forbidden actions** | 直接切换物流商或确认未审核的账单 |
| **Audit fields** | agent_id, shipment_id, carriers_compared, recommendation, audit_findings |
| **Schedule / Trigger** | manual, shipping.bill.created event |
| **Success metrics** | shipping cost reduction, billing audit accuracy, carrier reliability score |

---

## A11 — 售后管理 (Aftersales Management)

| Field | Value |
|---|---|
| **Business job** | 分析退货原因，提供退款决策建议，管理纠纷与售后报告 |
| **Input data** | return request details, order history, platform policy, customer communication |
| **Tools / APIs** | `return_analysis` (return cause analysis), `refund_decision` (refund recommendation), `dispute_manage`, `aftersales_report` |
| **Outputs** | return cause breakdown, refund decision (approve/reject/escalate), dispute summary |
| **Allowed actions** | suggest refund decision, categorize return reason, flag abuse pattern |
| **Approval required** | 执行退款操作、修改订单售后状态 |
| **Forbidden actions** | 直接执行退款、修改订单状态 |
| **Audit fields** | agent_id, return_id, decision, reason, refund_amount, action_id |
| **Schedule / Trigger** | manual, aftersales.request.created event |
| **Success metrics** | refund decision accuracy, dispute win rate, return rate reduction |

---

## G0 — 系统健康员 (System Health)

| Field | Value |
|---|---|
| **Business job** | 监控系统全局健康状态，检测异常指标，触发告警和升级 |
| **Input data** | service metrics, error rates, scheduler health, agent execution stats, anomaly count |
| **Tools / APIs** | `system_health` (health evaluation); Observability API, Metrics API |
| **Outputs** | system health score, anomaly list, escalation recommendation |
| **Allowed actions** | flag degraded services, suggest restart/remediation, escalate to G1 |
| **Approval required** | 重启服务、修改系统配置 |
| **Forbidden actions** | 自行重启服务或修改系统配置 |
| **Audit fields** | agent_id, health_score, anomalies, escalation |
| **Schedule / Trigger** | scheduled (5 min); when anomalies > 3, triggers G1 |
| **Success metrics** | anomaly detection rate, false positive rate, MTTR improvement |

---

## G1 — 驾驶舱 (Dashboard Overview)

| Field | Value |
|---|---|
| **Business job** | 汇总系统全局概览数据，为 Owner 提供经营总控台数据 |
| **Input data** | risk summary, agent suggestions, platform sync status, pending approvals |
| **Tools / APIs** | `dashboard_overview` (aggregated view); Owner API, Dashboard API |
| **Outputs** | aggregated business overview, risk count, action‑needed summary |
| **Allowed actions** | render read‑only dashboard, summarize state |
| **Approval required** | 不适用（只读） |
| **Forbidden actions** | 任何写操作（只读 Agent） |
| **Audit fields** | agent_id, snapshot_ts, risk_count, suggestion_count |
| **Schedule / Trigger** | scheduled (5 min), triggered by G0 on anomaly > 3 |
| **Success metrics** | dashboard freshness, Owner action rate from dashboard |

---

## G2 — 仓储专员 (Warehouse & Customs)

| Field | Value |
|---|---|
| **Business job** | 分析仓储路由选择，处理报关清关建议，优化库存分布 |
| **Input data** | warehouse inventory levels, customs regulations, destination country, transit times |
| **Tools / APIs** | `warehouse_routing` (routing optimization), `customs_declare` (customs guidance); Logistics API, Warehouse API |
| **Outputs** | warehouse routing suggestion, customs documentation checklist, cost‑time tradeoff |
| **Allowed actions** | suggest warehouse routing, flag customs risk, generate customs doc checklist |
| **Approval required** | 修改仓储配置、执行报关操作 |
| **Forbidden actions** | 直接修改仓储配置或提交报关材料 |
| **Audit fields** | agent_id, shipment_id, routing_suggestion, customs_risk, checklist |
| **Schedule / Trigger** | scheduled (1 hr), manual |
| **Success metrics** | customs clearance time reduction, warehouse utilization improvement |

---

## G3 — 折扣风控 (Discount Risk Control)

| Field | Value |
|---|---|
| **Business job** | 检测折扣、促销活动的利润风险和违规风险，阻止亏损上架 |
| **Input data** | promotion details, discount rate, product margin, platform policy, historical campaign data |
| **Tools / APIs** | `discount_risk_check` (risk evaluation), `promotion_validation` (policy compliance); Price API, Platform API |
| **Outputs** | risk assessment (block/pass/review), margin impact estimate, policy compliance status |
| **Allowed actions** | block high‑risk discount, flag policy violation, pass safe promotions |
| **Approval required** | 审批高风险折扣、修改促销价格 |
| **Forbidden actions** | 直接修改促销价格或折扣率 |
| **Audit fields** | agent_id, promotion_id, discount_rate, risk_level, decision, margin_impact |
| **Schedule / Trigger** | scheduled (30 min), triggered by A5 stock_alert (red); triggers A6 on block |
| **Success metrics** | false block rate, prevented margin‑eroding promotions, compliance pass rate |

---

## trustscore — 信任分引擎 (Trust Score)

| Field | Value |
|---|---|
| **Business job** | 计算 Agent 信任分，基于历史决策质量、采纳率、准确率 |
| **Input data** | agent action history, approval/adoption records, accuracy metrics, violation log |
| **Tools / APIs** | trust score CRUD API, action policy API |
| **Outputs** | per‑agent trust score (0‑100), score trend, gating recommendation |
| **Allowed actions** | recalculate scores, recommend autonomy‑level adjustment |
| **Approval required** | 提高 Agent 自主化等级 |
| **Forbidden actions** | 未经记录直接修改 Agent 自主化等级 |
| **Audit fields** | agent_id, score_before, score_after, calculation_formula, timestamp |
| **Schedule / Trigger** | scheduled (1 hr) |
| **Success metrics** | score correlation with actual decision quality, autonomy‑level accuracy |

---

## entropy — 熵防御 (Entropy Defense)

| Field | Value |
|---|---|
| **Business job** | 自净化系统：检测 Agent 规则老化、退化、冲突，执行 TTL 清理和预算约束 |
| **Input data** | agent rule database, action logs, timestamps, budget usage, decay metrics |
| **Tools / APIs** | entropy metrics API, TTL sweeper, budget enforcer, decay scheduler, merge detector, regret analyzer, rule health scorer, SPC controller |
| **Outputs** | entropy summary, stale rule cleanup list, budget violations, health scores |
| **Allowed actions** | drop stale TTL‑expired rules, flag budget overruns, report rule health |
| **Approval required** | 清理仍然活跃但过期的规则、调整预算上限 |
| **Forbidden actions** | 删除正在被引用的规则、全局禁用 Agent |
| **Audit fields** | agent_id, rule_id, action (sweep/budget/decay/merge), outcome |
| **Schedule / Trigger** | scheduled (6 hr) |
| **Success metrics** | rule health score improvement, stale rule reduction, budget violation decrease |

---

## M1 — 代谢评分 (Metabolism / Excretion Scoring)

| Field | Value |
|---|---|
| **Business job** | 评估 Agent 代谢健康度，标记低效/休眠 Agent 建议清理或升级 |
| **Input data** | agent activity logs, execution count, success rate, last active timestamp, trust score |
| **Tools / APIs** | `excretion_scoring` (metabolism evaluation); Metabolism API, Agent Activity API |
| **Outputs** | metabolism score per agent, excretion recommendation (keep/warn/deprecate) |
| **Allowed actions** | flag low‑metabolism agents, suggest deprecation or upgrade |
| **Approval required** | 废弃 Agent、移除 Agent 注册 |
| **Forbidden actions** | 自行注销或移除 Agent |
| **Audit fields** | agent_id, metabolism_score, activity_count, recommendation, last_active |
| **Schedule / Trigger** | scheduled (1 hr) |
| **Success metrics** | early detection of degenerating agents, false deprecation rate |

---

## Agent Pipeline Chain

Events chain referenced across agents:

```
A5 stock_alert (red)              → G3 discount_risk_check
G3 discount_risk_check (block)     → A6 profit_watch
A6 profit_watch (loss/threshold)   → A2 listing_optimize
G0 system_health (anomaly > 3)    → G1 dashboard_overview
```

All scheduled agents: G0 / A4 / G1 / A5 / G3 / A6 / A3 / G2 / A7 / M1 / trustscore / entropy

## Risk Level Summary

| Agent | Highest Risk Action | Approval Required |
|-------|-------------------|-------------------|
| A1 | 创建采购订单 / 上架 | Yes |
| A2 | 修改已上架 Listing | Yes |
| A3 | 调整广告预算 | Yes |
| A4 | 退款 / 改价回复 | Yes |
| A5 | 创建采购订单 | Yes |
| A6 | 调整商品售价 | Yes |
| A7 | 下架商品 | Yes |
| A8 | 创建上架任务 | Yes |
| A9 | 批量修改价格/库存 | Yes |
| A10 | 切换物流商 | Yes |
| A11 | 执行退款 | Yes |
| G0 | 重启服务 | Yes |
| G1 | 不适用（只读） | No |
| G2 | 修改仓储配置 / 报关 | Yes |
| G3 | 审批高风险折扣 | Yes |
| trustscore | 修改自主化等级 | Yes |
| entropy | 清理活跃规则 / 调整预算 | Yes |
| M1 | 废弃 Agent | Yes |

## Forbidden Actions (All Agents)

The following actions are **forbidden for every Agent** unless explicitly approved by Owner via written policy:

- 直接修改商品价格、库存、订单状态
- 直接执行退款
- 直接发布商品到外部平台
- 修改权限、凭证、RBAC 设置
- 销毁或删除业务数据
- 绕过审批系统执行高风险的 Action
- 修改 Agent 自主化等级
- 执行未经授权的系统配置变更
- 操作资金、结算、平台费用数据
