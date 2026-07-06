# AI Traffic System Architecture

> LingMirror AI AgentOS 的城市交通系统架构。
> 本文档把 AI AgentOS 设计成城市交通系统：道路、车辆、红绿灯、检查站、调度中心和监控中心共同决定整座城市是否畅通。
> 最后更新：2026-07-03

## 1. 定位

LingMirror 不是简单的 ERP 加 AI 功能，而是一座由 AI 参与运营的跨境电商城市。订单、商品、库存、利润、物流、合规、客服和外部平台都是城市里的区域；Agent 是在城市中运行的车辆；Platform Kernel 是道路、规则、调度和监控系统。

这个交通系统是 AI 架构的核心命脉：

- 交通系统做好，更多 Agent 可以接入城市而不造成混乱。
- 交通系统不好，Agent 越多，系统越容易拥堵、误判、越权和失控。
- 交通系统必须先保证可控、可审计、可观测，再追求自动化速度。

本架构文档约束的是 Platform Kernel 和 Agent Workflows，不替代具体业务模块设计。

## 2. 核心原则

### 2.1 Agent 是车辆，不是城市规则

Agent 可以观察、分析、建议和发起动作，但不能绕过城市交通规则直接改变关键业务数据。

所有 Agent 动作必须进入统一链路：

```text
Business Signal
  -> Event
  -> Agent Decision
  -> Agent Action
  -> Risk / Policy Check
  -> Approval if needed
  -> Command / Tool Execution
  -> Audit
  -> Observability
```

### 2.2 城市道路必须统一

Agent 之间不允许私下直连形成隐藏依赖。跨 Agent 协作必须通过 EventBus、Pipeline DAG、Command Dispatcher 或 ToolBridge 等内核契约完成。

正确方向：

```text
A5 Stock Alert -> publishes event -> G3 observes -> A6 observes -> A2 acts through command
```

错误方向：

```text
A5 directly calls G3 internal logic
G3 directly mutates A6 data
A6 directly publishes listing changes externally
```

### 2.3 高风险路段必须有红绿灯和检查站

涉及价格、库存、订单状态、钱、外部平台发布、账号权限、凭证、破坏性数据变更的动作，默认需要审批和审计。

AI 可以把车开到路口，但不能闯红灯。

### 2.4 Owner 看交通，不看发动机

Owner 不需要理解每个 Go interface 或数据库表，但必须能看懂：

- 哪些 Agent 正在运行。
- 哪些业务区域堵塞。
- 哪些动作等待审批。
- 哪些风险被拦截。
- 哪些外部平台失败。
- 哪些自动化已经执行并留下记录。

## 3. 城市交通映射

| 城市交通概念 | LingMirror 组件 | 责任 |
|---|---|---|
| 城市道路 | EventBus | 承载业务信号和 Agent 协作信号 |
| 主干路网 | Pipeline DAG | 定义 Agent 决策链路和事件流向 |
| 车辆 | Agent Action / Command | 一次明确的建议或执行请求 |
| 车牌 | Action identity / idempotency key | 标识动作，防止重复执行 |
| 红绿灯 | Approval / RBAC / Action Policy | 决定动作是否允许继续 |
| 检查站 | Risk Control / Guardrails | 拦截高风险或不合规动作 |
| 交通警察 | Audit | 记录谁、为什么、做了什么、结果如何 |
| 调度中心 | Scheduler / Orchestrator | 决定什么时候触发哪个 Agent |
| 地图 | Agent Registry / Tool Registry | 说明有哪些 Agent 和工具、能做什么 |
| 高速路 | Low-risk standard workflow | 低风险、高频、标准化自动流 |
| 危险品通道 | High-risk workflow | 价格、库存、订单、钱和外部发布 |
| 摄像头和传感器 | Observability / Metrics / Trace / Sentry | 发现拥堵、失败、异常和延迟 |
| 城市规划局 | Governance docs / Kernel Contracts | 规定长期平台边界和扩展规则 |

## 4. 标准交通流

### 4.1 事件：城市发生了什么

Event 是城市道路上的信号，表达事实或可观察状态。

示例：

```text
inventory.low
order.created
profit.margin_dropped
listing.risk_detected
scheduler.tick.A5
agent.decided.A5.stock_alert
```

事件规则：

- 事件名必须稳定、命名空间清晰。
- Payload 必须结构化，可追踪来源、租户、实体、时间和 correlation ID。
- 事件本身不应偷偷修改业务数据。
- 订阅者必须可重试，或明确说明不可重试风险。

### 4.2 Agent 决策：车辆准备出发

Agent 接收到事件后，可以生成决策。决策必须说明：

- 触发原因。
- 使用的数据。
- 业务判断。
- 建议动作。
- 风险等级。
- 置信度或证据。

示例：

```text
A5 发现 SKU-1001 当前库存低于安全库存。
建议创建补货建议，风险等级 medium。
不直接创建采购单，不直接推送外部库存。
```

### 4.3 Agent Action：车辆进入道路

