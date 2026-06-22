# SPEC: 凌镜 LingMirror AI Agent 模块垂直深挖

> 版本: v2.0.0-ai-deep-dive | 日期: 2026-06-22
> 基于对 Hermes Agent 系统（OBSERVATION → FULL_AUTONOMOUS 四阶演进）的架构审计与扩张设计

---

## 1. 概述与目标

### 1.1 背景

现有 Hermes 智能体系统覆盖了 7 个 Agent（G1-G3 Governors、A3-A7 Analysts/Specialists），具备：
- 四阶段演进（OBSERVATION / SUGGESTION / SEMI_AUTONOMOUS / FULL_AUTONOMOUS）
- 个人规则系统（threshold / strategy / style / veto）
- 动作中枢桥接（AgentAction → ActionProposal → Approval/Execute）
- 定时调度引擎（`AgentScheduler`，asyncio 循环）
- 熵防御系统（TTL / Budget / Decay / Merge / Regret / SPC）

### 1.2 目标

对 AI Agent 模块进行垂直深挖，覆盖以下能力缺口：

1. **Agent 间协作与冲突仲裁** — 多 Agent 对同一业务对象做出矛盾决策时的裁决机制
2. **审批与回滚** — 操作补偿通道、审批超时升级链路
3. **规则健康治理** — 冲突检测、容量硬化、手动编辑冷却期
4. **LLM 调用韧性** — 模型降级链、熔断器、轻量输出验证
5. **多租户隔离** — 按 store_id 维度拆分规则/决策/操作/提案
6. **性能扩展** — 调度并发、历史数据归档

---

## 2. Agent 间协作与冲突仲裁

### 2.1 架构

分层仲裁：**Policy Matrix（规则层）→ Arbiter Agent（智能层）→ 人工（兜底层）**

```
Conflict Detected
    │
    ├─→ Policy Matrix 命中 → 按预设优先级执行 (0 LLM cost)
    │
    └─→ Policy Matrix 未覆盖 → Arbiter Agent 裁决 (1 LLM call)
            │
            ├─→ Arbiter 确定 → 写入 DecisionLog，标记 "arbitrated"
            │
            └─→ Arbiter 不确定 (confidence < 0.7) → 推 ActionProposal 等人工
```

### 2.2 Policy Matrix

新表 `conflict_policies`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | PK | |
| agent_id_a | VARCHAR | 冲突参与方 A |
| agent_id_b | VARCHAR | 冲突参与方 B |
| decision_point | VARCHAR | 适用的决策点 (支持通配符 `*`) |
| condition | JSON | 触发条件（可选，如金额范围） |
| winner | VARCHAR | 冲突时谁的决策优先 |
| reason | TEXT | 预置理由 |
| priority | INT | 匹配优先级 |
| created_at | TIMESTAMP | |

查询逻辑：按 `priority` 排序，取第一条 `agent_id_a/b + decision_point` 匹配的记录。

**约束**：每个 Agent 对最多 20 条 policy，超出则创建时提示清理。

### 2.3 Arbiter Agent

- **Agent ID**: `G0`
- **触发条件**: 任一 `ActionCenterService.create_proposal` 调用时，若相同 `business_object_type + business_object_id` 在最近 30 分钟内已有其他 Agent 的 `pending/suggested` 状态提案，自动触发 G0 仲裁
- **输入**: 各方决策原文（`agent_output`）+ 上下文 + 已触发的 Policy Matrix 结果
- **输出**: `{verdict: "adopt_A5" | "adopt_A6" | "merge" | "escalate", reason: str, confidence: float}`
- **结果**: 写入新表 `arbitration_logs`（所有输入输出完整保留），仲裁后的最终胜出方决策进入正常 ActionProposal 流程

**不要做的**：不创造 Global State——Arbiter 不持有决策权，只做裁决建议。Policy Matrix 没有覆盖的判断才升级到它。

**Prompt 模板结构**（实现时具体措辞可调）：

