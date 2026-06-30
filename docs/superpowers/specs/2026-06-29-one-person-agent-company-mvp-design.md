# 一人 Agent 公司 MVP — 设计文档

日期: 2026-06-29
版本: v0.1 (Day 0)
状态: ✅ Owner 已确认

## 1. 目标

7 天内用 Claude Code Agent View + Subagents 并行开发出"一人 Agent 公司 MVP"的最小可运行经营闭环。

### 演示目标

```
20 个候选商品
-> 资料完整度检查
-> 成本 / 物流 / 平台费 / 利润计算
-> Agent 给出上架建议
-> 生成任务 / 待审批
-> Owner 总控台看到风险、建议、审批、失败
-> 平台只读或模拟数据进入
-> 所有关键动作有记录
```

## 2. 现有能力评估

| 能力 | 状态 | 说明 |
|------|------|------|
| ProductHub (版本/关系/新鲜度) | ✅ 已存在 | 可记录决策、版本、关系 |
| SKU / 品牌 / 类目 / 价格 / 库存 / 供应商 | ✅ 已存在 | 独立 domain |
| 平台费用 (platformfee) | ✅ 已存在 | 费用规则和计算 |
| 物流 (logistics, A10) | ✅ 已存在 | 四种定价模式，YAML 配置 |
| 选品盈利分析 (sourcing, A8) | ✅ 已存在 | 利润公式计算 + 评估 |
| 刊登任务 (listingtask) | ✅ 已存在 | 任务队列、状态、缺少项 |
| 刊登 (listing) | ✅ 已存在 | 平台发布记录 |
| 审批 (approval) | ✅ 已存在 | 审批请求 + 审查流程 |
| AgentOS 总控台 | ⚠️ 部分存在 | 有 cockpit，但缺少风险/建议视图 |
| 商品完整度检查 | ❌ 不存在 | 需要新建 |
| 端到端经营闭环接线 | ❌ 不存在 | 需要串联现有模块 |
| 种子数据 (20 产品) | ❌ 不存在 | 需要创建 |
| 平台模拟数据 | ❌ 不存在 | 需要创建 |
| Ozon 适配器 | ✅ 已存在 | 只读可用 |

**结论：MVP 80% 是接线 + 增强，20% 是新功能（完整度检查 + 总控台视图）。**

## 3. 5 条并行线

### A — 经营闭环线

构建完整经营链路：

```
商品资料 → 完整度检查 → 成本汇总 → 利润测算 → Agent 建议 → 任务生成 → 审批
```

核心交付：
- `candidate_product` 模型 + API（20 候选商品）
- `completeness_check` 服务 + API
- `profit_summary` 利润汇总 API（整合 supplier cost + logistics + platformfee）
- `listing_recommendation` 建议生成 + 写入 listingtask
- 端到端接线（通过 EventBus 或直接 service call）

### B — Owner 总控台线

为 Owner 提供日常管理视图：

- 风险面板（数据缺失、利润过低、异常状态）
- 待审批建议列表
- Agent 建议回顾
- 平台同步状态
- 操作日志展示

核心交付：
- `frontend-next/src/app/(main)/agentos/risk/` — 风险视图
- `frontend-next/src/app/(main)/agentos/suggestions/` — Agent 建议视图
- `frontend-next/src/app/(main)/agentos/approvals/` — 审批视图
- 后端 API：风险汇总、建议列表、审批统计

### C — Agent 建议线

让 Agent 从"能做"变成"会建议"：

- 商品完整度 Agent：分析哪些商品缺资料，给出补充建议
- 利润评估 Agent：基于成本/费用/售价判断是否值得上架
- 上架建议 Agent：综合给出建议上架/不建议上架/需要更多信息

核心交付：
- Agent 规则/提示词（agentrule 或 pipeline）
- Agent 建议的生成 → 存储 → 展示链路
- 与 A 线串联（评估结果 → listing_task）

### D — 平台数据线

让演示有真实感，但不受真实平台依赖：

- Mock 平台数据生成（Ozon 风格的订单、结算、费用）
- 只读同步展示（模拟同步动态）
- 在总控台展示平台数据状态

核心交付：
- `seed_data/` — mock 数据生成脚本
- 平台数据展示组件
- 只读同步状态视图

### E — QA / Review / 文档线

贯穿全程的质量保障：

- 每个 Day 的交付物验证
- 代码风险审查（是否符合平台宪法）
- 文档同步更新
- 每天最终验收日志

核心交付：
- 每日 QA 运行
- Review checklist 执行
- Daily acceptance log 更新
- Battle board 状态更新

## 4. Claude Code 工作方式

### 主会话 (Lead Agent)

当前会话。职责：
- 分解任务
- 分发 subagents
- 审核 subagent 产出
- 维持 battle board
- 向 Owner 汇报

### Subagents (通过 Agent 工具)

| Agent 类型 | 何时使用 | 隔离 |
|-----------|----------|------|
| lingmirror-research | 读代码、查模式、不写文件 | 不需要 |
| lingmirror-qa | 运行测试、验证行为 | 不需要 |
| lingmirror-review | 审查 diff、检查合规 | 不需要 |
| lingmirror-docs | 更新文档 | 不需要 |
| lingmirror-risk-guard | 风险合规检查 | 不需要 |

当多条线可以并行时，通过 Agent 工具同时启动多个 subagent。

### 并发策略

- **独立任务**（A线 vs D线）→ 同时启动 subagent，互不依赖
- **顺序任务**（A线完成后 → C线使用其结果）→ 先完成 A，再启动 C
- **随时任务**（E线）→ 与所有线并行，持续运行
- 使用 `isolation: "worktree"` 保护脏工作区

## 5. 禁止事项

1. 不多平台接入（一个 Ozon mock 足够）
2. 不做真实自动发布
3. 不做自动改价
4. 不做自动改库存
5. 不做自动退款 / 赔付
6. 不新增无关页面（只做经营闭环所需页面）
7. 不绕过审批审计
8. 不以代码完成代替 Owner 能看懂

## 6. 技术决策

- **候选产品表**: 新建 `candidate_product` 表在 `new domain candidate/`
- **完整度检查**: 新建 `internal/domain/completeness/`，纯计算逻辑
- **利润汇总**: 增强现有 sourcing/finance 能力，新增 profit_summary API
- **种子数据**: Go 脚本 (cmd/seed/) + SQL migration
- **前端口**: 现有 `agentos` 路径下新增 risk/suggestions/approvals 子页面
- **所有新功能写在 backend-go/ 和 frontend-next/**

## 7. 移交到实施计划

确认后通过 writing-plans 技能创建详细实施计划。
