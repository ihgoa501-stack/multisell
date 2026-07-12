# Experiment 模块技术质量审计

> 日期：2026-07-12
> 范围：`backend-go/internal/domain/experiment/`、相关迁移、路由注册、前端 `/experiments` 及聚焦测试
> 目标：评价模块开发质量，不评价真实成交或利润是否发生
> 基线：当前未提交工作区快照；`closure_validation.go`、`owner_closure.go` 等为在途未跟踪文件，本报告按当前可见源码审计

## 1. 结论

**总评：3.0 / 5，后半段校验有技术含量，但状态机、对象引用和数据库约束尚未形成一个可靠整体。**

模块已经具备较好的基础：十阶段模型清楚，创建时清除客户端注入的终局金钱字段，阶段前进要求当前 gate pass；证据、闸门、对象关联分层明确；利润和现金路径会读取结算、订单利润和财务交易，使用事务写 gate 与派生金额；Owner 查询全程带 owner 条件。当前聚焦测试和 race 检测通过。

最严重的技术问题是核心状态机仍能产生自相矛盾结果：`continue` 与 `completed` 不要求最终利润为正；利润/现金 gate 使用“最新一个同类链接”而不是唯一引用；现金不校验金额和币种一致性；订单、签收和售后 gate 仍只验证通用文本证据，而不验证链接对象的确定性状态。数据库也没有 CHECK 约束保护这些枚举和跨字段状态，绕过 Service 的写入可制造非法状态。

## 2. 本模块“开发得好 / 不好”的定义

开发得好必须满足：

- 状态机是显式、单调、可重放的；所有状态转换有前置条件，回退会使后续 gate 失效并留痕；
- 每个高风险 gate 使用对应领域对象的确定性校验，不以通用文本替代订单、售后和财务事实；
- 每种关键对象引用唯一、存在、属于同一 Owner/订单链，数据库与 Service 同时保护；
- 金额使用适合财务的精确类型，币种和对账口径明确；利润与现金的一致性可验证；
- JWT、Owner 角色、行级隔离、审计和批准边界完整；错误不会泄漏内部数据库细节；
- 测试覆盖正向、负向、越权、重复提交、并发、PostgreSQL 约束、页面和 E2E；
- 能从日志、指标和审计记录定位 gate 决策、失败原因与对象链。

开发得不好包括：状态可自由倒退或互相矛盾；多个对象链接时静默选一条；财务用浮点且不核币种；任何已登录用户都拥有 Owner 决策权；HTTP 返回裸错误；测试没有证明最危险的错误状态被拒绝。

## 3. 七轴评分

| 轴 | 分数 | actual 证据与判断 |
|---|---:|---|
| 正确性 | 2.4 | 阶段前进、证据等级、利润/现金基本关联校验较好；但负利润可 continue/completed，多链接静默择新，现金金额/币种不核，订单/履约/售后只靠文本。 |
| 可读性 / 复杂度 | 3.1 | 领域词汇和 canonical gate 集中；但 `service.go` 441 行承担状态机、CRUD、证据、gate 和跨域 SQL，`owner_closure.go` 又形成一套相似但更严格的读取逻辑。 |
| 架构边界 | 3.0 | Handler/Service/Model 分层、只读闭环投影和共享纯谓词方向正确；但跨域表名/字段通过匿名 struct 硬编码，profit/cash gate 未复用唯一链接和闭环读取规则。 |
| 安全 / 权限 | 2.6 | JWT 和 owner 条件完整，返回闭环视图会脱敏 PII；但无 Owner RBAC，所有 mutation 标为 standard，Delete 方法存在但未注册，错误响应不统一。 |
| 性能 / 数据库 | 2.5 | 常见 experiment_id 有索引，查询规模通常小；但列表无 size 上限，latest gate 缺 `(experiment_id, stage, id)` 复合索引，金额用 float64，迁移缺大量 CHECK/唯一约束。 |
| 测试质量 | 3.5 | 服务负向测试、利润/现金校验、Owner 闭环、race 检测均存在并通过；但缺负利润终局、多同类链接 gate、现金一致性、PostgreSQL 迁移约束、HTTP/RBAC 和页面 E2E。 |
| 可运维性 | 2.0 | gate 与派生金额事务写入，路由变更进入全局审计中间件；但 logger 未用于关键决策，无稳定错误码/指标，跨域表契约漂移只能在运行时暴露。 |
| **总评** | **3.0** | 七轴等权平均，四舍五入。 |

## 4. 发现（按严重度）

### P0 — 终局状态允许负利润或零利润 `continue`

`service.go:47-57` 的 `validateCase` 对 continue 只要求 `FinalProfitStatus == final` 和 `CashRecoveryStatus == recovered`；completed 也不要求 `FinalProfitAmount > 0`。`service.go:333-384` 会接受 final 且 missing costs 为空的负利润记录。

影响：状态机能保存“最终亏损 + 继续”的矛盾状态。这是模型正确性缺陷，不能靠 UI 文案修复。当前工作区测试未覆盖该负向条件。

### P0 — 利润和现金 gate 在多个同类链接中静默选择最新记录

`service.go:321-330` 的 `linkedNumericID` 使用 `Order("id DESC").First`。因此同一实验可同时链接多个 settlement、order 或 profit_record；数据库唯一约束仅覆盖三元组 `(experiment_id, object_type, object_id)`，不限制每类一个。`owner_closure.go:220-238` 已有更保守的 `uniqueLinkedID`，但 gate 路径没有复用。

