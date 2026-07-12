# 凌镜外部优秀系统对标总览（2026-07-12）

## 结论

外部标杆表明，凌镜不缺“可选的新框架”，真正缺的是把优秀系统共同遵守的不变量做成不可绕过的代码和数据库规则。

当前最适合凌镜的组合不是照搬某一个大系统，而是：

- 用 Microsoft DDD 的聚合和模块边界组织领域代码；
- 用 PostgreSQL 事务、约束、锁、outbox 和备份恢复作为单一可靠基础；
- 用 GitHub protected environments 的职责分离和禁止自审管理 Owner 批准；
- 用 Stripe 的稳定幂等键与未知结果对账处理外部副作用；
- 用 Cedar 的默认拒绝语义统一路由授权；
- 用 Testing Library、Playwright、Go race/fuzz 建立以风险为中心的质量门。

Temporal、AWS Step Functions、Kafka、Debezium、OpenFGA、Keycloak、Kubernetes 都是优秀标杆，但当前整体引入会扩大运维面和认知负担。凌镜应先复制它们的正确性语义，而不是复制它们的分布式平台。

## 优秀标杆及适用判断

| 领域 | 优秀标杆 | 值得立即借鉴 | 当前不照搬 |
|---|---|---|---|
| 领域与模块 | Microsoft DDD、PostgreSQL | 聚合边界、单一 command 写入口、短事务、DB 不变量、version/行锁 | 微服务拆分、复杂 CQRS 平台 |
| 长流程 | Temporal、AWS Step Functions、Saga | 持久步骤、错误分类、超时、补偿、恢复测试 | 独立工作流集群、云状态机、分布式 Saga 编排 |
| 授权 | Cedar、Zanzibar/OpenFGA | 默认拒绝、显式 permit、对象和动作绑定、集中 route policy | 策略语言平台、关系图权限服务 |
| 身份 | Keycloak | token 生命周期、会话撤销、禁用账号失效 | 独立 IAM 服务及其运维面 |
| 审批 | GitHub protected environments/branches | 申请人与审核人分离、禁止自审、内容变化批准失效 | 多团队、多级企业审批 |
| 外部副作用 | Stripe | 稳定幂等键、同键重放、timeout 后查状态而非盲重试 | 把所有内部读写包装成远程支付式 API |
| 审计 | AWS CloudTrail | 可信主体、不可变记录、外部签名/锚点、审计不可由客户端伪造 | 全套云审计数据湖 |
| 事件 | Debezium Outbox、Kafka | 数据变更与 outbox 同事务、至少一次、消费端幂等 | Kafka 集群、CDC 平台、笼统 exactly-once 宣称 |
| 调度/恢复 | PostgreSQL advisory lock、WAL/PITR | 单 leader + job 幂等、真实恢复测 RPO/RTO | 多区域热备和 Kubernetes 控制面 |
| 可观测性 | Prometheus/Alertmanager、probe contract | backlog、oldest age、DLQ、unknown outcome、恢复耗时和告警送达 | 大型多集群观测平台 |
| 前端测试 | Testing Library、Playwright、Next.js | 用户可见行为、隔离数据、生产式 E2E、web-first assertion | 第二套重叠测试框架和全页面 E2E |
| Go 质量 | race detector、native fuzzing | 并发路径 race；外部输入、body、状态枚举 fuzz | 为覆盖率对所有 CRUD 做 fuzz |

## 优秀系统的共同定义

### 1. 正确性由不变量保证

优秀系统不会只在 Service 的某个 if 中记住关键规则。不可重复、禁止自审、状态只能单向推进、证据必须有快照、利润必须为正等规则，会同时体现在：

- 类型和 command 输入；
- 单一领域写入口；
- 数据库约束或事务锁；
- 负向、并发和 PostgreSQL 集成测试；
- 审计与可恢复状态。

### 2. 默认拒绝，而不是缺配置就放行

授权、审批、审计、幂等和外部写保护在生产环境必须是必需依赖。模块不知道调用者是否合法时应拒绝，不能把 nil 当作“不启用检查”。

### 3. 不承诺不存在的 exactly-once

网络超时后，系统通常无法知道外部副作用是否发生。优秀设计使用稳定幂等键、至少一次投递和 `reconcile_required`，通过查询外部状态完成对账，而不是把 HTTP error 当成“肯定没执行”。

### 4. 恢复能力通过演练证明