```
=== CONTEXT ===
业务对象: {business_type}({business_id})
决策窗口: 过去 30 分钟内 {N} 个 Agent 提交了冲突提案

=== CONFLICTING DECISIONS ===
1. [{agent_id_A}] {summary_A}
   理由: {reasoning_A}
   置信度: {confidence_A}

2. [{agent_id_B}] {summary_B}
   理由: {reasoning_B}
   置信度: {confidence_B}

=== APPLICABLE RULES ===
{命中的 Policy Matrix 条目，如 "A5_priority > A6_priority"}
{若无则为 "无预设规则"}

=== TASK ===
输出:
- verdict: "adopt_A5" | "adopt_A6" | "merge" | "escalate"
- reason: 中文理由
- confidence: 0.0-1.0
```

SPEC 不定死措辞，交由实现时调优。

### 2.4 仲裁日志

```sql
CREATE TABLE arbitration_logs (
    id            SERIAL PRIMARY KEY,
    business_type VARCHAR(50)  NOT NULL,     -- 'sku' / 'campaign' / 'order'
    business_id   VARCHAR(100) NOT NULL,
    conflict_keys TEXT[]       NOT NULL,     -- 冲突的 decision_id 列表
    stage         VARCHAR(20)  NOT NULL,     -- 'policy' / 'arbiter' / 'manual'
    policy_id     INT,                       -- 命中的 policy (如果走 policy)
    verdict       VARCHAR(20),               -- 仲裁结论
    arbiter_output JSON,                     -- G0 的完整输出 (如果走了 arbiter)
    resolved_by   VARCHAR(100),              -- 'system' / 'arbiter_G0' / 用户名
    created_at    TIMESTAMP    DEFAULT NOW()
);
```

---

## 3. 审批流程与回滚

### 3.1 补偿操作（替代状态回滚）

**原则**：不做数据库级别的状态回滚，只做业务补偿。

每条 `CommandExecution` 表新增两列：

```sql
ALTER TABLE command_executions
  ADD COLUMN compensation          JSON,       -- 预填补偿操作定义
  ADD COLUMN compensated_by        INT         REFERENCES command_executions(id);  -- 补偿记录 ID
```

补偿操作定义示例：
```json
{
  "type": "cancel_purchase_order",
  "params": {"ref_id": "PO-42", "reason": "user_undo"},
  "risk_level": "medium"
}
```

**流程**：
1. Agent 执行补货 → 系统生成 `CommandExecution` 时，自动预填 `compensation` 字段
2. 用户在 Action Center 点击"撤销" → 系统以补偿定义为 payload 创建新的 ActionProposal
3. 新提案走正常审批/执行流程（不能跳过审批，防误操作）
4. 补偿操作执行后，原 `CommandExecution.compensated_by` 指向新记录 ID

**前端实现**：复用现有 Action Center 组件，不做新页面。
- 在 `CommandExecution` 详情区域（点击已执行记录展开）添加 `[↩ 撤销]` 按钮
- 仅当 `status = executed` 且 `compensation != null` 时显示
- 点击弹出确认对话框 → 调 `POST /action-proposals`（payload 来自 compensation 定义）→ 走现有审批流程
- 组件：Naive UI `NButton` + `NModal`，估算约半天工作量
- 算入 Phase 5 实施

### 3.2 审批超时升级

`ActionProposal` 表新增字段：

```sql
ALTER TABLE action_proposals
  ADD COLUMN approval_deadline   TIMESTAMP,     -- 审批截止时间
  ADD COLUMN escalation_level    INT DEFAULT 0, -- 当前升级层级 (0=初始)
  ADD COLUMN auto_decision       VARCHAR(20) DEFAULT 'reject';  -- 超时后动作
```

分层规则：

| 风险级别 | 首层超时 | 升级路径 | 超时决策 |
|----------|----------|----------|----------|
| critical / high | 30 分钟 | → 部门主管 | 自动拒绝 |
| medium | 4 小时 | → 直属上级 | 自动拒绝 |
| low | 24 小时 | → 上级 | 自动执行（仅限金额 < 100 元时） |

