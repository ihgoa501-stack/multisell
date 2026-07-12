# Hermes Agent 深度解析 & 自进化Agent集成方案

> **定位**：本文档是对 Hermes Agent（NousResearch）核心架构的深度技术解析，以及将其自进化机制集成到跨境电商 7+3 Agent 体系中的完整设计方案。
>
> **前置文档**：本文档是 final-integrated-solution.md（7+3 Agent综合方案）的扩展。
>
> **版本**：v1.0 | **日期**：2026-06-16

---

## 目录

1. Hermes Agent 五柱架构全景
2. 三层记忆系统 GEPA
3. Honcho 辩证用户建模
4. 技能闭环：从执行到创造
5. 自进化引擎：GEPA + DSPy
6. Nudge 机制：定期行为对齐
7. Cache-Aware 性能优化
8. 集成架构总览
9. 三层分离设计
10. 四类可学习信号
11. 四阶段渐进进化策略
12. 暖启动机制
13. 复杂度控制红线
14. 决策日志数据库Schema
15. 规则冲突仲裁逻辑
16. 完整进化示例：广告调价Agent
17. 熵管理系统

---

## 1. Hermes Agent 五柱架构全景

Hermes Agent 由五个相互咬合的子系统组成，形成一个自给自足的闭环生态：

```
┌──────────────────────────────────────────────────────────────────┐
│                     Hermes Agent Architecture                      │
│                       (Five-Pillar Design)                         │
├────────────┬────────────┬────────────┬────────────┬───────────────┤
│  MEMORY    │  SKILLS    │   SOUL     │   CRONS    │ SELF-IMPROVE  │
│  (GEPA)    │ (Procedural│ (Persona)  │(Scheduling)│   MENT        │
│            │  Memory)   │            │            │  (Gateway)    │
├────────────┼────────────┼────────────┼────────────┼───────────────┤
│ • Working  │ • YAML     │ • Honcho   │ • Periodic │ • DSPy + GEPA │
│   Memory   │   Bundles  │   User     │   Task     │   Prompt      │
│ • Episodic │ • Auto-    │   Model    │   Trigger  │   Evolution   │
│   Memory   │   creation │ • Cross-   │ • Blog     │ • Eval        │
│ • Semantic │   from     │   session  │   Posts    │   Datasets    │
│   Memory   │   workflows│   Identity │ • Commit   │ • PR-based    │
│ • Honcho   │ • Slash    │ • Multi-   │   Logs     │   Validation  │
│   User     │   Command  │   agent    │ • Social   │ • Constraint  │
│   Model    │   Interface│   profiles │   Media    │   Gates       │
└────────────┴────────────┴────────────┴────────────┴───────────────┘
```

**五柱关系表**：

| 支柱 | 核心职责 | 对应跨境电商Agent需求 |
|------|---------|---------------------|
| **Memory** | 跨会话信息持久化，三层记忆架构 | 用户偏好记忆、历史决策回溯、业务知识积累 |
| **Skills** | 程序性记忆，可复用工作流模板 | 广告调价策略模板、选品流程模板 |
| **Soul** | 人格/用户模型，通过Honcho辩证建模 | 用户沟通风格偏好、决策风格学习 |
| **Crons** | 定时任务调度，周期性触发 | 自动日报、库存定时检查、nudge对齐 |
| **Self-Improvement** | 自我进化引擎 | Agent策略进化、规则自动调参 |

