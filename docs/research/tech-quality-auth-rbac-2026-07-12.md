# Auth + RBAC 技术质量审计

> 审计日期：2026-07-12
> 范围：`backend-go/internal/auth/`、`backend-go/internal/rbac/`、JWT/Auth/RBAC 中间件、相关迁移与前端登录接线
> 不在范围：经营结果、生产账号真实性、未实际连接的生产环境
> 证据等级：源码为 `implemented`；本次聚焦测试为 `automated_verified`；生产行为为 `unknown`

## 结论

**总评：22/35，基础可用，但授权边界尚不够扎实，不能称为“安全基础已经做好”。**

认证本身已有密码哈希、短/长 Token 分离、刷新 Token 轮换、重放后撤销整个会话族、密钥轮换、发布模式弱密钥校验和限流，质量明显高于简单 JWT 样板。主要问题不在“能不能登录”，而在“登录后到底能做什么”：RBAC 只包住部分路由，大量模块仅要求任意有效 JWT；Owner 专属边界没有形成统一、可审计的授权策略。禁用账号的既有 access token 最长仍可继续使用 24 小时。

## 好与不好如何定义

好：身份不可伪造；会话可撤销、可轮换、可发现重放；账号禁用及时生效；所有敏感路由默认拒绝并明确声明权限；Owner 专属操作无法被 operator/viewer 执行；权限变更只有一个事实源；测试覆盖攻击与失败路径；运行时能观察登录失败、刷新重放和权限拒绝。

不好：JWT 正确但授权覆盖靠人工记忆；新增路由默认只需登录；角色同时存在于 JWT 字段和 RBAC 表；禁用用户继续持有长时间有效权限；Owner 与普通账号没有确定性隔离；测试只证明服务函数，不证明整套路由矩阵。

## 七轴评分

| 轴 | 分数 | 判断 |
|---|---:|---|
| 正确性 | 3/5 | 登录、刷新、轮换、注销主路径完整；失效账号 access token 未即时失效，且遗留前端 auth store 契约错误。 |
| 可读性/复杂度 | 4/5 | auth/rbac 服务较小、命名清楚；JWT 校验在 service 与 middleware 重复，身份类型转换也多处重复。 |
| 架构边界 | 2/5 | 认证与 RBAC 模块分开是对的，但路由授权覆盖不统一；legacy `user.role` 与 RBAC 表形成双事实源。 |
| 安全 | 2/5 | 刷新重放、发布配置校验较好；Owner 隔离、全路由最小权限、access token 撤销不足。 |
| 性能/数据库 | 4/5 | 权限单请求缓存、查询有明确 join；刷新会话有 family/user/expiry 索引。未见跨请求权限缓存陈旧问题。 |
| 测试质量 | 4/5 | 聚焦包含大量正常/异常/并发行为测试，race 通过；缺少完整路由权限矩阵和 Owner/operator 端到端越权测试。 |
| 可运维性 | 3/5 | 支持密钥轮换、发布配置 fail-fast；缺少会话清理任务、认证安全指标和集中式撤销机制。 |

## 发现（P0-P3）

### P1 — RBAC 没有覆盖整个受保护 API，授权默认不是最小权限

`router.go` 只对 RBAC 管理、product、inventory、listing、shipping、order、settlement、finance、report 等少数组显式加 `RequirePermission`；approval、operation-log、settings、supplier、purchase、aftersales 等大量模块直接挂在仅有 `Auth + ApprovalRequired` 的 `protected` 组。`ApprovalRequired` 对普通写操作直接放行，它不是权限中间件。

影响：任意已登录账号能访问哪些模块取决于每个注册点是否记得再套一层权限，而不是统一默认拒绝。后续新增模块很容易“有 JWT 即有权”。这会使 RBAC 看似存在，但不是系统级安全边界。

验证方法：生成 Owner、operator、viewer 三种账号，对所有 `/api/v1` 路由建立方法 × 权限矩阵；未声明权限的敏感路由应测试失败。建议把路由目录本身扩展为权限必填字段并在启动时拒绝缺失声明。

