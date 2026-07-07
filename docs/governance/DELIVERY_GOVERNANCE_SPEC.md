# Spec: LingMirror 交付验收与生产就绪治理体系

## Objective

建立"可证明完成、可控上线、可审计运行、可恢复事故"的 AgentOS 交付治理体系。

**不是只管测试**，而是覆盖：需求清晰度、开发完成度、测试全绿、业务闭环跑通、风险受控、生产可运行、事故可恢复、文档一致性、Owner 决策能力。

**长期目标（3-6个月）：** 任何功能、Agent、平台写回、业务闭环、版本发布都必须通过证据链证明才能叫完成或可上线。不能有"口头完成"。

## Scope

### Phase 1 — 完成定义和红线（完善）
- ACCEPTANCE_GATE.md — 完成定义、验收等级、发布通道、Owner 决策日志
- ACCEPTANCE_MATRIX.md — 环境矩阵、角色矩阵、权限回归矩阵、高风险动作证据
- KNOWN_ISSUES.md — 已知问题追踪、过期升级策略、deadline 自动化检查
- PR 模板 — 风险等级、发布通道、数据质量影响
- CI required checks — E2E 不允许静默 skip、governance-checks 必过

### Phase 2 — 业务闭环验收体系
- PRODUCT_LOOP_ACCEPTANCE.md — 候选商品→完整度→成本→审批→上架全链路
- ORDER_LOOP_ACCEPTANCE.md — 订单→结算→异常→Agent→Owner 全链路
- HIGH_RISK_ACTION_ACCEPTANCE.md — 价格/库存/订单/退款/平台发布等门禁
- OWNER_DECISION_LOG.md — 风险接受决策日志

### Phase 3 — 生产就绪体系
- PRODUCTION_READINESS_CHECKLIST.md — 配置/secrets/observability/cost control/kill switch/试运行边界
- RELEASE_READINESS_CHECKLIST.md — 风险分级发布通道
- INCIDENT_DRILL_CHECKLIST.md — 事故等级/演练/人工接管
- ROLLBACK_AND_RECOVERY.md — DB/迁移/平台写回/发布回滚

### 扩展补充
- AGENT_TRUST_MARKERS.md — 5 级可信度标记
- DATA_QUALITY_GATES.md — 数据完整度阈值
- AUDIT_READABILITY.md — 审计日志可读性

### Phase 4 — 持续治理自动化
- scripts/verify_all.sh — 全局验证（E2E 必须运行）
- scripts/check_doc_drift.sh — 文档漂移检查
- scripts/check_known_issues.sh — known issue 过期检查
- scripts/daily_health_report.sh — 每日健康报告
- scripts/weekly_acceptance_report.sh — 每周验收报告
- .github/workflows/ci.yml — governance-checks、daily-health-report 定时任务

## Current State

已全部完成 Phase 1-4 文档创建和自动化脚本编写。
