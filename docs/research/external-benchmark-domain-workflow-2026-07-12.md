# 外部标杆调研：模块化后端、领域模型、状态机与工作流

> 日期：2026-07-12
> 范围：只研究优秀系统如何划分模块、维护领域不变量、执行状态转换、处理长流程与失败。
> 证据边界：外部结论只采用官方文档、官方源码或原始论文；对凌镜的适用判断均标为 `inferred`，不是已实施事实。
> 当前产品边界：凌镜是单 Owner 使用的 Go/PostgreSQL 内部系统。本报告不建议微服务化、外部 SaaS、多租户或增加自治 Agent。

## 1. 结论先行

`inferred`：对凌镜而言，“优秀”不是拥有最先进的工作流引擎，而是满足以下七点：

1. 每个模块代表一个清晰的业务问题，拥有自己的术语、状态和写入入口；
2. 关键业务规则由领域服务/聚合入口和数据库共同保护，不能只靠前端按钮；
3. 每次状态转换在一个短事务内完成：锁定当前对象、验证旧状态与前置条件、写新状态、写不可变事件/审计记录；
4. 外部调用不放在数据库事务中，调用使用稳定幂等键，允许安全重试；
5. 可逆失败重试，不可逆或永久性错误停止并等待 Owner；补偿必须是显式业务动作，不是假装“回滚现实”；
6. 长等待保存为持久化状态和截止时间，进程重启后仍能继续；
7. 测试不仅覆盖“成功走完”，还覆盖非法转换、并发竞争、重复请求、超时、重启恢复、补偿失败和历史兼容。

最小推荐：继续使用模块化单体 + PostgreSQL；先统一三个核心模块的状态转换命令、数据库约束、幂等收据和恢复任务。现阶段不引入 Temporal、AWS Step Functions、Saga 框架、事件溯源或拆分微服务。

## 2. 证据等级与来源

本文使用：

- `quoted`：来源明确表达的含义；为避免版权问题，正文主要转述，仅保留极短原词。
- `inferred`：根据多个来源及凌镜当前边界作出的工程判断。
- `unknown`：没有通过本次外部调研或运行验证确认。

主要一手来源：

