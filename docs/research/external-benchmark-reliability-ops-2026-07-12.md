# 可靠性与运维外部标杆调研（2026-07-12）

## 1. 结论先行

凌镜不需要为了“看起来先进”引入 Kafka、Debezium、Temporal 或 Kubernetes。对当前单机/小规模 Go + PostgreSQL 系统，优秀基础的最低定义是：

1. 业务写入和 outbox 事件在**同一个 PostgreSQL 事务**提交；数据库是待投递事件的唯一权威事实源，内存队列只负责加速。
2. 采用“**至少一次投递 + 消费端幂等**”，不声称端到端 exactly-once（恰好一次）。只有外部服务也接受并持久化同一个幂等键时，才能防止重复外部副作用。
3. 重试状态、尝试次数、下次执行时间和终局失败均持久化；临时失败回到可恢复状态，只有耗尽重试才进入 DLQ（死信区），任何重新入队失败都必须可见。
4. Scheduler（定时任务）继续使用 PostgreSQL session advisory lock 做单 leader，但每个任务本身仍须幂等；leader 只减少重复，不消除崩溃边界。
5. health 只回答“进程是否活着”，readiness 回答“是否可以安全接流量/执行任务”；恢复 outbox、retry queue 或取得 scheduler leader 失败时保持 not-ready。
6. 备份好不好，不以文件存在判断，而以真实恢复能否在约定的 RPO/RTO 内完成判断。RPO 是最多可丢多少时间的数据，RTO 是多久恢复服务。

对照现状，凌镜已有 outbox、幂等 claim、重试、DLQ、leader lease、readiness、备份/恢复/回滚脚本和 Prometheus 基础，方向总体正确。当前最值得做的不是换基础设施，而是修复 EventBus 状态分叉与静默重新入队失败，并完成三类故障注入和一次真实恢复演练。

## 2. 范围、方法与证据等级

本报告研究：事件系统、transactional outbox、幂等、重试/DLQ、scheduler leader、可观测性、部署、备份、灾难恢复和 rollback。

来源限定为官方文档和项目官方资料：PostgreSQL、Debezium、Apache Kafka、Temporal、Kubernetes、Prometheus、AWS 和 Google Cloud。搜索结果和官方做法是外部参考，不证明凌镜已经实现或验证。

- `actual`：本轮直接读取到的凌镜审计记录或官方文档内容。
- `quoted`：官方来源的明确主张。
- `inferred`：把一手资料应用到凌镜后的工程判断。
- `recommended`：适合当前规模的建议，尚未实施。
- `unknown`：本轮没有用真实运行或故障演练证明。

内部对照基于：

- [平台基础设施技术质量审计](tech-quality-platform-infrastructure-2026-07-12.md)
- [部署、备份与恢复技术质量审计](tech-quality-deploy-recovery-2026-07-12.md)
- [Kernel Contracts](../governance/KERNEL_CONTRACTS.md)

## 3. 五个真正优秀的外部标杆及适用条件

这里的“标杆”是其在特定问题上的设计值得学习，不代表整套产品都适合凌镜。

