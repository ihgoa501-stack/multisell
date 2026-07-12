# 凌镜系统开发质量与技术基础总评（2026-07-12）

## 结论

**系统不是空架子，也不是基础很差；它有真实、较广、测试数量可观的工程基础。但关键安全门、状态机不变量和失败恢复仍存在 P0/P1，所以目前只能评为 3.3/5：中等偏上、可修复，不适合在未修核心问题前继续扩大功能。**

这次评价只回答“开发得好不好、系统基础好不好”。不以真实成交或经营效果评分，也不把工程测试解释为经营结果。

## 统一评价标准

每个模块按七轴评价：

1. 正确性：正常、异常、边界和状态转换是否正确；
2. 可读性与复杂度：代码能否被继续理解和修改；
3. 架构边界：领域职责、依赖方向和唯一事实源是否清楚；
4. 安全：认证、授权、Owner 隔离、审批、输入和敏感数据是否安全；
5. 性能与数据库：查询、索引、并发、事务和数据类型是否可靠；
6. 测试质量：测试是否能捕获真实回归，而不只是主路径通过；
7. 可运维性：日志、指标、审计、部署、恢复和失败状态是否可解释。

### 开发得好

功能契约清楚，关键不变量同时由 Service、数据库和测试保护；权限默认拒绝；失败不会假成功或重复副作用；模块边界清楚；测试覆盖负向、并发、PostgreSQL 和跨层路径；部署、恢复和审计可实际验证。

### 开发得不好

页面/API 存在但关键规则可绕过；同一事实有多条路径且口径不同；权限只做登录不做角色；错误被吞掉；数据库约束弱于业务规则；测试全绿却没有覆盖会造成错误决策、越权或重复副作用的路径。

## 八个模块/基础单元评分

| 模块 | 得分 | 当前评价 | 最高风险 |
|---|---:|---|---|
| 候选市场 DemandCase | 2.7/5 | 骨架清楚，但关键证据不变量不可信 | P0：弱证据 API 可绕快照/独立 run；unknown conflict 不阻断 |
| 经营实验 Experiment | 3.0/5 | 后半段规则有深度，状态机和对象约束未形成可靠整体 | P0：负/零利润可 continue；多链接静默择新；现金金额/币种不核 |
| 1688 Sourcing | 3.9/5 | 当前业务模块中工程质量最好的一组 | P1：关键 PostgreSQL 约束测试默认跳过；大文件；缺跨层验收 |
| 小Q | 3.9/5 | 只读、权限和追溯设计较好 | P1：5 秒超时未覆盖所有 active 路径；trace 多步非原子 |
| Auth + RBAC | 3.1/5 | JWT/RBAC 有基础，但覆盖和 Owner 边界不统一 | P1：部分路由无 RBAC；禁用账号 token 最长仍有效 24h |
| Approval + Audit | 2.6/5 | 当前系统基础中最危险的部分 | **P0：任意已认证用户可创建并审核自己的审批** |
| Platform 基础设施 | 3.5/5 | primitives 真实且测试密集，但持久化失败语义有缺口 | P1：EventBus retry/outbox 状态可能丢事件；Command audit 未装配 |
| 部署与恢复 | 3.7/5 | 机制较完整，真实灾难恢复演练不足 | P1：恢复能力未本次实测；脚本弱默认；回滚中途失败策略不足 |

平均约 **3.3/5**。分数不是精确科学测量，用于比较模块相对成熟度和修复顺序。

## 系统基础到底好不好

### actual：已经确认的好基础

- 后端 build、vet 通过；118 个包、3131 项测试通过。
- 前端 25 个测试文件、133 项测试通过；lint 0 error/10 warnings；91 页面生产构建通过。
- 领域模块采用较一致的 routes/handler/service/model 结构。
- 有 JWT、RBAC、审批、operation log、EventBus、Command、Scheduler、ToolBridge、readiness、metrics、Sentry、备份和恢复机制。
- Command/ToolBridge 已有审批目标绑定、持久化幂等和失败状态；Scheduler 有 PostgreSQL leader lease 和 retry store。
- 生产 Compose 端口边界、不可变异地备份要求和 rollback contract 验证通过。

