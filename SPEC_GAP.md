# SPEC Gap Analysis — AI Agent 模块垂直深挖

> 状态追踪：已落地 / 半落地 / 未落地
> 生成日期: 2026-06-22

## 总览

| 维度 | SPEC 评分 | 实现评分 | 状态 |
|------|----------|---------|------|
| 需求规格完整度 | 7/10 | — | 架构草案级，部分细节未定义 |
| 工程可落地性 | — | 5/10 | 多租户、LLM 降级、撤销闭环未完成 |
| 当前实现成熟度 | — | 4/10 | 核心路径可跑，边界场景未覆盖 |

## 逐项追踪

### §2 Agent 间协作与冲突仲裁

| 条目 | 状态 | 说明 |
|------|------|------|
| 分层仲裁架构 (Policy → Arbiter → Human) | ✅ 已落地 | 三层链路实现 |
| Policy Matrix CRUD | ✅ 已落地 | 增删查 + resolve_conflict |
| Policy Matrix 查询逻辑 (priority 排序) | ✅ 已落地 | |
| Arbiter Agent G0 | ✅ 已落地 | decide() 三层逻辑 |
| 冲突检测 30 分钟窗口 | ✅ 已落地 | AutoArbitrator.detect_and_arbitrate |
| 仲裁日志写入 ArbitrationLog | ✅ 已落地 | |
| **仲裁 user_id** | ⚠️ 半落地 | detect_and_arbitrate 和 resolve_conflict 已接受 user_id 参数；但 create_proposal → arbitrator 链路尚未从 HTTP 层传递 user_id，默认使用 1 |
| **不确定时人工兜底闭环** | ⛔ 未落地 | G0 escalate 后没有流程将仲裁结果推入 ActionProposal 审批流 |

### §3 审批流程与回滚

| 条目 | 状态 | 说明 |
|------|------|------|
| 补偿操作预填 compensation 字段 | ✅ 已落地 | |
| 撤销端点 POST .../undo | ✅ 已落地 | |
| 前端撤销按钮 | ✅ 已落地 | WorkItemCard.vue |
| 审批超时三层升级 | ✅ 已落地 | |
| **undo Action Adapter** | ✅ 已修复 | COMMAND_ADAPTERS 添加 undo_ 前缀回退 + HANDLER_MAP 注册 handle_undo |

### §4 规则健康治理

| 条目 | 状态 | 说明 |
|------|------|------|
| 冲突检测 detect_conflicts | ✅ 已落地 | |
| 熵指数公式新增 conflict_ratio | ✅ 已落地 | |
| 硬顶 20 条/决策点 | ✅ 已落地 | |
| 软顶 50 条/Agent → 触发防御 | ✅ 已落地 | |
| 警告线 200 条 | ✅ 已落地 | |
| 冷却期 72h | ✅ 已落地 | defenses.py + service.py 双实现 |
| **冷却期双实现维护风险** | ⛔ 未落地 | 两份独立代码，变更需同步 |

### §5 LLM 调用链路韧性

| 条目 | 状态 | 说明 |
|------|------|------|
| 熔断器 CircuitBreaker | ✅ 已落地 | |
| 熔断粒度 (agent_id, decision_point) | ✅ 已落地 | |
| 决策缓存 AgentDecisionCache | ✅ 已落地 | 300s TTL |
| 输出验证 _sanitize_output | ✅ 已落地 | 含日志 + validation_error 信号 |
| **LLM 降级链** | ✅ 已修复 | settings.LLM_MODEL 运行时切换，try/finally 恢复 |
| **llm_resilient 装饰器死代码** | ⛔ 未落地 | 定义但未使用 |
| **缓存写回在 sanitize 之前** | ⛔ 未落地 | 先缓存再验证，缓存可能存无效数据 |

### §6 多租户隔离

| 条目 | 状态 | 说明 |
|------|------|------|
| Store 模型 | ✅ 已落地 | stores 表 |
| 四表加 store_id 列 | ✅ 已落地 | personal_rules, agent_decisions, agent_actions, action_proposals |
| **上下文构建器 store_id 过滤** | ⛔ 未落地 | 5 个 build_* 只接收参数，需新建 StoreSKU 关联表 |

### §7 性能与扩展性

| 条目 | 状态 | 说明 |
|------|------|------|
| 生产者/消费者调度架构 | ✅ 已落地 | |
| Worker Pool (20) | ✅ 已落地 | |
| 周期重叠保护 | ✅ 已落地 | |
| 失败重试 ScheduleFailure | ✅ 已落地 | |
| 90 天归档 | ✅ 已落地 | |
| 归档计数精确性 | ✅ 已落地 | 已修复 result.rowcount |
| 归档 ID 预取内存浪费 | ⛔ 未落地 | count 查询拉取全部 ID |

### §8 数据模型变更

| 条目 | 状态 | 说明 |
|------|------|------|
| 4 张新表 | ✅ 已落地 | |
| 14 个新列 | ✅ 已落地 | |
| Store 模型 | ✅ 已落地 | |

## 阻断项（优先级排序）

| P | 项 | 影响 | 预估工作量 | 状态 |
|---|-----|------|-----------|------|
| 0 | **StoreSKU 关联表 + 上下文过滤** | 多租户隔离完全失效 | 1 天 | ⛔ 待建 |
| 0 | **仲裁 user_id 从 HTTP 层传递** | detect_and_arbitrate 默认 user_id=1 | 0.5 天 | ⚠️ 接口已就绪，待路由器串联 |
| 0 | **仲裁 escalate 后人工兜底闭环** | 不确定时无审批流 | 0.5 天 | ⛔ 待建 |
| 1 | 冷却期双实现合并 | 维护风险 | 0.5 天 | ⛔ 待建 |
| 1 | 归档 ID 预取优化 | 内存浪费 | 0.5 天 | ⛔ 待建 |
| 2 | 移除 llm_resilient 死代码 | 混淆 | 0.5 天 | ⛔ 待删 |
| 2 | 缓存写回移到 sanitize 之后 | 无效数据缓存 | 0.5 天 | ⛔ 待改 |

## 升级建议

1. SPEC.md 首页添加状态标签："设计草案·部分实现"
2. 下次迭代优先修复 4 个 P0 阻断项
3. 建立实现状态自动化追踪（对照此表 PR 检查时逐项确认）
