# 凌镜 LingMirror — 文档索引

> 跨境電商 AI Agent 运营平台（技术名：MultiSell）
> 更新日期：2026-07-07

---

## 🚀 快速入门 (Quick Start)

| 文档 | 说明 |
|------|------|
| [入门教程](tutorial-getting-started.md) | 🆕 从零搭建开发环境到创建第一个商品——新人首选 |
| [端到端教程](tutorial-first-workflow.md) | 🆕 从安装到运行业务闭环，30 分钟端到端体验 |
| [API 快速参考](reference-api-quick.md) | 🆕 路由、认证、响应格式、中间件栈速查 |
| [模块目录](reference-module-catalog.md) | 🆕 全部 60+ 后端领域模块一览 |
| [配置参考](reference-configuration.md) | 🆕 config.yaml + 环境变量完整说明 |
| [Agent Pipeline 解释](explanation-agent-pipeline.md) | 🆕 Agent 间如何通过 EventBus 通信和协作 |
| [AI & Agent 系统参考](reference-ai-agent-system.md) | 🆕 LLM 编排、Agent 注册表、AgentOS 控制台、Trace 系统 |

## 🛠️ 操作指南 (How-to)

| 文档 | 说明 |
|------|------|
| [添加新领域模块](howto-add-domain-module.md) | 🆕 添加完整 CRUD 模块 step-by-step |
| [配置平台集成](howto-platform-integrations.md) | 🆕 接入 Ozon / Shopee API |
| [创建自定义 Agent 规则](howto-agent-rules.md) | 🆕 控制 Agent 决策边界和触发条件 |
| [运行测试与验证](howto-test-and-verify.md) | 🆕 Go 测试、前端测试、E2E |
| [配置与部署](howto-deploy.md) | 🆕 Docker 生产部署、Nginx/Caddy |
| [使用 WebSocket 流式更新](howto-websocket.md) | 🆕 连接 /ws 端点接收实时数据 |
| [执行第一个业务闭环](howto-first-business-loop.md) | 🆕 候选商品→完整度→利润→审批→上架的操作指南 |

## 🧠 解释 (Explanation)

| 文档 | 说明 |
|------|------|
| [两个核心业务闭环](explanation-business-loops.md) | 🆕 商品→上架与订单→履约→结算两个主循环的设计 |
| [领域模块架构](explanation-domain-architecture.md) | 🆕 60+ 模块的组织方式、协作模式和依赖关系 |

## 📋 项目概览

当前执行口径以 [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) 和 `docs/governance/` 为准。
下表中的历史计划、蓝图和研究材料保留作参考，不覆盖当前优先级。

| 文档 | 说明 |
|------|------|
| [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) | 🆕 当前执行口径：产品方向、AgentOS 安全优先级、文档清理规则 |
| [设计系统](DESIGN.md) | 🆕 UI 设计规范：色彩、字体、间距、动画、组件风格、无障碍 |
| [项目状态](PROJECT_STATUS.md) | 当前版本、验证状态、更新历史 |
| [AI 无人公司长远战略愿景](LONG_TERM_VISION_AND_STRATEGY.md) | 🆕 系统终极形态：“全自动驾驶”无人公司操作系统的四道门槛与落地路线图 |
| [模块目录](reference-module-catalog.md) | 🆕 模块、API 路由和前端页面唯一事实源 |
| [一人 Agent 公司长期作战地图](ONE_PERSON_AGENT_COMPANY_STRATEGY.md) | 长期方向、阶段路线、Owner 控制规则和 Agent 提案检查表 |
| [7 天一人 Agent 公司 MVP 计划](7_DAY_AGENT_COMPANY_MVP_PLAN.md) | Day 1-7 完整开发计划，5 条并行线 |
| [每日战情板](7_DAY_BATTLE_BOARD.md) | 7 天每日进度追踪 |
| [Agent Commerce OS 完整蓝图](LINGMIRROR_AGENT_COMMERCE_OS_BLUEPRINT.md) | 完整产品定位、系统分层、开发路径 |
| [产品愿景与 MVP](PRODUCT_VISION_AND_MVP.md) | 产品定位、第一可用版本定义 |
| [路线图](ROADMAP.md) | Phase 0–8 详细阶段规划 |
| [项目治理与 Agent 协作规范](PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md) | 协作规则、验收标准 |
| [Active Stack Policy](ACTIVE_STACK_POLICY.md) | Go + Next 活跃技术栈和旧栈边界 |
| [AIOS 基础设施架构](aios-architecture.md) | AIOS 内核层 11 个基础设施模块的接口契约与实现路径 |
| [经营闭环审计](ONE_PERSON_AGENT_COMPANY_LOOP_AUDIT.md) | 8 环节经营闭环当前可用性评估 |
| [每日验收日志](DAILY_ACCEPTANCE_LOG.md) | Day 0-7 每日交付验收记录 |
| [Claude Code 工作流](CLAUDE_CODE_AGENT_WORKFLOW.md) | Subagent 定义、并行策略、日终验证 |

