# 安全治理外部标杆：优秀系统怎样定义

> 日期：2026-07-12
> 范围：Auth、RBAC/ABAC、Owner 审批、幂等审批消费、审计完整性
> 适用对象：单 Owner、Go、PostgreSQL 的内部经营系统
> 方法：只采用官方文档、标准、原始论文和官方开源项目；本报告不证明凌镜生产配置或生产行为。

## 结论

**优秀的安全治理，不等于安装 Keycloak、OpenFGA、OPA 或 Cedar。** 对凌镜现阶段，更好的定义是：关键规则在服务端形成统一且默认拒绝的边界；高风险批准由正确身份作出、绑定不可变请求、只能安全消费一次；外部结果不确定时停止自动推进并对账；审计记录的来源不可伪造、写入失败可见且可恢复、事后篡改可检测。

凌镜已经具备刷新 Token 轮换与重放撤销、显式 route catalog、审批目标绑定、一次消费状态机、operation_log 哈希链等不错的局部机制；但还没有达到上述系统级定义。当前最重要的差距是：授权没有覆盖所有路由、审批没有 Owner/禁止自审边界、access token 不能及时失效、高风险审计写失败仍可成功、审计入口可由客户端伪造，以及 HTTP 审计会截断大请求。

## 证据等级

- `quoted`：外部一手来源明确陈述的设计或行为。
- `actual`：本次直接读取当前仓库源码确认的事实。
- `inferred`：根据外部原则和当前代码作出的适配判断。
- `unknown`：未连接生产环境、生产数据库或外部平台，无法确认。

## 1. 外部标杆教会我们的“好”与“不好”