关键洞察来自 NousResearch 创始人：五个子系统形成递归递进关系 —— **Memory 为 Skills 提供数据源，Skills 通过 Soul 被个性化执行，Crons 定期触发 Self-Improvement 评估，Self-Improvement 结果又写入 Memory**。这是一个真正的闭环学习架构。[citation:NousResearch Hermes Agent](https://github.com/NousResearch/hermes-agent)

---

## 2. 三层记忆系统 GEPA

GEPA（Generic Episodic-Personal Agentic Memory）是 Hermes Agent 的记忆子系统。它不是简单的"数据库"，而是一个**分层检索架构**。

### 2.1 三层架构详解

```
请求到达
    │
    ▼
┌──────────────────────────────────────────────────┐
│  Layer 1: Working Memory (工作记忆)               │
│  ────────────────────────────────────             │
│  • 存储位置: 系统提示词 (System Prompt)             │
│  • 生命周期: 单会话 (Session-scoped)              │
│  • 淘汰策略: TTL (Time-to-Live) + LRU             │
│  • 容量: ~8K-32K tokens                           │
│  • 内容: 当前会话摘要、最近N轮对话上下文、          │
│          当前任务关键参数                           │
│  • 写入时机: 每轮对话后实时更新                      │
│  • 类比: 人脑的"前额叶工作记忆"                    │
└──────────────────────────────────────────────────┘
    │ 未命中 → 向下一层查询
    ▼
┌──────────────────────────────────────────────────┐
│  Layer 2: Episodic Memory (情景记忆)               │
│  ────────────────────────────────────             │
│  • 存储位置: sqlite-vec + FTS5 混合索引            │
│  • 生命周期: 跨会话永久 (Persistent)               │
│  • 检索方式:                                     │
│     - 向量相似度搜索 (sqlite-vec, cosine)         │
│     - 全文关键词搜索 (FTS5 BM25)                  │
│     - 混合重排序 (Reciprocal Rank Fusion)         │
│  • 粒度: 每15个任务为一个 Episode                  │
│  • 内容: 历史决策记录、成功/失败模式、              │
│          用户反馈、技能使用日志                     │
│  • 特有字段: success_score, skill_used,            │
│              user_feedback_rating, task_complexity  │
│  • 类比: 人脑的"海马体情景记忆"                    │
└──────────────────────────────────────────────────┘
    │ 语义查询 → 向下一层查询
    ▼
┌──────────────────────────────────────────────────┐
│  Layer 3: Semantic Memory (语义记忆)               │
│  ────────────────────────────────────             │
│  • 存储位置: 结构化知识图谱或向量数据库              │
│  • 生命周期: 永久 (Persistent)                    │
│  • 内容:                                          │
│     - 关于世界的事实 (产品知识、平台规则)          │
│     - 关于用户的长周期偏好 (Honcho模型)            │
│     - 关于Agent自身的元知识 (能力边界认知)         │
│  • 更新方式: LLM 提取 + 人工确认 (半自动)          │
│  • 类比: 人脑的"皮层语义网络"                     │
└──────────────────────────────────────────────────┘
```

### 2.2 FTS5 跨会话召回机制

这是 Hermes Agent 实现"记住上次对话"的关键技术：

```sql
-- Hermes 情景记忆表 (简化版)
CREATE VIRTUAL TABLE episodic_memory_fts USING fts5(
    session_id,
    episode_summary,       -- LLM 自动生成的情景摘要
    task_description,      -- 原始任务描述
    user_feedback,         -- 用户显式反馈
    key_decisions,         -- 关键决策点
    tags                   -- 自动打标
);

-- 跨会话查询示例:
-- 用户说 "上次那个ACoS超标的广告怎么样了"
-- Agent 执行:
SELECT episode_summary, key_decisions
FROM episodic_memory_fts
WHERE episodic_memory_fts MATCH '"ACoS" AND "超标"'
ORDER BY rank
LIMIT 5;
-- 然后 LLM 对 Top-5 结果进行语义重排序和摘要
```

流程：**FTS5 快速关键词召回 → LLM 语义重排序 Top-K → 最相关1-3条注入 Working Memory → Agent 获得跨会话上下文**。

### 2.3 记忆写入触发条件

| 触发条件 | 写入层级 | 写入内容 |
|---------|:------:|---------|
| 每轮对话结束 | Working | 对话摘要增量更新 |
| 每15个任务完成 | Episodic | Episode 摘要（LLM 自动生成） |
| 用户给出显式反馈 | Episodic | 反馈内容 + 评分 |
| Agent 发现新事实 | Semantic | LLM 提取 → 人工确认 → 写入 |
| Nudge 触发 | Episodic + Semantic | 行为对齐后的洞察 |

---

## 3. Honcho 辩证用户建模

Honcho 是 Hermes Agent 的用户建模子系统。与传统用户画像不同，Honcho 采用**辩证方法**：Agent 不仅观察用户行为，还主动与用户对话来验证和细化理解。

### 3.1 核心思想：不猜测，去验证

```
传统用户画像 (静态、被动):
  "用户偏好低价商品"  ← 从行为推断，可能错误

Honcho 辩证建模 (动态、主动):
  1. Agent 观察: 用户连续5次选择低价商品
  2. Agent 形成假设: "用户似乎偏好低价策略"
  3. Agent 主动验证: "我注意到您最近5次选品都倾向低价区间。
     这是长期策略偏好，还是当前清仓期的特殊选择?"
  4. 用户确认/修正: 形成精确偏好模型
```

### 3.2 多Agent档案系统

Honcho 支持为同一个用户维护**多个Agent配置档案**，每个独立存储但可交叉引用。具体 JSON 结构包含广告优化器(ACoS阈值、风险容忍度)、选品专员(品类偏好、价区间)、客服Agent(回复风格、退款阈值)等独立档案，每个档案内包含该Agent各决策点的进化阶段标记。

---

## 4. 技能闭环：从执行到创造

技能系统是 Hermes Agent 的程序性记忆。Agent 不仅能执行预定义技能，更能**从成功工作流中提取模式、自主打包为新技能**。

### 4.1 技能生命周期

```

用户需求 ──→ Agent 执行 ──→ 成功完成 ──→ 模式提取
                                              │
                    ┌──────────────────────────┘
                    ▼
              技能创建 ──→ 用户审核 ──→ 注册到技能库
                    │
            ┌───────┘
            ▼
      下次类似需求 ──→ 自动加载技能 ──→ 更快执行
            │
    ┌───────┘
    ▼
技能使用反馈 ──→ 自进化引擎优化 ──→ 技能版本更新
```

### 4.2 技能提取触发条件

| 触发条件 | 说明 |
|---------|------|
| **复杂度门槛** | 任务涉及 3+ 个工具调用 且 跨多个步骤 |
| **重复性检测** | 相似任务在 Episodic Memory 中出现 ≥3 次 |
| **成功率门槛** | 过去 5 次同类任务成功率 ≥80% |
| **用户显式请求** | 用户说 "记住这个流程" / "下次直接这么做" |

### 4.3 技能包格式 (简化YAML)

技能包以 YAML 定义，包含名称、版本、slash命令、触发条件(cron/event)、输入参数(支持从Honcho档案读取默认值)、决策树(condition→action映射)、进化日志等字段。

### 4.4 技能阴影 (Skill Shadowing)

新创建的技能在"阴影模式"下运行 5-10 个周期，不执行实际操作，而是记录"如果使用这个技能会怎样"。阴影期结束后对比实际人工操作与技能建议，评估质量，通过后方可"点亮"。

---

## 5. 自进化引擎：GEPA + DSPy

GEPA（Genetic-Pareto Prompt Evolution）是 Hermes Agent 的核心自进化算法，结合遗传算法和帕累托优化来进化 prompts、工具描述和代码。

### 5.1 GEPA 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│                    GEPA Evolution Cycle                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 种群初始化                                                │
│     ├── 种子 prompt (当前版本)                                │
│     ├── 变异体 × N (GPT-4 生成的 prompt 变体)                │
│     └── 控制组 (上一版本作为 baseline)                        │
│                                                              │
│  2. 适应度评估 (Fitness)                                      │
│     ├── 在 eval dataset 上测试所有变体                        │
│     ├── 多目标评分:                                          │
│     │   ├── accuracy (决策准确率)                             │
│     │   ├── latency (响应时间)                                │
│     │   ├── token_cost (Token 消耗)                           │
│     │   └── user_satisfaction (用户反馈)                      │
│     └── 计算帕累托前沿 (Pareto Front)                        │
│                                                              │
│  3. 约束门控 (Constraint Gates)                              │
│     ├── 单元测试通过率 ≥ 95%                                  │
│     ├── prompt 长度 < 8K tokens                               │
│     ├── 准确率不低于 baseline                                 │
│     └── 新增错误类型 = 0                                      │
│                                                              │
│  4. 选择与交叉 (Selection & Crossover)                        │
│     ├── 从帕累托前沿选择精英                                  │
│     ├── GPT-4 驱动的 prompt 交叉重组                         │
│     └── 变异率: 15%                                          │
│                                                              │
│  5. PR 验证 (Pull Request Validation)                        │
│     ├── 进化结果作为 PR 提交                                  │
│     ├── 自动运行完整测试套件                                  │
│     ├── 人工 review (仅 Phase 1-2)                           │
│     └── 合并后自动部署                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 三个进化层级

| 层级 | 进化对象 | 周期 | 人工介入 |
|:----:|---------|:----:|:------:|
| **L1: Prompt 进化** | System prompt, few-shot examples, tool descriptions | 每周 | PR Review |
| **L2: 参数进化** | ACoS阈值, 调价幅度, 预警阈值 | 每日 | 自动（置信度≥95%） |
| **L3: 代码进化** | Agent 决策逻辑代码 | 每2周 | 强制人工Review |

### 5.3 Eval 数据集来源

1. **历史决策回放**：过去30天真实决策，人工最终决策为 ground truth
2. **用户显式反馈**："这个建议不错/不对" → 标记正/负样本
3. **Nudge 对比**：Agent 行为 vs 用户期望行为 → 差距即改进方向
4. **合成测试用例**：GPT-4 生成的边界场景 → 覆盖率补充

---

## 6. Nudge 机制：定期行为对齐

Nudge 是 Hermes Agent 最独特的设计 —— Agent 不是被动工具，而是一个主动寻求反馈的行为对齐系统。

### 6.1 Nudge 触发与对话示例

```
触发频率: 每 ~15 个任务
时机: 任务间隙（不在用户忙碌时打断）
形式: 自然对话

典型 Nudge:
  Agent: "我注意到过去15个任务中，有3个涉及广告调价的场景。
         我使用的ACoS阈值是15%，实际效果是ACoS从18%降到16.5%。
         不过有1次你手动覆盖了我的建议（从-15%改为-10%）。
         你希望我把默认调价幅度改小一点吗？"

  用户: "那次是因为竞品也在降价，不敢降太多。"

  Agent: "明白。新增规则：竞品同时降价>5%时，调价幅度自动减半。"

  用户: "可以。"
```

### 6.2 Nudge 分类

| 类型 | 触发条件 | 示例 |
|------|---------|------|
| **阈值校准** | 偏差 >20% | "你的ACoS容忍度比我设定的高，要调整吗？" |
| **策略发现** | 一致行为模式 | "你连续5次选海运而非空运，设为默认偏好？" |
| **技能推荐** | 可技能化的重复工作流 | "过去一周你手动做了7次类似竞品分析，创建自动技能？" |
| **空白发现** | 未覆盖场景 | "刚才的TikTok广告无法处理，需要新增能力吗？" |
| **性能报告** | 定期汇总 | "过去30天决策准确率87%，比上月+3%。主要改进..." |

### 6.3 节制设计

- 每次 Nudge 最多 3 个话题
- 用户可选择"跳过"、"本周不提醒"、"永久关闭该类"
- 连续 2 次被跳过的类型自动降频 50%
- Nudge 不在 09:00-11:00 核心工作时段触发

---

## 7. Cache-Aware 性能优化

```
传统方案: 每轮重新构建完整 system prompt
→ 无法复用 LLM provider 的 KV Cache
→ token 消耗和延迟均高

Hermes Cache-Aware:
  会话开始 → 冻结 system prompt 快照 → 计算 hash
  → 后续请求前缀一致 → provider 识别 hash → 复用 KV Cache
  → 仅计算新增 user/assistant 消息
  → token 降低 40-60%，响应延迟降低 30-50%
```

对跨境电商 Agent 的实用价值：
- 高频调用（广告每4h、库存实时）大幅降低 API 成本
- 同一天内多次调用因 Working Memory 前缀一致，直接受益
- 适合支持 Prompt Caching 的 provider（Anthropic Claude、Google Gemini）

---

# 第二部分：跨境电商 7+3 Agent 自进化集成

## 8. 集成架构总览

核心原则：**最小侵入性**——不改变现有 Agent 决策逻辑，以"个人偏好过滤器"方式叠加。

```
┌──────────────────────────────────────────────────────────────────┐
│          自进化跨境电商Agent 架构（Hermes集成版）                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────┐                         │
│  │        通用Agent核心（稳定层）        │  ← 不修改               │
│  │  ┌───────┐ ┌───────┐ ┌───────────┐  │                         │
│  │  │A3广告 │ │A4客服 │ │A5库存...   │  │                         │
│  │  └───┬───┘ └───┬───┘ └─────┬─────┘  │                         │
│  └──────┼─────────┼───────────┼────────┘                         │
│         │         │           │                                   │
│         ▼         ▼           ▼                                   │
│  ┌─────────────────────────────────────┐                         │
│  │     个人偏好过滤器（进化层）          │  ← 新增层               │
│  │  ┌──────────────────────────────┐   │                         │
│  │  │  Rule Override Engine        │   │  用户偏好规则           │
│  │  │  Strategy Router             │   │  策略选择偏好           │
│  │  │  Style Adapter               │   │  沟通风格适配           │
│  │  │  Veto Gate                   │   │  否决过滤器             │
│  │  │  Evolution Tracker           │   │  进化阶段追踪           │
│  │  └──────────────────────────────┘   │                         │
│  └─────────────────────────────────────┘                         │
│         │                                                         │
│         ▼                                                         │
│  ┌─────────────────────────────────────┐                         │
│  │        团队共享规则池（可选）         │  ← 团队级复用           │
│  └─────────────────────────────────────┘                         │
│                                                                   │
│  ┌─────────────────────────────────────┐                         │
│  │      Hermes 自进化内核               │  ← 新增                 │
│  │  ┌──────┐ ┌──────┐ ┌───────────┐   │                         │
│  │  │Memory│ │Skills│ │Self-Evolve│   │                         │
│  │  │(三层)│ │(闭环)│ │(GEPA)     │   │                         │
│  │  └──────┘ └──────┘ └───────────┘   │                         │
│  │  ┌──────┐ ┌──────┐                 │                         │
│  │  │Honcho│ │Nudge │                 │                         │
│  │  │(用户)│ │(对齐)│                 │                         │
│  │  └──────┘ └──────┘                 │                         │
│  └─────────────────────────────────────┘                         │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### 集成原则

| 原则 | 说明 |
|------|------|
| **最小侵入** | 不改动现有 Agent 代码，新层以拦截器/过滤器模式嵌入 |
| **优雅降级** | 个人规则层故障时，自动回退到通用 Agent 输出 |
| **用户可控** | 用户可随时查看、修改、删除任何个人规则 |
| **阶段分离** | 每个决策点独立进化，不全局化 |

---

## 9. 三层分离设计

```
Layer 3: 团队共享规则池
├── 共享范围: 同一团队/品类组
├── 写入权限: 管理员审批
├── 示例: "家居品类ACoS目标统一12%"
└── 优先级: 最低（个人规则可覆盖）

Layer 2: 个人规则库 ← ★ 核心进化层
├── 归属: 单个用户
├── 写入: Agent建议 → 用户确认
├── 示例: "新品推广ACoS容忍度45%而非30%"
└── 优先级: 中（可覆盖团队规则）

Layer 1: 通用Agent核心
├── 归属: 系统级，全用户共享
├── 写入: 系统管理员 + PR Review
├── 示例: 广告调价决策树 v3.0
└── 优先级: 最高基础值，但可被Layer 2覆盖
```

### 规则优先级解析

```
最终决策 = Layer1 通用输出
         → 经 Layer2 个人规则覆盖
         → 经 Layer3 团队规则覆盖 (仅当Layer2未覆盖)

覆盖逻辑:
  FOR each rule in [Layer2, Layer3] ordered by priority DESC:
    IF rule.condition_matches(decision_context):
      decision = rule.apply(decision)
      IF rule.type == "veto":
        BREAK  // 否决规则不可被覆盖
  RETURN decision
```

---

## 10. 四类可学习信号

类比 Hermes Agent 的技能提取机制，跨境电商 Agent 从用户行为中学习四类信号：

| 信号类型 | 学习内容 | 存储位置 | 示例 |
|---------|---------|:------:|------|
| **阈值偏好** | 数值参数的个性化上下界 | Episodic Memory → 规则库 | "不把出价调到 $3 以上" |
| **策略选择** | 在多个可行方案中的偏好排序 | Episodic Memory → 规则库 | "优先推联盟产品而非闪购" |
| **沟通风格** | 回复的正式度、长度、数据密度 | Honcho 用户模型 | "正式、数据驱动、加图表" |
| **否决模式** | 用户明确拒绝的操作类型 | Veto Gate | "永远不自动发布这类社媒内容" |

### 信号提取流程

```
1. 监测: Agent 做决策 → 用户接受/修改/拒绝
2. 差异分析:
   - 用户修改参数 → "阈值偏好信号"
   - 用户选B不选A → "策略选择信号"
   - 用户调整措辞 → "沟通风格信号"
3. 累积: 同类信号 ≥3 次
4. 建议: Agent 通过 Nudge 询问确认
5. 确认: 用户确认后写入个人规则库
```

---

## 11. 四阶段渐进进化策略

每个决策点独立进化，不全局化升级：

```
Stage 0: Observation (观察期)
  周期: 1-2周 或 ≥50个决策样本
  Agent 不参与决策，仅记录人工决策输入/输出
  积累 baseline 数据

    ↓ 样本量足够 + 模拟准确率 ≥85%

Stage 1: Suggestion (建议期)
  周期: 1-3周
  Agent 生成建议但不执行，推送消息流
  记录采纳率

    ↓ 采纳率 ≥80% 持续2周

Stage 2: Semi-Autonomous (半自治)
  Agent 自动执行低风险操作
  高风险操作推送确认卡片
  30分钟未响应 → 按建议执行 + 通知

    ↓ 确认拒绝率 <5% 持续4周

Stage 3: Full-Autonomous (全自治)
  Agent 完全自主，仅异常/边界场景通知
  Nudge 定期对齐
```

### 每个Agent的决策点初始进化阶段

| Agent | 决策点 | 默认初始阶段 | 最高可达 |
|-------|--------|:----------:|:------:|
| **A3 广告调价** | ACoS调整 | Stage 1 | Stage 3 |
| | 关键词发现 | Stage 0 | Stage 2 |
| | 广告活动暂停 | Stage 1 | Stage 3 |
| | 预算分配 | Stage 0 | Stage 2 |
| **A4 客服** | FAQ回复 | Stage 2 | Stage 3 |
| | 复杂投诉回复 | Stage 1 | Stage 2 |
| | 退款决策 | Stage 0 | Stage 2 |
| **A5 库存预警** | 补货量计算 | Stage 1 | Stage 3 |
| | 物流方式选择 | Stage 1 | Stage 2 |
| | 紧急补货触发 | Stage 2 | Stage 3 |
| **A6 利润监控** | 异常检测 | Stage 2 | Stage 3 |
| | 归因分析 | Stage 0 | Stage 2 |
| **G1 驾驶舱** | 日报生成 | Stage 2 | Stage 3 |
| | KPI预警阈值 | Stage 1 | Stage 3 |
| **G3 折扣风控** | 叠加检测 | Stage 3 | Stage 3 |
| | 放行审批 | Stage 2 | Stage 2 |

---

## 12. 暖启动机制

### 角色模板克隆

新用户注册后选择角色模板（保守型/激进型/多平台均衡型/独立站DTC/1688选品型），系统克隆模板到个人规则库。

### 5分钟校准问卷（8-10题）

| 问题 | 校准内容 | 影响Agent |
|------|---------|----------|
| 更关注利润还是GMV增长？ | ACoS阈值浮动 | A3 |
| 多少金额以下退款自动放行？ | 退款阈值 | A4 |
| 每日报表接收时间？ | Cron调度 | G1 |
| 最不能容忍的风险？(断货/封号/亏损) | 预警优先级 | A5, A6 |
| 沟通语言风格？(简洁/详细/数据多) | 回复风格 | G1 |
| 主要市场？(北美/欧洲/日本/东南亚) | 市场默认值 | 全局 |
| 月广告预算？ | 预算告警线 | A3 |
| 是否愿Agent自动调广告出价？ | 直接进Stage 1 | A3 |

---

## 13. 复杂度控制红线

| 红线 | 说明 | Hermes 类比 |
|------|------|------------|
| ❌ 无跨用户协同过滤 | 不分析"类似用户"偏好推荐 | Honcho 单用户档案 |
| ❌ 无实时在线学习 | 批量处理规则更新 | GEPA 周期性触发 |
| ❌ 无NL规则解释 | 规则存储为结构化 JSON | 技能为 YAML 定义 |
| ❌ 无自动A/B实验 | 策略变更需人工确认 | PR Review 机制 |
| ❌ 无情感/性格建模 | 延后到 v2.0 | Soul 层可但不优先 |

---

# 第三部分：实现细节

## 14. 决策日志数据库Schema

参照 Hermes Agent 的 sqlite-vec + FTS5 混合索引设计，在 PostgreSQL 中实现。

### 14.1 决策日志主表

```sql
-- 决策日志主表 (类比 Hermes Episodic Memory)
CREATE TABLE agent_decisions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    agent_id        VARCHAR(20) NOT NULL,   -- 'A3','A4','A5','A6','A7','G1','G2','G3'
    decision_point  VARCHAR(50) NOT NULL,    -- 'acos_adjustment','faq_reply','stock_alert'等

    -- 决策内容
    context_json    JSONB NOT NULL,           -- 决策上下文
    agent_output    JSONB NOT NULL,           -- Agent 原始输出
    final_decision  JSONB NOT NULL,           -- 最终执行的决策 (经规则覆盖后)

    -- 用户交互
    user_action     VARCHAR(20) NOT NULL,     -- 'accepted','modified','rejected','ignored'
    user_overrides  JSONB,                    -- 用户修改内容 (若 modified)
    user_feedback   TEXT,                     -- 用户显式反馈

    -- 规则追踪
    rules_applied   JSONB,                    -- 本次应用的规则ID列表
    rule_overrides  INT DEFAULT 0,

    -- 进化追踪
    evolution_stage VARCHAR(20) NOT NULL,     -- 'observation','suggestion','semi_autonomous','full_autonomous'
    confidence      DECIMAL(4,3),             -- 置信度 (0.000-1.000)

    -- 性能指标
    response_time_ms INT,
    token_count     INT,

    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_id      UUID NOT NULL,
    episode_id      UUID,                     -- 每15个任务一批

    CONSTRAINT valid_user_action CHECK (
        user_action IN ('accepted','modified','rejected','ignored')
    )
);