| 标杆 | 真正优秀之处 | 适用条件 | 凌镜当前裁决 |
|---|---|---|---|
| **Debezium Outbox Event Router** | 业务事务只写数据库和不可变 outbox，由 CDC（读取数据库变更日志）可靠转发；事件有唯一 ID、aggregate identity 和明确类型，天然支持下游去重 | 已拆成多个独立服务；事件量明显超过简单 PostgreSQL poller；团队能运维 Kafka Connect/CDC offset/schema | **借设计，不引组件**：先把现有 PostgreSQL outbox 状态机做正确；当前没有引入 Kafka/Connect 的规模证据 |
| **Apache Kafka idempotent/transactional processing** | 官方明确拆分 producer、broker、consumer 的保证；Kafka Streams 能把 input offset、state store 和 Kafka output 原子提交，exactly-once 边界清楚 | 高吞吐持久日志、分区并行、多消费者、事件保留/重放是实际需求，且副作用主要留在 Kafka 事务域 | **作为语义标尺，不部署**：帮助凌镜拒绝笼统 exactly-once；它不能替凌镜保证外部电商平台写入不重复 |
| **Temporal Activity + durable retry** | retry/backoff 由持久历史驱动，worker 重启不丢计时；官方明确 Activity 可能执行多次，要求业务写幂等、外部服务持久化幂等键 | 跨分钟/天的长流程、等待人工/外部回调、补偿分支很多，现有数据库状态机维护成本已成为真实瓶颈 | **借 Activity 原则，不部署**：采用持久化 retry、permanent error、stable idempotency key、unknown outcome；当前流程复杂度不足以证明需要 Temporal cluster |
| **PostgreSQL WAL/PITR + advisory locks** | 同一数据系统同时提供事务原子性、时间点恢复和 session 生命周期 leader lease；组件少、故障边界较窄 | 单体或少量服务、PostgreSQL 是事实源；Owner 可接受单数据库架构，并愿意监控 WAL 归档/磁盘 | **最匹配凌镜**：继续使用 advisory lock；真实关键订单出现且 RPO 24h 不可接受后启用/采用托管 PITR，而非现在盲目上多区域 |
| **Prometheus + Alertmanager + Kubernetes probe contract** | 指标把 attempts/failures/latency/backlog 暴露出来；告警做分组、去重、路由；liveness/readiness/startup 各自职责明确 | 任何需要值守或故障诊断的服务；不要求实际运行 Kubernetes，也可复用探针语义 | **直接借鉴**：补 outbox age、enqueue failure、DLQ、unknown outcome、last successful backup/restore；沿用 Compose/systemd，不部署 Kubernetes |

选择原则：当一个标杆解决的问题还没有被真实测量到，就只借其契约和测试方法；当现有简单方案已被吞吐、隔离或流程复杂度证明确实不够，再引入对应组件。

## 4. 优秀事件与 outbox 系统怎么定义

### 4.1 业务状态和事件必须原子提交

`quoted`：Debezium 官方把 outbox pattern 定义为避免服务内部数据库状态与下游消费事件不一致的方法；标准 outbox 事件含唯一 `id`，下游可用该 ID 去重。[Debezium Outbox Event Router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html)

`inferred`：对凌镜而言，创建/更新业务对象与插入 outbox 行必须使用同一个数据库事务。不能先提交业务数据再“尽力 publish”，也不能把内存队列是否成功当作事件是否存在的事实。

优秀状态机的最小形态：

```text
pending --claim/lease--> processing
processing --success--> delivered
processing --transient failure--> pending(next_attempt_at, attempts+1)
processing --lease expired after crash--> pending 或被新 worker reclaim
processing --attempts exhausted/permanent failure--> dead_letter
dead_letter --Owner/reconciler explicit replay--> pending
```

核心不变量：

- `delivered` 和 `dead_letter` 是明确终局，不能由普通重试随意覆盖。
- handler 首次失败不等于终局失败。
- 事件恢复扫描查询数据库的可恢复状态，不依赖旧进程内存。
- claim 必须带 lease/过期时间或用短事务 `FOR UPDATE SKIP LOCKED`；进程死亡后不能永远卡在 `processing`。
- attempts、last_error、next_attempt_at、delivered_at 必须由同一权威行记录。

### 4.2 “恰好一次”必须拆开说