### P1 — 禁用账号的 access token 最长仍可用 24 小时

Auth 中间件只校验 JWT 签名、过期、类型和 `user_id`，不会查询用户当前状态或会话版本。`LogoutAll` 的注释也明确 access JWT 要等短期过期；当前默认 `expiry_hours: 24`。

影响：账号禁用、密码重置、发现凭据泄露后，既有 access token 仍可继续调用接口。对于审批、发布、财务等高风险操作，24 小时不是“短期”。

建议：access token 缩短到 5–15 分钟，并增加 `session_version`/`token_not_before` 或高风险请求的实时账号状态检查；明确密钥泄露时的全局撤销流程。

### P1 — Owner 专属边界没有统一实现

当前身份模型只有 `admin/operator/user` legacy role 和 RBAC role；JWT 的 `Role` 不参与 `RequirePermission`，项目也没有统一的 `RequireOwner`。Owner 专属能力依赖各领域自行检查 user id 或权限。结合上一项路由覆盖缺口，不能证明 operator 无法触达 Owner 应独占的审批、审计或治理接口。

这不是多租户要求；即使系统只供 Owner 使用，只要保留 operator/viewer 账号或 Agent 代操作，就必须确定谁能代表 Owner 作最终决定。

### P2 — `user.role` 与 RBAC 表是两个权限事实源

注册时写 `user.role`，再尝试映射到 `user_role`；映射表/角色不存在时 `assignDefaultRBACRole` 会静默成功。access JWT 也携带 legacy role，但服务端权限检查读取 RBAC 表。

影响：界面、日志或遗留代码可能相信 JWT role，实际授权却来自另一套表；迁移缺失时可产生“显示 operator、实际无权限”的账号。建议明确 RBAC 为唯一授权事实源，legacy role 只作展示兼容并逐步移除。

### P2 — 授权服务允许删除基础角色/权限，缺少系统对象保护

`DeleteRole` 和 `DeletePermission` 会事务删除关联后再删实体，没有保护 `admin`、`rbac.manage` 等系统关键对象，也没有检查是否删除最后一个管理者。虽路由要求 `rbac.manage`，误操作仍可能把 Owner 锁在系统外。

建议：系统角色/权限不可删除；权限变更采用受控审批；保证至少一个活跃 Owner/admin 持有管理权限。

### P3 — 存在未使用且契约错误的前端 auth store

`frontend-next/src/stores/auth-store.ts` 调用 `/auth/login`（缺 `/v1`），并读取 `{token}`，而后端返回 `{access_token, refresh_token}`。当前登录页面另有正确实现，因此这是遗留死路径而非当前登录 P0，但会误导后续开发且没有保存 refresh token。

## 已验证事实

- `actual`：2026-07-12 运行 `go test -race ./internal/auth ./internal/rbac ./internal/httpx/middleware ./internal/domain/approval ./internal/domain/operationlog`，5 个包共 213 项测试通过。
- `implemented`：bcrypt、JWT HS256 算法锁定、`kid` 密钥轮换、刷新会话持久化、原子轮换与重放撤销、登录/注册/刷新 IP 限流、发布模式弱密钥和公开注册拒绝。
- `unknown`：生产是否使用强密钥、实际 access token 时长、是否存在非 Owner 有效账号、生产反向代理后的真实客户端 IP、生产会话撤销演练。

## 推荐修复顺序

1. 建立全路由权限目录并启动时 fail closed，先封住未声明授权的敏感路由。
2. 为 Owner 最终决定建立统一 Owner 权限/中间件，并做 Owner/operator/viewer 越权矩阵测试。
3. 缩短 access token 并实现高风险请求即时撤销检查。
4. 保护系统角色、权限与最后一个 Owner；统一 RBAC 为权限事实源。
5. 删除或修正遗留 auth store，补认证安全指标和过期刷新会话清理。
