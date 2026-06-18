# 凌镜 LingMirror — 文档索引

> 跨境電商 AI Agent 运营平台（技术名：MultiSell）
> 最后更新：2026-06-18

---

## 📋 项目概览

| 文档 | 说明 |
|------|------|
| [项目状态](PROJECT_STATUS.md) | 当前完成进度、功能清单 |
| [产品愿景与 MVP](PRODUCT_VISION_AND_MVP.md) | 产品定位、第一可用版本定义 |
| [路线图](ROADMAP.md) | Phase 0–8 详细阶段规划 |
| [项目治理与 Agent 协作规范](PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md) | 协作规则、验收标准 |

## 🧠 知识库 (Knowledge Base)

### AI Agent 设计

| 文档 | 说明 |
|------|------|
| [Hermes 自演化 Agent 设计](aiagent/hermes-self-evolving-agent-design.md) | 核心架构：10 个 Agent、4 个演化阶段 |
| [Hermes 熵值管理系统](aiagent/hermes-entropy-management-system.md) | 自净化、SPC 控制、规则健康评分 |
| [跨境电商 AI Agent 业务场景需求](aiagent/跨境电商AI_Agent业务场景需求文档.md) | 业务场景分析 |
| [跨境电商 AI Agent 深度调研报告](aiagent/跨境电商AI_Agent深度调研报告.md) | 行业调研 |
| [跨境电商岗位与业务场景调研](aiagent/跨境电商岗位与业务场景调研报告.md) | 岗位分析 |
| [跨境电商岗位业务场景调研](aiagent/跨境电商岗位业务场景调研报告.md) | 岗位分析（续） |
| [跨境电商行业岗位职责体系深度研究](aiagent/跨境电商行业岗位职责体系深度研究报告.md) | 电商岗位职责 |
| [跨境電商角色分析](aiagent/cross-border-ecommerce-roles.md) | 角色定义 |
| [最终集成方案](aiagent/final-integrated-solution.md) | 全系统集成方案 |
| [AI Agent 可行开发规格](AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md) | 技术可行性、开发规格 |
| [自净化系统设计规格书](跨境电商AI_Agent_自净化系统设计规格书.md) | 自净化规格 |

### 竞品与市场

| 文档 | 说明 |
|------|------|
| [竞品调研 2026-06](research/competitor-research-2026-06.md) | 竞品分析 |
| [竞品来源](research/competitor-sources-2026-06.md) | 信息来源 |
| [凌镜能力决策](research/lingmirror-capability-decisions-2026-06.md) | 能力选择决策 |

## 🏗️ 开发文档 (Development)

| 文档 | 说明 |
|------|------|
| [AgentOS 开发文档](AGENTOS_DEVELOPMENT_GUIDE.md) | AgentOS 团队系统工程 |
| [AgentOS Phase 1 实现计划](superpowers/plans/2026-06-18-lingmirror-agentos-phase-1.md) | 第一版工程骨架总控台 |
| [核心模块架构](PERMISSIONS_AND_AUDIT.md) | 权限系统 + 审计日志设计 |
| [物流与运费 PRD](LOGISTICS_AND_SHIPPING_PRD.md) | 物流需求 |
| [物流与运费技术规格](LOGISTICS_SHIPPING_TECH_SPEC.md) | 物流技术实现 |
| [报价规则示例](LOGISTICS_QUOTE_RULE_EXAMPLES.md) | 运费报价案例 |
| [平台费用规则方案](platform-fee-rules-plan.md) | 平台费用设计 |

### Demo & 验收

| 文档 | 说明 |
|------|------|
| [演示场景](DEMO_SCENARIO.md) | 完整演示流程 |
| [验收报告](DEMO_ACCEPTANCE_REPORT.md) | 验收结果 |
| [订单导入冒烟检查](ORDER_IMPORT_SMOKE_CHECKLIST.md) | CSV 订单导入测试清单 |

### Demo 数据