`quoted`：Kafka 官方把语义分成 at-most-once、at-least-once 和 exactly-once，并特别提醒很多 exactly-once 声明在消费者、生产者或磁盘失败时并不成立。Kafka 的幂等 producer 解决的是 producer 重试导致同一日志分区重复写入；Kafka Streams 的端到端 exactly-once 依赖把输入 offset、状态存储更新和输出 topic 写入一起原子提交。[Kafka design: Message Delivery Semantics](https://kafka.apache.org/36/design/design/)；[Kafka Streams Core Concepts](https://kafka.apache.org/10/streams/core-concepts/)

`quoted`：Temporal 明确区分“平台观察到 Activity 完成一次”和“Activity 实际可能执行、甚至部分完成多次”；worker 在副作用成功后、上报完成前崩溃时会重试，因此写操作必须幂等，并由被调用服务持久化幂等键。[Temporal Activity Definition](https://docs.temporal.io/activity-definition)

`inferred`：凌镜应对 Owner 表述为：

- 数据库内事务可做到原子性；
- 事件交付采用至少一次，因此 handler 可能重复；
- 本地数据库副作用可通过唯一幂等键和同事务 claim 去重；
- 外部平台副作用只有在平台接受同一幂等键，或能用稳定 external reference 查询/对账时，才能安全重试；
- timeout/unknown response 不能记成功或盲目重试，应进入 `reconcile_required`。

### 4.3 重试与 DLQ

`quoted`：Temporal 的官方 retry policy 使用声明式重试、指数退避、最大间隔、最大尝试次数和 non-retryable errors；永久失败不应重复尝试，应直接暴露。[Temporal Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)

适合凌镜的最小标准：

| 情况 | 行为 | 不允许 |
|---|---|---|
| 临时网络/数据库错误 | 指数退避 + jitter，持久化 `next_attempt_at` | 紧密循环或仅内存重试 |
| 输入/权限/业务规则错误 | 标记 permanent，直接终止或 DLQ | 重试到耗尽资源 |
| 外部写超时、结果未知 | 保留同一幂等键，先查询/对账 | 生成新键再次写入 |
| 重试耗尽 | DLQ + 告警 + 可审计 replay | 静默丢弃 |
| replay | Owner/受控 reconciler 显式操作，沿用事件身份 | 修改 payload 后冒充原事件 |

“DLQ 有数据”本身不是可靠性；优秀系统还必须能回答：为什么失败、影响哪个对象、已经尝试几次、是否可能产生外部副作用、谁决定 replay、replay 的结果是什么。

## 5. 对照凌镜 EventBus：最小借鉴

### 已有基础

`actual`（来自 2026-07-12 内部技术审计，非本轮重新运行）：EventBus 已有 PostgreSQL outbox、worker pool、重试、DLQ、幂等上下文、panic recovery、指标、优雅关闭和启动恢复；Scheduler 已有 PostgreSQL advisory leader lease 和持久化 retry store。

### 与标杆不符的关键点

1. `actual`：当前审计发现 handler 第一次失败就把 outbox 从 `pending` 更新为 `failed`，同时只在内存重新入队；后续成功只允许 `pending → delivered`，启动恢复也只读取 `pending`。这会造成数据库与内存状态分叉，并在进程重启时丢失可恢复事件。
2. `actual`：重试重新入队使用忽略错误的调用，队列满或 backend 失败可能静默丢事件。
3. `actual`：EventBus 在订阅读锁内调用任意 handler，存在慢 handler 阻塞订阅变化或 handler 重入导致死锁的风险。

### 推荐的最小改造（不更换组件）

1. PostgreSQL outbox 行成为唯一调度真相；内存 queue 只保存 event ID，enqueue 失败不改变数据库可恢复状态。
2. 首次/中间失败更新为 `pending + attempts + next_attempt_at`，只有永久失败或耗尽才 `dead_letter`。
3. claim 使用数据库原子状态转换和 lease；成功、失败转换都检查受影响行数，CAS 失败必须记录指标。
4. 先复制匹配 subscriber 快照，释放锁后调用 handler。
5. 增加至少三项 PostgreSQL 崩溃测试：
   - 业务事务提交后、首次 dispatch 前崩溃；
   - handler 完成副作用后、ack/delivered 前崩溃；
   - handler 失败并安排重试后、真正重新执行前崩溃。
6. 每项测试都验证：重启后事件不静默丢失、允许重复投递但没有重复业务副作用、最终数据库状态与实际结果一致。

## 6. Scheduler leader 怎么定义为好

`quoted`：PostgreSQL session-level advisory lock 会一直保持到显式释放或 session 结束；连接异常断开时会自动释放。它是 application-defined（应用自行约定）的锁，数据库不会替应用强制所有参与者都遵守。[PostgreSQL Advisory Lock Functions](https://www.postgresql.org/docs/17/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS)；[PostgreSQL Explicit Locking](https://www.postgresql.org/docs/17/explicit-locking.html#ADVISORY-LOCKS)

`inferred`：凌镜当前用 PostgreSQL advisory lock 做 scheduler leader，适合当前规模，没必要引入 etcd/Consul/Kubernetes Lease。但要满足：

- leader lock 使用专用长连接；连接健康与 leader 状态可观测；连接丢失立即停止发新 tick。
- standby 不报告 scheduler ready，并持续有限退避抢锁。
- 注册任务完成、retry store 恢复完成、leader 获得后才进入 running。
- 每个 tick 带稳定的 `job_name + scheduled_time` 幂等键；即使两个实例短暂都执行，业务副作用仍能去重。
- 记录 `scheduler_is_leader`、最近成功 tick 时间、延迟、失败、retry backlog。

leader election 是减少重复执行的优化，不是 exactly-once 保证。

## 7. 可观测性与避免静默失败

`quoted`：Prometheus 官方对离线处理建议记录每阶段进入量、处理中数量、最后处理时间和输出量；对 thread pool 建议记录队列长度、使用线程数、处理总数、执行时间和排队时间；每次失败都应递增计数器，并同时有总尝试数才能计算失败率。[Prometheus Instrumentation](https://prometheus.io/docs/practices/instrumentation/)

`quoted`：Alertmanager 的职责是告警去重、分组、路由、静默和抑制；其 HA 设计在网络分区时宁愿重复通知也不漏关键告警，语义是至少一次。[Prometheus Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/)；[Alertmanager High Availability](https://prometheus.io/docs/alerting/latest/high_availability/)

凌镜最小必需指标：

| 范围 | 指标/告警 |
|---|---|
| outbox | pending/processing/DLQ 数量、最老 pending 年龄、claim 冲突、状态 CAS 失败、recovery 数 |
| handler | attempts、success、transient/permanent failures、duration、duplicate claims |
| queue | depth、capacity、enqueue failures、queue wait duration |
| scheduler | leader 状态、最后 tick/成功时间、tick lag、retry backlog |
| external mutation | attempts、timeouts、unknown outcomes、reconcile backlog、provider latency |
| backup | 最后成功时间、备份年龄、对象校验失败、异地上传失败、最后恢复演练时间/耗时 |

告警优先级：

- 立即通知 Owner：DLQ 新增、unknown external outcome、备份/异地上传失败、恢复演练失败、审计链失败。
- 延迟后通知：最老 pending 超过阈值、scheduler 很久无成功 tick、WAL/磁盘空间逼近上限。
- 仅仪表盘：短暂重试、单次 transient error。

指标必须低基数；event ID、order ID、用户 ID 不应作为 Prometheus label，放在结构化日志/审计并通过 correlation ID 查询。

## 8. Health、readiness 与部署/rollback

`quoted`：Kubernetes 官方区分三种探针：liveness 决定何时重启容器，readiness 决定是否接收流量，startup 防止慢启动期间过早执行前两者；错误的 liveness 可能造成级联故障。[Kubernetes Probes](https://kubernetes.io/docs/concepts/workloads/pods/probes/)

`inferred`：即使凌镜不使用 Kubernetes，这个契约仍适用于 Compose/systemd：

- `/api/health`：进程 event loop/HTTP server 是否活着，必须轻量，不能因暂时数据库故障触发无限重启。
- `/api/ready`：数据库可用、迁移版本匹配、必要配置加载、outbox/retry recovery 完成；scheduler 实例还需 leader 条件。
- shutdown：先 readiness=false，停止接收新 mutation，再 drain HTTP、scheduler、EventBus，超时后留下持久化可恢复状态。

优秀 rollback（版本回退）不是“git 切回去”：

1. 发布使用不可变 commit/image 标识，记录当前版本与目标版本。
2. 数据库迁移优先 backward-compatible 的 expand/contract（先扩展兼容结构、代码切换、最后另一次发布删除旧结构）。
3. 默认只回滚应用，不自动 down migration。
4. 回滚前备份并记录 migration version；失败后保留现场和一条明确恢复到原 commit 的路径。
5. 回滚演练同时验证 readiness 和一个最小业务读/写 smoke test，不能只看容器 running。

## 9. 备份、RPO/RTO 与灾难恢复

`quoted`：Google Cloud 对业务语言的定义是：RTO 回答“灾难后多久恢复运行”，RPO 回答“最多能承受丢多少数据”；接近零的目标会显著改变成本和架构。[Google Cloud Disaster Recovery](https://docs.cloud.google.com/architecture/disaster-recovery)

`quoted`：AWS Well-Architected 明确要求定期恢复备份，在新环境验证数据可访问、完整且未损坏，并测量是否满足 RTO/RPO；“假设备份存在/可恢复/足够快”属于反模式。[AWS REL09-BP04](https://docs.aws.amazon.com/wellarchitected/latest/framework/rel_backing_up_data_periodic_recovery_testing_data.html)

`quoted`：PostgreSQL 提供 SQL dump、文件系统备份和连续归档三类方案。WAL 连续归档配合 base backup 可做 PITR（时间点恢复）；归档落后会扩大灾难时的数据损失并可能填满 `pg_wal`，必须监控。`pg_dump` 是逻辑备份，不能作为 WAL replay 的 base backup。[PostgreSQL Backup and Restore](https://www.postgresql.org/docs/17/backup.html)；[PostgreSQL Continuous Archiving and PITR](https://www.postgresql.org/docs/17/continuous-archiving.html)

### 适合凌镜当前阶段的目标建议

这是 `recommended`，不是 Owner 已批准的 SLA，也未被演练证明：

| 等级 | 建议目标 | 最小机制 |
|---|---|---|
| 当前无真实关键经营数据 | RPO 24h、RTO 8h | 每日逻辑备份、异地不可变副本、每月恢复演练 |
| 开始产生真实订单/结算/审批 | RPO 1h、RTO 4h | 更高频备份；评估 PostgreSQL WAL/PITR；每月恢复演练 |
| Owner 明确无法承受 1h 数据损失 | 重新决策 | WAL 连续归档/托管 PostgreSQL、多故障域成本评估 |

不要未经真实损失评估就追求 near-zero RPO/RTO。对一人内部系统，可靠恢复通常比自动跨区域热切换更合适。

### 每次恢复演练的通过标准

- 使用最新真实备份恢复到全新隔离数据库，不覆盖生产。
- 记录 backup ID、SHA-256、创建时间、恢复开始/结束时间。
- 核对 schema/migration version、关键表、代表性行数与关键不可变约束。
- 应用连接恢复库后 `/api/ready` 通过，Owner 身份可登录并读取关键记录。
- 记录实际 data age（测得 RPO）和 elapsed restore time（测得 RTO）。
- 演练失败必须告警并形成修复项；成功记录不可只存在终端滚屏。

## 10. 对照凌镜部署与恢复：最小借鉴

### 已有且应保留

`actual`（来自内部技术审计）：

- 唯一 Owner/AI deployment runbook；Caddy 是唯一公开入口。
- backup 使用 partial 文件、`pg_restore --list`、SHA-256、异地上传检查，并可要求 Object Lock。
- restore verifier 使用安全命名的隔离库并核对关键表。
- migration verifier 执行 up → down all → up。
- rollback 前强制备份，验证目标网络边界，最后检查 health/readiness。

### 当前最小补齐

1. 用一份最新真实备份做一次完整隔离恢复，记录实测 RPO/RTO；现状仍是 `unknown`。
2. 明确恢复档位：当前每日逻辑备份；有真实关键订单后评估 WAL/PITR，而不是默认立即建设。
3. 备份脚本先完成异地上传与远端验证，再清理旧本地恢复点。
4. release/远端连接拒绝 `postgres/postgres` 默认凭据。
5. rollback runbook 增加每个失败步骤的裁决表：如何回原 commit、何时只恢复应用、何时必须从备份恢复数据库。
6. 实际发送一次测试告警并确认 Owner 收到，记录渠道和延迟。

## 11. 明确不照搬的优秀方案

| 方案 | 为什么优秀 | 为什么当前不照搬 |
|---|---|---|
| Kafka cluster + transactions | 高吞吐日志、分区复制、Kafka 内原子语义 | 凌镜没有相应吞吐和独立服务规模；仍不能自动解决外部平台副作用；增加 broker 运维和故障面 |
| Debezium CDC + Kafka Connect | 无需应用 poller 即可把 outbox CDC 到 broker | 当前单体 PostgreSQL 内部消费足够；会引入 Connect/Kafka/offset 运维。应借鉴 event ID、不可变 outbox 和去重，而非组件 |
| Temporal cluster | 长流程历史、durable timers、声明式 retry 很强 | 当前工作流和并发规模不足以抵消部署、持久化、学习和迁移成本；先借鉴幂等 Activity、持久化退避与 unknown outcome 模型 |
| Kubernetes | probes、滚动发布和调度成熟 | 单机 Compose/systemd 已满足当前部署边界；借鉴 health/readiness/startup 语义即可 |
| 多区域热备/自动故障切换 | 可逼近低 RTO/RPO | 成本和复杂度高，且不能防逻辑错误/误删除；当前应先证明备份可恢复 |
| exactly-once 宣称 | 在同一受控日志/事务域内可成立 | 凌镜跨 PostgreSQL、Go handler 和外部电商平台，无法笼统保证；应承诺至少一次 + 幂等 + 对账 |

## 12. 推荐执行顺序与验收

### P0：先关闭静默丢事件路径

1. 修正 outbox 权威状态机。
2. 禁止忽略 retry enqueue/state transition 错误。
3. handler 移出订阅锁。
4. 新增三类崩溃恢复 PostgreSQL 集成测试。

通过标准：故障注入后事件要么最终 delivered，要么明确 dead_letter/reconcile_required；不能消失，不能出现数据库写 failed 但实际已成功且无从对账。

### P1：证明运维机制真实可用

1. 最新真实备份隔离恢复并记录 RPO/RTO。
2. 非生产应用 rollback 演练，不回滚数据库。
3. 一次模拟不兼容迁移的人工恢复桌面演练。
4. 一次真实告警送达测试。

### P2：真实数据量和容忍度变化后再升级

只有当 Owner 已有真实订单/结算数据，且每日备份的 24 小时潜在损失不可接受时，再评估 WAL/PITR；只有当单 PostgreSQL poller 的吞吐、隔离或服务拆分成为已测瓶颈时，再评估 Debezium/Kafka；只有当持久长流程复杂度已经真实出现时，再评估 Temporal。

## 13. 最终判断

`inferred`：凌镜的可靠性基础不是“差到要推倒重来”，而是“机制已经很多，但最关键的失败语义尚未完全闭合”。优秀系统与当前系统的主要差距，不在组件名，而在四件可验证的事：

1. 数据库是否始终是 outbox 权威事实源；
2. 崩溃窗口是否经过测试且不静默丢事件；
3. 重复执行是否不会重复产生业务/外部副作用；
4. 备份是否真的恢复过，并达到 Owner 接受的 RPO/RTO。

这四项完成前，不能称为生产级可靠；完成后，即使仍是单机 Go + PostgreSQL，也可以是适合当前规模的优秀基础。