-- 复合索引：按用户+Agent+决策点查询历史
CREATE INDEX idx_decisions_user_agent_point
    ON agent_decisions(user_id, agent_id, decision_point, created_at DESC);

-- 部分索引：仅记录被修改/拒绝的决策（用于信号提取）
CREATE INDEX idx_decisions_modified_rejected
    ON agent_decisions(user_id, agent_id, decision_point, created_at DESC)
    WHERE user_action IN ('modified', 'rejected');

-- JSONB 索引：按进化阶段过滤
CREATE INDEX idx_decisions_stage
    ON agent_decisions(evolution_stage, created_at DESC);
```

### 14.2 个人规则库表

```sql
CREATE TABLE personal_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    agent_id        VARCHAR(20) NOT NULL,
    decision_point  VARCHAR(50) NOT NULL,

    -- 规则定义
    rule_type       VARCHAR(20) NOT NULL,     -- 'threshold','strategy','style','veto'
    rule_name       VARCHAR(100) NOT NULL,
    rule_condition  JSONB NOT NULL,
    rule_action     JSONB NOT NULL,
    priority        INT DEFAULT 100,

    -- 规则来源
    source          VARCHAR(20) NOT NULL,     -- 'manual','nudge','auto_extracted','template'
    source_decisions UUID[],

    -- 规则状态
    status          VARCHAR(20) DEFAULT 'active',  -- 'active','shadow','paused','retired'
    confidence      DECIMAL(4,3) DEFAULT 0.000,

    -- 使用统计
    times_applied   INT DEFAULT 0,
    times_overridden INT DEFAULT 0,
    last_applied_at TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_rule_type CHECK (
        rule_type IN ('threshold','strategy','style','veto')
    ),
    CONSTRAINT valid_source CHECK (
        source IN ('manual','nudge','auto_extracted','template')
    ),
    CONSTRAINT valid_status CHECK (
        status IN ('active','shadow','paused','retired')
    )
);