- [Microsoft Learn：DDD-oriented microservice](https://learn.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/ddd-oriented-microservice)
- [PostgreSQL：Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
- [PostgreSQL：Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html)
- [PostgreSQL：Explicit Locking](https://www.postgresql.org/docs/current/explicit-locking.html)
- [Temporal：Workflow Execution](https://docs.temporal.io/workflow-execution)
- [Temporal：Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)
- [Temporal Go SDK：Testing](https://docs.temporal.io/develop/go/testing-suite)
- [Temporal 官方 Go 示例](https://github.com/temporalio/samples-go)
- [AWS Step Functions：Error handling](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-error-handling.html)
- [AWS Step Functions：Best practices](https://docs.aws.amazon.com/step-functions/latest/dg/sfn-best-practices.html)
- [AWS Step Functions：Testing state machines](https://docs.aws.amazon.com/step-functions/latest/dg/test-state-isolation.html)
- [Garcia-Molina、Salem：Sagas 原始论文](https://doi.org/10.1145/38713.38742)

## 2.1 五个真正值得看的标杆及适用条件

| 标杆 | 为什么优秀（`quoted`/官方能力） | 适用条件 | 对凌镜的裁决 |
|---|---|---|---|
| Microsoft DDD 分层与聚合根 | 用 bounded context 分离问题域；聚合根是规则与不变量的单一入口；应用层只协调，不承载领域规则 | 业务规则开始复杂、多个入口会修改同一对象、需要长期维护共同语言 | **立即借鉴原则**；只做模块化单体，不照搬微服务部署 |
| PostgreSQL 原生约束与事务控制 | `NOT NULL/CHECK/UNIQUE/FK/EXCLUDE` 提供不可绕过的数据保护；行锁和 Serializable 可控制并发异常 | 以 PostgreSQL 为事实库、写入必须原子一致、存在重复请求或并发操作 | **当前首选实现基础**；成本最低且与现栈一致 |
| Temporal Durable Execution | 事件历史 replay、持久 timer/signal、Activity 重试、失败后从已记录步骤恢复；官方 Go SDK 有时间跳跃和历史 replay 测试 | 大量跨服务、运行数天以上、人工等待、频繁重启恢复、复杂版本兼容 | **优秀但当前过度设计**；把其幂等、错误分类、恢复测试思想移植到 Postgres job |
| AWS Step Functions | 托管状态机，声明式 `Retry/Catch/Timeout`，执行历史和 AWS 服务集成 | 基础设施已主要在 AWS/Lambda，希望由 AWS 承担编排运维并接受平台绑定/计费 | **优秀但当前不适合**；可借鉴超时、退避、结果未知分支 |
| Saga 原始模型 + Temporal 官方 Go saga 示例 | 把长事务拆成已提交子事务；失败后执行语义明确的补偿动作 | 真正跨多个独立事务/服务，无法用一个本地数据库事务完成，且每一步有可定义的补偿 | **仅作为未来跨外部副作用参考**；当前内部状态流不应 Saga 化 |

`inferred`：前两个是凌镜现在应直接采用的“实现标杆”；后三个是“可靠性设计标杆”，目前只借鉴原则并设置引入阈值。一个方案优秀，不等于它适合当前规模。

## 3. 优秀的模块边界是什么样

### 3.1 标杆原则

`quoted`：Microsoft 的 DDD 指南把独立问题域称为 bounded context（限界上下文）；每个上下文识别自己的实体、值对象和聚合。应用层只协调用例，不应持有领域状态或定义核心业务规则；业务不变量应由领域模型和聚合根保护，领域层不直接依赖表现层或基础设施框架。

`inferred`：虽然该指南以微服务为示例，但“上下文边界”和“规则归属”不要求部署成微服务。单 Owner、单数据库系统更适合把边界落实为同一 Go 进程内的独立 package/module：

- 模块公开少量命令和查询，不允许其他模块任意改它的表；
- 模块内部拥有状态定义、转换规则、错误类型和持久化代码；
- 跨模块只传 ID、不可变快照或明确的领域结果，不共享一个无边界的巨型 model；
- HTTP handler 只做鉴权、解析、错误映射；Service/Command 执行业务规则；数据库做不可绕过的最终保护。

### 3.2 “好”与“不好”的判定

| 维度 | 好 | 不好 |
|---|---|---|
| 模块目的 | 一句话能说明它裁决什么业务问题 | 一个模块同时承担采集、研究、审批、发布、利润等不相干职责 |
| 写边界 | 只有模块自己的命令能改变核心状态 | handler、定时器、Agent、其他模块都直接更新状态列 |
| 术语 | 同一个词在代码、API、页面、文档中含义一致 | 技术命名暗示了业务上并未成立的结论 |
| 依赖方向 | 领域逻辑不依赖 Gin/GORM/页面 DTO | 业务规则散落在 handler、React 按钮和 ORM hook 中 |
| 跨模块协作 | 通过受控查询/命令和稳定 ID | 多模块联表后直接越权写入 |

### 3.3 对凌镜的适用判断

`inferred`：`demandcase`、现有技术名 `experiment`、`sourcing1688` 应继续是三个独立模块，不应合并成一个“大经营工作流”。推荐边界：

- `demandcase`：只负责候选市场定义、研究快照、证据充分性和 Owner 市场裁决；
- `experiment`：按当前方向只负责经营事实核验案卷，不宣称因果实验或反馈闭环；
- `sourcing1688`：负责私人采集箱、可信采集快照、商品整理、草稿准备与独立发布审批；
- 跨模块前置条件由目标模块在命令执行时重新读取并验证，不能信任前端传来的 `passed=true`。

## 4. 优秀的状态转换与领域不变量

### 4.1 一个可靠转换应包含什么

`inferred`：每个写命令应采用下列固定结构：

```text
输入 command_id / actor / expected_version / payload
→ 开启短数据库事务
→ SELECT 当前对象（必要时 FOR UPDATE）
→ 验证身份、当前状态、expected_version、领域前置条件
→ 执行唯一允许的 from → to 转换
→ 更新实体并增加 version
→ 同事务写 transition_receipt / audit / outbox
→ 提交
→ 事务外执行外部副作用
```

`quoted`：PostgreSQL 官方文档说明，`FOR UPDATE` 会阻止其他事务修改或锁定相同行，直到当前事务结束；行锁不阻止普通读取。文档也提醒显式锁可能产生死锁，多个对象应按一致顺序加锁。

`inferred`：单 Owner 不等于没有并发。浏览器重复点击、网络重试、后台任务、Agent 与 Owner 同时操作都会制造竞争，因此状态转换仍需 `expected_version`（乐观并发版本）或行锁。

### 4.2 不变量应放在哪里

`quoted`：PostgreSQL 支持 `NOT NULL`、`CHECK`、`UNIQUE`、主键、外键和排斥约束；官方建议能用 `UNIQUE`、`EXCLUDE` 或 `FOREIGN KEY` 表达的跨行/跨表限制优先使用这些机制。PostgreSQL 明确不支持用引用其他行的 `CHECK` 可靠保证跨行规则。

`inferred`：推荐三层防护，而不是任选其一：

1. 领域代码：生成清楚、可读的错误，表达为什么不允许；
2. 数据库约束：防止绕过 service、并发竞争或迁移脚本写出非法数据；
3. 测试：证明两层规则一致，并覆盖数据库实际并发。

适合数据库直接保护的规则：

- 必填字段：`NOT NULL`；
- 状态枚举与同一行组合：`CHECK`；
- 同一 Owner/业务对象只能有一个当前有效记录：部分唯一索引；
- 引用对象必须存在且属于正确复合键：外键/复合外键；
- command、采集请求、外部提交不能重复：稳定键上的 `UNIQUE`；
- 时间区间不可重叠：必要时使用排斥约束。

需在事务中的领域服务或约束触发器保护的规则：跨多表证据齐全、同一订单的结算/利润/现金一致性、批准内容哈希等。不要伪装成跨表 `CHECK`。

### 4.3 隔离级别的选择

`quoted`：PostgreSQL 的 `SERIALIZABLE` 模拟事务串行执行，但应用必须准备重试 SQLSTATE `40001` 的序列化失败；官方也指出监控依赖和事务重启有成本，并建议事务只包含完整性所需的最少工作。

`inferred`：凌镜不需要全局改成 `SERIALIZABLE`。推荐：

- 单对象状态转换：短事务 + `FOR UPDATE` 或 `version` 条件更新；
- 少数“先统计/检查多行，再据此写入”的关键闸门：可选择 `SERIALIZABLE`，并统一处理 `40001` 重试；
- 外部 HTTP、LLM、浏览器、平台调用绝不放在数据库事务和锁中。

## 5. 长流程、重试与补偿

### 5.1 Temporal 和 Step Functions 真正优秀在哪里

`quoted`：Temporal 将 Workflow Execution 定义为可持久恢复的执行；失败后通过事件历史 replay，从最后记录的事件继续。Workflow 代码必须确定性，容易失败或非确定性的 API/LLM 调用应放入 Activity。Activity 默认指数退避重试，而整个 Workflow 默认不重试；永久性错误应设为不可重试。

`quoted`：AWS Step Functions 为 Task/Parallel/Map 提供声明式 `Retry` 与 `Catch`，可设置最大次数、退避和抖动。AWS 还建议为任务设置合理超时；没有超时，任务可能永久等待。Express 工作流按至少一次模型执行，步骤可能运行多次；非幂等操作应谨慎选择执行模型。

这些系统的共同优秀点不是“画状态图”，而是：

- 每一步都有持久化身份和历史；
- 等待、超时、重试、取消是明确状态；
- 失败只重试失败步骤，不盲目重跑整个流程；
- 外部副作用默认按“可能重复执行”设计；
- 人工等待可跨进程重启；
- 运维人员能看到当前步骤、尝试次数、最后错误和下一次执行时间。

### 5.2 凌镜现在如何低成本借鉴

`inferred`：在出现大量、跨服务、数天运行且必须自动恢复的流程前，不需要引入专用引擎。用 PostgreSQL 建立轻量持久工作项即可：

```text
workflow_job
- id / type / business_id
- state: pending | running | waiting_owner | retry_wait | succeeded | failed | reconcile_required
- step
- attempt_count / max_attempts
- next_attempt_at / deadline_at
- idempotency_key UNIQUE
- locked_at / locked_by
- last_error_code / last_error_summary
- payload_hash / result_receipt_id
- created_at / updated_at
```

执行器每次只领取一个到期工作项，更新租约；步骤成功写收据后推进，进程重启后可重新领取超时租约。外部调用必须使用 `idempotency_key`，或在本地用唯一约束记录“该业务动作已提交”。未知结果不能标成功，应进入 `reconcile_required`。

### 5.3 重试分类

| 错误 | 例子 | 推荐处理 |
|---|---|---|
| 瞬时 | 连接重置、429、服务 5xx | 有上限的指数退避 + 抖动 |
| 可等待条件 | 等 Owner、等争议期结束、等平台异步结果 | 持久化等待和截止时间，不计为失败重试 |
| 永久输入错误 | URL 非法、市场未批准、证据缺失 | 不重试，返回明确阻断原因 |
| 结果不确定 | 提交后超时，不知平台是否接收 | 禁止直接重试，先 reconcile（查询/对账） |
| 安全/权限 | 无 Owner 授权、哈希变化 | 立即停止并记录审计 |

### 5.4 补偿不是数据库回滚

`quoted`：1987 年 Saga 原始论文提出把长事务拆成一系列可独立提交的子事务；失败时通过对应补偿事务处理已经提交的步骤。AWS 的 Saga 指南同样区分向前恢复（继续/重试）和向后恢复（补偿）。

`inferred`：补偿只能撤销“业务效果”，不能抹去现实事实。例如：

- 已生成草稿可作废草稿；
- 已预留库存可释放预留；
- 已提交平台但结果未知，不能用本地状态回滚伪装成未提交，必须对账；
- 已发出的消息、已被第三方读取的数据、已发生的付款不能真正撤销，只能发更正、退款或建立相反业务记录。

对凌镜当前大多数 Owner 审批/研究/草稿流程，显式的 `cancelled`、`superseded`、`reconcile_required` 状态比通用 Saga 框架更准确、更便宜。

## 6. 优秀测试是什么样

`quoted`：Temporal Go SDK 官方测试框架支持隔离测试 Workflow 和 Activity、mock Activity、控制/跳过长时间、发送 Signal，并建议在 CI 中用真实历史 replay 新代码；如果历史与新 Workflow 定义不兼容，CI 应失败。AWS 提供 `TestState` API，可单独测试一个状态及其错误处理。

`inferred`：无论是否使用工作流引擎，凌镜都应复制这种测试思想：

### 6.1 每个模块的状态机契约测试

- 列出全部合法 `from → command → to`；
- 对每个状态验证所有非法命令均被拒绝；
- 每个前置条件分别缺失一次，证明不能放行；
- 终局状态不可被普通命令重新打开；
- 状态、审批、审计、收据要么一起提交，要么全部回滚。

### 6.2 并发与幂等测试（必须在 PostgreSQL）

- 两个 goroutine 同时审批，只能一个成功；
- 同一 `command_id`、采集请求 ID、发布幂等键重复提交，业务效果只有一次；
- 事务在更新后、写审计前故障，不留下半状态；
- `40001`/死锁重试有上限，且不会重复外部副作用；
- SQLite 单元测试不能替代部分索引、锁、约束触发器和隔离级别测试。

### 6.3 恢复与故障注入测试

- 外部调用前崩溃；
- 外部调用成功但写回前崩溃；
- 外部返回 429/500/永久错误/超时；
- 进程重启后从持久化步骤继续；
- 等待 Owner 超时；
- 补偿本身失败后进入可观察、可人工处理状态；
- 历史记录能解释谁、何时、基于哪个版本/哈希执行了什么。

### 6.4 不要用测试数代替质量

`inferred`：一个模块有数百个 happy-path 测试，仍可能不如十几个覆盖非法转换、并发、重复和恢复的契约测试可靠。核心验收应按风险矩阵报告，而不是只报告测试总数。

## 7. 三个模块分别值得借鉴什么

以下均为 `inferred`，需后续结合代码逐项确认，不能当作已实施状态。

### 7.1 demandcase

最值得借鉴：

1. 把“候选定义”“研究 run”“不可变证据快照”“Owner 裁决”视为不同聚合/记录，不混成一个可任意改写的大对象；
2. `experiment_ready` 只能由单一领域命令生成，事务中重新计算八维证据和独立反证条件；
3. 数据库用唯一键保证同 run 幂等、payload hash 不重复漂移、同一裁决版本不可重复生效；
4. 裁决保存输入版本/快照哈希，后续证据变化不能静默改写历史裁决；
5. 并发测试：研究写入与 Owner 裁决同时发生时，必须明确使用哪个证据版本。

不建议：为少量人工研究引入 Temporal；把每条证据做成事件溯源系统；拆成研究、反证、裁决三个微服务。

### 7.2 experiment（经营事实核验案卷）

最值得借鉴：

1. 先纠正领域语言：模块不应通过状态名暗示因果实验或经营反馈闭环；
2. 每个闸门是纯领域判定：输入是明确版本的事实集合，输出是 `passed/blocked/unknown` 和逐条理由；
3. 事实记录追加而非覆盖，纠错用 supersede/reconcile 记录；
4. 同订单、结算、利润、现金的关联同时由事务内服务与数据库复合外键/唯一约束保护；
5. 终局命令使用版本锁，避免后台对账和 Owner 操作同时推进；
6. 长等待（签收、争议期、对账）保存 deadline 和 wake-up job，不依赖进程内 timer。

不建议：用通用工作流 DSL 取代清晰的 Go 领域代码；把交易事实关联包装成 Saga；把“流程走完”当作因果验证。

### 7.3 sourcing1688

最值得借鉴：

1. 私人采集箱与受控草稿是两个清楚边界：前者允许 `unverified_lead/quoted`，后者必须重新通过经营前置条件；
2. 采集、图片处理、规则查询、平台提交都是可能重复的外部 Activity，应有稳定幂等键和结果收据；
3. 平台发布使用 `prepared → approved → submitting → submitted | reconcile_required | rejected` 之类的显式状态，不以 HTTP 无错误等同“已上线”；
4. 批准绑定不可变 payload hash；执行前重新核验 hash、权限和审批仍有效；
5. 超时后的第一动作是按幂等键/平台对象查询结果，不是盲目再次发布；
6. 图片/规则处理可使用 PostgreSQL 工作项和租约恢复，达到规模阈值后才评估 Temporal。

不建议：为了一个 Owner 的少量商品部署 Temporal Server；使用 Step Functions 会引入 AWS 绑定、IAM、日志和按状态转换计费；把采集箱保存也建成多步骤 Saga。

## 8. 何时才值得引入 Temporal 或 Step Functions

### 8.1 Temporal

只有同时出现多项信号才建议评估：

- 数十种跨多个外部服务的长流程；
- 大量流程持续数天/数月并等待人工 Signal；
- 进程重启后精确恢复已经成为频繁事故；
- 自建 PostgreSQL job/outbox 的重试、timer、版本兼容和可观察性维护成本持续上升；
- 团队有能力维护 Temporal Server/Worker，或愿意承担 Temporal Cloud 成本和平台依赖；
- 愿意接受确定性 Workflow、Activity 幂等、历史 replay 和工作流版本管理的开发约束。

当前判断：`inferred` 为过度设计。凌镜单 Owner、模块化单体、单 PostgreSQL 尚可用更小机制解决。

### 8.2 AWS Step Functions

适合已有大量 AWS Lambda/服务集成、需要托管状态机、IAM 与 CloudWatch 运维的系统。成本包括 AWS 绑定、状态定义维护、状态转换/日志费用、数据大小与历史配额，以及本地测试复杂度。

当前判断：`inferred` 为过度设计，除非未来生产基础设施已经全面位于 AWS，且长流程成为主导负担。

### 8.3 成熟 Go 状态机库

`inferred`：状态机库能减少 `switch` 样板，但不能自动解决事务、数据库约束、跨模块权限、幂等、副作用、审计和恢复。因此不应把“采用库”当作质量升级。凌镜的状态数量可控时，显式 transition table + typed command + PostgreSQL 契约测试更透明。

## 9. 成本与风险

| 方案 | 初始成本 | 持续成本 | 主要收益 | 当前适合度 |
|---|---:|---:|---|---|
| 现有 Go/Postgres 内统一转换命令与约束 | 低到中 | 低 | 直接关闭绕过、并发和重复问题 | 高 |
| Postgres 持久 job/outbox + 幂等收据 | 中 | 低到中 | 重启恢复、有限重试、可观察 | 高（仅外部流程） |
| 通用 Go 状态机库 | 低 | 中 | 减少部分样板 | 低到中；收益有限 |
| Temporal Cloud | 中到高 | 中到高 | durable workflow、timer、signal、history | 当前低 |
| 自建 Temporal | 高 | 高 | 同上且自控基础设施 | 当前极低 |
| AWS Step Functions | 中 | 中到高 | 托管编排、AWS 集成 | 当前低 |
| 微服务 + Saga | 很高 | 很高 | 独立部署/扩缩和跨服务恢复 | 当前不适合 |
| 全事件溯源 | 很高 | 很高 | 完整事件重建和时态模型 | 当前不适合 |

## 10. 最小借鉴清单（推荐执行顺序）

### 第一阶段：不增加新基础设施

1. 为三个模块各写一张状态转换表：状态、命令、操作者、前置条件、目标状态、审计字段；
2. 所有核心状态只能通过 typed command/service 修改；禁止 handler、Agent、scheduler 直接更新；
3. 每个命令携带 `command_id` 和 `expected_version`；数据库建立唯一约束和版本条件更新；
4. 把 `NOT NULL/CHECK/UNIQUE/FK/partial unique index` 能表达的不变量下沉数据库；
5. 状态更新与 transition receipt/audit/outbox 同事务提交；
6. 增加 PostgreSQL 并发、幂等、非法转换和事务回滚测试；
7. 每个闸门返回逐条 `passed/blocked/unknown`，保存判定时的输入版本/哈希。

### 第二阶段：只覆盖真实外部副作用

8. 建最小 `workflow_job`/outbox：租约、尝试次数、next attempt、deadline、错误分类；
9. 采集、图片、平台提交使用稳定幂等键；
10. 区分 `retry_wait`、`waiting_owner`、`failed`、`reconcile_required`；
11. 超时不等于失败或成功；对未知外部结果先对账；
12. 增加崩溃点和重启恢复测试。

### 暂不做

- 不拆微服务；
- 不引入 Temporal/Step Functions；
- 不建设通用低代码工作流平台；
- 不把所有表改为事件溯源；
- 不为内部纯数据库状态转换使用 Saga；
- 不增加 Agent/MoA 自治层来掩盖领域规则不清。

## 11. 通过、失败与升级条件

### 本轮调研通过条件

- `actual`：已形成一份仅引用一手来源的外部标杆报告；
- `actual`：明确了哪些原则可低成本复用，哪些方案属于当前过度设计；
- `unknown`：本轮没有逐行审查三个模块当前代码，不能断言清单中的缺口均存在；
- `unknown`：没有实施或运行测试，不能提高任何工程事实等级。

### 后续工程质量通过条件

`inferred`：只有在以下证据存在后，才能称一个模块的状态基础“好”：

1. 状态转换矩阵完整且与代码一致；
2. 非法转换、重复命令、并发竞争、崩溃恢复测试通过；
3. 数据库约束可阻止绕过 service 的非法数据；
4. 外部副作用有幂等收据和未知结果对账；
5. 生产式 PostgreSQL 验证，而非只用 SQLite；
6. 审计历史能重建每次关键决定的 actor、输入版本、理由和结果。

### 重新评估专用工作流引擎的触发条件

当自建持久 job/outbox 已经反复出现 timer、signal、长历史版本兼容、复杂补偿、跨服务恢复和运维可视化负担，并且这些是实际事故/成本而不是预测时，再用 Temporal 做小型概念验证。概念验证应比较真实故障恢复成本，而不是比较功能列表。
