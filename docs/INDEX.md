# 凌镜 LingMirror — 文档索引

> **唯一开发路径（2026-07-13）**：先读 [ADR-001：完整 Owner 自用跨境电商平台](decisions/ADR-001-owner-complete-commerce-platform.md)，再读 [ADR-002：实践—认识项目运行方法](decisions/ADR-002-practice-cognition-operating-method.md)和 [真实可运行系统完善计划](plan/REAL_OPERATION_READINESS_PLAN.md)。完整平台是目的地，按系统建设验收与受控真实经营两类循环推进；凌镜只供 Owner 本人使用，外部 SaaS、多租户和商业化文档均为历史材料。

> **当前事实入口**：先读 [真实经营就绪基线 2026-07-13](research/real-operation-readiness-baseline-2026-07-13.md)与 [本地运行验收事实审计 2026-07-13](research/project-truth-audit-2026-07-13.md)，再读 [经营闭环模型纠偏](research/project-truth-audit-2026-07-12-business-loop-correction.md)与 [方向事实审计 2026-07-12](research/project-truth-audit-2026-07-12.md)；历史工程边界继续追溯到 [项目真相审计 2026-07-11](research/project-truth-audit-2026-07-11.md)。

> 跨境電商 AI Agent 运营平台（技术名：MultiSell）
> 更新日期：2026-07-13

---

## 🚀 快速入门 (Quick Start)

| 文档 | 说明 |
|------|------|
| [入门教程](tutorial-getting-started.md) | 🆕 从零搭建开发环境到创建第一个商品——新人首选 |
| [端到端教程](tutorial-first-workflow.md) | ⛔ 历史教程；“业务闭环”命名已过时，只能用于了解旧工程流程 |
| [API 快速参考](reference-api-quick.md) | 当前核心路由、认证、响应格式和中间件栈速查 |
| [完整 API 参考](reference-api-complete.md) | ⛔ 2026-07-12 历史运行时快照；当前 Agent 路由以小Q契约和 Router 测试为准 |
| [完整经营闭环与系统边界](research/commerce-loop-system-boundaries.md) | ⛔ 历史研究；其“完整经营闭环”定义已被 2026-07-12 纠偏，不得作为当前工程闭环定义 |
| [模块目录](reference-module-catalog.md) | 🆕 全部 60+ 后端领域模块一览 |
| [配置参考](reference-configuration.md) | 🆕 config.yaml + 环境变量完整说明 |
| [Agent Pipeline 解释](explanation-agent-pipeline.md) | ⛔ 已退役 A/G EventBus Pipeline 的历史说明 |
| [AI & Agent 系统参考](reference-ai-agent-system.md) | 历史参考：旧Agent注册表、AgentOS与Trace；运行边界以小Q架构为准 |

## 🛠️ 操作指南 (How-to)

| 文档 | 说明 |
|------|------|
| [添加新领域模块](howto-add-domain-module.md) | 🆕 添加完整 CRUD 模块 step-by-step |
| [配置平台集成](howto-platform-integrations.md) | 🆕 接入 Ozon / Shopee API |
| [创建自定义 Agent 规则](howto-agent-rules.md) | ⛔ 已退役旧 Agent 规则运行面的历史说明 |
| [运行测试与验证](howto-test-and-verify.md) | 🆕 Go 测试、前端测试、E2E |
| [Owner 与 AI 统一部署测试手册](ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md) | ⭐ 服务器初始化、部署、恢复、测试、回滚与交付的唯一运行手册 |
| [使用 WebSocket 流式更新](howto-websocket.md) | 🆕 连接 /ws 端点接收实时数据 |
| [执行第一个业务闭环](howto-first-business-loop.md) | ⛔ 历史操作指南；其流程不构成当前定义的经营反馈闭环 |

## 🧠 解释 (Explanation)

| 文档 | 说明 |
|------|------|
| [两个核心业务闭环](explanation-business-loops.md) | ⛔ 历史流程设计；“闭环”命名已过时，不得作为当前领域模型 |
| [领域模块架构](explanation-domain-architecture.md) | 🆕 60+ 模块的组织方式、协作模式和依赖关系 |

## 📋 项目概览

当前执行口径以 [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) 和 `docs/governance/` 为准。
下表中的历史计划、蓝图和研究材料保留作参考，不覆盖当前优先级。