- 每次升级写一条 `AgentOSOperationLog`（`action = "escalate"`）
- 自动拒绝：提案状态 → `rejected` + `rejection_reason = "审批超时自动拒绝"`
- `auto_decision = 'auto_execute'` 只允许 `risk_level = low && amount < 100` 的场景

---

## 4. 规则健康治理

### 4.1 冲突检测

在 `EntropyService.run_defenses()` 中新增 `detect_conflicts()` 步骤：

```python
async def detect_conflicts(self, db, user_id, store_id=None):
    """
    检测同一 (user_id, agent_id, decision_point) 下条件相同但行为冲突的规则
    条件相同: field + op 一致
    行为冲突: action.override 对同一字段赋值相反
    """
    # 按 (field, op) 分组 → 每组 > 1 条 → 检查 action 是否冲突
    # 冲突 → 优先级低的标记为 shadow
    # 记录 RuleMarkChange: "与规则R{n}自动冲突检测"
```

冲突率计入熵指数公式：

```
entropy_index = unhealthy_ratio × 0.4
              + shadow_ratio × 0.3
              + (1 - avg_score) × 0.3
              + conflict_ratio × 0.3    ← 新增
```

### 4.2 容量硬化

| 层级 | 类型 | 上限 | 触发行为 |
|------|------|------|---------|
| 单决策点 | 硬顶 | 20 条规则 | 拒绝创建，提示"请合并或清理" |
| 单 Agent | 软顶 | 50 条规则 | 立即触发一轮熵防御扫描（跳过 6h 定时） |
| 全局 | 警告线 | 200 条规则 | 仪表盘警告，建议用户审查 |

在 `PersonalRule.create` 和 `update` 入口检查硬顶。

### 4.3 手动编辑冷却期

```sql
ALTER TABLE personal_rules
  ADD COLUMN last_manual_edit_at TIMESTAMP;  -- 仅用户手动操作时更新
```

熵防御规则：

- `last_manual_edit_at` > `NOW() - 72h` → 熵系统跳过该规则
- 规则创建后至少被 Agent 调用 **3 次**，熵才允许对其进行自动操作
- 已有 `min_applied >= 5` 机制，扩展为 `min_applied >= 3 AND (last_manual_edit_at IS NULL OR last_manual_edit_at < NOW() - 72h)`

---

## 5. LLM 调用链路韧性

### 5.1 模型降级链 + 熔断器

**降级链**：

```
gpt-4o → gpt-4o-mini → cached → skip
```

每层降级发生在：超时 / HTTP 429 / HTTP 500 / 空响应

**实现**：`BaseAgent.decide()` 上加装饰器 `@llm_resilient`，代码逻辑：

```python
class LLMCircuitBreaker:
    # 单例
    state: dict[str, CircuitState]  # key = f"{agent_id}:{decision_point}"
    
    # 5分钟内连续3次失败 → 熔断
    # 5分钟后尝试半开 → 1次试探请求 → 成功闭合 / 失败回全开
```

在 `AgentDecision` 表新增：

```sql
ALTER TABLE agent_decisions
  ADD COLUMN llm_model_used VARCHAR(50),    -- 实际使用的模型
  ADD COLUMN llm_errors     JSON,           -- [] 或 [{timestamp, error_type, model}]
  ADD COLUMN cached         BOOLEAN DEFAULT false;  -- 是否命中缓存
```

### 5.2 决策缓存

新表 `agent_decision_cache`：

```sql
CREATE TABLE agent_decision_cache (
    cache_key     VARCHAR(64) PRIMARY KEY,
    decision_json JSON        NOT NULL,
    expires_at    TIMESTAMP   NOT NULL
);
```