影响：同一案件的事实主体不再唯一，后插入链接可以改变 gate 读取对象，历史 gate 难以重放和解释。

### P0 — 现金 gate 不验证币种和金额口径

`service.go:386-415` 只要求正金额 revenue、银行/现金账户、日期、同订单和同结算；没有比较交易币种与结算币种，也没有检查金额与结算净额或应收口径。`owner_closure.go:343` 明确把该一致性保持 unknown，但 gate 仍可 pass。

影响：错币种或部分金额记录可以使现金状态进入 recovered，造成财务状态错误。

### P1 — order / fulfillment / aftersales gate 没有领域对象校验

`actualRequired` 要求这三阶段使用 actual，但 `EvaluateGate` 在这些阶段只检查 EvidenceRecord 的 Owner 核验、来源和时间，不读取链接订单的 paid/delivered/cancelled 状态，也不读取售后/争议观察窗口。通用证据系统承担了本应由确定性领域校验承担的责任。

### P1 — 状态机允许无条件后退且不失效后续 gate

`service.go:95-114` 只限制向前跳级；`nextIndex <= currentIndex` 直接允许。回退没有理由、版本、后续 gate 作废或派生金额清理。OwnerSummary 会继续把历史各阶段最新 gate 计入通过数，可能展示与当前阶段不一致的状态。

### P1 — 数据库约束明显弱于 Service 规则

`000076_create_experiment_core.up.sql` 对 stage、status、truth_status、result、object_type、final_decision 等均无 CHECK；金额也无跨字段约束。GORM 模型同样没有 enum check 标签。任何迁移脚本、管理任务或未来新 Service 直接写表，都可绕过状态机。

### P1 — 财务金额使用 `float64`

`ExperimentCase`、利润闭包和现金闭包均用 float64，而 PostgreSQL 是 NUMERIC。扫描、比较和 JSON 往返可能引入二进制浮点误差。当前虽然没有直接等值比较，但未来一致性校验若建立在 float64 上会脆弱，应使用 decimal 或最小货币单位。

### P1 — 权限仅为认证 + 行级隔离

`router.go:592-593,860` 只把模块挂入 JWT protected 组，没有 `RequirePermission`。Handler 将 user ID 直接作为 owner；所有已认证账号都能创建实验、核验证据和评估 gate。路由策略表把所有 mutation 标为 standard，未表达 Owner 专属决策权限。

### P2 — Service 过度集中并重复跨域规则

`service.go` 同时负责 CRUD、状态转换、证据、对象图、gate、结算、利润、现金与摘要；`owner_closure.go` 又重复 settlement/profit/cash 查询，并在链接唯一性上比 gate 更严格。相同概念出现两套规则，已经发生语义漂移。应提取小型只读 validator，不需要大规模重构。

### P2 — 分页、索引和 HTTP 合约不一致

- Experiment List 只对 size `<1` 设默认值，没有 100 上限，单个请求可拉取任意大页面。
- latest gate 查询按 experiment + stage + id 排序，但迁移只有 experiment_id 单列索引。
- Handler 有 Delete 方法，routes 没注册，属于死入口；部分错误用裸 `e.Error()` 返回 500/400，未统一使用 `InternalError`，可能泄漏内部信息且客户端无法稳定分类。

### P2 — 前端没有核心页面测试或 E2E

仓库搜索未发现 experiments 页面组件测试或浏览器 E2E；相关测试主要是展示映射和小Q链接。最关键的阶段推进、证据核验、错链和终局阻断没有浏览器级防回归证据。

## 5. 做得好的工程基础

- Create 主动清除客户端注入的终局利润、现金和决定字段，并强制从 opportunity 开始。
- 向前推进只能逐级，且要求当前阶段最后一个 gate 为 pass。
- 普通录入不能直接声明 actual；Owner 核验要求来源与观察时间。
- profit gate 校验可信结算来源、完整对账、同订单 final 利润记录和 missing costs 为空；cash gate 要求同订单和结算。
- gate 与派生金额在同一事务提交，避免只写一半。
- Owner 闭环读取使用 repeatable-read read-only 事务，并脱敏订单号、不暴露 PII、账户余额和原始结算 payload。
- SQL 全部使用参数绑定，未发现字符串拼接注入。

## 6. 实际验证

- `actual`：先用 CodeGraph 读取当前 `service.go`、`owner_closure.go`、`closure_validation.go`、`model.go`、`routes.go`，再核对 Handler、迁移、router 和路由策略表。
- `automated_verified`：`rtk go test ./internal/domain/demandcase ./internal/domain/experiment` 返回 2 包共 42 项通过。
- `automated_verified`：`go test -race ./internal/domain/demandcase ./internal/domain/experiment` 两包通过。
- `unknown`：未运行真实 PostgreSQL 迁移集成、前端构建、浏览器 E2E、生产数据、财务精度或负载测试。

## 7. 推荐的最小修复顺序

1. 封死三项 P0：正利润才能 continue；每种关键链接唯一；现金金额和币种按明确口径一致。
2. 为 order、fulfillment、aftersales 建立小型确定性 validator，gate 调用它们；文本证据只作为引用和解释。
3. 明确状态回退命令：原因、版本、后续 gate 失效和派生字段清理；禁止通用 Update 任意改状态。
4. 给迁移增加 enum CHECK、复合索引和每类关键链接唯一约束；财务值改用 decimal/最小货币单位。
5. 增加 Owner permission、稳定错误码、结构化 gate 日志/指标、PostgreSQL 约束测试和一条核心 E2E。
