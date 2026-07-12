# Approval + Audit 技术质量审计

> 审计日期：2026-07-12
> 范围：审批领域、审批执行消费、中间件高风险门、operation_log、HTTP Audit 中间件及相关迁移
> 不在范围：经营结果、真实外部平台副作用、生产数据库/部署
> 证据等级：源码为 `implemented`；本次聚焦测试为 `automated_verified`；生产行为为 `unknown`

## 结论

**总评：18/35。审批执行的“绑定 + 一次消费”基础较强，但 Owner 审批身份边界存在 P0，审计目前是可校验的追加日志，不是完整可信审计系统。**

系统已经实现高风险路由默认分类、审批与动作/目标绑定、幂等键、数据库唯一约束、执行状态机、成功终态不可回退、operation_log 哈希链和 append-only 触发器。这些是好的底座。但任何已登录用户都能创建并审核审批，且没有禁止申请人自审；因此技术上满足“approved”不等于 Owner 实际批准。审计写入失败又不会阻止业务成功，客户端还能直接创建任意内容的 operation_log，哈希链只能证明记录插入后没改，不能证明记录最初真实。

## 好与不好如何定义

好：只有 Owner 可审批；申请人与审核人边界明确；审批绑定动作、对象、内容快照和有效期；每个批准最多消费一次；并发与崩溃可恢复；副作用和执行状态一致；所有写操作审计 fail closed 或进入可恢复队列；审计主体不可伪造、敏感值不泄漏、日志可验证且有告警。

不好：任意登录用户可自批；批准只绑定一个宽泛类型；业务成功而审计丢失；幂等状态与真实副作用分叉；用户可伪造“系统日志”；哈希链被误当成真实性证明；大请求因审计中间件而改变。

## 七轴评分

| 轴 | 分数 | 判断 |
|---|---:|---|
| 正确性 | 2/5 | 审批消费状态机完整，但审核主体不受限；HTTP 审计读取 2KB 后把截断 body 交给下游。 |
| 可读性/复杂度 | 3/5 | 分层和命名总体清楚；审批绑定逻辑同时存在 middleware 与 domain service，重复规则增加漂移风险。 |
| 架构边界 | 3/5 | route catalog、approval execution、operationlog 职责基本清楚；审批路由权限和审计写入信任边界错误。 |
| 安全 | 1/5 | P0 自审批使 Owner gate 可被绕过；审计入口允许伪造主体与内容，写失败 fail open。 |
| 性能/数据库 | 3/5 | 唯一约束、索引、事务、advisory lock 正确；全局哈希链每次插入串行化，日志增长后吞吐与校验成本上升。 |
| 测试质量 | 4/5 | 并发消费、重试、状态机、红action与哈希链已有聚焦测试并通过 race；缺自审批、审计 DB 失败、超大 body、伪造日志等关键测试。 |
| 可运维性 | 2/5 | 有写入指标和完整性 gauge；没有审计失败阻断/补偿、外部锚点、定期校验与告警闭环证据。 |

## 发现（P0-P3）

### P0 — 任意已认证用户都能创建并审核自己的审批

`approval.RegisterRoutes(protected, approvalSvc)` 没有 `RequirePermission` 或 Owner 中间件。handler 从 JWT 强制填写 requester/reviewer，这是防冒名的优点，但 service/handler 没有要求 reviewer 是 Owner，也没有禁止 `ReviewerUserID == RequesterUserID`。

影响：operator 只要能登录，就能创建一个覆盖高风险 action/target 的请求，再由同一账号 approve，随后把该审批交给 `ApprovalRequired` 消费。系统严格验证了一个并非 Owner 决定的“approved”记录，Owner-first 安全边界实质失效。

验证方法：用 operator token 依次 POST `/approval`、PUT `/approval/:id/review`，再调用匹配的高风险路由；当前路由接线预期可通过。修复必须让创建、查看、审核分权，其中审核仅 Owner；服务层也要重复执行不可绕过的 reviewer policy，并禁止自审（除非 Owner 明确政策允许）。

### P1 — 审计写入失败时业务请求仍返回成功

HTTP Audit 在 handler 完成后以 3 秒 deadline 写 operation_log；失败只写应用 logger。approval `Review` 的结构化审计也显式忽略 `LogStructured` 错误。因而数据库触发器、超时、磁盘或连接故障时，业务变更已经提交且客户端仍收到成功。

影响：治理契约要求的“mutation logging”可能出现不可恢复缺口。对于审批、权限、发布、钱和凭据类写操作，这是安全与取证风险。

建议：高风险变更把业务状态与 audit/outbox 放在同一数据库事务；普通 HTTP 审计至少进入 durable outbox，失败触发 readiness/告警。明确哪些操作 fail closed，不能统一静默放行。

### P1 — `/operation-log` 允许任意已认证用户直接伪造审计记录

operationlog 路由直接挂在 `protected`，包含 POST Create；handler 接受客户端提供的完整 `OperationLog`，包括 operator、user_id、result、trigger_type、approval_id 等。数据库哈希链会忠实地给伪造记录签入链中。

影响：append-only 与 SHA-256 只证明“插入后没被普通 UPDATE/DELETE 改”，不证明“插入内容来自可信事件”。伪造行会通过完整性校验，污染取证与 Owner 判断。