- `cache_key = SHA256(f"{agent_id}:{decision_point}:{json.dumps(context, sort_keys=True)}")`
- TTL: 300 秒（对齐 LLM prompt cache TTL）
- 命中时：不调用 LLM，直接返回缓存决策，`cached = true`
- 不存历史——`expires_at` 后自动清理

### 5.3 输出验证（轻量）

在 `AgentService.apply_rules()` 后追加 `_sanitize_output(agent_id, decision)`：

| 检查 | 规则 | 失败处理 |
|------|------|---------|
| 存在性 | `action_payload.sku_code` 在 DB 中存在 | `confidence × 0.5` + `validation_warnings` |
| 存在性 | `action_payload.campaign_id` 在 DB 中存在 | `confidence × 0.5` + `validation_warnings` |
| 非负 | `suggested_price` / `suggested_qty` ≥ 0 | 裁掉该 action |
| 合理性 | `suggested_qty` ≤ 仓库容量 | 裁掉该 action |
| 合理性 | `final_price` > `cost_price`（折扣场景除外） | 附 warning，不阻断 |

不引入独立验证 Agent，不增加 LLM 调用。

---

## 6. 多租户隔离

### 6.1 数据模型

四张表各加 `store_id` 列：

```sql
ALTER TABLE personal_rules    ADD COLUMN store_id INT REFERENCES stores(id);
ALTER TABLE agent_decisions   ADD COLUMN store_id INT REFERENCES stores(id);
ALTER TABLE agent_actions     ADD COLUMN store_id INT REFERENCES stores(id);
ALTER TABLE action_proposals  ADD COLUMN store_id INT REFERENCES stores(id);
```

- `store_id` 允许 `NULL`（向后兼容：NULL = 全店铺通用规则）
- 新建记录时 `store_id NOT NULL`

### 6.2 查询隔离

所有涉及以上四张表的查询改为：

```python
# 基础过滤
WHERE (store_id IS NULL OR store_id = :sid)

# 熵防御扫描按 (user_id, store_id, agent_id) 组合
```

### 6.3 调度层

`SchedulerContextBuilder` 接受可选 `store_id` 参数：

```python
async def build_stock_alert_contexts(self, db, store_id=None):
    stmt = select(...)
    if store_id:
        stmt = stmt.where(StoreSKU.store_id == store_id)
    ...
```

`AgentScheduler._run_cycle()` 改为遍历用户的所有 `store_id`：

```python
async for store_id in self._get_active_stores(db, user_id):
    await self._run_cycle(agent_id, db, decision_points, store_id=store_id)
```

### 6.4 不离

- 不改造 `BaseAgent`——Agent 不感知 store
- `EntropyService` 核心逻辑不受影响，查询加过滤即可
- ActionProposal 桥接写入时带上当前 store_id

---

## 7. 性能与扩展性

### 7.1 调度并发 — 生产者/消费者 + Worker Pool

从第一天起使用 `asyncio.Queue` 生产者/消费者模式，**不做先 gather 后优化的过渡方案**。

#### 架构

```
Scheduler Cycle Trigger
    │
    ▼
Producer: 枚举所有活跃店铺 × Agent × 决策点 → 生成工作任务 → 投递到 Queue
    │
    ▼
Queue (asyncio.Queue, maxsize=CAP)
    │
    ▼
Consumer Pool (N 个 worker, 默认 N=20)
    ├─ Worker 1: 拉取任务 → LLM 调用 → 结果写入 DB
    ├─ Worker 2: 拉取任务 → LLM 调用 → 结果写入 DB
    ├─ ...
    └─ Worker N: ...
    │
    ▼
Result Collector: 等待所有任务完成 → 汇总 summary → 记录调度周期
```

#### 关键参数（可配置，热生效）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `MAX_WORKERS` | 20 | 最大并发 LLM 调用数，控制 token 消耗和 DB 连接池压力 |
| `QUEUE_CAPACITY` | 500 | 队列缓冲上限，防止内存堆积；满了则生产者阻塞 |
| `BATCH_SIZE` | 50 | 生产者一次预取多少店铺（分批加载，不一次全量） |
| `WORKER_TIMEOUT` | 30s | 单个任务超时（一次 decide 调用），超时标记失败不阻塞 worker |