Agent Action 是所有 Agent 建议或执行的标准车辆。它不能只是自然语言，必须是结构化记录。

标准字段：

```text
action_type:
version:
agent_id:
actor:
tenant_id:
target_type:
target_id:
risk_level:
approval_required:
approval_id:
mode:
idempotency_key:
input:
rollback_note:
```

执行模式：

| Mode | 含义 |
|---|---|
| dry_run | 只验证，不改变数据 |
| sandbox | 在测试或沙箱环境执行 |
| production | 在真实业务环境执行，必须带护栏 |

### 4.4 风险与审批：红绿灯放行

风险分级决定车辆能不能继续。

| 风险 | 例子 | 默认规则 |
|---|---|---|
| Low | 看板摘要、库存提醒、数据读取、日报 | 可自动执行 |
| Medium | Listing 草稿、合规标记、价格建议、补货建议 | 可生成建议，不直接改关键数据 |
| High | 改价、改库存、取消订单、退款、外部发布、权限变化 | 必须审批和审计 |

红灯规则：

- 没有权限，不能执行。
- 需要审批但没有 approval ID，不能执行。
- 审批过期、被拒绝或目标不一致，不能执行。
- 风险等级不明，按更高风险处理。

### 4.5 Command / Tool：正式执行动作

Command 是城市里的正式通行请求，Tool 是外部工具或平台通道。

Command 适合内部业务动作：

```text
stock_alert
replenish
price_review
listing_optimize
compliance_check
```

Tool 适合外部能力：

```text
fetch_page
publish_listing
sync_inventory
analyze_price_trend
```

执行规则：

- Command 必须校验输入。
- 关键业务修改必须检查权限、审批和审计。
- ToolBridge 不能绕过审批直接执行外部副作用。
- 外部失败必须安全降级，不能造成重复提交或隐藏成功。

### 4.6 Audit：城市黑匣子

所有重要动作必须留下审计记录。

审计至少要能回答：

- 谁触发的？
- 哪个 Agent 判断的？
- 目标是什么？
- 为什么这么做？
- 需要审批吗？
- 谁批准或拒绝？
- 执行了什么？
- 改变前后是什么？
- 成功、失败还是被拦截？

推荐字段：

```text
actor_type:
actor_id:
action:
target_type:
target_id:
risk_level:
approval_id:
request_id:
correlation_id:
before:
after:
result:
created_at:
```

### 4.7 Observability：交通监控中心

AgentOS 总控台必须能看到城市交通状态：

- 正在运行的 Agent。
- 最近触发的事件。
- 当前等待审批的动作。
- 已执行动作和结果。
- 被策略拦截的动作。
- 外部平台调用失败。
- Agent 异常、熔断和延迟。

每条 Agent workflow 必须带 correlation ID，以便从触发事件一路追踪到执行结果。

## 5. 交通分层

### 5.1 城市普通道路：低风险自动化

适用于读数据、生成摘要、风险提醒、看板聚合等动作。

特征：

- 不改变关键业务数据。
- 不调用外部 mutation 工具。
- 不影响价格、库存、订单、钱或权限。

例子：

```text
G1 dashboard_overview
G0 system_health summary
A5 stock_alert notification
A6 profit_watch report
```

### 5.2 城市主干道：中风险建议流

适用于需要业务判断，但暂不直接改关键数据的动作。

特征：

- 可以创建建议、草稿、任务、标记。
- 需要证据和可解释理由。
- 不得直接做不可逆或外部动作。

例子：

```text
A2 creates listing optimization draft
A6 proposes price review
A8 creates sourcing recommendation
A10 proposes logistics route optimization
```

### 5.3 危险品通道：高风险执行流

适用于真实影响业务资产、外部平台或客户体验的动作。

特征：

- 必须审批。
- 必须审计。
- 必须可追踪。
- 需要回滚说明或恢复指引。

例子：

```text
price_update
inventory_change
order_cancel
refund_issue
listing_publish
sync_inventory
credential_change
```

## 6. 典型交通路线

### 6.1 库存风险路线

```text
scheduler.tick.A5
  -> A5 checks inventory
  -> agent.decided.A5.stock_alert
  -> if red: G3 discount_risk_check
  -> if block: A6 profit_watch
  -> if margin risk: A2 listing_optimize
  -> create recommendation / approval request
  -> Owner reviews high-risk action if needed
  -> Command executes after approval
  -> Audit and dashboard update
```

业务含义：

库存告警不会直接变成采购单或外部库存同步。系统先判断风险，再生成建议；涉及采购、库存修改或外部同步时进入审批通道。

### 6.2 利润风险路线

```text
profit.margin_dropped
  -> A6 profit_watch
  -> create price_review recommendation
  -> policy checks risk
  -> price suggestion shown to Owner
  -> Owner approves if price change is desired
  -> price_update command executes
  -> audit records before / after price
```

业务含义：

AI 可以发现亏损并建议调价，但不能未经审批自动改价。

### 6.3 Listing 发布路线