### 当前事实源与历史报告边界

| 文档 | 当前用途 |
|------|----------|
| [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) | 当前产品方向和下一阶段优先级 |
| [双产品十星架构设计](designs/dual-product-cathedral.md) | 🆕 CEO Review 通过的 Intelligence + Portfolio Launch OS 双产品边界、Phase 2 Gate、协议、安全、资金与交付计划 |
| [AI-Native AgentOS 长期愿景与架构方向](AI_NATIVE_DEVELOPMENT_PLAN.md) | 🆕 面向多领域可信自进化 AgentOS 的长期愿景；不覆盖当前执行优先级 |
| [AI-Native AgentOS 执行路径规格](specs/2026-07-09-ai-native-agentos-execution-path.md) | 🆕 将长期愿景拆成底层、上层、AIOS 和验证门禁的可执行规划 |
| [AI-Native AgentOS 执行计划](../tasks/plan.md) | 🆕 Product Loop E2E -> Action Gate -> CI/E2E -> Cockpit 的 canonical 阶段计划 |
| [AI-Native AgentOS 任务清单](../tasks/todo.md) | 🆕 可逐项执行的 Product Loop / Action Gate / E2E canonical 任务列表 |
| [商品出海决策与执行层](features/product-export-decision-execution-layer.md) | 🆕 未来半年商品出海业务层开发总纲：目标、阶段、架构 and 验收标准 |
| [验收门禁](ACCEPTANCE_GATE.md) | 🆕 Dev Done / Test Green / Business Verified / Beta Accepted 的完成定义和证据要求 |
| [验收矩阵](ACCEPTANCE_MATRIX.md) | 🆕 环境、角色、业务闭环、高风险动作和运行时证据矩阵 |
| [已知问题台账](KNOWN_ISSUES.md) | 🆕 红灯问题 owner / deadline / impact 跟踪，防止长期合理化 |
| [实现路线图](implementation-roadmap.md) | 🆕 19 个 Open Issue × 6 轮迭代的整体路线图 |
| [Iteration 1 Spec: 安全速赢 + 数据地基](specs/iteration-1-security-quick-wins.md) | 🆕 #280 JWT_SECRET / #281 DB备份 / #130 汇率硬编码 |
| [项目状态](PROJECT_STATUS.md) | 当前事实快照、当前验证状态、历史更新入口 |
| [模块目录](reference-module-catalog.md) | 模块、API 路由、前端页面清单 |
| [Agent 项目病历归档 2026-07-06](archive/AGENT_PROJECT_MEDICAL_RECORD_2026-07-06.md) | 从 AGENTS.md 移出的历史快照；不作为当前优先级或执行指令 |
| [测试说明](TEST_SUMMARY.md) | 2026-06-24 历史测试报告 |
| [前端测试报告](FRONTEND_TEST_REPORT.md) | 2026-06-24 前端历史测试报告 |

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
| [Owner-First 开发协议](governance/OWNER_FIRST_PROTOCOL.md) | 非技术 Owner 如何提需求，Agent 如何澄清、拆解、验收和汇报 |
| [平台宪法](governance/PLATFORM_CONSTITUTION.md) | 平台优先最高规则：系统分层、风险等级、禁止事项、Owner 决策边界 |
| [Agent 开发协议](governance/AGENT_DEVELOPMENT_PROTOCOL.md) | 多 Agent 角色、开工检查、review、QA、交接规则 |
| [Kernel 契约](governance/KERNEL_CONTRACTS.md) | EventBus、Command、Scheduler、ToolBridge、Approval、Audit 等内核契约 |
| [验收大门](ACCEPTANCE_GATE.md) | 🆕 完成定义、验收等级、发布通道、Owner 决策日志 |
| [验收矩阵](ACCEPTANCE_MATRIX.md) | 🆕 环境矩阵、角色矩阵、权限回归矩阵、高风险动作证据 |
| [已知问题](KNOWN_ISSUES.md) | 🆕 未解决风险和失败项追踪、过期升级策略 |
| [Owner 决策日志](governance/OWNER_DECISION_LOG.md) | 🆕 风险接受决策记录模板，带有效期的 Owner 签收 |
| [产品闭环验收](governance/PRODUCT_LOOP_ACCEPTANCE.md) | 🆕 候选商品→完整度→成本→审批→上架全链路验收证据 |
| [订单闭环验收](governance/ORDER_LOOP_ACCEPTANCE.md) | 🆕 订单→结算→异常检测→Agent→Owner 全链路验收 |
| [高风险动作验收](governance/HIGH_RISK_ACTION_ACCEPTANCE.md) | 🆕 价格/库存/订单/退款/平台发布等高风险门禁逐项验证 |
| [交付治理体系 Spec](governance/DELIVERY_GOVERNANCE_SPEC.md) | 🆕 交付验收与生产就绪治理体系完整规范 |
| [Agent 可信度标记](governance/AGENT_TRUST_MARKERS.md) | 🆕 5 级可信度：STUB→DETERMINISTIC_RULE→REAL_LLM→HUMAN_APPROVED→PRODUCTION_EXECUTED |
| [数据质量门禁](governance/DATA_QUALITY_GATES.md) | 🆕 商品/成本/物流费/结算完整度阈值，数据不全 Agent 不给强结论 |
| [审计可读性要求](governance/AUDIT_READABILITY.md) | 🆕 审计日志必须 Owner 能看懂：谁、何时、对什么、为什么、结果 |
| [生产就绪检查清单](governance/PRODUCTION_READINESS_CHECKLIST.md) | 🆕 配置/secrets/migration/observability/cost control/kill switch/试运行边界 |
| [发布就绪检查清单](governance/RELEASE_READINESS_CHECKLIST.md) | 🆕 风险分级发布通道（read-only/suggestion/approval-required/production-write）|
| [事故演练检查清单](governance/INCIDENT_DRILL_CHECKLIST.md) | 🆕 事故等级、演练场景、人工接管机制、事后复盘模板 |
| [回滚与恢复手册](governance/ROLLBACK_AND_RECOVERY.md) | 🆕 DB/迁移/平台写回/发布的逐项回滚流程和决策树 |
| [LingMirror Development Loop](DEVELOPMENT_LOOP.md) | Intake → Translate → Discover → Slice → Implement → Verify → Review → Record → Repeat 的项目开发闭环 |
| [验收门禁](ACCEPTANCE_GATE.md) | Dev Done / Test Green / Business Verified / Beta Accepted 的统一完成定义 |
| [验收矩阵](ACCEPTANCE_MATRIX.md) | 环境、角色、业务闭环、高风险动作和运行时证据矩阵 |
| [已知问题台账](KNOWN_ISSUES.md) | 已知失败项 owner / deadline / impact 跟踪 |
| [开发指南](DEVELOPMENT_GUIDE.md) | Go + Next 本地启动、结构、验证和开发流程 |
| [E2E Docker 环境](docker-compose.e2e.yml) | 🆕 全栈 E2E 测试专用 Docker Compose（Postgres + 后端 + 前端）|
| [运维手册](ops/RUNBOOK.md) | 运维命令速查、启动停止、日志、备份恢复、迁移 |
| [全局验证脚本](scripts/verify_all.sh) | 🔄 全局验证脚本：build→vet→test→lint→build→E2E（E2E 默认必须运行）|
| [文档漂移检查](scripts/check_doc_drift.sh) | 🆕 验证 INDEX.md/AGENTS.md/CLAUDE.md 所有 .md 引用是否存在 |
| [已知问题过期检查](scripts/check_known_issues.sh) | 🆕 扫描 KNOWN_ISSUES.md 中超过 Target Fix 日期的项 |
| [静态代码健康报告](scripts/daily_health_report.sh) | 🆕 仓库新鲜度、构建校验、KNOWN_ISSUES 过期检查、迁移完整性 |
| [每周验收报告](scripts/weekly_acceptance_report.sh) | 🆕 扫描决策日志过期项、已知问题截止日期 |
| [迁移回滚检查](scripts/check_migrations.sh) | 🆕 验证最新迁移的 down.sql 可回滚 |
| [E2E 种子数据](scripts/e2e_seed.sh) | 🆕 为 E2E 全栈测试预填种子数据 |
| [告警规则](ops/ALERT_RULES.md) | 🆕 各服务告警规则定义、阈值和响应流程 |
| [备份策略](ops/BACKUP_POLICY.md) | 🆕 数据库/配置/文件备份策略、保留周期和恢复流程 |
| [AgentOS 开发文档](AGENTOS_DEVELOPMENT_GUIDE.md) | AgentOS 团队系统工程 |
| [前端页面与路由](FRONTEND_PAGES_AND_ROUTING.md) | Next App Router 页面结构、菜单覆盖和 API 路径规则 |
| [UI 覆盖审计](UI_FRAMEWORK_GAP_ANALYSIS.md) | 当前 Next App Router 页面覆盖、菜单覆盖和 UI 风险 |
| [LingMirror Development Loop](DEVELOPMENT_LOOP.md) | Intake → Translate → Discover → Slice → Implement → Verify → Review → Record → Repeat 的项目开发闭环 |
| [AgentOS Phase 1 实现计划](superpowers/plans/2026-06-18-lingmirror-agentos-phase-1.md) | 第一版工程骨架总控台 |
| [Agent Commerce 动作中枢计划](superpowers/plans/2026-06-22-agent-commerce-action-center.md) | Phase 2 动作提案、审批、执行、复盘 |
| [核心模块架构](PERMISSIONS_AND_AUDIT.md) | 权限系统 + 审计日志设计 |
| [物流与运费 PRD](LOGISTICS_AND_SHIPPING_PRD.md) | 物流需求 |
| [物流与运费技术规格](LOGISTICS_SHIPPING_TECH_SPEC.md) | 物流技术实现 |
| [报价规则示例](LOGISTICS_QUOTE_RULE_EXAMPLES.md) | 运费报价案例 |
| [平台费用规则方案](platform-fee-rules-plan.md) | 平台费用设计 |
| [AI 选品使用指南](sourcing-guide.md) | A8 选品引擎：利润计算、质量评分、API 参考 |
| [物流费率引擎指南](logistics-guide.md) | Logistics Rate Engine：四种定价模式、YAML 配置、A10 接线 |
| [ToolBridge 工具桥接](toolbridge-guide.md) | 插件驱动工具执行、添加 Driver、降级策略 |
| [Chrome 扩展指南](chrome-extension-guide.md) | 凌镜选品助手扩展：安装、WebSocket 协议、内容脚本 |
| [测试说明](TEST_SUMMARY.md) | 2026-06-24 历史测试状态、已知问题 and 覆盖面 |
| [前端测试报告](FRONTEND_TEST_REPORT.md) | 2026-06-24 `frontend-next` build/test/lint 历史状态 |
| [AI-Native 统一开发框架总览](guides/ai-native-framework-overview.md) | 🆕 AI 程序员开发规范入口，概述系统机制 |
| [ARCS 上下文同步指南](guides/ai-native-arcs-guide.md) | 🆕 解决 AI 记忆 amnesia 的 manifests 与交接账本规范 |
| [沙盒 staging 与 E2E 测试指南](guides/ai-native-sandbox-guide.md) | 🆕 Docker 容器沙盒与 Playwright E2E 测试门禁规则 |
| [状态模拟器与网络阻回拦截](guides/ai-native-mocking-guide.md) | 🆕 Stateful Mock DB 设计与 Fail-Safe 网络防火墙拦截规则 |
| [AI 研发死循环监控与熔断](guides/ai-native-loop-prevention-guide.md) | 🆕 归一化报错哈希、Ping-Pong 震荡与 Stagnation 熔断算法 |

| [API 清单](api-inventory.md) | 🆕 全部 71+ 后端模块 API 路由、方法、参数清单 |
| [Swagger API 文档](http://localhost:8080/swagger/index.html) | 🆕 交互式 OpenAPI 文档 — `GET /swagger/index.html` |


### Demo & 验收

| 文档 | 说明 |
|------|------|
| [演示场景](DEMO_SCENARIO.md) | 新栈 demo seed 待重建说明 |
| [验收报告](DEMO_ACCEPTANCE_REPORT.md) | 旧栈验收归档与新栈验收待办 |
| [订单导入冒烟检查](ORDER_IMPORT_SMOKE_CHECKLIST.md) | Go 新栈订单导入 smoke 待重建说明 |
| [Beta 验收报告](BETA_ACCEPTANCE_REPORT.md) | 🆕 Beta 试运行验收状态和已知问题记录 |

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

- [商品出海决策与执行层](features/product-export-decision-execution-layer.md) — 未来半年商品出海业务层开发总纲
- [功能需求模板](features/TEMPLATE.md)
- [Phase 1: 商品出海Dry-Run闭环修复](features/phase1-dry-run-closed-loop-spec.md)

---

## 时间线

→ 详见 [时间线](TIMELINE.md)