CREATE INDEX idx_rules_user_agent_point
    ON personal_rules(user_id, agent_id, decision_point)
    WHERE status = 'active';
```

### 14.3 Episode 汇总表

```sql
CREATE TABLE agent_episodes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    agent_id        VARCHAR(20) NOT NULL,

    episode_number  INT NOT NULL,
    decision_count  INT NOT NULL,

    -- LLM 生成摘要
    episode_summary TEXT,
    key_insights    JSONB,
    improvement_suggestions JSONB,

    -- 统计
    acceptance_rate DECIMAL(4,3),
    avg_confidence  DECIMAL(4,3),
    avg_response_ms INT,
    total_tokens    INT,

    -- Nudge
    nudge_triggered BOOLEAN DEFAULT FALSE,
    nudge_topics    JSONB,
    nudge_response  TEXT,

    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 14.4 Honcho 用户模型表

```sql
CREATE TABLE honcho_user_profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE,

    risk_tolerance  VARCHAR(30) DEFAULT 'moderate',
    communication_style VARCHAR(20) DEFAULT 'balanced',
    notification_prefs JSONB,

    agent_profiles  JSONB NOT NULL DEFAULT '{}',

    hypothesis_count INT DEFAULT 0,
    confirmed_count  INT DEFAULT 0,
    last_dialectic_at TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 14.5 全文搜索索引 (FTS)

```sql
ALTER TABLE agent_decisions ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_decision_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        to_tsvector('simple',
            COALESCE(NEW.context_json::text, '') || ' ' ||
            COALESCE(NEW.agent_output::text, '') || ' ' ||
            COALESCE(NEW.user_feedback, '')
        );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_decision_search_vector
    BEFORE INSERT OR UPDATE ON agent_decisions
    FOR EACH ROW EXECUTE FUNCTION update_decision_search_vector();

