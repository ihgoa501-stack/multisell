# DemandCase 模块技术质量审计

> 日期：2026-07-12
> 范围：`backend-go/internal/domain/demandcase/`、相关迁移、路由注册、前端 `/demand-cases` 及聚焦测试
> 目标：评价模块开发质量，不评价候选市场是否真实成立
> 基线：当前未提交工作区快照；该模块文件已有他人在途改动，本报告未修改业务代码

## 1. 结论

**总评：2.7 / 5，骨架清楚、能运行，但关键不变量没有落到可信的数据约束和单一路径上。**

模块的优点是体量小、领域词汇集中、Owner 行级隔离明确，研究导入使用事务、SHA-256 和幂等检查，42 项两模块聚焦测试中 DemandCase 测试全部通过，race 检测也通过。主要问题不是“没有代码”，而是同一业务事实存在两条强度不同的写入路径：受控研究导入要求快照和哈希，普通证据接口却能绕过快照，只靠调用者填写 `run_id`。裁决随后信任这些弱记录，导致核心状态机可被合法 API 输入错误放行。

这意味着当前基础适合继续收紧，不适合在其上直接增加更多研究 Agent、平台或自动决策。

## 2. 本模块“开发得好 / 不好”的定义

开发得好必须同时满足：

- 所有可改变裁决的证据只能从一个受控入口进入，并可追溯到同案、同 Owner、同 run 的不可变快照；
- 服务层、数据库约束和测试共同保护同一组不变量，而不是只靠 Go 分支；
- `Evaluate` 对未知、冲突、证据等级和独立性采取保守、可解释且可复验的规则；
- JWT 之外还有符合“唯一 Owner”方向的角色权限；所有变更可审计；
- 查询有与访问模式匹配的索引、分页上限和稳定排序；
- 正常、异常、越权、并发、数据库约束和 HTTP 合约都有测试；
- 错误有稳定分类，日志/指标能定位导入、裁决和哈希失败。

开发得不好包括：API 能绕过受控链；不同层规则不一致；关键约束只存在于应用代码；任意已登录用户可执行 Owner 决策；测试只证明当前实现而没有攻击错误放行路径；运维人员无法从稳定错误码和指标定位失败。

## 3. 七轴评分

| 轴 | 分数 | actual 证据与判断 |
|---|---:|---|
| 正确性 | 2.0 | `Evaluate` 只按字符串 run ID 判断独立反证，普通写入口不要求快照；unknown conflict 可不阻断，estimated 可满足所有维度。核心裁决存在错误放行路径。 |
| 可读性 / 复杂度 | 3.8 | 主服务 214 行、模型 118 行，枚举和八维集中，流程容易跟踪；但裁决规则散在 `AddEvidence`、`ImportResearchResult`、`Evaluate`，没有显式策略类型。 |
| 架构边界 | 3.0 | 领域目录、Handler/Service/Model 分层清楚，导入事务内复用 Service；但普通证据写入与受控研究契约并存，破坏单一事实入口。 |
| 安全 / 权限 | 2.5 | 所有路由位于 JWT `protected` 组，查询均按 `owner_id` 隔离；但无 Owner 角色/RBAC，任何已认证用户可创建、导入和裁决自己的“Owner”案件。 |
| 性能 / 数据库 | 2.8 | 列表有 100 上限，常用 case/owner 列有索引；但快照引用无外键，候选自然键无唯一约束，按六字段 `.First` 可在重复候选中产生歧义。 |
| 测试质量 | 3.0 | 服务、研究契约和 Handler 有聚焦测试；本次 `go test` 与 `go test -race` 通过。缺普通接口伪造独立 run、unknown conflict 并存、仅 estimated、跨快照归属、真实 PostgreSQL 约束和页面 E2E。 |
| 可运维性 | 1.9 | 返回标准响应包且事务失败会回滚；但业务错误仅自由文本，无稳定错误码、结构化裁决指标、导入/哈希失败观测或专属运行手册。 |
| **总评** | **2.7** | 七轴等权平均，四舍五入。 |

## 4. 发现（按严重度）

### P0 — 普通证据 API 绕过受控研究契约，可错误生成 `experiment_ready`

- `routes.go:15-16` 暴露普通证据与反证写入口。
- `service.go:35-48` 只要求 `run_id` 非空，不要求 `snapshot_id > 0`，也不验证快照属于同案、同 Owner、同 run。
- `service.go:77-116` 仅以反证 run 字符串没有出现在支持 run 集合中，认定“独立”。
- 与之相对，`research_contract.go:46-109` 才要求来源、时间、原始 payload、哈希、run 类型和幂等。