```text
listing.optimization_needed
  -> A2 listing_optimize
  -> create optimized draft
  -> A7 compliance_check
  -> if pass: propose listing_publish
  -> approval required
  -> ToolBridge publishes after approval
  -> audit records external reference
```

业务含义：

AI 可以优化 Listing 内容，但发布到外部平台属于高风险动作，必须经过合规检查、审批和审计。

## 7. 交通拥堵与事故治理

城市交通系统必须处理拥堵和事故，而不是假设所有 Agent 永远正常。

| 问题 | 表现 | 治理方式 |
|---|---|---|
| 事件堵塞 | 同类事件大量堆积 | 限流、合并、延迟重试、优先级 |
| Agent 误判 | 频繁产生低质量建议 | Entropy / TrustScore 降权、暂停、复查 |
| 重复执行 | 同一动作多次提交 | idempotency key、Command 幂等 |
| 审批堆积 | 高风险动作无人处理 | Owner dashboard 汇总、过期策略 |
| 外部平台失败 | 发布或同步失败 | 安全降级、重试、外部引用记录 |
| 隐藏副作用 | 工具调用改变外部状态但无记录 | Tool category、approval_id、audit 强制要求 |
| 跨 Agent 依赖混乱 | 一个 Agent 改动导致多处异常 | EventBus + Pipeline DAG，禁止私下直连 |

## 8. Owner 控制台视图

Owner 应该看到的是城市交通状态，而不是技术日志。

建议的总控台区域：

1. **城市总览**
   - 今日 Agent 决策数。
   - 已自动完成的低风险动作。
   - 等待审批的高风险动作。
   - 被拦截动作数。
   - 当前异常区域。

2. **待放行路口**
   - 动作摘要。
   - 影响对象。
   - 风险等级。
   - AI 建议原因。
   - 预期业务影响。
   - 批准、拒绝、稍后处理。

3. **交通事故记录**
   - 失败的 Command 或 Tool。
   - 失败原因。
   - 是否已重试。
   - 是否影响外部平台或客户。

4. **Agent 车队健康**
   - 每个 Agent 的运行状态。
   - 最近一次运行时间。
   - 建议采纳率。
   - 误报率。
   - TrustScore / Entropy 状态。

5. **审计回放**
   - 从一个业务结果反查完整链路：
   - 哪个事件触发。
   - 哪个 Agent 决策。
   - 哪个动作被审批。
   - 哪个 Command / Tool 执行。
   - 最终改变了什么。

## 9. 架构落地路线

### Phase 1：统一交通语言

- 将 Agent 输出统一映射为 Agent Action。
- 所有动作标记 risk_level、mode、approval_required。
- Dashboard 区分 suggested、pending_approval、executing、completed、failed、blocked。

### Phase 2：统一红绿灯

- 所有高风险动作进入 Approval Contract。
- Command Dispatcher 和 ToolBridge 执行前检查 approval_id。
- 生产 mutation 工具无 approval_id 时必须失败。

### Phase 3：统一黑匣子

- Agent decision、approval、command、tool call、external reference 统一 correlation ID。
- 关键动作记录 before / after。
- Owner 可以从总控台回放一次动作。

### Phase 4：统一交通监控

- Agent workflow 暴露状态、延迟、失败率、拦截率。
- G0 / Entropy / TrustScore 对异常 Agent 降权或暂停。
- Scheduler 和 Pipeline DAG 支持拥堵治理。

### Phase 5：扩展多城市道路

当进程内 EventBus 不再足够时，再引入跨进程消息系统。引入前必须保持同一套事件、动作、审批、审计契约，不能让新消息系统成为第二套交通规则。

## 10. 禁止事项

以下行为会破坏城市交通系统：

- Agent 直接修改价格、库存、订单、钱、权限或外部平台状态。
- Agent 之间私下同步调用，绕过 EventBus 或 Pipeline DAG。
- ToolBridge mutation 工具在 production 模式下绕过审批。
- Command 执行关键业务修改但不记录审计。
- 用自然语言结果替代结构化 Agent Action。
- UI 只展示“AI 已处理”，但不展示建议、审批、执行和失败状态。
- 在 Platform Kernel 中写入具体业务规则。
- 为某个 Agent 单独发明一套事件、审批或审计机制。

## 11. 与现有文档的关系

本文档是 AI AgentOS 的上层架构蓝图，必须与以下文档保持一致：

- `docs/governance/PLATFORM_CONSTITUTION.md`
- `docs/governance/KERNEL_CONTRACTS.md`
- `docs/explanation-agent-pipeline.md`
- `docs/aios-architecture.md`

当本文档与治理文档冲突时，以治理文档为准；当具体实现与本文档冲突时，应优先检查实现是否绕过了交通系统契约。

## 12. 一句话总结

LingMirror 的 AI 架构不是让每个 Agent 自己修路，而是建设一套统一城市交通系统：事件是道路，动作是车辆，审批是红绿灯，审计是黑匣子，观测是监控中心。只有交通系统稳定，整座 AI 电商城市才能持续扩张而不失控。