CREATE INDEX idx_decisions_fts ON agent_decisions USING GIN(search_vector);
```

---

## 15. 规则冲突仲裁逻辑

### 15.1 仲裁优先级体系

```
优先级 (从高到低):

1. Veto 规则 (否决规则)           ← 最高，不可覆盖
2. 用户手动创建的规则              ← 显式意图优先
3. Nudge 确认的规则                ← 被对话确认过的
4. 自动提取的规则 (置信度≥90%)     ← 高置信自动
5. 模板克隆的规则                  ← 初始值
6. 自动提取的规则 (置信度<90%)     ← 低置信自动
7. 团队共享规则                    ← 最低
```

### 15.2 同优先级冲突解决

```python
def resolve_conflicts(matched_rules, decision_context):
    """
    当多个同优先级规则冲突时的解决策略
    """
    # 策略 1: 按规则类型排序
    type_order = {'veto': 0, 'threshold': 1, 'strategy': 2, 'style': 3}
    matched_rules.sort(key=lambda r: type_order[r.rule_type])

    # 策略 2: 同类型规则 - 取"最新创建"的（用户最近意图优先）
    if len(matched_rules) > 1:
        matched_rules.sort(key=lambda r: r.created_at, reverse=True)

    # 策略 3: "最新规则胜出" + 冲突记录
    winner = matched_rules[0]
    if len(matched_rules) > 1:
        log_conflict(decision_context, matched_rules, winner)
        # 通过 Nudge 询问用户
        schedule_nudge_conflict_resolution(matched_rules)

    return winner
```

### 15.3 规则冲突日志表

```sql
CREATE TABLE rule_conflicts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id     UUID NOT NULL REFERENCES agent_decisions(id),

    conflicting_rules UUID[] NOT NULL,
    winner_rule_id  UUID NOT NULL,
    resolution      VARCHAR(20) NOT NULL,     -- 'auto_priority','user_choice','latest_wins'

    nudge_sent      BOOLEAN DEFAULT FALSE,
    nudge_resolved  BOOLEAN DEFAULT FALSE,
    user_choice     UUID,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 16. 完整进化示例：广告调价 Agent (A3)

以 A3 广告调价 Agent 的 **ACoS调整** 决策点为示例，展示从 Day 0 部署到 Stage 3 全自治的完整进化路径。

### 16.1 初始状态 (Day 0)

从"激进型亚马逊运营"模板克隆，初始规则：新品ACoS超标（>45%）时降低出价15%。初始进化阶段为 Suggestion。

### 16.2 Stage 1 建议期 — 信号采集 (Day 1-21)

**Day 3**：Agent建议ACoS=48%时降15%，用户接受。

**Day 5**：Agent建议ACoS=42%时降15%，用户**修改为降10%**，理由"竞品也在降，稳一点"。

