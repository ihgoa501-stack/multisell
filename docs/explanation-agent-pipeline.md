# Agent Pipeline 和事件驱动编排

> 凌镜 LingMirror 的 Agent 间是如何通信和协作的
> 更新日期: 2026-06-30

## 问题

传统电商运营系统里，业务逻辑是人在操作系统上点来点去完成的——人看库存报表，人决定打折，人调价。凌镜要的是 AI 运营，也就是多个 Agent 像团队成员一样协作。但 Agent 之间不能直接调来调去——那会变成意大利面条式的依赖，一个 Agent 崩了牵连一片。

怎么让十几个 Agent 安全、可观测地协作？

## 方案

凌镜选择了 **事件总线 + 声明式管道 DAG** 作为 Agent 间通信的唯一通道。核心原则：Agent 不直接调用其他 Agent，只发布事件。谁关心这个事件，谁订阅它。

```
┌───────┐  事件总线 (EventBus)       ┌───────┐
│ Agent A│───────agent.decided.A───▶│ Agent B│
└───────┘  (发布决策事件)             └───────┘
                                        │
                                        │ 订阅 agent.decided.A.*
                                        │ （A 的决策触发 B 的行为）
                                        ▼
```

### 三个基础设施

1. **EventBus** (`internal/platform/eventbus/`) — 进程内发布/订阅。支持 glob 模式匹配主题，支持 outbox 持久化。约 15 个活跃订阅。
2. **Scheduler** (`internal/platform/scheduler/`) — 周期性任务触发器。每个 Agent 有自己的调度间隔，到期发布 `scheduler.tick.{agent_id}` 事件。
3. **Pipeline DAG** (`internal/agent/pipeline/`) — 声明式决策有向无环图。比手写订阅链更清晰：引擎读取边定义，自动路由 `agent.decided.*` 事件。

### 事件种类

| 事件主题模式 | 发布者 | 说明 |
|-------------|--------|------|
| `scheduler.tick.A5` | Scheduler | 定时触发 Agent 运行 |
| `agent.decided.A5.stock_alert` | Agent A5 | Agent 决策完成，附带决策上下文 |
| `order.created` | Order 模块 | 业务领域事件 |
| `inventory.low` | Inventory 模块 | 库存告警 |
| `entropy.agent.unhealthy.A5` | Entropy | 自净化系统发现异常 |

### 数据飞轮

物流费率的实际履约数据（承运商绩效）通过 `supplychain.flywheel` 事件回流，更新 A10 物流引擎的承运商绩效评分，和 A8 选品引擎的品类绩效评分。这是闭环反馈——运得越好的承运商，后续报价权重越高。

```
A5 库存检查 → [库存低] → G3 折扣风控 → [需干预] → A6 利润看护 → [利润低] → A2 刊登优化
                                                              ↓
                                                    (触发 A10 物流报价)
                                                              ↓
                                                   supplychain.flywheel (履约数据回流)
```

## 权衡

**选的方案：事件总线 + DAG**

- ✅ Agent 间解耦：A5 不知道谁在听它的决策事件。可以随时加新的订阅者而不改 A5 一行代码。
- ✅ 可观测：每条 `agent.decided.*` 事件都被审计日志自动捕获。AgentOS 总控台可以看到完整决策链。
- ✅ 异步：Agent 运行耗时（含 LLM 调用）不阻塞其他 Agent。
- ❌ 弱一致性：事件是最终送达。如果 G3 挂了，A5 的库存警报可能没人处理。当前靠调度器的周期性重试兜底。
- ❌ 调试困难：事件流比同步调用更难跟踪，需要 Trace ID 关联（当前通过 `trace_id` 字段串联）。

**不选 RPC 直调的原因**：A5 直接调 G3 的 API 会构造紧耦合——每次加订阅者都得改 A5。这在 Agent 数量增长后不可持续。

**不选消息队列的原因**：进程内 EventBus 延迟更低（微秒级 vs 毫秒级），部署更简单。当需要跨进程 Agent 通信时再考虑 NATS/RabbitMQ。

## 相关文档

- [装饰式管道 DAG 实现](../../backend-go/internal/agent/pipeline/) — Pipeline engine 源码
- [模块目录 - 平台基础设施](reference-module-catalog.md#1-平台基础设施-internalplatform)
- [Kernel Contracts - Event Contract](governance/KERNEL_CONTRACTS.md#2-event-contract)
- [系统架构 - Agent 层](system-architecture-design-v1.md#5-agent-编排模式)