#### 代码骨架

```python
class AgentScheduler:
    _queue: asyncio.Queue
    _workers: list[asyncio.Task]
    
    async def _produce(self, db):
        """生产者：枚举店铺 + Agent + 决策点"""
        cursor = None
        while True:
            stores = await self._fetch_store_batch(db, cursor, BATCH_SIZE)
            if not stores:
                break
            for store in stores:
                for agent_id, cfg in self._schedules.items():
                    if not cfg.get("enabled"):
                        continue
                    for dp in cfg.get("decision_points", []):
                        contexts = await self._build_contexts(agent_id, dp, db, store_id=store.id)
                        for ctx in contexts:
                            await self._queue.put({
                                "agent_id": agent_id, "dp": dp,
                                "context": ctx, "store_id": store.id,
                                "user_id": store.user_id,
                            })
            cursor = stores[-1].id
    
    async def _consume(self, worker_id):
        """消费者 worker：从队列拉任务，执行决策"""
        while self._running:
            task = await asyncio.wait_for(
                self._queue.get(), timeout=IDLE_TIMEOUT
            )
            # 执行决策（包含 LLM 降级链、缓存检查、输出验证）
            result = await self._execute_single_decision(**task)
            self._results.append(result)
            self._queue.task_done()
    
    async def _run_cycle(self):
        """一轮调度周期"""
        self._results = []
        producer = asyncio.create_task(self._produce(db))
        workers = [asyncio.create_task(self._consume(i)) for i in range(MAX_WORKERS)]
        
        await producer          # 所有任务投递完成
        await self._queue.join()  # 所有任务执行完毕
        for w in workers:
            w.cancel()
        await asyncio.gather(*workers, return_exceptions=True)
```

#### 容错策略

| 故障场景 | 行为 |
|----------|------|
| 单个任务超时 | 跳过该任务，worker 继续取下一个 |
| 单个店铺连续 3 次失败 | 该店铺本轮跳过，记录到 `schedule_failures` 表供事后重试 |
| Worker 崩溃 | asyncio.Task 异常被捕获，自动重启一个新 worker |
| 队列满 | 生产者阻塞在 `queue.put()`，自然反压 |
| 一个周期未完成又到下一个触发时间 | 跳过下一个触发（确保不会叠任务），现有 `_run_agent_loop` 的 await sleep 间隔天然防叠 |
| 数据库断连 | 所有 worker 的 DB session 报错 → 本轮周期标记失败 → 下周期重试 |

#### 失败重试表

```sql
CREATE TABLE schedule_failures (
    id            SERIAL PRIMARY KEY,
    agent_id      VARCHAR(20)  NOT NULL,
    store_id      INT          NOT NULL,
    decision_point VARCHAR(50) NOT NULL,
    error_msg     TEXT,
    failed_at     TIMESTAMP    DEFAULT NOW(),
    retry_count   INT          DEFAULT 0,
    last_retry_at TIMESTAMP,
    resolved      BOOLEAN      DEFAULT false
);
```

调度周期结束后检查此表，失败的 `retry_count < 3` 的任务自动入队下一轮重试。

**失败通知策略**（分三级，不做过度告警）：

| 失败程度 | 方式 | 说明 |
|----------|------|------|
| 首次失败 | 仅记表，不通知 | 重试可能成功，不打扰用户 |
| 重试 3 次仍失败 | 站内通知推送 | WebSocket 推送到前端右下角通知中心："Agent A5 对 Store#42 库存决策失败，已跳过本轮" |
| 同一 Agent 连续多店铺大面积失败 | 控制台标记 | Agent 调度状态卡颜色标记：黄（部分失败）/ 红（大面积失败） |

**不加**邮件、短信、企微通知——调度异常是系统内部事件，不是用户业务告警。大面积失败在控制台可见即足够。