**Day 8, 12**：类似场景，用户连续两次修改为降10%。

**Day 15 — 信号累积触发 Nudge**：
- Agent 检测到同类信号 3 次："用户偏好将调价幅度从15%降为10%"
- Nudge 对话后用户确认："可以改成10%，新品期不太想降太猛影响曝光"
- 规则更新为置信度0.95的新规则，来源标记为"nudge"

### 16.3 Stage 2 半自治期 (Day 22-56)

升级条件满足：采纳率87%≥80%。

**Day 22**：Agent自动按10%调整，高风险操作（幅度>20%）仍需确认。

**Day 28**：ACoS=52%（新品）+ 竞品降价10%，Agent建议降10%，用户**改为降5%**。

**Day 35 — 第二次 Nudge**：
- Agent："竞品降价时你的调价策略会更保守。新增规则：竞品同时降价>5%时，调价幅度自动减半？"
- 用户确认后创建高优先级(110)策略规则

**规则叠加效果**：正常新品超标→降10%；竞品同时降价→降5%（覆盖）

### 16.4 Stage 3 全自治期 (Day 57+)

升级条件满足：确认拒绝率2.3%<5%，持续4周。

Agent 每小时自动扫描所有广告活动，根据 ACoS + 库存 + 竞品 + 产品阶段自主决策调价。仅异常情况通知用户（调价>30%、连续3天ACoS不降反升、需暂停活动）。每15个任务 Nudge 对齐一次。

### 16.5 每月自我性能报告（自动生成）

```
## A3 广告调价 Agent — 2026年6月性能报告

### 进化历程
- 当前阶段: Stage 3 (全自治)
- 个人规则数: 7 条（1模板 + 3 Nudge + 3 自动提取）
- 本月Nudge: 2次（1次阈值校准, 1次空白发现）

### 性能指标
| 指标 | 本月 | 上月 | 变化 |
|------|:---:|:---:|:---:|
| 决策采纳率 | 97.1% | 95.3% | +1.8pp |
| 平均ACoS | 16.2% | 17.8% | -1.6pp |
| 用户手动覆盖 | 2次 | 4次 | -50% |
| 广告花费节省 | $3,240 | $2,850 | +13.7% |
| Token消耗 | 142K/天 | 158K/天 | -10% (Cache优化) |

### 学到的新模式
1. 用户对"品牌防守关键词"的出价更激进（+25% vs 默认+10%）
2. 用户偏好在美国东部时间 02:00-05:00 不调价
3. 用户不接受自动创建新的广告活动（已加入否决规则）

### 改进建议
- [GEPA] ACoS阈值可从15%微调至14%（基于近30天数据，预期再降1.2pp）
- [技能] 检测到"竞价调整+否定关键词添加"的重复模式（8次/月），建议打包技能
```

---

## 17. 熵管理系统

> **定位前情**：在前述 Hermes 自进化体系（第5-6章）中，GEPA 负责进化 Agent 行为，Nudge 负责行为对齐。但在一个持续自治的系统中，进化本身也会引入退化——过时规则堆积、Agent 间策略矛盾、反馈信号衰减。本章定义熵管理系统作为进化机制的**免疫层**：它不决定 Agent 如何进化，但监控进化是否正在失控。

### 17.1 核心隐喻：熵增是不可逆的物理定律

类比热力学第二定律：封闭系统必然走向无序。自进化 Agent 系统同样面临这条铁律——**每一条新增规则、每一次自动优化、每一个 Nudge 确认，都在向系统注入局部优化，但全局秩序在持续衰减**。

将系统状态定义为：

```
SystemOrder(t) = f(coherence, consistency, predictability)

SystemEntropy(t) ∝ -SystemOrder(t)
```

低熵态特征：规则数量适中，各 Agent 策略一致，决策可预测且可解释，用户介入率低且稳定。

高熵态特征：规则数量膨胀但大量闲置，不同 Agent 对同一场景输出矛盾建议，不可预测的边界行为激增，用户频繁手动覆盖。

关键洞察：**不做熵管理的自进化系统，注定从"聪明的助手"退化为"混乱的来源"**。Hermes 未显式定义熵管理层，这是我们在跨境电商场景下的必要补充。

### 17.2 五种熵增形态

```
         ┌──────────────────────────────────────────────┐
         │              五 种 熵 增 形 态                 │
         ├──────────┬──────────┬──────────┬──────────────┤
         │ 决策漂移  │ 规则膨胀  │上下文稀释│ 技能退化     │
         │(Decision │(Rule     │(Context  │(Skill Rot)  │
         │ Drift)   │ Inflation)│Dilution) │             │
         ├──────────┼──────────┼──────────┼──────────────┤
         │ 反馈衰减  │          │          │              │
         │(Feedback │          │          │              │
         │ Decay)   │          │          │              │
         └──────────┴──────────┴──────────┴──────────────┘
```

**形态1：决策漂移 (Decision Drift)**。随着进化周期推进，Agent 的同一决策点输出逐渐偏离用户最初校准的意图。例如 Day 0 用户设定 ACoS 容忍度 15%，Day 90 GEPA 自动调参后实际阈值滑移至 24%，用户未曾察觉。根因是进化信号仅关注局部指标（如"ACoS降低了"），缺乏全局约束锚点。

**形态2：规则膨胀 (Rule Inflation)**。每个 Nudge 确认都创建新规则，但从不清除过时规则。3 个月后个人规则库从 7 条膨胀至 80+ 条，其中 60% 从未被触发，30% 与其他规则冲突。用户无法理解 Agent 为什么做某个决策——因为中间层嵌入了太多无形规则。

**形态3：上下文稀释 (Context Dilution)**。Working Memory（第2.1节）的容量上限是 8K-32K tokens。过多的 Episodic Memory 召回、个人规则注入、Honcho 偏好片段挤占 prompt 空间，核心任务上下文被挤到边缘，决策质量反而下降。这是 Hermes 自身的 Working Memory 设计未考虑的退化路径。

**形态4：技能退化 (Skill Rot)**。技能包（第4章）创建后长期未使用或底层平台 API 已变更但技能未更新。下次触发时输出错误结果但用户未察觉。

**形态5：反馈衰减 (Feedback Decay)**。用户初期积极给反馈（前 30 天 80% 决策有反馈信号），随着信任建立或疲劳，反馈密度降至 20%，但 GEPA 仍在持续进化——此时进化方向已脱离用户真实意图。

### 17.3 反应式五道防线

五道防线按触发时机排列，越靠前越轻量、越靠后越重：

```
防线1: TTL         ──→  防线2: Merge     ──→  防线3: Regret
(强制过期)              (规则合并)              (遗憾检测)

防线4: Decay       ──→  防线5: Budget
(衰减降权)              (预算硬限制)
```