| 文档 | 说明 |
|------|------|
| [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) | 🆕 当前执行口径：产品方向、AgentOS 安全优先级、文档清理规则 |
| [真实可运行系统完善计划](plan/REAL_OPERATION_READINESS_PLAN.md) | ⭐ 当前唯一执行计划：从工程基线、正式环境到真实经营、对账和下一轮行动 |
| [真实经营就绪基线 2026-07-13](research/real-operation-readiness-baseline-2026-07-13.md) | P0-1 结果：完整经营路径的权威入口、证据等级、现场阻塞和下一验收动作 |
| [正式运行底座检查点 2026-07-13](research/production-readiness-checkpoint-2026-07-13.md) | P0-2 现场证据：备份恢复与 111→151 隔离迁移已通过，异地不可变备份和告警仍阻塞 |
| [设计系统](DESIGN.md) | 🆕 UI 设计规范：色彩、字体、间距、动画、组件风格、无障碍 |
| [项目状态](PROJECT_STATUS.md) | 当前版本、验证状态、更新历史 |
| [AI 无人公司长远战略愿景](LONG_TERM_VISION_AND_STRATEGY.md) | 🆕 系统终极形态：“全自动驾驶”无人公司操作系统的四道门槛与落地路线图 |
| [模块目录](reference-module-catalog.md) | 🆕 模块、API 路由和前端页面唯一事实源 |
| [平台真相合同审计](research/platform-truth-contract-audit-2026-07-12.md) | 平台真相只读合同、领域处置和证据限制 |
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
| [ADR-001：完整 Owner 自用跨境电商平台](decisions/ADR-001-owner-complete-commerce-platform.md) | Accepted；唯一开发路径、完整平台结构、纵向单元顺序和范围边界 |
| [ADR-002：实践—认识项目运行方法](decisions/ADR-002-practice-cognition-operating-method.md) | Accepted；区分工程实践与经营实践，规定真实可运行门槛和循环推进方法 |
| [Owner 自用经营方向](SELF_USE_OPERATING_DIRECTION.md) | 当前最高优先级产品边界、资金纪律与开发路线；明确无外部软件用户目标 |
| [经营闭环模型纠偏](research/project-truth-audit-2026-07-12-business-loop-correction.md) | 当前有效裁决：现有 experiment 是经营事实核验案卷，不构成因果实验或反馈闭环 |
| [外部经营反馈标杆综合](research/external-benchmark-operating-feedback-synthesis-2026-07-12.md) | 三路外部调研综合：Amazon 单变量实验、eBay 单人工作台及凌镜首个现实验证建议 |
| [真实反馈系统与实验平台标杆](research/external-benchmark-real-feedback-systems-2026-07-12.md) | Microsoft、Booking、Uber、Netflix、Airbnb、Spotify、Amazon 与 GA4 对照 |
| [一人电商经营系统标杆](research/external-benchmark-solo-commerce-operations-2026-07-12.md) | Shopify、Amazon、eBay、Etsy、Odoo 官方能力比较 |
| [经营决策循环工程边界](research/external-benchmark-operating-decision-loops-2026-07-12.md) | OODA、PDSA、Build-Measure-Learn、因果推断与电商反馈机制 |
| [当前方向与优先级](CURRENT_DIRECTION_AND_PRIORITIES.md) | 当前产品方向和下一阶段优先级 |
| [真实付费需求发现循环设计](superpowers/specs/2026-07-11-paid-demand-discovery-loop-design.md) | 历史命名；仅保留商品消费者付款、反证和交易裁决边界，不代表凌镜有外部软件需求 |
| [真实付费需求发现循环实施计划](superpowers/plans/2026-07-11-paid-demand-discovery-loop.md) | ⛔ 历史实施计划；不得覆盖 ADR-001 的完整平台路径，仅保留证据边界参考 |
| [付费需求信号地图](../deliverables/research/paid-demand-signal-map.md) | 代理信号与真实付款证据的边界 |
| [付费需求反证协议](../deliverables/research/paid-demand-falsification-protocol.md) | 独立反证、污染和停止规则 |
| [需求数据可得性现实审计](../deliverables/research/demand-data-access-reality.md) | 平台字段、权限和 unknown 边界 |
| [选市场、选类目、选产品统一方法](../deliverables/research/market-category-product-selection-synthesis.md) | 历史研究材料；固定漏斗数量已冻结，当前只保留其证据边界参考 |
| [市场选择研究](../deliverables/research/market-selection-method.md) | 国家市场、平台市场与需求市场的选择方法和 Ozon 实证闸门 |
| [类目选择研究](../deliverables/research/category-selection-method.md) | 类目硬淘汰、七维筛选、评分与反例 |
| [产品选择研究](../deliverables/research/product-selection-method.md) | 需求、竞品、供应商、成本、合规与真实商品机会定义 |
| [双产品十星架构设计](designs/dual-product-cathedral.md) | ⛔ 2026-07-11 已冻结的历史决策，不得作为当前开发指令 |
| [AI-Native AgentOS 长期愿景与架构方向](AI_NATIVE_DEVELOPMENT_PLAN.md) | 冻结历史愿景；不得作为当前多Agent开发指令 |
| [AI-Native AgentOS 执行路径规格](specs/2026-07-09-ai-native-agentos-execution-path.md) | 冻结历史规格；当前只执行小Q单Agent架构 |
| [AI 基础建设总规划：小Q与 Evidence Workshop](../tasks/plan.md) | 小Q作为唯一 Owner Agent 的长期目标、能力目录、安全边界、阶段门禁、预算与停止条件 |
| [小Q与 Evidence Workshop 执行清单](../tasks/todo.md) | 小Q对话、Capability 接入和经营闭环的垂直任务；每阶段需 Owner 单独批准 |
| [本地运行验收事实审计 2026-07-13](research/project-truth-audit-2026-07-13.md) | PostgreSQL 145、跨层浏览器验收、Schema Drift 与图片服务契约的最新本地证据 |
| [真实经营就绪基线 2026-07-13](research/real-operation-readiness-baseline-2026-07-13.md) | 正式服务器只读现场检查与全经营路径 P0-1 基线；当前最早门槛为 P0-2 |
| [方向事实审计 2026-07-12](research/project-truth-audit-2026-07-12.md) | 最新产品边界：Owner 单人自用，不验证外部软件需求 |
| [项目真相审计 2026-07-11](research/project-truth-audit-2026-07-11.md) | 上一版代码、测试、经营事实、mock 和未闭环部分的只读审计快照 |
| [商品出海决策与执行层](features/product-export-decision-execution-layer.md) | ⛔ 已被 2026-07-11 自用方向替代，仅供历史追溯 |
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
| [小Q Capability Contract](governance/XIAOQ_CAPABILITY_CONTRACT.md) | 小Q唯一身份、能力字段、风险、权限、审批，以及新增功能同步接入规则 |
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
| [小Q Agent Runtime Architecture v1](architecture/XIAOQ_AGENT_RUNTIME_V1.md) | 小Q唯一真实运行循环、能力目录、安全控制、第一湖验收与旧架构迁移的第一版权威设计 |
| [小Q Agent 架构官方资料调研](research/xiaoq-agent-architecture-2026-07-12.md) | 单一主Agent、动态能力目录、模型外审批和审计的官方资料依据 |
| [小Q产品需求质量审计](research/xiaoq-product-requirement-quality-2026-07-12.md) | 原始愿景的合理性、产品经理版One-pager、指标、非目标和停止条件 |
| [小Q多领域路由规范 v1](specs/xiaoq-multi-domain-routing-v1.md) | `superseded`：固定能力路由设计，保留历史参考；新实现遵循Agent Runtime v1 |
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
| [Chrome扩展指南](chrome-extension-guide.md) | 1688页面一键保存到Owner私人采集箱：安装、权限、HTTPS接口和故障提示 |
| [1688采集助手重构规格](features/1688-browser-evidence-collector-refactor.md) | Owner页面一键采集、私人采集箱、后续选品任务与受控草稿 |
| [市场与商品机会 Owner 流程](features/market-opportunity-owner-flow.md) | 三类研究、系统评估、Owner 市场决定、商品机会及安全边界 |
| [Owner 经营决策与反馈案卷](features/business-decision-feedback-owner-flow.md) | 权威事实快照、exact Owner 决定、受控执行、观测与可恢复反馈链 |
| [市场与商品机会进展审计](research/market-opportunity-progress-audit-2026-07-12.md) | 第2单元已实现证据、验证结果与剩余缺口 |
| [商品、货源与渠道准备缺口审计](research/product-supply-channel-gap-audit-2026-07-12.md) | 第3单元已有证据、权威缺口与实施顺序 |
| [订单、库存、履约与售后纵向单元审计](research/order-inventory-fulfillment-aftersales-gap-audit-2026-07-12.md) | 第4单元事实链、自动验证与仍然未知的外部经营结果 |
| [结算、最终利润与现金纵向单元审计](research/settlement-profit-cash-gap-audit-2026-07-12.md) | 第5单元单一金额权威链、自动验证与真实金额证据边界 |
| [Owner 经营决策与反馈纵向单元审计](research/business-decision-feedback-gap-audit-2026-07-12.md) | 第6单元事实快照、Owner决定、受控行动、结果反馈与因果边界 |
| [小Q Owner 协作层审计](research/xiaoq-owner-collaboration-audit-2026-07-12.md) | 第7单元 active/deferred Capability、Owner交互与安全边界 |
| [测试说明](TEST_SUMMARY.md) | 2026-06-24 历史测试状态、已知问题 and 覆盖面 |
| [前端测试报告](FRONTEND_TEST_REPORT.md) | 2026-06-24 `frontend-next` build/test/lint 历史状态 |
| [AI-Native 统一开发框架总览](guides/ai-native-framework-overview.md) | 🆕 AI 程序员开发规范入口，概述系统机制 |
| [ARCS 上下文同步指南](guides/ai-native-arcs-guide.md) | 🆕 解决 AI 记忆 amnesia 的 manifests 与交接账本规范 |
| [沙盒 staging 与 E2E 测试指南](guides/ai-native-sandbox-guide.md) | 🆕 Docker 容器沙盒与 Playwright E2E 测试门禁规则 |
| [状态模拟器与网络阻回拦截](guides/ai-native-mocking-guide.md) | 🆕 Stateful Mock DB 设计与 Fail-Safe 网络防火墙拦截规则 |
| [AI 研发死循环监控与熔断](guides/ai-native-loop-prevention-guide.md) | 🆕 归一化报错哈希、Ping-Pong 震荡与 Stagnation 熔断算法 |