#### 适用规模

此架构原生支持 **1000+ 店铺** 并发调度，上限在 DB 连接池和 LLM API 限流。瓶颈不在调度层。

### 7.2 不合并 LLM 调用

各 Agent 保持独立 prompt 结构，不尝试跨 Agent 共享 context。原因：
- 各 Agent prompt 结构不同，强行合并 token 消耗相近但排查难度翻倍
- 后台调度不阻塞用户请求，延迟容忍度高

### 7.3 历史数据归档

不建新表，用布尔归档：

```sql
ALTER TABLE agent_decisions ADD COLUMN archived BOOLEAN DEFAULT false;
ALTER TABLE agent_decisions ADD COLUMN archived_at TIMESTAMP;

CREATE INDEX idx_decisions_active 
  ON agent_decisions(user_id, agent_id) 
  WHERE archived = false;
```

**策略**：

| 时间段 | 状态 | 说明 |
|--------|------|------|
| 0-90 天 | `archived = false` | 在线查询默认覆盖 |
| 90 天 - 2 年 | `archived = true` | 可查但需要显式 `WHERE archived = true` |
| > 2 年 | 可删除 | 保留还是清除取决于合规要求 |

每月 1 日定时任务：

```sql
UPDATE agent_decisions 
SET archived = true, archived_at = NOW() 
WHERE created_at < NOW() - INTERVAL '90 days' 
  AND archived = false;
```

---

## 8. 数据模型变更总览

| 表 | 变更 | 关键 |
|----|------|------|
| `conflict_policies` | 新建 | 用户预设仲裁规则 |
| `arbitration_logs` | 新建 | 所有仲裁记录 |
| `agent_decision_cache` | 新建 | LLM 上下文缓存 |
| `command_executions` | 加 2 列 | `compensation`, `compensated_by` |
| `action_proposals` | 加 3 列 | `approval_deadline`, `escalation_level`, `auto_decision` |
| `personal_rules` | 加 2 列 | `store_id`, `last_manual_edit_at` |
| `agent_decisions` | 加 5 列 | `store_id`, `llm_model_used`, `llm_errors`, `cached`, `archived`, `archived_at` |
| `agent_actions` | 加 1 列 | `store_id` |
| `action_proposals` | 加 1 列 | `store_id` |

**总计：4 张新表 + 14 个新列 + 3 个索引**

---

## 9. 实施顺序（建议）

| 阶段 | 内容 | 依赖 | 预估工作量 |
|------|------|------|-----------|
| Phase 1 | 数据层：所有新表 + 新列 + 迁移 + 索引 | 无 | 1-2 天 |
| Phase 2 | 调度并发：生产者/消费者 + Worker Pool + store_id 循环 | Phase 1 | 1.5 天 |
| Phase 3 | 冲突检测 + 容量硬化 + 冷却期（改 EntropyService） | Phase 1 | 1 天 |
| Phase 4 | LLM 韧性：降级链 + 熔断器 + 缓存 + 输出验证 | Phase 1 | 1.5 天 |
| Phase 5 | 补偿操作 + 审批超时升级 | Phase 1 | 1 天 |
| Phase 6 | Policy Matrix + Arbiter Agent G0 + 仲裁日志 | Phase 1, 5 | 1.5 天 |
| Phase 7 | 历史归档定时任务 + 仪表盘告警 | Phase 1 | 0.5 天 |

**总计约 7-9 天工期，按 1 人全职计算。**

---

## 10. 边界与不做的

- **不做**独立的事实验证 Agent（成本 > 收益）
- **不做**跨 Agent 的 prompt 合并复用
- **不做**数据库级别的状态回滚（补偿操作为准）
- **不做**规则自动合并（熵只生成候选，人工确认合并）
- **不做**语义级决策去重（context hash 去重已足够）
- **不做**超过 2 层仲裁链（Policy → Arbiter → Human 已封顶）