有备份脚本、有 outbox 表、有 retry 队列，都不等于可靠。必须测试进程在每个关键阶段崩溃，然后验证重启能够恢复；备份必须实际恢复并测出 RPO/RTO。

### 5. 测试数量服从风险

优秀测试体系会优先覆盖会造成越权、错误决策、重复资金动作、数据丢失和无法恢复的路径。3131 个测试若没有阻止审批自审，测试体系仍需要调整。

## 对凌镜八个模块的具体借鉴

| 模块 | 当前最值得借鉴 | 暂时不做 |
|---|---|---|
| DemandCase | DDD 聚合 + PostgreSQL：所有裁决证据只走 typed command；快照/run/Owner 关系用 FK、唯一约束和事务保护 | 新研究 Agent、工作流平台 |
| Experiment | 显式状态机、version 锁、同类型 link 唯一、金额 decimal、gate 与审计同事务 | Temporal/Saga 集群 |
| Sourcing1688 | 把已有 PostgreSQL 不可变触发器测试变成默认 CI；批准绑定内容 hash；外部 publish timeout 进入 reconcile | 批量铺货、自动采购、更多平台 |
| 小Q | runtime 可验证 Capability schema、所有路径执行 timeout、trace 写入事务或明确可恢复状态 | 通用 Agent 平台、直接写数据库 |
| Auth/RBAC | Cedar 式全路由默认拒绝；token version/session revoke；唯一 Owner policy | OpenFGA/Keycloak 服务 |
| Approval/Audit | GitHub 式禁止自审、内容变化批准失效；Stripe 式幂等/对账；CloudTrail 式客户端不可伪造 | 多级企业审批和审计数据湖 |
| Platform | PostgreSQL outbox 作为唯一权威；至少一次 + 稳定幂等；崩溃恢复、队列满和 DLQ 测试 | Kafka/Debezium/Temporal/K8s |
| Deploy/Recovery | 真实恢复测 RPO/RTO；告警送达；记录原版本/迁移/备份恢复点 | 多区域热备 |

## 推荐的最小借鉴路线

### 第一阶段：把安全和正确性语义落地

1. 全路由默认拒绝；只有显式登记 policy 才允许访问。
2. 审批禁止自审，Owner 身份独立校验；批准绑定 action、target、idempotency key 和内容 hash。
3. DemandCase/Experiment 使用单一 typed command 写入口，并把 P0 不变量落到数据库。
4. 外部写超时统一进入 `reconcile_required`，禁止盲重试。

### 第二阶段：把可靠性变成可恢复状态机

5. 修 PostgreSQL outbox：`pending → processing → pending/delivered/dead_letter`；数据库是唯一权威。
6. 增加三个崩溃点测试：业务提交/outbox 前后、外部副作用返回前后、审批消费/完成之间。
7. 所有高风险审计与业务 mutation 同事务，或使用同事务 outbox；禁止客户端直接创建可信 operation log。

### 第三阶段：把测试门对准最高风险

8. PostgreSQL 约束、race、关键 fuzz 进入默认 CI。
9. 只增加 6 条生产式 E2E：登录权限、DemandCase、Experiment、1688、小Q、审批审计。
10. 用最新备份完成隔离恢复并记录实际 RPO/RTO；发送一次真实测试告警。

## 引入大型标杆的触发条件

只有出现以下事实之一，才重新评估 Temporal/Kafka/OpenFGA/Keycloak/Kubernetes：

- 多个独立服务需要跨进程、跨天持久工作流；
- PostgreSQL outbox 吞吐或锁竞争被实际测量为瓶颈；
- 权限关系扩展到多个组织、角色层级和共享对象，现有 route policy 无法清楚表达；
- 单机/Compose 的部署、恢复或可用性无法满足已经明确的 RPO/RTO；
- 当前简化实现已经重复发生无法恢复的事故。

在触发条件出现前，引入这些系统不是“更优秀”，而是用额外复杂度提前支付尚未发生的问题。

## 独立报告

- `docs/research/external-benchmark-domain-workflow-2026-07-12.md`
- `docs/research/external-benchmark-security-governance-2026-07-12.md`
- `docs/research/external-benchmark-reliability-ops-2026-07-12.md`
- `docs/research/external-benchmark-frontend-testing-quality-2026-07-12.md`

各独立报告均引用官方文档、原始论文或官方源码，并区分外部资料陈述与针对凌镜的推断。本轮只做调研和对照，没有安装新组件或修改业务代码。