建议：删除外部 Create 路由；所有审计主体字段由服务器上下文生成；读取要求 `audit.read`/Owner；内部写入使用受限数据库角色。若需更强防篡改，定期把链头锚定到数据库外的不可变存储。

### P1 — 审计中间件会截断下游请求体

Audit 使用 `io.LimitReader(c.Request.Body, 2048)` 读取后，只把已读取的 2KB 放回 `c.Request.Body`。任何被审计且合法超过 2KB 的 JSON，handler 接收到的是截断内容，而非原请求。

影响：审计横切逻辑改变业务语义，可能造成解析失败；更危险的是某些流式/部分解析 handler 可能处理不完整输入。注释声称只是日志截断，但实际截断了请求。

建议：由全局 max-body middleware 先限制总大小；Audit 读取完整但受总上限保护的 body，日志仅截取副本，或使用 tee/spool 保持下游字节完全一致；补 >2KB 回归测试。

### P1 — “副作用成功、HTTP 返回失败”时审批可被再次执行

中间件在 handler 前把 execution 置为 `processing`，HTTP 状态 >=400 就置 `failed`；failed 状态允许同一 key 重试。如果 handler 已提交数据库或外部副作用，之后才返回 500，审批会被重新开放。反向情况是业务成功但 `CompleteExecution` 持久化失败，记录也可能滞留 processing。

影响：approval execution 的 exactly-once 仅覆盖本地 claim，不自动覆盖真实副作用。对于未实现自身幂等/结果回放的 handler，重试可能重复发布、扣减或更新。

建议：业务 mutation 与 CompleteExecution 同事务；外部调用使用同一个 provider idempotency key 和 durable result；为 processing 增加租约/人工 reconcile 状态，不能仅按 HTTP status 推断副作用事实。

### P2 — 无约束审批可覆盖任意对象

`approvalBindingMatches` 在 ProductID、TargetID、EntityID 全为 0 时返回 true；CreateApproval 只要求 `product_id` 的 binding required，但 Go validator 对数字 required 会拒绝 0，因此 HTTP 主路径通常有 product 约束，内部 service 调用仍可创建无约束审批。中间件也允许空 `TargetType`。

影响：内部调用或未来 handler 变化可能产生覆盖整个 action type 的 bearer approval。建议所有 high-risk approval 强制 target type、target id、内容摘要和短有效期，service 层而非仅 handler 校验。

### P2 — 哈希链有完整性价值，但不是独立防篡改证明

PostgreSQL trigger 禁止普通 UPDATE/DELETE，并用 advisory lock 串行接续 SHA-256；实现本身合理。但拥有函数/触发器修改权或高权限数据库账号的人可重建整条链。当前也未见链头外部锚定、定时 VerifyIntegrity 或告警接线证据。

应把它准确描述为“数据库内篡改可检测”，不能描述为不可抵赖审计。生产需要最小权限 DB 角色、迁移权限隔离、定期验证和外部链头锚定。

### P2 — 审计敏感信息清理对非 JSON 内容 fail open

`RedactSensitive` 只处理合法 JSON；非 JSON 原样返回。HTTP middleware 对 JSON body 会清理，但 service 的 `LogStructured` 接受自由文本，review note 也直接记录。Token、密码或密钥出现在非 JSON 文本时不会被遮盖。

建议：结构化字段优先；对自由文本增加保守模式（明确字段白名单或通用 secret pattern），并限制审计读取权限与保留期。

### P3 — 审批规则存在重复实现

request-type/action 映射通过 domain helper共享了一部分，但 middleware 仍再次执行状态、过期、类型、target 和 requester 检查，随后 service 又 Authorize。双重校验是纵深防御，但当前是两套业务判断而非单一 policy 结果，未来字段扩展容易一处更新、一处遗漏。

建议：middleware 只负责提取可信请求身份，domain policy 返回结构化授权结果；保留独立二次校验时用同一纯函数和契约测试证明一致。

## 已验证事实

- `actual`：2026-07-12 运行 `go test -race ./internal/auth ./internal/rbac ./internal/httpx/middleware ./internal/domain/approval ./internal/domain/operationlog`，5 个包共 213 项测试通过。
- `implemented`：mutation route 未分类 fail closed；high-risk 要求审批 ID 与 8–255 字符幂等键；审批绑定状态、期限、action type、target；approval_execution 主键/唯一键与数据库状态机；operation_log append-only 哈希链和完整性校验。
- `unknown`：生产数据库是否已迁移至 000100、应用 DB 用户是否能修改 trigger/function、真实外部适配器是否透传同一幂等键、审计完整性是否定时验证和告警、生产故障恢复演练。

## 推荐修复顺序

1. 立即把 review 权限收紧到 Owner，并在 service 层禁止非 Owner/自审批；补 operator 越权集成测试。
2. 移除客户端 operation-log Create，所有主体字段由服务器生成；读取限制到 Owner/audit 权限。
3. 修复 2KB body 截断；高风险业务与审计/outbox 同事务。
4. 让审批 execution 与真实副作用共享事务或 provider 幂等结果，增加 reconcile/lease 机制。
5. 强制所有高风险审批绑定对象、内容哈希和期限；接入定时完整性校验、告警和外部链头锚定。