影响：系统有两条证据事实链，弱链可以满足强链裁决。开发质量上属于核心不变量被公开 API 绕过，不是普通输入校验缺陷。

### P0 — unknown 冲突在已有支持证据时可能不阻断

`service.go:99-101` 只在 conflict 同时满足 `usableEvidence` 时加入 blocker；`service.go:132-134` 又将 unknown 判为不可用。结果是某维已有 quoted support 时，再录入明确 unknown conflict，unknown 本身不会取消支持。

影响：状态机对“证据不足”和“没有证据”处理不一致，可输出过强状态。现有测试覆盖 quoted conflict，未覆盖 unknown conflict 与 support 并存。

### P1 — 数据库没有保护证据溯源和候选唯一性

- `000079_create_demand_research_snapshots.up.sql:17-18` 把 `snapshot_id` 定义为默认 0 的 bigint 和普通索引，没有外键。
- `000078_create_demand_cases.up.sql` 没有 Owner + 地区 + 消费者 + 场景 + 渠道 + locale 的唯一约束。
- `research_contract.go:80-88` 每个新 scout 都创建案件，后续 run 再用六字段 `.First` 查找。

影响：并发或重复导入可制造重复候选，后续证据可能附着到非预期记录；应用校验不能替代持久层完整性。

### P1 — 权限模型只做认证和行级隔离，没有 Owner 角色门

`router.go:592-593,861` 表明路由在 JWT 保护组，但不像 product/order 等模块，没有 `RequirePermission` 子组。Handler 直接把 JWT user ID 当 Owner ID。行级隔离是优点，但未实现“仅 Owner 本人”的系统边界。

### P1 — `estimated` 对八个维度拥有同等放行能力

`usableEvidence` 只排除 unknown/mock/inferred，因此 estimated 与 quoted 等价。代码没有按维度声明最低证据等级。这是可维护性问题：规则隐含在一个宽泛布尔函数中，新增维度时很容易继续错误继承。

### P2 — 错误与观测能力不足

Handler 将大多数领域错误统一映射为 422，自由文本直接返回；Service 的 logger 字段几乎未使用。没有稳定错误码、裁决耗时/结果指标、重复 run 冲突计数或哈希失败日志，生产问题只能依赖请求日志和数据库排查。

### P2 — 前端核心流程缺专属测试

仓库搜索未发现 demand-cases 页面组件测试或浏览器 E2E；现有相关前端测试只是小Q链接。后端通过不能证明页面能正确展示 blocker、来源和跨状态刷新。

## 5. 做得好的工程基础

- `ImportResearchResult` 用数据库事务包住 batch、snapshot、evidence 创建，失败不会留下部分事实。
- 原始 payload 在入口复算 SHA-256；重复 run 的来源或摘要变化会拒绝，具备基本幂等语义。
- `requireOwner`、List、Get 均使用 owner 条件，未发现跨 Owner 直接读取路径。
- 状态和证据枚举同时有 Go 校验与 PostgreSQL CHECK（快照归属除外）。
- 分页对 size 做 100 上限；查询全部使用参数绑定，没有发现 SQL 拼接注入。

## 6. 实际验证

- `actual`：先使用 CodeGraph 读取当前 `service.go`、`research_contract.go`、`model.go`、`routes.go` 及调用关系，再核对 Handler、迁移、router 和路由策略表。
- `automated_verified`：`rtk go test ./internal/domain/demandcase ./internal/domain/experiment` 返回 2 包共 42 项通过。
- `automated_verified`：`go test -race ./internal/domain/demandcase ./internal/domain/experiment` 两包通过。
- `unknown`：未运行真实 PostgreSQL 集成、前端构建、浏览器 E2E、生产流量或数据库负载测试。

## 7. 推荐的最小修复顺序

1. 关闭或降权普通证据写入口；裁决只接受绑定有效 snapshot 的证据，并在数据库增加外键/归属约束。
2. 将“维度状态”建成一个明确纯函数：unknown/conflict 必阻断，按维度声明最低真实性级别，并补 P0 负向测试。
3. 建立候选自然键或显式 case identity，消除 `.First` 歧义和并发重复。
4. 为写入和裁决增加 Owner 权限、稳定错误码、结构化日志和结果指标。
5. 补 PostgreSQL 约束测试和一条页面 E2E；在此之前不扩大 Agent 或平台范围。