| [API 清单](api-inventory.md) | ⛔ 2026-07-03 历史 API 快照，仅兼容旧链接 |
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

- [商品出海决策与执行层](features/product-export-decision-execution-layer.md) — ⛔ 已被当前 Owner 自用方向替代，仅供历史追溯
- [功能需求模板](features/TEMPLATE.md)
- [经营实践—认识循环底座（架构草案）](features/operating-practice-cognition-loop.md) — `draft`；经营事实、构思、Owner决定、真实行动、结果与下一轮的概念设计
- [Phase 1: 商品出海Dry-Run闭环修复](features/phase1-dry-run-closed-loop-spec.md)
- [1688 货源到待上架草稿受控闭环](features/1688-controlled-draft-workflow.md) — Owner 自用，草稿与独立发布审批严格分离；真实商品验收待完成
- [商品视觉生产与学习系统开发规格](features/multi-provider-product-image-system.md) — 单 SKU 配方冻结、反馈、返工和统计已完成隔离浏览器验收；Owner 真实 SKU/场景图验收后再进入3 SKU对照
- [AI 商品图片系统的长期价值、提示词资产与建设边界](research/ai-product-image-system-value-and-scope-2026-07-13.md) — 提示词不是孤立核心资产；定义长期数据资产、做/不做边界和最小真实验证
- [AI 电商作图核心卡点、成本与方案](research/ai-commerce-image-core-bottlenecks-cost-options-2026-07-13.md) — 商品保真、人审返工、成本模型与多路线建议
- [Image Service 与 MCP 技术合同](features/image-service-mcp-contract.md) — 独立服务、HTTP/MCP边界和一次性执行令牌已进入代码；付费 Provider 尚未开放
- [AI 商品图片系统工程验证记录（2026-07-12）](research/ai-image-system-engineering-verification-2026-07-12.md) — 当前代码、自动验证与未完成边界
- [AI 商品图片 Provider 官方合同核验（2026-07-12）](research/ai-image-provider-contract-verification-2026-07-12.md) — Photoroom、Adobe、OpenAI 官方合同矩阵与 sandbox/生产门禁
- [Photoroom Sandbox 单次 Canary 运行手册](ops/PHOTOROOM_SANDBOX_CANARY_RUNBOOK.md) — Owner 凭据、权利、一次配额、停止条件与结果裁决
- [OpenAI 商品图片 Owner 单次付费验收手册](ops/OPENAI_PRODUCT_IMAGE_OWNER_CANARY_RUNBOOK.md) — 真实 SKU、一次付费、禁止重试、费用对账与事实裁决
- [Prism → Image Service 迁移盘点（2026-07-12）](research/prism-to-image-service-inventory-2026-07-12.md) — MultiSell 旧 Prism 运行路径已退役；历史数据与独立 Prism 仓库保留
- [真实 SKU 的 AI 辅助图片草稿方案](features/ai-assisted-product-image-draft.md) — `superseded`；历史收缩方案
- [1688 受控草稿链工程事实审计（2026-07-12）](research/project-truth-audit-2026-07-12-1688-draft-workflow.md) — 工程实现与外部事实边界

---

## 时间线

→ 详见 [时间线](TIMELINE.md)