**防线1：TTL 强制过期**。每条个人规则创建时即绑定 TTL（默认 90 天）。到期后自动进入 `expired` 状态，通知用户"以下规则已过期，是否续期？"。这是最轻量的防御——不分析规则质量，仅依赖时间维度。

**防线2：Merge 规则合并**。每周扫描规则库，检测语义相似规则（基于 embedding 余弦相似度 ≥0.85），合并为一条并保留历史溯源。合并后规则数 ≤ 原数量的 60%。

**防线3：Regret 遗憾检测**。对每条规则生成反事实模拟：如果未应用此规则，决策结果是否更好？例如规则"竞品降价时调价幅度减半"，在过去 30 天产生了 12 次效果，其中 4 次导致 ACoS 反升——标记为"遗憾"。遗憾率 >30% 且样本量 ≥10 的规则自动暂停。

**防线4：Decay 衰减降权**。规则优先级随"未使用天数"自动衰减。30 天未被触发 → 优先级降 20%，60 天 → 降 50%，90 天 → 触发 TTL（防线1）或用户确认是否保留。衰减仅影响自动提取和模板克隆的规则，用户手动创建的不衰减。

**防线5：Budget 预算硬限制**。每个决策点可应用的规则数硬上限：小决策（如FAQ分类）≤3 条，中决策（如关键词出价）≤5 条，大决策（如广告活动暂停）≤8 条。超限时按"优先级 × 最近触发时间"排序，低分规则自动被裁剪。绝对上限：每个用户所有 Agent 规则总数 ≤200 条。

### 17.4 五层预测架构

区别于反应式防线"事后处理"，预测层在退化发生前发出预警：

```
Layer 5: 系统级熵指数     ← 全局视图，CEO仪表盘
Layer 4: 进化速度监控     ← 进化是否过快/过慢
Layer 3: 跨Agent矛盾检测  ← A3和A5对同一场景的建议是否冲突
Layer 2: 模式异常检测     ← SPC统计过程控制
Layer 1: 规则健康评分     ← 每条规则的实时健康度
```

**Layer 1：规则健康评分**

每条规则独立计算健康分（0-100），综合三维度：
- **新鲜度 (40%)**：基于最近触发时间，7天内 100 分，30天 50 分，90天 0 分
- **有效性 (40%)**：1 - 遗憾率。遗憾率 10% → 90 分，30% → 70 分
- **一致性 (20%)**：与用户最新 20 个决策的向量距离。距离越近分越高

主动规则（status=active）健康分 <50 → 自动降级为 `shadow`（阴影模式，不生效但继续记录）。连续 2 周健康分 >70 → 可恢复为 `active`。

**Layer 2：SPC 统计过程控制**

将工业 SPC（Statistical Process Control）应用于 Agent 决策质量监控。

对每个关键决策点建立统计基线（前 30 天数据），计算均值 μ 和标准差 σ。监控三条控制线：

| 控制线 | 阈值 | 含义 | 响应 |
|--------|:----:|------|------|
| UCL/LCL (控制上下限) | μ ± 3σ | 统计显著异常 | 立即暂停该决策点自治，回退至 Stage 1 |
| UWL/LWL (警告上下限) | μ ± 2σ | 趋势偏移预警 | 通知管理员，增加采样频率 |
| 连续规则 | 7 点同侧 | 系统性漂移 | 触发全面规则审查 |

监控指标包括：用户采纳率、平均置信度、手动覆盖率、响应延迟。

**Layer 3：跨Agent矛盾检测**

当两个 Agent 对同一场景输出相互矛盾的决策建议时触发。检测方法：将 A3（广告调价）和 A5（库存预警）对同一产品的建议分别向量化 → 计算余弦相似度。相似度 <0.3 且两个 Agent 都在 Stage 2+ 时标记为矛盾。矛盾累积 ≥3 次同一类型 → 推送 Nudge 让用户仲裁，仲裁结果写入全局一致性约束（优先级高于任何单个 Agent 的个人规则）。

**Layer 4：进化速度监控**

监控每个决策点的进化频率。过快（每周 >3 次规则变更）→ 系统不稳定，可能震荡。过慢（30 天零变更但有遗憾规则）→ 系统僵化，进化停滞。理想区间：每周 0.5-2 次变更。偏离理想区间时减速/加速进化步长。

**Layer 5：系统级熵指数**

聚合指标为单一 0-100 分数，提供给驾驶舱 Agent（G1）用于日报。

```
EntropyIndex = w1 × RuleHealth(avg)
             + w2 × (1 - ContradictionRate)
             + w3 × EvolutionStability
             + w4 × AdoptionTrend
             + w5 × (1 - OverrideRate)
```

| 熵指数 | 状态 | 建议动作 |
|:------:|------|---------|
| 80-100 | 健康 | 正常运行 |
| 60-79 | 警告 | 触发 Layer 1-3 检查 |
| 40-59 | 退化 | 暂停最高遗憾率的规则 |
| 0-39 | 危机 | 冻结所有自动进化，人工审计 |

### 17.5 闭环数据流

```
                    ┌─────────────┐
                    │  决策执行    │
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            ▼              ▼              ▼
     ┌────────────┐ ┌────────────┐ ┌────────────┐
     │ 决策日志写入 │ │ 规则触发记录│ │ 用户反馈捕获│
     └──────┬─────┘ └──────┬─────┘ └──────┬─────┘
            │              │              │
            └──────────────┼──────────────┘
                           ▼
                  ┌─────────────────┐
                  │  熵分析引擎       │
                  │  ├ 五道防线巡检   │
                  │  ├ 五层预测扫描   │
                  │  └ 矛盾检测       │
                  └────────┬────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │ 自动处理  │ │ Nudge建议│ │ 紧急冻结  │
       │(TTL/Merge│ │(规则续期 │ │(暂停自治) │
       │ /Decay)  │ │ /矛盾仲裁)│ │          │
       └──────────┘ └──────────┘ └──────────┘
              │            │            │
              └────────────┼────────────┘
                           ▼
                  ┌─────────────────┐
                  │  熵指数更新      │
                  │  → 写入驾驶舱    │
                  └─────────────────┘
```

### 17.6 核心数据模型

**Mark Change Table：标记变更的完整审计日志**

这是熵管理系统最底层的数据基础设施。与 agent_decisions 表（第14.1节）不同，Mark Change 表记录的不是"Agent 做了什么决策"，而是"系统的规则/策略/参数发生了什么变更"，以及——更重要的是——**这个变更是谁以什么身份建议的**。

