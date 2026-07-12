# 平台基础设施技术质量审计（2026-07-12）

## 范围与裁决

范围：`internal/platform/eventbus`、`command`、`scheduler`、`toolbridge`、`routecatalog`、`actioncatalog` 及其生产装配。

结论：这套基础设施不是空壳，持久化、幂等、审批消费、leader lease、重试、DLQ、优雅关闭和 readiness 都有真实实现，测试密度也较高；但 EventBus 的持久化状态机存在可能丢失失败事件或留下错误状态的 P1，Command 成功高风险动作的专用 audit callback 在生产装配中没有接入。当前可评价为 **基础较强，但尚不能称为生产级可靠**。

本次实际运行：6 个平台包共 236 项测试通过。覆盖率：EventBus 84.0%、Command 82.9%、Scheduler 73.8%、ToolBridge 77.0%、RouteCatalog 81.3%、ActionCatalog 96.8%。这证明测试环境中的工程行为，不证明长期生产稳定性。

## 什么叫开发得好 / 不好

开发得好：消息和动作不静默丢失；失败状态可恢复；重复执行不会产生重复副作用；审批绑定目标并只能消费一次；关闭时拒绝新任务并排空在途任务；多实例只有 leader 执行定时任务；安全组件在生产装配中不可缺失；指标、日志、审计能解释每次执行。

开发得不好：错误被记录后仍丢弃；数据库状态与内存状态分叉；安全依赖为 nil 时静默放行；高风险成功没有审计；回调在基础设施锁内执行导致死锁；测试只覆盖内存实现而没有 PostgreSQL 约束和崩溃恢复。

## 七轴评分

| 轴 | 分数（0-5） | 依据 |
|---|---:|---|
| 正确性 | 3.3 | 主路径完整，但 EventBus 失败/重试持久化状态存在缺口 |
| 可读性与复杂度 | 3.0 | 命名清楚；`bus.go` 932 行、状态与并发职责过多 |
| 架构边界 | 3.7 | 接口隔离较好，领域通过接口接入；生产装配仍有可选安全件 |
| 安全 | 3.5 | Command/ToolBridge 高风险审批与幂等较强；Command audit 未接入 |
| 性能与并发 | 3.2 | worker pool、优先级和限流存在；锁内调用订阅者有死锁/阻塞风险 |
| 测试质量 | 4.3 | 236 项、覆盖率较高，含并发、失败和持久化测试 |
| 可运维性 | 3.6 | readiness、leader、retry store、DLQ、指标和 shutdown 已实现；崩溃恢复语义仍需验证 |
| **总评** | **3.5/5** | **较强工程基础，有必须修复的可靠性缺口** |

## 关键发现

### P1：EventBus 先把 outbox 标成 failed，再只允许 pending 变 delivered

worker 在任一 handler 失败后执行 `UPDATE ... status='failed' ... WHERE status='pending'`，随后又把同一事件重新放回内存队列；如果后续重试成功，成功更新仍是 `WHERE status='pending'`，数据库行不会从 failed 变 delivered。服务若在内存重试前崩溃，启动恢复只查询 `status='pending'`，这个事件不会恢复。证据：`eventbus/bus.go:778-833, 898-931`。

结构性修复：明确 outbox 状态机，例如 `pending → processing → pending(retry) → delivered`，只有重试耗尽才 `failed/DLQ`；用数据库尝试次数作为权威状态，并增加“首次 handler 失败后进程重启”的 PostgreSQL 集成测试。

### P1：重试重新入队错误被忽略

普通重试和 DLQ 持久化失败后的回退都使用 `_ = pool.backend.Enqueue(evt)`。队列满或后端失败时事件会被静默丢失，同时日志还可能写“re-enqueuing”。证据：`eventbus/bus.go:817-834`。

修复：检查 enqueue 错误；持久化保持可恢复状态；指标和日志明确记录 retry enqueue failure，不能把内存队列当最后事实源。

### P1：Command 的高风险成功审计回调未在生产装配接入

`DispatchSafe` 只有 `auditRecorder != nil` 才记录高风险成功；`NewDispatcher` 默认 nil。`router.go:174-178` 接入 catalog、rate limiter、idempotency，但没有 `WithAuditRecorder`。HTTP mutation audit 和 approval 记录可能覆盖部分路径，但不能替代 AgentAction envelope 的专用执行审计。证据：`command/command.go:38-41,252-265`、`internal/httpx/router.go:174-178`。

修复：在创建 operationlog service 后完成必需装配，或让生产构造函数缺少审计器时启动失败；增加装配测试，而不是只测带 option 的 isolated dispatcher。

### P1：EventBus 持有订阅读锁执行任意 handler

worker 遍历订阅者时持有 `b.mu.RLock()` 并同步调用 handler。若 handler 尝试订阅/退订需要写锁，会自锁；慢 handler 也会长时间阻塞订阅变更。证据：`eventbus/bus.go:742-778`。

修复：锁内复制匹配订阅快照，释放锁后执行回调。

### P2：安全组件由通用构造函数定义为 optional/fail-open

Command 的 catalog、audit、rate limiter、idempotency 均可为 nil；测试和 sandbox 需要这种灵活性，但生产没有一个类型或构造阶段证明“安全装配完整”。ToolBridge 对生产 mutation 在 verifier/store 缺失时会 fail-closed，设计更稳健。证据：`command/command.go:34-87`、`toolbridge/bridge.go:241-315`。

修复：新增生产装配验证或 `NewProductionDispatcher`，启动时检查 catalog、audit、idempotency 均存在；保留轻量构造函数给测试。

### P2：Event schema 注册明显不完整

生产只注册 `stock.alert`、`profit.*`、`compliance.*`，旁边注释仍写 “add more as event contracts stabilize”，而系统有大量 scheduler 和 pipeline topics。未登记事件是否有同等 schema 保障需要逐项确认。证据：`internal/httpx/router.go:160-166`。

### P2：大文件和多职责提高修改风险

`eventbus/bus.go` 932 行，同时承担事件模型、队列、worker、outbox、idempotency、生命周期和恢复；测试文件 1346 行。不是仅凭行数判坏，但当前 P1 正发生在多个状态职责交叠处，说明拆分状态机/存储适配器会直接降低正确性风险。

## 已做得好的基础

- Command/ToolBridge 都把审批绑定 action、target 和 idempotency key，并有消费、完成、失败状态。
- PostgreSQL 下 Scheduler 使用 advisory leader lease；retry store 恢复失败时不会报告 running。
- EventBus 有优先级 worker、panic recovery、DLQ、outbox、schema registry、correlation/idempotency context 和 shutdown drain。
- ToolBridge 以注册 driver 的 category 为权威，调用者不能把 mutation 伪装成 read。
- 生产装配为 Command 和 ToolBridge 接入持久化幂等，Scheduler 接入持久化 retry；PostgreSQL 才启用 leader lease。

## 最小修复与验证顺序

1. 修正 EventBus outbox/retry 状态机并增加崩溃恢复集成测试。
2. 所有 retry enqueue 错误必须可见且保持持久化可恢复。
3. handler 调用移出订阅锁，增加 handler 内 Subscribe/Unsubscribe 的回归测试。
4. 把 Command audit recorder 设为生产必需并增加 router 装配测试。
5. 补齐实际生产 topics 的 schema 清单；暂不增加新的平台 primitive。

通过标准不是“236 项继续通过”，而是新增的失败重启、队列满、锁重入和生产装配测试能够先复现问题、修复后通过。
