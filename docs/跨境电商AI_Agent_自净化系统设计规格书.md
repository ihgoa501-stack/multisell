# 跨境电商 AI Agent 自净化系统 — 设计规格书

> **版本**: v1.0 | **日期**: 2026-06-17
> **文档定位**: 基于 Hermes 架构（NousResearch）的深度集成方案，覆盖 7+3 Agent 矩阵、个人规则系统、Hermes 自净化（熵管理）体系、大数据基础设施及完整实施路线图。
>
> **前置文档**: 本文档整合 final-integrated-solution.md（7+3 Agent 综合方案）、hermes-self-evolving-agent-design.md（Hermes 架构）、hermes-entropy-management-system.md（熵管理扩展）三份设计文档。
>
> **实现状态声明**: 本文档中标记为 [已实现] 的内容已在代码中完成；[部分实现] 为框架骨架已就位但需扩展；[规划中] 尚未开始。

---

## 目录

1. [系统总览](#1-系统总览)
2. [Agent 框架架构](#2-agent-框架架构)
3. [7+3 Agent 矩阵](#3-7-3-agent-矩阵)
4. [Agent 协同链](#4-agent-协同链)
5. [个人规则系统](#5-个人规则系统)
6. [Hermes 自净化系统（核心）](#6-hermes-自净化系统核心)
7. [大数据基础设施](#7-大数据基础设施)
8. [数据库 Schema 总览](#8-数据库-schema-总览)
9. [API 设计一览](#9-api-设计一览)
10. [实施路线图](#10-实施路线图)
11. [成功指标](#11-成功指标)
12. [跨文档引用映射](#12-跨文档引用映射)

---

## 1. 系统总览

### 1.1 项目背景

跨境电商行业面临数据孤岛、库存困境、广告成本攀升、合规压力、多语言障碍等十大痛点（详见 final-integrated-solution.md §3）。本系统通过在现有 multisell 平台基础上，以**最小侵入性**原则新增 AI Agent 框架层，实现 7+3 个智能 Agent 覆盖全运营链路。

核心创新点：Agent 系统引入 Hermes 自净化（熵管理）架构，确保系统在持续进化过程中不退化——这是区别于传统 AI Agent 方案的关键差异化能力。

### 1.2 核心设计原则

| 原则 | 含义 |
|------|------|
| **生长必须伴随凋亡** | 每个新规则必有一个 TTL，每条新记忆必有一个衰减函数 |
| **压缩优先于扩展** | 当两个功能可通过合并减少 50% 复杂度时，触发合并 |
| **复杂度红线不可逾越** | 硬限制是最后防线——熵减在阈值以下可自动，超过则必须人工介入 |
| **隐私圣线** | 无跨用户协同过滤、无实时在线学习、无 NL 规则解释、无自动 A/B 实验、无情感/性格建模 |
| **最小侵入性** | 不改动现有 Agent 决策逻辑，以"个人偏好过滤器"方式叠加 |
| **优雅降级** | 个人规则层故障时自动回退到通用 Agent 输出 |
| **阶段分离** | 每个决策点独立进化，不全局化升级 |

### 1.3 三个核心子系统

```
┌──────────────────────────────────────────────────────────────────────┐
│                  跨境电商 AI Agent 自净化系统                          │
├────────────────┬──────────────────────┬─────────────────────────────┤
│  自进化         │  自净化（核心）       │  自学习                      │
│  (4-stage      │  (Entropy Mgmt)      │  (Honcho + Nudge)           │
│   lifecycle)   │                      │                             │
├────────────────┼──────────────────────┼─────────────────────────────┤
│ OBSERVATION    │ 5道反应式防线         │ Honcho 辩证用户建模           │
│ SUGGESTION     │ (TTL/Budget/Decay/   │ (观察→假设→验证→确认)        │
│ SEMI_AUTONOMOUS│  Merge/Regret)       │                             │
│ FULL_AUTONOMOUS│                      │ Nudge 定期行为对齐            │
│                │ 5层预测式架构          │ (阈值校准/策略发现/技能推荐/  │
│ 设计来源:       │ (遥测/SPC/漂移/因果/  │  空白发现/性能报告)           │
│ Hermes Ch11    │  跨用户)              │                             │
│                │                      │ 设计来源: Hermes Ch3+Ch6     │
│                │ 设计来源: Hermes Ch17 │                             │
│                │  + 熵管理扩展文档      │                             │
└────────────────┴──────────────────────┴─────────────────────────────┘
```

### 1.4 实现状态总览

| 子系统 | 状态 | 说明 |
|--------|:----:|------|
| Agent 框架底座 | [已实现] | BaseAgent 基类、AgentRegistry 注册、决策管线 |
| A5 库存预警 Agent | [已实现] | 三级预警 + 补货计算 + 物流推荐 |
| G3 折扣风控 Agent | [已实现] | 折扣叠加检测 + 促销验证 + 阻断逻辑 |
| 个人规则系统 | [已实现] | 四类规则 CRUD + 条件匹配 + 优先级仲裁 |
| Honcho 用户模型 | [已实现] | JSON 档案 CRUD |
| Episode 管理 | [已实现] | 批量查询 + LLM 摘要预留 |
| 熵管理系统 Phase 1 | [已实现] | 5 道防线 + Layer 1 健康评分 + Layer 2 SPC + Mark Change 审计 |
| 前端框架 | [已实现] | AgentList/AgentDetail/AgentRules/EntropyCockpit |
| A3 广告调价 Agent | [规划中] | 核心完整决策树 + Ads API 对接 |
| A4 多语言客服 Agent | [规划中] | 意图识别 + FAQ 引擎 + Amazon Messaging API |
| A6 利润监控 Agent | [规划中] | 多平台利润归集 + 异常检测 |
| A1/A2/A7/G1/G2 Agent | [规划中] | Phase 2-3 规划 |
| 大数据遥测层 | [规划中] | TimescaleDB + 遥测采集 Cron |
| 预测式熵对抗 | [规划中] | 分布漂移 + 因果归因 + 跨用户模式 |
| 全自动自净化 | [规划中] | 闭环熵管理（自动检测→诊断→处理→验证） |

---

## 2. Agent 框架架构

### 2.1 核心框架

#### 2.1.1 BaseAgent 基类

```
BaseAgent (backend/app/agent/base.py)          [已实现]
├── agent_id           — 唯一标识（如 'A3', 'G3'）
├── name               — 中文名称
├── description        — 功能描述
├── decision_points    — 决策点列表
├── version            — 版本号
│
├── get_stage(decision_point)                 — 获取当前进化阶段
├── get_confidence_threshold(decision_point)  — 置信度阈值
├── decide(decision_point, context) [abstract]— 核心决策方法
└── build_decision_record(...)                — 构建决策日志
```

**四阶段进化生命周期**（Hermes Ch11）:

```
Stage 0: OBSERVATION (观察期)
  周期: 1-2周 或 ≥50个决策样本
  Agent 不参与决策，仅记录人工输入/输出
  积累 baseline 数据
  → 转移条件: 样本量足够 + 模拟准确率 ≥85%

Stage 1: SUGGESTION (建议期)
  周期: 1-3周
  Agent 生成建议但不执行，推送消息流
  记录采纳率
  → 转移条件: 采纳率 ≥80% 持续2周

Stage 2: SEMI_AUTONOMOUS (半自治)
  Agent 自动执行低风险操作
  高风险操作推送确认卡片
  30分钟未响应 → 按建议执行 + 通知
  → 转移条件: 确认拒绝率 <5% 持续4周

Stage 3: FULL_AUTONOMOUS (全自治)
  Agent 完全自主，仅异常/边界场景通知
  Nudge 定期行为对齐
```

#### 2.1.2 AgentRegistry 注册中心

```
AgentRegistry (backend/app/agent/registry.py)  [已实现]

register(agent_cls)         — 注册 Agent 类（验证 agent_id 唯一性）
get_agent_class(agent_id)   — 按 ID 获取 Agent 类
list_agents()               — 列出所有已注册 Agent 元数据
get_metadata(agent_id)      — 获取单个 Agent 元数据
```

**自动发现机制**: 新建 Agent 模块只需继承 BaseAgent 并设置 `agent_id`，系统自动注册路由。无需修改已有代码。

#### 2.1.3 决策执行管线

```
decide(decision_point, context) → 决策执行管线          [已实现]

1. Agent.decide() 生成原始决策 (agent_output)
2. AgentService.apply_rules() 匹配个人规则
   ├── 按优先级排序 (veto > threshold > strategy > style)
   ├── 条件匹配 (gt/gte/lt/lte/eq/neq/in/contains)
   └── 动作应用 (percentage/absolute override)
3. 生成最终决策 (final_decision)
4. 记录决策日志 (AgentDecision)
5. [预留] 触发熵检查 (entropy_check)
```

### 2.2 三层分离设计

```
Layer 3: 团队共享规则池
  ├── 共享范围: 同一团队/品类组
  ├── 写入权限: 管理员审批
  └── 优先级: 最低（个人规则可覆盖）

Layer 2: 个人规则库 ← ★ 核心进化层      [已实现]
  ├── 归属: 单个用户
  ├── 写入: Agent建议 → 用户确认
  └── 优先级: 中（可覆盖团队规则）

Layer 1: 通用Agent核心                      [已实现]
  ├── 归属: 系统级
  ├── 写入: 系统管理员 + PR Review
  └── 优先级: 最高基础值，但可被 Layer 2 覆盖
```

### 2.3 暖启动机制

[规划中] 新用户注册后：

1. **角色模板克隆**: 选择角色模板（保守型/激进型/多平台均衡型/独立站 DTC/1688 选品型），系统克隆模板到个人规则库
2. **5 分钟校准问卷**（8-10 题）:
   - 更关注利润还是 GMV 增长？ → ACoS 阈值浮动 (A3)
   - 多少金额以下退款自动放行？ → 退款阈值 (A4)
   - 每日报表接收时间？ → Cron 调度 (G1)
   - 最不能容忍的风险？(断货/封号/亏损) → 预警优先级 (A5, A6)
   - 沟通语言风格？ → 回复风格 (G1)
   - 主要市场？(北美/欧洲/日本/东南亚) → 市场默认值 (全局)
   - 月广告预算？ → 预算告警线 (A3)
   - 是否愿 Agent 自动调广告出价？ → 直接进 Stage 1 (A3)

---

## 3. 7+3 Agent 矩阵

### 3.1 完整矩阵

| # | Agent 名称 | 类型 | AI 级别 | 覆盖岗位群 | 设计文档 | 实现状态 |
|:-:|-----------|:----:|:------:|-----------|---------|:-------:|
| A1 | 选品扫描 Agent | 核心 | L2/L3 | SCM-01 | agents-doc §2 | [规划中] |
| A2 | Listing 优化 Agent | 核心 | L2 | OPS-07, MKT-09 | agents-doc §3 | [规划中] |
| A3 | 广告调价 Agent | 核心 | L1/L2 | MKT-01 | agents-doc §4 | [规划中] |
| A4 | 多语言客服 Agent | 核心 | L1/L2 | CS-01~04 | agents-doc §5 | [规划中] |
| A5 | 库存预警 Agent | 核心 | L1/L2 | SCM-04/06 | agents-doc §6 | [已实现] |
| A6 | 利润监控 Agent | 核心 | L1/L3 | FIN-01, OPS | agents-doc §7 | [规划中] |
| A7 | 合规检测 Agent | 核心 | L1/L2 | FIN-04~07 | agents-doc §8 | [规划中] |
| G1 | 运营驾驶舱 Agent | 空白填补 | L1/L2 | OPS, MGT | roles-syn §6.2 | [规划中] |
| G2 | 仓储海关 Agent | 空白填补 | L1/L2 | SCM-04~08 | panorama Tier 1 | [规划中] |
| G3 | 折扣风控 Agent | 空白填补 | L1 | OPS, FIN | panorama §7.1 | [已实现] |

### 3.2 各 Agent 决策点与初始进化阶段

| Agent | 决策点 | 默认初始阶段 | 最高可达 | 状态 |
|-------|--------|:----------:|:------:|:----:|
| **A3 广告调价** | ACoS 调整 | Stage 1 | Stage 3 | [规划中] |
| | 关键词发现 | Stage 0 | Stage 2 | [规划中] |
| | 广告活动暂停 | Stage 1 | Stage 3 | [规划中] |
| | 预算分配 | Stage 0 | Stage 2 | [规划中] |
| **A4 客服** | FAQ 回复 | Stage 2 | Stage 3 | [规划中] |
| | 复杂投诉回复 | Stage 1 | Stage 2 | [规划中] |
| | 退款决策 | Stage 0 | Stage 2 | [规划中] |
| **A5 库存预警** | 库存状态检查 | Stage 1 | Stage 3 | [已实现] |
| | 补货量计算 | Stage 1 | Stage 3 | [已实现] |
| | 物流方式选择 | Stage 1 | Stage 2 | [已实现] |
| **A6 利润监控** | 异常检测 | Stage 2 | Stage 3 | [规划中] |
| | 归因分析 | Stage 0 | Stage 2 | [规划中] |
| **G1 驾驶舱** | 日报生成 | Stage 2 | Stage 3 | [规划中] |
| | KPI 预警阈值 | Stage 1 | Stage 3 | [规划中] |
| **G3 折扣风控** | 折扣叠加检测 | Stage 3 | Stage 3 | [已实现] |
| | 促销验证 | Stage 2 | Stage 2 | [已实现] |

### 3.3 已实现 Agent 详情

#### A5 库存预警 Agent `backend/app/agent/agents/inventory_alert.py`

**三级预警体系**:

| 级别 | 触发条件 (可售天数) | 响应动作 |
|:----:|------|---------|
| 🔴 红色 | ≤7 天 (快消品≤3天) | 暂停广告 → 通知采购紧急补货 → 通知运营考虑提价 → 推送管理层 |
| 🟡 黄色 | 7-14 天 (快消品3-7天) | 生成补货建议 → 推送采购确认 → 广告酌情降投 |
| 🟢 绿色 | >14 天 | 日常监控，无需干预 |

**补货数量公式**:

```
补货建议量 = 
    安全库存覆盖天数 × 日均销量预测
  - 当前可售库存
  - 在途库存
  + 缓冲量(max(7天销量, 最低MOQ))

其中:
  日均销量预测 = Holt-Winters 时序模型 (考虑季节因子 + 趋势)
  安全库存覆盖天数 = 采购提前期 × 1.5 (含缓冲)
  最低MOQ = 供应商最低起订量
```

**物流决策树**: 红色预警→空运/快递；黄色预警→海运/铁路；绿色→常规海运

#### G3 折扣风控 Agent `backend/app/agent/agents/discount_risk.py`

**检测逻辑**:

```
每次促销设置变更 → 扫描所有生效折扣:
├── 同一ASIN上生效折扣数 ≥ 2 → 模拟叠加后价格
│   ├── 折后价 < 成本价 → 🚫 阻断 + 推送运营确认
│   ├── 折后价 < 成本价×1.1 → ⚠️ 预警（利润率不足10%）
│   └── 折后价 ≥ 成本价×1.1 → ✅ 放行
└── 跨平台折扣冲突检测 (如亚马逊和独立站同时大幅降价)
    └── 价差>30% → ⚠️ 可能触发平台"最低价"条款
```

---

## 4. Agent 协同链

[已规划，尚未实现独立编排层]

### 4.1 断货应急链（A5→A3→A4）

```
A5 检测到断货(可售<3天)
  → 推送采购: "SKU X 即将断货，紧急补货方案"
  → 触发 A3: 自动暂停该 SKU 所有广告
  → 触发 A4: 加载"延迟发货FAQ话术"准备
  → 通知 G1: 标记红色预警 SKU
```

### 4.2 质量异常追溯链（A4→A6→A5→A2）

```
A4 检测到某 SKU 退货/差评激增 (>2倍周均值)
  → 触发 A6: 核算该 SKU 退货损失
  → 触发 A5: 质检在途/在库库存
  → 通知 A2: 检查评论是否含"缺陷/损坏"关键词
  → 推送运营+质检: "SKU X 质量问题预警"
```

### 4.3 大促备战链（全 Agent 协同）

```
T-30天: A6 历史大促ROI → A5 备货量模拟 → G3 折扣策略模拟
T-7天:  A1 竞品策略扫描 → A2 A+内容生成 → A4 大促FAQ加载
T-Day:  A3 大促调价模式 → A5 实时库存(每30分钟) → A4 全量FAQ自动回复
T+3天:  A6 大促ROI终算 → A4 售后集中处理
```

### 4.4 竞品响应链（A1→A2→A3→A6）

```
A1 检测到竞品降价>15% 或推出高度相似新品
  → 触发 A2: 分析竞品Listing差异 → 生成优化建议
  → 触发 A3: 竞品防御策略 → 调整出价/增加防守关键词
  → 触发 A6: 模拟降价后利润影响
  → 推送运营决策
```

---

## 5. 个人规则系统

### 5.1 四类规则引擎

| 规则类型 | 作用 | 优先级 | 示例 |
|---------|------|:-----:|------|
| **veto** (否决) | 阻断特定操作，不可被覆盖 | 最高 | "永远不自动发布这类社媒内容" |
| **threshold** (阈值) | 数值参数的个性化上下界 | 高 | "不出价 $3 以上" |
| **strategy** (策略) | 多个可行方案中的偏好选择 | 中 | "优先推联盟产品而非闪购" |
| **style** (风格) | 沟通风格适配 | 低 | "正式、数据驱动的回复风格" |

### 5.2 规则生命周期状态机

[已实现: 基础 active/shadow/paused/retired 四态]
[规划中: 完整的 active→shadow→stale→tombstone 四态 + 宽限期机制]

```
          ┌──────────────────────────────────────────┐
          │          规则生命周期状态机                │
          ├──────────────────────────────────────────┤
          │                                          │
          │  [*] → active (规则创建)                  │
          │                                          │
          │  active → shadow (置信度<0.5 或 创建后     │
          │                    5周期无触发)            │
          │  active → stale (30天未触发 或 被覆盖>50次 │
          │              或 上下文漂移>0.3)            │
          │                                          │
          │  stale → tombstone (14天宽限期满)         │
          │  stale → active (Nudge 中明确保留)         │
          │                                          │
          │  shadow → active (5-10周期验证通过)        │
          │  shadow → tombstone (验证失败)             │
          │                                          │
          │  tombstone → [*] (硬删除, 7天后)          │
          │                                          │
          └──────────────────────────────────────────┘
```

### 5.3 规则冲突仲裁

```
仲裁优先级 (从高到低):
1. Veto 规则 (否决规则)               ← 最高，不可覆盖
2. 用户手动创建的规则                  ← 显式意图优先
3. Nudge 确认的规则                    ← 被对话确认过的
4. 自动提取的规则 (置信度≥90%)         ← 高置信自动
5. 模板克隆的规则                      ← 初始值
6. 自动提取的规则 (置信度<90%)         ← 低置信自动
7. 团队共享规则                        ← 最低

同优先级: 按规则类型排序 (veto > threshold > strategy > style)
同类型: 取最新创建的（用户最近意图优先）
```

[已实现: 基础优先级仲裁；规划中: 完整冲突日志 + Nudge 推送冲突解决]

### 5.4 差异化 TTL 策略

[规划中]

```
DEFAULT_TTL_MAP:
  (threshold, manual):       180 天
  (strategy, manual):        180 天
  (style, manual):           365 天
  (veto, manual):            365 天
  (threshold, nudge):        120 天
  (strategy, nudge):         120 天
  (style, nudge):            180 天
  (threshold, auto_extracted): 60 天 (高置信) / 30 天 (低置信)
  (strategy, auto_extracted):  60 天 (高置信) / 30 天 (低置信)
  (style, auto_extracted):   90 天
  (threshold, template):     90 天
  (strategy, template):      90 天
  (style, template):         180 天

动态调整:
  每次规则成功应用且未被覆盖 → TTL *= 1.1 (上限 2x)
  每次规则被覆盖 → TTL *= 0.8 (下限 0.5x)
```

### 5.5 四类可学习信号

| 信号类型 | 学习内容 | 存储位置 | 示例 |
|---------|---------|:------:|------|
| **阈值偏好** | 数值参数的个性化上下界 | Episodic Memory → 规则库 | "不把出价调到 $3 以上" |
| **策略选择** | 在多个可行方案中的偏好排序 | Episodic Memory → 规则库 | "优先推联盟产品而非闪购" |
| **沟通风格** | 回复的正式度、长度、数据密度 | Honcho 用户模型 | "正式、数据驱动、加图表" |
| **否决模式** | 用户明确拒绝的操作类型 | Veto Gate | "永远不自动发布这类社媒内容" |

**信号提取流程**: 监测→差异分析→累积(≥3次)→Nudge确认→写入规则库

---

## 6. Hermes 自净化系统（核心）

> 这是本系统的核心差异化能力。基于 Hermes 架构第 17 章（熵管理系统）和熵管理扩展文档的完整设计。

### 6.1 核心洞察

```
熵增不是状态切换，而是时间序列上的累积过程
  → 规则数量↑不可怕，规则数量的加速度↑才可怕

熵增在系统的生产数据中留下可追踪的信号
  → 只要采集了正确维度的遥测数据

引入大数据视角的本质不是"存更多数据"
  → 而是将自省能力从"看当前状态快照"升级为"看趋势、看分布、看因果"
```

### 6.2 五种熵增形态

| 形态 | 定义 | 指标 | 根源 |
|:----:|------|------|------|
| **① 规则膨胀** | 规则数量无上限增长 | R(t), R'(t), R''(t) | 自动提取过敏感、Nudge 频率过高、模板与自动规则不合并 |
| **② 规则冲突** | 同决策点多条规则输出互斥动作 | conflict_rate, override_rate | 模板 vs 自动、旧规则 vs 新规则、通用 vs 场景特化 |
| **③ 语义漂移** | 规则前提已变但规则未更新 | 上下文分布 KL 散度 | 业务环境变化、市场迁移、品类切换 |
| **④ 记忆污染** | 低质/过时/错误样本进入情景记忆 | regret_marked 占比、低权重样本占比 | 用户犯错、外部冲击、分阶段标准不同 |
| **⑤ 认知过载** | Agent 和用户双方决策负荷超标 | 延迟 P99、Nudge 频率、规则嵌套深度 | 规则链过长、Nudge 过频、用户决策疲劳 |

### 6.3 反应式五道防线

[已实现: 5 道防线框架 (backend/app/agent/entropy/defenses.py) 含基础实现]
[部分实现: 需要完善 Mark Change 审计集成和 Nudge 推送联动]

```
防线1: TTL 强制过期                       防线2: Merge 规则合并
(每条规则绑定 TTL, 到期自动进入 retired)   (每周扫描语义相似规则, 合并重复)
         │                                          │
         └──────────────┬───────────────────────────┘
                        ▼
防线3: Regret 遗憾检测
(反事实模拟: 未应用此规则是否更好?
 遗憾率 >30% 且样本量 ≥10 → 自动暂停)
         │
         ▼
防线4: Decay 衰减降权
(规则优先级随未使用天数自动衰减)
         │
         ▼
防线5: Budget 预算硬限制
(小决策 ≤3 条, 中决策 ≤5 条, 大决策 ≤8 条;
 绝对上限: 所有 Agent 规则总数 ≤200 条)
```

#### 6.3.1 防线实现详情

**防线1 — TTLSweeper** [已实现]:
- 扫描 90 天未触发的活跃规则
- 自动标记为 `retired`
- 记录 Mark Change 审计日志

**防线2 — BudgetEnforcer** [已实现]:
- 按规则类型设上限: veto≤10, threshold≤20, strategy≤15, style≤10
- 超出规则的按最近触发时间降序裁剪至影子模式

**防线3 — DecayScheduler** [已实现]:
- 每次运行对 30 天未使用规则优先级降权
- 单次衰减率: 0.05 (下限 0.1)

**防线4 — MergeDetector** [已实现]:
- 检测同 Agent+决策点+类型的重复规则
- 合并: 累加计数、取最大置信度、保留历史溯源

**防线5 — RegretAnalyzer** [已实现]:
- 检测变更导致置信度降幅 >0.15 的事件
- 执行回滚并记录 Mark Change

### 6.4 五层预测式架构

[规划中: 需要大数据基础设施支持]

```
Layer 5: 系统级熵指数     ← 全局视图，G1 驾驶舱仪表盘
    EntropyIndex = w1 × RuleHealth(avg) + w2 × (1 - ContradictionRate)
                 + w3 × EvolutionStability + w4 × AdoptionTrend
                 + w5 × (1 - OverrideRate)
    | 熵指数  | 状态  | 动作          |
    |:-------:|:----:|--------------|
    | 80-100  | 健康  | 正常运行      |
    | 60-79   | 警告  | 触发 Layer 1-3 |
    | 40-59   | 退化  | 暂停遗憾率最高规则 |
    | 0-39    | 危机  | 冻结所有自动进化 |

Layer 4: 进化速度监控     ← 每个决策点的进化频率
    过快(>3次/周) → 系统震荡
    过慢(30天零变更但有遗憾规则) → 系统僵化
    理想: 每周 0.5-2 次变更

Layer 3: 跨Agent矛盾检测  ← A3 和 A5 对同一产品的建议是否冲突
    向量化建议 → 余弦相似度 <0.3 且双方在 Stage 2+
    → 矛盾 ≥3 次 → Nudge 用户仲裁

Layer 2: SPC 控制图       ← 统计过程控制 [已实现: 基础框架]
    四张控制图: 冲突率/覆盖(采纳)率/延迟/Gini系数
    Western Electric 规则:
    Rule-1: 单点超出 3σ → 立即告警
    Rule-2: 连续 7 点同侧 → 系统性偏移
    Rule-3: 连续 7 点单调上升/下降 → 趋势失控
    Rule-4: 连续 3 点中 2 点超出 2σ → 接近失控
    Rule-5: 连续 15 点在 ±1σ 内 → 过度稳定

Layer 1: 规则健康评分     ← 每条规则的实时健康度 [已实现]
    5 维度加权:
    - 采纳率 (30%) : 1 - 覆盖次数/应用次数
    - 置信度 (25%) : 原始置信度
    - 新鲜度 (20%) : 近 180 天衰减
    - 频率 (15%)   : sigmoid(应用次数)
    - 类型权重 (10%): veto > threshold > strategy > style
    风险等级: unhealthy(<0.40), warning(<0.60), healthy(≥0.60)
```

### 6.5 后悔驱动机制

[已实现: RegretAnalyzer；规划中: 完整 Regret Detector + Shadow Trial 对比]

**后悔检测原则**: 不是"用户覆盖了规则"就算后悔，而是"执行结果与用户期望偏差≥20%"才算后悔。

```
Regret Detection:
1. 仅检查 user_action == 'modified' 的决策
2. 计算原始输出与实际执行的偏差 (deviation)
3. deviation ≥ 20% → 归责到对应规则
4. 对归责规则:
   - 置信度 ×= 0.7
   - 置信度 < 0.5 → 降为 shadow
   - 同一规则 30 天内连续 3 次后悔 → 触发 Nudge
```

**Shadow Trial** [规划中]:

```
规则降入 shadow 模式后:
1. 继续同时执行正式规则和 shadow 规则（仅记录，不暴露）
2. 对比 5-10 个周期的效果差异
3. 效果 ≥ 基线 → 恢复 active
4. 效果 < 基线 → 升级 tombstone
```

### 6.6 复杂度预算硬约束

| 限制项 | 阈值 | 突破后行为 | 审查周期 |
|--------|:----:|-----------|:--------:|
| 单决策点匹配规则上限 | 20 条 | 只加载 Top-20（按优先级+置信度） | 实时 |
| 规则嵌套/链式深度 | 3 层 | 拒绝第 4 层规则触发 | 实时 |
| 决策延迟 P99 | 200ms | 触发降级（跳过规则链，直接用 base） | 实时 |
| 单次 Nudge 话题数 | 3 个 | 只推送优先级最高的 3 个 | 每次 Nudge |
| 每日 Nudge 次数 | 2 次 | 超出的 Nudge 合并到次日 | 每日 |
| 用户总规则数 | 200 条 | 触发硬性清理：自动 stale 最低 10% | 每月 |
| 情景记忆每决策点样本数 | 500 条 | 按权重保留 Top-500 | 每月 |
| 用户模型假设(Honcho)数 | 30 个 | 优先保留 confirmed+最新的 | 每月 |

### 6.7 月度全局熵审计

[规划中]

```
monthly_entropy_audit(user_id) → AuditReport:
  1. 规则统计: 总数/按状态分布/按类型分布
  2. 熵增速度: R'(t), R''(t), 拟合增长模式
  3. 冲突检测: 30 天冲突率
  4. 生成建议:
     - 规则增长加速度为正 → 调整自动提取阈值
     - 冲突率 > 5% → 触发规则合并审计
     - stale 规则 > 20 → 推送批量清理 Nudge
```

### 6.8 闭环数据流

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  决策执行    │────→│ 数据采集      │────→│ 熵分析引擎    │────→│ 自动/人工动作  │
│             │     │ 遥测/规则/   │     │ 防线/预测/   │     │ 续期/合并/回滚│
│             │     │ 冲突/后悔    │     │ 矛盾/审计    │     │ /冻结/Nudge  │
└─────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
       ↑                                                          │
       └──────────────────────────────────────────────────────────┘
```

---

## 7. 大数据基础设施

### 7.1 时序遥测层

[规划中: 待部署 TimescaleDB]

**采集粒度与保留策略**:

```
采集粒度: 每分钟 1 行/决策点 × 10 Agent × 20 决策点 = 200 行/分钟
每日行数: 200 × 60 × 24 = 288,000 行
年行数: ~1.05 亿行

原始数据保留: 90 天 (~2600 万行)
TimescaleDB 压缩率: 10-20x
实际存储占用: < 5 GB/年
```

**遥测指标体系**:

```
每条遥测 (user_id, agent_id, decision_point, timestamp):

entropy_core:
  · rules_active / rules_stale / rules_shadow / rules_tombstone

conflict_signals:
  · conflict_rate_1h / conflict_rate_24h
  · override_rate_1h / override_rate_24h

performance_signals:
  · decision_latency_p50/p95/p99_ms
  · token_per_decision / cache_hit_rate

distribution_signals:
  · rule_trigger_gini / rule_trigger_entropy
  · context_embedding_centroid / context_embedding_variance

quality_signals:
  · regret_rate_24h / acceptance_rate_24h
  · avg_confidence / nudge_skip_rate

推导指标 (不存储，实时计算):
  entropy_velocity: d(rules_active)/dt, d(conflict_rate)/dt, etc.
  entropy_acceleration: d²(rules_active)/dt², etc.
```

### 7.2 分布漂移检测

[规划中: 需要 pgvector 支持]

```
漂移检测: KL 散度 + Wasserstein 距离

| KL 散度范围 | 级别 | 动作 |
|:----------:|:----:|------|
| < 0.1      | STABLE | 无动作 |
| 0.1-0.3   | MILD | 标记规则为"需复审" |
| 0.3-0.5   | SIGNIFICANT | 降为 shadow 模式 |
| > 0.5     | SEVERE | 紧急 Nudge 推送 |
```

### 7.3 因果归因

[规划中: 需要 ≥50 个规则触发样本 + ≥100 个对照样本]

```
方法 1: 差分法 (Difference-in-Differences)
  ATT = (Y_treat_post - Y_treat_pre) - (Y_control_post - Y_control_pre)
  区分: 规则本身有问题 vs 外部因素 vs 效果有滞后

方法 2: 合成控制法 (Synthetic Control)
  为每个触发决策构造一个"未触发规则的平行宇宙"
  合成控制 = 加权平均"未触发规则的相似决策"
```

### 7.4 跨用户模式发现

[规划中: 严格遵守隐私红线]

```
安全区内可做（v1.0）:
  · 熵增速度分群 (K=3): 快速收敛 / 缓慢增长 / 指数膨胀
  · 规则废弃路径分析: 哪种规则最容易 stale
  · TTL 默认值自动调优
  · 异常行为模式标记 (偏离自身基线)

红线外不可做:
  ❌ 协同过滤 ("类似用户也有这条规则，推荐你也创建")
  ❌ 跨用户规则共享/推荐
  ❌ 基于用户相似度的任何自动操作
  ❌ 个人数据跨用户泄露
```

### 7.5 Cron 调度表

[已实现: 基础防御运行；规划中: 完整 Cron 调度]

```python
ENTROPY_CRON_SCHEDULE = {
    'every_minute': [
        'collect_telemetry',           # 采集遥测数据
        'check_spc_rules',             # SPC 异常检测
    ],
    'daily (UTC 02:00)': [
        'compute_entropy_velocity',    # 计算熵增速度
        'check_rule_staleness',        # 规则 TTL 检查
        'detect_distribution_drift',   # 分布漂移检测
        'update_entropy_scores',       # 更新规则熵值评分
    ],
    'weekly (Mon UTC 02:00)': [
        'run_rule_consolidation_scan', # 规则合并候选扫描
        'run_regret_analysis',         # 后悔模式分析
        'prune_episodic_memory',       # 情景记忆预算维护
        'run_spc_baseline_recalc',     # SPC 控制限重新计算
    ],
    'monthly (1st UTC 02:00)': [
        'run_causal_attribution',      # 完整因果归因分析
        'run_cross_user_pattern',      # 跨用户模式发现
        'generate_monthly_audit',      # 月度熵审计报告
        'update_ttl_defaults',         # TTL 默认值全局调优
        'check_complexity_budget',     # 复杂度预算合规检查
    ],
}
```

### 7.6 数据基础设施架构

```
单机 PostgreSQL 16+ (16GB RAM + 100GB SSD) 完全满足

┌──────────────────────────────────────────────┐
│              PostgreSQL 16+                   │
│ ┌────────────────────────────────────────┐   │
│ │  关系表 (agent_decisions, personal_rules, │   │
│ │  rule_conflicts, agent_episodes, etc.)   │   │
│ ├────────────────────────────────────────┤   │
│ │  pgvector 扩展 (上下文 embedding 向量)    │   │
│ │  FTS 全文搜索 (决策日志关键词召回)         │   │
│ ├────────────────────────────────────────┤   │
│ │  TimescaleDB 扩展 (agent_telemetry 时序)  │   │
│ │  - Hypertable (按天分片)                │   │
│ │  - 自动压缩 (10-20x)                    │   │
│ │  - 连续聚合 (日/周/月降采样)              │   │
│ │  - 90 天保留策略                         │   │
│ └────────────────────────────────────────┘   │
└──────────────────────────────────────────────┘

不需要: Hadoop, Spark, Flink, Kafka, 分布式数据库集群
年均数据增量 < 10 GB
```

---

## 8. 数据库 Schema 总览

### 8.1 AI Agent 系统模型

以下为 `backend/app/models.py` 中已实现的 7 个 AI Agent 模型:

| 模型 | 表名 | 核心字段 | 用途 |
|:----:|:----:|---------|:----:|
| **AgentDecision** | `agent_decision` | user_id, agent_id, decision_point, context_json(JSONB), agent_output(JSONB), final_decision(JSONB), user_action, user_overrides(JSONB), user_feedback, rules_applied(JSONB), rule_overrides, evolution_stage, confidence, response_time_ms, token_count, session_id, episode_id | 决策日志主表，类比 Hermes Episodic Memory |
| **PersonalRule** | `personal_rule** | user_id, agent_id, decision_point, rule_type, rule_name, rule_condition(JSONB), rule_action(JSONB), priority, source, source_decisions(UUID[]), status, confidence, times_applied, times_overridden, last_applied_at | 个人规则库，四类规则(threshod/strategy/style/veto) |
| **AgentEpisode** | `agent_episode` | user_id, agent_id, episode_number, decision_count, episode_summary, key_insights(JSONB), improvement_suggestions(JSONB), acceptance_rate, avg_confidence, nudge_triggered, nudge_topics(JSONB), nudge_response | 每 15 个任务汇总为 Episode |
| **HonchoProfile** | `honcho_profile` | user_id(UNIQUE), risk_tolerance, communication_style, notification_prefs(JSONB), agent_profiles(JSONB), hypothesis_count, confirmed_count, last_dialectic_at | 用户模型辩证档案 |
| **RuleConflict** | `rule_conflict` | decision_id(FK), conflicting_rules(UUID[]), winner_rule_id, resolution, nudge_sent, nudge_resolved, user_choice | 规则冲突日志 |
| **RuleMarkChange** | `rule_mark_change` | target_type, target_id, field_path, old_value, new_value, source_type, source_id, change_summary, parent_change_id, related_decision_ids(UUID[]), context_json(JSONB) | 规则变更审计 (Mark Change Pattern) |
| **SpcControlLimit** | `spc_control_limit` | user_id, agent_id, decision_point, metric_name, baseline_mean, baseline_stddev, ucl, lcl, uwl, lwl, consecutive_same_side, last_breach_at, baseline_recalc_at, next_recalc_at | SPC 控制限基线 (UNIQUE on 四维) |

### 8.2 待新增模型

| 表名 | 用途 | 优先级 | 依赖 |
|:----:|------|:----:|:----:|
| `rule_merge_candidates` | 合并候选记录 (rule_a_id, rule_b_id, similarity, shadow_result, status, merged_rule_id) | Phase 2 | MergeDetector |
| `rule_shadow_trials` | Shadow 模式对比数据 (shadow_output, formal_output, actual_decision, shadow_effect, formal_effect) | Phase 2 | RegretAnalyzer |
| `agent_telemetry` | TimescaleDB Hypertable 时序遥测 (rules_active, conflict_rate, latency_p95, gini, etc.) | Phase 3 | TimescaleDB |
| `agent_telemetry_daily` | 连续聚合物化视图 (日均指标汇总) | Phase 3 | TimescaleDB |
| `system_self_monitoring` | 系统自身熵监控 (db_size, cron_health, alert_counts) | Phase 3 | — |

### 8.3 索引策略

| 索引 | 表 | 字段 | 用途 |
|:----:|:--:|:----|:----:|
| PK | agent_decision | id | 主键查询 |
| idx1 | agent_decision | (user_id, agent_id, decision_point, created_at DESC) | 按用户+Agent+决策点查询历史 |
| idx2 | agent_decision | (user_id, agent_id, decision_point, created_at DESC) WHERE user_action IN ('modified','rejected') | 仅被修改/拒绝的决策 |
| idx3 | agent_decision | (evolution_stage, created_at DESC) | 按进化阶段过滤 |
| idx_fts | agent_decision | GIN(search_vector) | 全文搜索 |
| idx1 | personal_rule | (user_id, agent_id, decision_point) WHERE status='active' | 活跃规则查询 |
| pk_time | agent_telemetry | (time, user_id, agent_id, decision_point) | 时序主键 |
| idx_tel | agent_telemetry | (user_id, agent_id, time DESC) | 用户+Agent 时序查询 |

---

## 9. API 设计一览

### 9.1 Agent 基础 API

所有 Agent API 注册于 `/api/v1/agents` 前缀下。

| 方法 | 路径 | 用途 | 权限 | 状态 |
|:----:|------|:----|:---:|:----:|
| GET | `/agents` | 列出所有已注册 Agent | agent:view | [已实现] |
| GET | `/agents/{agent_id}` | 获取 Agent 元数据 | agent:view | [已实现] |
| POST | `/agents/{agent_id}/decide` | 执行 Agent 决策 | agent:execute | [已实现] |
| GET | `/agents/decisions` | 分页查询决策日志 | agent:view | [已实现] |
| POST | `/agents/decisions/{id}/feedback` | 提交决策反馈 | agent:execute | [已实现] |
| GET | `/agents/rules` | 列出个人规则 | agent:view | [已实现] |
| POST | `/agents/rules` | 创建个人规则 | agent:execute | [已实现] |
| PUT | `/agents/rules/{rule_id}` | 更新个人规则 | agent:execute | [已实现] |
| DELETE | `/agents/rules/{rule_id}` | 删除个人规则 | agent:execute | [已实现] |
| GET | `/agents/profile` | 获取 Honcho 用户档案 | agent:view | [已实现] |
| PUT | `/agents/profile` | 更新 Honcho 用户档案 | agent:execute | [已实现] |
| GET | `/agents/episodes` | 分页查询 Episode | agent:view | [已实现] |

### 9.2 熵系统 API

| 方法 | 路径 | 用途 | 权限 | 状态 |
|:----:|------|:----|:---:|:----:|
| GET | `/entropy/dashboard` | 熵管理驾驶舱摘要 | agent:view | [已实现] |
| POST | `/entropy/defend` | 运行 5 道防线 | agent:execute | [已实现] |
| GET | `/entropy/health` | 规则健康评分列表 | agent:view | [已实现] |
| GET | `/entropy/spc` | SPC 控制图状态 | agent:view | [已实现] |
| GET | `/entropy/changes` | 变更审计日志 | agent:view | [已实现] |

### 9.3 前端路由

| 路径 | 组件 | 用途 | 状态 |
|:----:|:----|:----|:----:|
| `/agent` | AgentList.vue | Agent 列表主页 | [已实现] |
| `/agent/detail/:id` | AgentDetail.vue | Agent 详情 | [已实现] |
| `/agent/rules` | AgentRules.vue | 个人规则管理 | [已实现] |
| `/agent/entropy` | EntropyCockpit.vue | 熵管理驾驶舱 | [已实现] |

### 9.4 待新增 API

| 方法 | 路径 | 用途 | 阶段 |
|:----:|------|:----|:----:|
| POST | `/entropy/defend/{defense}` | 单道防线运行 | Phase 2 |
| GET | `/entropy/telemetry` | 遥测数据查询 | Phase 3 |
| GET | `/entropy/telemetry/velocity` | 熵增速度趋势 | Phase 3 |
| GET | `/entropy/drift` | 分布漂移检测报告 | Phase 4 |
| GET | `/entropy/attribution` | 因果归因分析 | Phase 4 |
| GET | `/entropy/audit/monthly` | 月度熵审计报告 | Phase 5 |

---

## 10. 实施路线图

### 10.1 三阶段总览

```
Phase 1 (当前)                Phase 2 (3-6月)               Phase 3 (6-12月)
─────────────────────         ─────────────────────         ────────────────────
框架底座 ████████████         A3 决策树完善 ░░░░░░░░       A1 选品扫描 ░░░░░░░░
A5 库存预警 ████████████       A4 FAQ引擎 ░░░░░░░░        全链路协同 ░░░░░░░░
G3 折扣风控 ████████████       A6 利润归集 ░░░░░░░░       预测式熵对抗 ░░░░░░░░
熵系统Phase1 ████████████      G1 驾驶舱 ░░░░░░░░         全自动熵审计 ░░░░░░░░
                              A2/A7/G2 ░░░░░░░░
                              规则系统完善 ░░░░░░░░
                              数据基础层 ░░░░░░░░
```

### 10.2 当前阶段 (Phase 1) 详细分解

#### M1: 框架底座 + 核心 Agent + 熵系统 Phase 1 [已完成]

| 交付物 | 状态 | 文件 |
|--------|:----:|------|
| BaseAgent 基类 + 4 阶段生命周期 | [已完成] | `backend/app/agent/base.py` |
| AgentRegistry 注册 + 自动发现 | [已完成] | `backend/app/agent/registry.py` |
| Pydantic 模型 + 请求/响应 Schema | [已完成] | `backend/app/agent/schemas.py` |
| AgentService 决策管线 + 规则引擎 | [已完成] | `backend/app/agent/service.py` |
| 12 个 RESTful 端点 | [已完成] | `backend/app/agent/router.py` |
| 7 个 DB 模型 (AgentDecision, PersonalRule, etc.) | [已完成] | `backend/app/models.py` |
| A5 库存预警 Agent (三级预警 + 补货 + 物流) | [已完成] | `backend/app/agent/agents/inventory_alert.py` |
| G3 折扣风控 Agent (叠加检测 + 促销验证) | [已完成] | `backend/app/agent/agents/discount_risk.py` |
| 2 个 Alembic 迁移 | [已完成] | `migrations/` |
| 熵系统 5 道防线 (TTL/Budget/Decay/Merge/Regret) | [已完成] | `backend/app/agent/entropy/defenses.py` |
| Layer 1 规则健康评分 | [已完成] | `backend/app/agent/entropy/health_score.py` |
| Layer 2 SPC 控制图 | [已完成] | `backend/app/agent/entropy/spc_control.py` |
| Mark Change 审计日志 | [已完成] | `backend/app/agent/entropy/service.py` |
| Entropy 驾驶舱 API (5 端点) | [已完成] | `backend/app/agent/entropy/router.py` |
| 前端 4 个 Vue 组件 | [已完成] | `frontend/src/views/agent/*.vue` |
| 前端 API 模块 + 路由 | [已完成] | `frontend/src/api/modules/agent.ts` |

#### M2: A3 广告调价 + A4 多语言客服 + 规则系统完善 [进行中]

| 交付物 | 状态 | 说明 |
|--------|:----:|------|
| A3 广告调价决策树 | [规划中] | ACoS 阈值决策 + Ads API 对接 |
| A4 FAQ 自动回复引擎 | [规划中] | 意图识别 + 知识库 RAG + Amazon Messaging API |
| 规则 TTL 差异化策略 | [规划中] | 按类型×来源的动态 TTL 表 |
| 规则生命周期状态流转 | [规划中] | shadow→stale→tombstone 完整流转 |
| Nudge 机制 | [规划中] | 后悔冲突解决 Nudge |
| 前端补充 | [规划中] | Honcho 档案页面、Episode 浏览页 |

#### M3: A6 利润监控 + G1 驾驶舱 + 熵系统 Phase 2

| 交付物 | 状态 | 说明 |
|--------|:----:|------|
| A6 多平台利润归集 | [规划中] | 收入矩阵 + 异常检测 + 汇率对冲 |
| G1 核心看板 | [规划中] | 销售/广告/库存/利润 KPI 看板 |
| merge_candidates 模型 | [规划中] | 合并候选记录表 |
| shadow_trials 模型 | [规划中] | Shadow 模式对比数据表 |
| 规则合并 Nudge 集成 | [规划中] | 相似规则推送 → 用户确认合并 |

### 10.3 Phase 2 规划 (3-6月)

| 月份 | 里程碑 | 关键交付物 |
|:----:|--------|-----------|
| M4 | A2 Listing 优化 MVP | 标题/五点/描述 AI 优化 + 关键词挖掘 + A/B 测试框架 |
| M4 | A7 合规检测 MVP | Amazon Account Health 监控 + 政策变更抓取 + 商标初筛 |
| M5 | G2 仓储海关 MVP | 多仓库库存同步 + 三单匹配报关 + 入库建议 |
| M5 | P0 Agent 算法优化 | A3 多目标优化 / A4 多轮对话 / A5 预测升级 |
| M6 | 跨 Agent 协同 MVP | 断货应急链 + 竞品响应链 + 质量异常追溯链 |

### 10.4 Phase 3 规划 (6-12月)

| 月份 | 里程碑 | 关键交付物 |
|:----:|--------|-----------|
| M7-8 | A1 选品扫描上线 | 1688+亚马逊全品类扫描 + 多维评分 + 利润预估 |
| M9-10 | 全 Agent 协同层 | 大促备战链 + 自动化策略联动 + 人工审核工作台 |
| M11-12 | AI 预测能力升级 | LSTM 时序预测 + 因果推断归因 + 全自动熵审计 |

### 10.5 熵管理系统实施路线图

```
阶段一: 反应式基础 (M1-M2) [已完成]
  ├── TTL stale 自动标记 ✓
  ├── 复杂度预算硬限制 ✓
  └── Layer 1 规则健康评分 ✓

阶段二: 反应式进阶 (M2-M4) [进行中]
  ├── 规则自动合并 (Nudge 集成)
  ├── 后悔回滚 (Shadow Trial)
  ├── 记忆自动衰减
  └── Layer 2 SPC 完善 + 前端可视化

阶段三: 预测式基础 (M4-M6) [规划中]
  ├── TimescaleDB 遥测部署
  ├── entropy_velocity 计算
  ├── SPC 控制图全自动化
  └── 熵值仪表盘完善

阶段四: 预测式进阶 (M7-M9) [规划中]
  ├── 分布漂移检测引擎 (KL 散度 + Wasserstein)
  ├── 因果归因 (DiD + 合成控制)
  └── Layer 5 熵指数

阶段五: 全域优化 (M10-M12) [规划中]
  ├── 跨用户模式聚类
  ├── TTL 全局自动调优
  └── 月度审计全自动化
```

---

## 11. 成功指标

### 11.1 Phase 1 指标

| 指标 | 基线 | 目标 |
|------|:---:|:----:|
| Agent 框架可用性 | N/A | 10 Agent 注册 + 自动发现 |
| 规则引擎采纳率 | N/A | 规则匹配延迟 < 50ms |
| Layer 1 健康评分覆盖 | N/A | 100% 规则健康可分 |
| Layer 2 SPC 覆盖 | N/A | 3 个决策点建立基线 |
| 折扣叠加事故 | 月均 1-2 次 | 0 次 |
| 熵防线运行 | N/A | 5 道防线全部可触发 |
| 前端仪表盘 | N/A | 4 个页面全部可用 |

### 11.2 Phase 2-3 指标

| 指标 | 基线 | 目标 |
|------|:---:|:----:|
| 客服 FAQ 自动回复率 | ~0% | ≥60% |
| ACoS 优化幅度 | 基线 | -15~25% |
| 库存预警准确率 | N/A | ≥90% |
| 利润核算时效性 | T+1~3 | T+0 |
| 熵指数(P50) | N/A | ≥75 (健康) |
| 规则自动清理率 | N/A | 每月 ≥10% stale |
| 跨 Agent 协同链 | N/A | 4 条链全部上线 |
| 全自动熵审计 | N/A | 月度自动完成 |

---

## 12. 跨文档引用映射

### 12.1 来源文档

| 文档 | 路径 | 关键章节引用 |
|------|------|-------------|
| Hermes Agent 深度解析 & 自进化方案 | `hermes-self-evolving-agent-design (1).md` | Ch2(GEPA), Ch3(Honcho), Ch11(四阶段), Ch17(熵管理) |
| 熵管理扩展文档 | `hermes-entropy-management-system.md` | Ch2(五道防线), Ch3(预测式架构), Ch5(数据基础设施) |
| 7+3 Agent 综合方案 | `final-integrated-solution.md` | §5(Agent矩阵), §6(P0设计), §9(路线图) |
| 业务场景需求文档 | `跨境电商AI_Agent业务场景需求文档.md` | §1(运营日流), §2(选品链路), §3(广告操作), §4(客服场景) |
| 深度调研报告 | `跨境电商AI_Agent深度调研报告.md` | §2(痛点排名), §4(7 Agent设计) |

### 12.2 本文档 vs 设计文档的关系

| 本文档章节 | 对应设计文档 | 变化说明 |
|-----------|------------|---------|
| §2 Agent 框架 | Hermes Ch11 + Ch14 | 简化了 GEPA 三层记忆（聚焦决策日志而非全 Hermes 实现） |
| §3 7+3 Agent | final-integrated-solution §5 + §6 | 保持完全一致 |
| §5 规则系统 | Hermes Ch10 + Ch15 | 扩展了 TTL 差异化策略和完整生命周期状态机 |
| §6 自净化 | Hermes Ch17 + 熵管理扩展 | 是本文档核心，整合了反应式+预测式层，增补实现状态标注 |
| §7 大数据 | 熵管理扩展 §3 + §5 | 保持设计一致，标注了实施优先级 |
| §8 Schema | Hermes Ch14 | 增补了待新增的三个表和索引策略 |
| §10 路线图 | final-integrated-solution §9 + 熵管理扩展 §6 | 合并了两份路线图，反映当前实现状态 |

### 12.3 与现有代码的关系

| 代码路径 | 对应文档章节 | 状态 |
|---------|-------------|:----:|
| `backend/app/agent/` | §2 | 已实现，覆盖框架底座 |
| `backend/app/agent/entropy/` | §6 | 已实现，覆盖 5 道防线 + Layer 1-2 |
| `backend/app/models.py (Agent 模型)` | §8 | 已实现，7 个模型 |
| `frontend/src/views/agent/` | §9.3 | 已实现，4 个页面 |
| `frontend/src/api/modules/agent.ts` | §9.1-9.2 | 已实现 |
| `frontend/src/router/modules/agent.ts` | §9.3 | 已实现 |

---

> **文档结束**
>
> **核心宣言**: 生长必须伴随凋亡。在跨境电商 AI Agent 系统中，熵管理（自净化）不是可选的锦上添花，而是确保系统在持续进化中不自我退化的必要条件。本文件作为统一设计规格书，完整覆盖了从实现状态到远期规划的完整路径。