```sql
-- 标记变更表 (基于 Mark Change Pattern)
CREATE TABLE rule_mark_changes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 变更目标
    target_type     VARCHAR(30) NOT NULL,  -- 'personal_rule','skill','threshold','honcho_profile'
    target_id       UUID NOT NULL,

    -- 变更内容
    field_path      VARCHAR(200) NOT NULL, -- JSON路径，如 '$.rule_action.adjustment_percentage'
    old_value       JSONB,
    new_value       JSONB NOT NULL,

    -- 变更来源（关键字段）
    source_type     VARCHAR(30) NOT NULL,  -- 'gds' | 'gds_proxy' | 'human' | 'nudge' | 'auto_extract'
    source_id       VARCHAR(100),           -- 具体来源标识

    -- 人类可读摘要
    change_summary  TEXT NOT NULL,           -- LLM 生成的变更说明

    -- 因果关联
    parent_change_id UUID,                   -- 关联的触发变更
    related_decision_ids UUID[],             -- 关联的决策ID

    -- 变更上下文
    context_json    JSONB,                   -- 触发此变更的上下文快照

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 按来源类型+时间查询（日常报告）
CREATE INDEX idx_mark_changes_source_time
    ON rule_mark_changes(source_type, created_at DESC);

-- 按目标查询变更历史
CREATE INDEX idx_mark_changes_target
    ON rule_mark_changes(target_type, target_id, created_at DESC);
```

**source_type 枚举说明**：

| source_type | 含义 | 示例 |
|-------------|------|------|
| `gds` | General Decision System — Agent 自身决策 | A3 Agent 建议调价 -15% |
| `gds_proxy` | GDS 代理 — Agent 代表用户执行但未经用户确认 | Agent 自动应用的规则覆盖 |
| `human` | 人工 — 用户手动操作 | 用户在面板上直接修改 ACoS 阈值 |
| `nudge` | Nudge — 通过对话确认的变更 | "你要我把默认调价幅度改小一点吗？" → 用户确认 |
| `auto_extract` | 自动提取 — 系统从行为模式自动提取 | 连续 5 次选海运 → 系统提取偏好规则 |

**SPC 控制表**：

```sql
CREATE TABLE spc_control_limits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    agent_id        VARCHAR(20) NOT NULL,
    decision_point  VARCHAR(50) NOT NULL,
    metric_name     VARCHAR(50) NOT NULL,     -- 'acceptance_rate','confidence','override_rate'

    -- 统计基线（基于前30天数据计算）
    baseline_mean   DECIMAL(10,4) NOT NULL,
    baseline_stddev DECIMAL(10,4) NOT NULL,
    baseline_samples INT NOT NULL,

    -- 控制线
    ucl             DECIMAL(10,4) NOT NULL,   -- μ + 3σ
    lcl             DECIMAL(10,4) NOT NULL,   -- μ - 3σ
    uwl             DECIMAL(10,4) NOT NULL,   -- μ + 2σ
    lwl             DECIMAL(10,4) NOT NULL,   -- μ - 2σ

    -- 连续规则检测
    consecutive_same_side INT DEFAULT 0,       -- 连续同侧点数
    last_breach_at  TIMESTAMPTZ,

    -- 基线更新
    baseline_recalc_at TIMESTAMPTZ NOT NULL,
    next_recalc_at  TIMESTAMPTZ NOT NULL,      -- 每周重新计算

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, agent_id, decision_point, metric_name)
);
```

### 17.7 实施路线图

| 阶段 | 工作内容 | 优先级 | 依赖 |
|------|---------|:------:|------|
| **Phase 1a (M1)** | Mark Change 表部署 + source_type 分类 | 🔴 P0 | 数据库 |
| **Phase 1b (M2)** | 五道防线（TTL + Budget + Decay） | 🔴 P0 | Phase 1a |
| **Phase 1c (M2)** | Layer 1 规则健康评分 | 🔴 P0 | Phase 1a |
| **Phase 2a (M3)** | 防线3 Regret 遗憾检测 + 防线2 Merge | 🟡 P1 | Phase 1b |
| **Phase 2b (M4)** | Layer 2 SPC 统计过程控制 | 🟡 P1 | Phase 1c |
| **Phase 2c (M5)** | Layer 3 跨Agent矛盾检测 | 🟡 P1 | Phase 1b |
| **Phase 3a (M6)** | Layer 4 进化速度监控 + Layer 5 熵指数 | 🟢 P2 | Phase 2b |
| **Phase 3b (M7)** | G1 驾驶舱集成熵指数仪表盘 | 🟢 P2 | Phase 3a |

> **Phase 1 即可覆盖 80% 的熵增风险**：TTL + Budget + 健康评分三条低成本防线，足以拦截规则膨胀、过期规则堆积和上下文稀释。

---

# 附录

## A. 与核心文档的关系

本文档是 `final-integrated-solution.md`（7+3 Agent 综合方案）的扩展。当与核心方案集成时：第1-7章（Hermes深度解析）作为新第14章插入，第8-13章（集成设计）作为新第15章插入，第14-16章（实现细节）作为技术附录插入，第17章（熵管理系统）作为独立附加章节插入。

## B. 实施优先级建议

| 阶段 | 工作内容 | 依赖 |
|------|---------|------|
| Phase 1a (M1) | 决策日志表 + 个人规则库表 部署 | 数据库 |
| Phase 1b (M2) | 个人偏好过滤器层嵌入 A3/A5 Agent | Phase 1a |
| Phase 1c (M2) | 角色模板库（5个模板）+ 校准问卷 | Phase 1a |
| Phase 2a (M3) | Nudge 引擎（阈值校准 + 策略发现） | Phase 1b |
| Phase 2b (M4) | 四阶段进化追踪器 | Phase 1b |
| Phase 3a (M5) | Honcho 辩证用户建模 | Phase 2a |
| Phase 3b (M6) | GEPA 参数进化（L2层级） | Phase 2b |
| Phase 4 (M9+) | GEPA Prompt 进化（L1层级） | Phase 3b |

> **Phase 1 即可见效**：决策日志 + 偏好过滤器 2 个月内上线，让用户感受到 Agent "越来越懂我"。

## C. 关键参考

- [NousResearch Hermes Agent](https://github.com/NousResearch/hermes-agent) - 官方源代码与文档
- [Hermes Agent Documentation](https://hermes-agent.com/docs) - 技术规范
- [LangGraph](https://langchain.com/langgraph) - Agent 编排框架
- [Anthropic Prompt Caching](https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching) - KV Cache 优化

---

> **文档结束**
>
> 核心思想：Hermes Agent 提供了从记忆到技能到进化的完整闭环架构。跨境电商 7+3 Agent 体系通过「个人偏好过滤器 + 四阶段渐进进化 + Nudge 行为对齐」实现 "Agent 随用户成长" 的目标，同时严格遵守复杂度控制红线（不做跨用户协同、不做实时在线学习、不做自动 A/B 测试）。