- [`docs/demo-data/order_import_demo.csv`](demo-data/order_import_demo.csv) — 订单导入示例 CSV
- [`docs/demo-data/platform_settlement_demo.csv`](demo-data/platform_settlement_demo.csv) — 平台结算示例 CSV
- [`docs/demo-data/shipping_bill_demo.csv`](demo-data/shipping_bill_demo.csv) — 运费账单示例 CSV

## 🗄️ 历史计划 (Archived)

已完成计划的执行记录：`docs/superpowers/plans/`

| 日期 | 计划 |
|------|------|
| 06-13 | [Auth + RBAC + 审计](superpowers/plans/2026-06-13-auth-rbac-audit.md) |
| 06-13 | [稳定化路线图](superpowers/plans/2026-06-13-multisell-stabilization-roadmap.md) |
| 06-13 | [订单管理](superpowers/plans/2026-06-13-order-management.md) |
| 06-13 | [平台发布适配器](superpowers/plans/2026-06-13-platform-listing-adapters.md) |
| 06-13 | [RBAC 审计上线](superpowers/plans/2026-06-13-reasonix-rbac-audit-rollout.md) |
| 06-13 | [运行与迁移](superpowers/plans/2026-06-13-runtime-and-migrations.md) |
| 06-14 | [物流属性 Phase 1](superpowers/plans/2026-06-14-logistics-attributes-phase-1.md) |
| 06-14 | [物流文档计划](superpowers/plans/2026-06-14-logistics-shipping-documentation-plan.md) |
| 06-14 | [订单运费快照利润](superpowers/plans/2026-06-14-order-shipping-snapshot-profit.md) |
| 06-14 | [物流 Phase 2 计算](superpowers/plans/2026-06-14-shipping-phase-2-calculation.md) |
| 06-15 | [批量上架前决策](superpowers/plans/2026-06-15-batch-prelisting-decision.md) |
| 06-15 | [竞品调研框架](superpowers/plans/2026-06-15-competitor-research-framework.md) |
| 06-15 | [决策到上架任务](superpowers/plans/2026-06-15-decision-to-listing-task.md) |
| 06-15 | [Excel 批量决策](superpowers/plans/2026-06-15-excel-batch-prelisting-decision.md) |
| 06-15 | [凌镜品牌](superpowers/plans/2026-06-15-lingmirror-branding.md) |
| 06-15 | [凌镜图标集成](superpowers/plans/2026-06-15-lingmirror-icon-integration.md) |
| 06-15 | [马帮 ERP 对标路线](superpowers/plans/2026-06-15-mabang-erp-benchmark-roadmap.md) |
| 06-15 | [下阶段执行路线图](superpowers/plans/2026-06-15-next-stage-execution-roadmap.md) |
| 06-15 | [订单库存闭环](superpowers/plans/2026-06-15-order-inventory-closure.md) |
| 06-15 | [平台费用规则](superpowers/plans/2026-06-15-platform-fee-rules.md) |
| 06-15 | [商品列表重构](superpowers/plans/2026-06-15-product-list-rebuild.md) |
| 06-15 | [商品运费预填](superpowers/plans/2026-06-15-product-shipping-prefill.md) |
| 06-15 | [项目稳定化与 MVP 交接](superpowers/plans/2026-06-15-project-stabilization-and-mvp-handoff.md) |
| 06-15 | [运费手动计算器](superpowers/plans/2026-06-15-shipping-manual-calculator.md) |
| 06-15 | [运费表导入](superpowers/plans/2026-06-15-shipping-rate-import.md) |
| 06-18 | [AgentOS Phase 1](superpowers/plans/2026-06-18-lingmirror-agentos-phase-1.md) |
| 06-16 | [订单导入运营链路](superpowers/plans/2026-06-16-order-import-operational-chain.md) |

## 🆕 功能需求 (Feature Requests)

> 新功能需求请添加到 `docs/features/` 目录，使用标准模板。

- [功能需求模板](features/TEMPLATE.md)

---

## 时间线

→ 详见 [时间线](TIMELINE.md)