| 能力 | 好的定义 | 不好的定义 | 一手证据 |
|---|---|---|---|
| 默认拒绝 | 没有明确 permit 就拒绝；显式 forbid 优先于 permit；策略异常不扩大权限 | 新路由只要登录就能访问；授权靠开发者逐个记住加中间件 | `quoted`：Cedar 的决策算法是 forbid 优先、其次 permit、否则 deny；[Cedar Authorization](https://docs.cedarpolicy.com/auth/authorization.html)。OPA 也提供显式 `default` 用于构建 default-deny；[OPA default keyword](https://www.openpolicyagent.org/docs/policy-reference/keywords/default)。 |
| 权限模型 | 主体、动作、资源和必要上下文均进入服务端授权；模型和业务对象保持一致 | 只凭 JWT 中一个 role 字段；显示角色与实际权限来自两套事实源 | `quoted`：Zanzibar 用统一关系模型表达对象 ACL，并强调授权变化与对象访问的因果一致性；[Google 原始论文入口](https://research.google/pubs/zanzibar-googles-consistent-global-authorization-system/)。OpenFGA 官方建模指南覆盖关系、条件和多重限制；[OpenFGA Modeling](https://openfga.dev/docs/modeling)。 |
| Token 生命周期 | access token 短寿命；refresh token 可撤销并轮换；禁用账号/泄露凭据后，高风险请求迅速失效 | “退出登录”只删前端 Token；长寿命 access JWT 在禁用后仍有效 | `quoted`：RFC 7009 定义撤销端点并要求客户端能处理 Token 随时失效；[RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009)。Keycloak 官方文档也明确“sign out all sessions”不必然撤销所有已签发 access token，仍需短有效期或撤销策略；[Keycloak Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)。 |
| 职责分离 | 发起、批准、执行是可区分的身份与记录；禁止申请人自审；高风险可禁止管理员绕过 | 同一登录用户创建、批准并消费自己的批准；管理员默认可以绕过 | `quoted`：GitHub protected environments 可禁止 self-review，且批准前 job 无法取得环境 secrets；也可禁止管理员绕过；[GitHub deployments and environments](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments)。 |
| 批准有效性 | 批准绑定动作、对象、请求内容摘要、环境、有效期和批准人；请求改变后批准失效 | 一个宽泛的 `approved` 状态长期覆盖不同参数或不同目标 | `quoted`：GitHub 分支保护可在新提交后撤销旧批准，并可要求最新 push 由另一人批准；[GitHub protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)。 |
| 幂等执行 | 同一逻辑操作使用同一 key；服务端绑定原始参数并回放结果；并发只允许一个执行者 | 只做本地“已消费”标记，却不约束真实外部副作用；换 key 盲重试 | `quoted`：Stripe 保存同一 idempotency key 的首次结果，复用时校验参数，连接错误可用同一 key 安全重试；[Stripe idempotent requests](https://docs.stripe.com/api/idempotent_requests)。 |
| 不确定副作用 | timeout/5xx 不写成失败或成功；保持 `reconcile_required`，用外部查询、webhook或人工凭证对账 | 超时即重试新请求，或把“无错误响应”当外部事实成功 | `quoted`：Stripe 明确把 500 视为 indeterminate，原操作可能已有副作用，需 reconciliation；[Stripe advanced error handling](https://docs.stripe.com/error-low-level)。 |
| 审计来源 | 主体、时间、动作、对象、结果和关联 ID 由服务端生成；业务变更与 audit/outbox 具备原子或可恢复关系 | 客户端可提交 operator/result；业务成功而审计写失败仅打普通日志 | `inferred`：这是职责边界与故障语义的直接要求；下述 CloudTrail 只解决“交付后是否被改”，不解决“最初记录是否真实”。 |
| 审计防篡改 | append-only、最小 DB 权限、链式摘要、独立签名/外部锚点、定期验证和告警 | 仅在同一数据库做 hash，然后声称“不可抵赖” | `quoted`：CloudTrail 对每个日志做 SHA-256，周期 digest 用 RSA 签名并串联前一 digest，可检测修改或删除；但“启用完整性验证并不等于已经执行验证”；[CloudTrail integrity validation](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-log-file-validation-intro.html)。 |

## 2. 对凌镜当前实现的逐项对照

### 2.1 Auth

`actual`：`internal/auth` 已实现 access/refresh 类型区分、持久化 refresh session、refresh 轮换、旧 token 重放时撤销整个 family、单会话 family 注销和全部 refresh session 注销。JWT 中还校验 HS256 与 key id；禁用用户不能刷新。

`actual`：`Auth` 中间件只验证签名、有效期、token type 和 user id，不查询账号实时状态或 session 版本。`RevokeAllRefreshSessions` 的源码注释也明确 access JWT 会继续有效到过期。

判定：**局部优秀，失效边界不足。** 推荐 access token 5–15 分钟；对审批、发布、权限变更、财务等高风险路由额外读取 `user.status` 与 `token_not_before/session_version`。无需现在迁移到 Keycloak；单 Owner 系统自行维护这套小模型更简单，也更容易审计。

### 2.2 RBAC / route policy

`actual`：`RequirePermission` 查询数据库 RBAC，查询错误时拒绝，是 fail closed；`ApprovalRequired` 对所有 mutation 要求 route catalog 明确分类，未分类时拒绝，也是好的默认拒绝机制。

`actual`：RBAC 并非 protected group 的统一默认边界。`router.go` 只有部分路由组套用了 `RequirePermission`；approval、operation-log、settings、supplier、purchase、aftersales 等大量模块直接挂在只要求认证的 group。新增读取路由尤其不会被 route mutation catalog 覆盖。

判定：**写操作分类基础好，但系统授权仍是 allow-by-registration。** 推荐建立一个服务端 route authorization manifest，CI 枚举所有 `/api/v1` 路由，要求每条路由声明 `public / authenticated / permission / owner`；没有声明就启动失败或 CI 失败。Owner 专属动作采用显式 `owner.approve`、`audit.read` 等权限，并在领域 service 再校验，不能只靠 handler。

### 2.3 Owner 与审批

`actual`：审批执行中间件已经校验状态、有效期、request type 到 action type 映射、target type/id、requester、idempotency key；审批消费有数据库级的原子状态机。这比“一个 approved 布尔值”可靠得多。

`actual`：`approval.RegisterRoutes(protected, approvalSvc)` 未要求 Owner 或审批权限。`Service.Review` 验证 pending/expiry/concurrency，却没有验证 reviewer 是否 Owner，也没有禁止 `ReviewerUserID == RequesterUserID`。审批结构化审计错误被忽略。

判定：**消费机制较好，批准主体边界不合格。** 对单 Owner 系统，推荐最小规则：

1. 普通 Agent/流程只能申请；只有 Owner 权限可批准或拒绝。
2. 服务层禁止申请人与审核人为同一身份；若确实只有 Owner 一个人，Owner 直接发起的高风险动作应建模为“Owner 明示确认执行”，而不是伪造第二个人的双人审批。该确认仍需重新认证或短时 step-up、内容摘要和有效期。
3. 批准绑定规范化请求 SHA-256、action、target、环境、幂等 key scope；任何字段变化使旧批准失效。
4. 执行结果分为 `not_started / executing / succeeded / failed_before_side_effect / reconcile_required`；不得用一个 failed 覆盖不确定外部结果。

### 2.4 幂等批准消费与外部写

`actual`：凌镜的 `ApprovalRequired` 先授权再原子消费，并把 idempotency key 放入上下文；这是可靠的本地并发防重基础。

`inferred`：本地 claim 只能证明“凌镜尝试过一次”，不能证明平台侧只发生一次。优秀实现必须让同一 key 贯穿 `approval_execution → tool_execution → provider request`，并保存外部 reference 与可查询状态。若 provider 不支持幂等，就必须查询外部结果；超时进入 `reconcile_required`，禁止自动换 key 再发。

### 2.5 Audit

`actual`：operation_log 已有 append-only 与哈希链机制，可用于检测数据库内已有记录的后续修改；HTTP audit 会等待最多 3 秒写入。

`actual`：写入失败只记 application error，不改变业务响应；approval review 直接忽略审计错误。`/operation-log` 直接注册在 authenticated group，现有专项审计确认客户端 Create 可提交主体字段。`Audit` 用 `io.LimitReader(..., 2048)` 后只把前 2KB 放回 request body，实际会改变超过 2KB 的业务请求。

判定：**有“篡改检测”雏形，不是可信审计系统。** 推荐：删除外部日志创建入口；身份与结果由服务器生成；修复 body 读取使完整 body 原样交给 handler、仅日志副本截断；高风险业务变更与 audit/outbox 同一 Postgres 事务；普通审计写失败进入 durable queue 并拉低 readiness/触发告警；每日生成链头摘要，签名后写到应用数据库权限之外的对象存储或其他外部锚点，并定期自动验证。

## 3. 哪些标杆值得借鉴，哪些不应照搬

### 3.1 五个真正优秀的标杆及适用条件

1. **AWS Cedar：授权决策语义标杆。** 优秀之处是 default-deny、forbid-overrides-permit、决策诊断和 schema 校验。适用于主体、动作、资源和条件逐渐复杂，且团队愿意把策略当代码测试的系统。凌镜现在适合复用它的决策语义与测试方法；只有当内置 Go policy manifest 难以维护时，才值得引入 Cedar runtime。
2. **Google Zanzibar / OpenFGA：对象关系授权标杆。** 优秀之处是统一关系模型、对象级权限与一致性设计。适用于多组织、多层级资源、共享/委托关系非常多的产品。凌镜当前单 Owner 不满足引入独立 ReBAC 服务的条件，只应借鉴“唯一授权事实源”和清晰关系词汇。
3. **Keycloak：标准身份与会话治理标杆。** 优秀之处是集中式 OIDC/OAuth 身份、会话、注销、撤销和管理能力。适用于多个应用、外部身份源、企业 SSO、MFA 或专门 IAM 运维已成为真实需求的场景。凌镜当前不应引入整套服务，但应达到短 access token、refresh rotation、实时禁用等基本语义。
4. **GitHub protected environments / branches：人类审批门标杆。** 优秀之处是禁止自审、批准前不释放 secrets、可禁止管理员 bypass、内容变化后旧批准失效。适用于部署、发布、资金或其他高风险不可逆动作。它与凌镜最贴近，应直接移植语义；但单 Owner 情况应用“Agent 申请 + Owner 明示确认”，不伪造双人审批。
5. **Stripe + AWS CloudTrail：副作用可靠性与审计完整性组合标杆。** Stripe 适用于任何有网络超时和外部写入的支付/发布/订单集成，核心是参数绑定幂等与 indeterminate 后对账；CloudTrail 适用于需要安全调查和篡改检测的审计，核心是独立签名、digest chain 和持续验证。凌镜应采用缩小版：provider key 贯通、`reconcile_required`、Postgres audit/outbox、每日外部签名锚点。

这五个标杆不是按“功能最多”选出的，而是分别代表授权语义、对象关系、身份会话、审批职责、副作用与审计五个成熟边界。对当前凌镜，优先级是 GitHub 审批语义与 Stripe 不确定副作用，其次 Cedar 默认拒绝，再其次 CloudTrail 外部锚定；OpenFGA 和 Keycloak 暂时只学原则。

| 标杆 | 现在应借鉴 | 现在不应照搬 |
|---|---|---|
| Cedar / OPA | 默认拒绝、显式 deny、策略测试、主体-动作-资源-上下文输入 | 暂不引入独立 policy runtime；当前权限规模不值得增加部署、缓存和策略同步故障面 |
| Zanzibar / OpenFGA | 统一资源关系词汇；避免 JWT role 和数据库权限双事实源 | 不建设独立 ReBAC 服务、全球一致性 token 或图关系基础设施；凌镜没有多租户、共享对象和世界级规模需求 |
| Keycloak | 短 access token、refresh rotation/revocation、账号禁用后的快速失效语义 | 不为单 Owner 当前阶段引入完整 IAM 服务；会增加运维、升级、备份和故障排查成本 |
| GitHub protected environment | 禁止自审、批准后才释放执行能力、请求改变即批准失效、可禁止管理员 bypass | 不机械要求“两个人审批”；单人系统应区分 Agent 申请与 Owner 明示确认，而不是制造虚假的第二审批人 |
| Stripe | 幂等 key 绑定参数、结果回放、timeout/500 为不确定、对账后定终态 | 不宣称 exactly-once；若渠道不接收同一 idempotency key，只能做到本地 at-most-once 尝试加外部对账 |
| CloudTrail | hash 链、独立签名、外部存储、持续验证 | 不自建 CloudTrail 级平台；Postgres append-only + 每日外部签名锚点已足够当前风险 |

## 4. 适合凌镜的最小优秀基线

以下全部满足，才可以称安全治理基础“好”；其中任何高风险项缺失，都不能靠测试数量抵消：

1. 100% API 路由有机器可检查的授权声明；未声明默认拒绝。
2. RBAC 数据库是唯一授权事实源；Owner 能力显式、最小且可审计。
3. access token 短寿命；禁用、泄露和权限变更能在高风险路径立即生效。
4. Agent 申请与 Owner 决定分离；禁止申请者自审；单 Owner 自己发起时使用明示确认而非伪双人审批。
5. 批准绑定不可变请求摘要、动作、目标、环境、有效期和执行幂等范围。
6. 本地消费、工具执行与 provider key 一路关联；不确定副作用进入 `reconcile_required`。
7. 高风险业务写与 audit/outbox 原子落库；审计故障不能静默。
8. 客户端不能创建或伪造权威审计主体；敏感字段白名单记录。
9. 日志 append-only、可验证，并定期把链头锚定到数据库权限之外。
10. 有权限矩阵、越权、自审、并发消费、崩溃恢复、超时对账、审计故障和篡改验证测试。

## 5. 推荐实施顺序（不扩大系统）

1. **P0：关闭权限与审批绕过。** approval read/create/review 分权；review 仅 Owner；service 禁止自审；所有路由进入可检查的 authorization manifest。
2. **P0：修复审计改变请求体。** 完整恢复 request body，只对日志副本做截断和脱敏。
3. **P1：补即时失效。** 缩短 access token；高风险路由检查账号状态和 session/token version。
4. **P1：让高风险审计可恢复。** 业务事务内写 audit/outbox；禁止外部创建权威 operation_log。
5. **P1：统一不确定副作用。** provider key 透传、外部 reference、`reconcile_required` 与对账命令。
6. **P2：外部锚定。** 每日签名 operation_log 链头并自动验证告警。

## 6. 事实边界

- `actual`：本次读取了当前 Auth、RBAC、Approval、Audit 和 router 接线源码；未修改业务代码。
- `quoted`：外部行为仅代表各官方系统或标准的公开契约，不代表凌镜已实现。
- `inferred`：推荐方案按当前“单 Owner、自用、Go/Postgres”边界裁剪；不是要求平台化。
- `unknown`：生产 access token 有效期、生产迁移状态、数据库角色隔离、外部 provider 的幂等支持、生产审计校验与恢复演练均未验证。