### actual：说明基础还不稳的证据

- Owner 审批门存在可直接绕过的 P0。
- RBAC 没有统一覆盖所有受保护路由，多个业务模块把“已登录用户”直接当 Owner。
- Audit middleware 对超过 2KB 的 body 会截断后再交给 handler，可能直接改变业务请求。
- 审计写失败时业务仍成功；客户端可创建具有可信外观的 operation log。
- 审批执行遇到“真实副作用成功但 HTTP 失败”时可能被标记 failed，随后重复执行。
- EventBus 内存 retry 与数据库 outbox 状态可能分叉，重启时无法恢复已标 failed 的事件。
- DemandCase 和 Experiment 的关键不变量主要在 Service 层，数据库和单一路径保护不足。
- 财务金额仍有 `float64`，关键对象多链接时存在静默择新。

因此，正确表述是：

> **凌镜已经有中等偏上的工程骨架和较强测试投入，但安全门与状态一致性还没有达到“可以放心扩建”的基础等级。**

## 修复优先级

### 第一批：立即停止扩建，先处理 P0

1. 审批必须只允许真正 Owner/授权审核人审核，禁止申请人审核自己的请求；数据库和事务同时约束。
2. DemandCase 只允许绑定受控快照和真实独立 run 的证据进入裁决；任何 unknown conflict 必须阻断。
3. Experiment 的 continue 必须要求正的最终利润；多同类链接必须阻断而不是择新；现金金额与币种必须和结算/利润一致。

### 第二批：修系统基础 P1

4. 修 Audit 2KB body 截断、伪造 operation log、审计失败语义和副作用不确定时的 reconcile 状态。
5. 建立统一 Owner/RBAC route policy，禁用账号能快速撤销 access token。
6. 修 EventBus outbox/retry 状态机、enqueue 错误和 handler 锁内执行；把 Command audit 设为生产必需。
7. 把关键 PostgreSQL 约束测试纳入默认 CI，补跨模块 HTTP/E2E 验收。

### 第三批：降低长期维护成本

8. 拆分过大的 Service/page/bus 文件，统一状态机和错误类型。
9. 财务金额迁移到 decimal/最小货币单位。
10. 完成一次真实备份恢复、回滚和告警送达演练。

## 通过标准

不能以“原有 3131 项测试仍通过”作为修复完成。每个 P0/P1 必须先增加能复现问题的负向或集成测试，再修复；最终至少需要：

- 自审审批、普通用户 Owner 动作、重复副作用全部被拒绝；
- DemandCase/Experiment 错误放行测试转为阻断；
- EventBus 失败后重启仍能恢复且最终状态一致；
- 大请求体经过 audit middleware 后字节不变；
- 关键 PostgreSQL 约束测试进入默认 CI；
- 最新真实备份在隔离库恢复成功并记录耗时。

## 独立报告

- `docs/research/tech-quality-demandcase-2026-07-12.md`
- `docs/research/tech-quality-experiment-2026-07-12.md`
- `docs/research/tech-quality-sourcing1688-2026-07-12.md`
- `docs/research/tech-quality-xiaoq-2026-07-12.md`
- `docs/research/tech-quality-auth-rbac-2026-07-12.md`
- `docs/research/tech-quality-approval-audit-2026-07-12.md`
- `docs/research/tech-quality-platform-infrastructure-2026-07-12.md`
- `docs/research/tech-quality-deploy-recovery-2026-07-12.md`

本轮只新增审计报告，没有修复代码、连接生产、修改经营数据或执行破坏性运维操作。工作区原本已有大量未提交改动，本报告评价的是 2026-07-12 本次读取到的当前文件状态。
