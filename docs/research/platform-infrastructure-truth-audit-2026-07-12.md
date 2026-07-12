# 凌镜底层基础设施事实审计

> 审计日期：2026-07-12
> 范围：活跃 Go 后端的 Platform Kernel 与工程运行基础设施
> 状态：进行中的海洋目标基线；代码或验证结果变化后追加新证据，不用“文件存在”代替完成

## 1. 目标与边界

本目标只完善底层系统：认证、权限、审批、审计、事件、命令、调度、ToolBridge、配置、密钥、迁移、数据库生命周期、可观测性、故障恢复、测试、CI 和运维手册。

不建设商品、市场、1688、页面、外部 SaaS、多租户、更多 Agent、更多平台连接器或展示性功能。底层机制不得包含具体经营决策。

“海洋目标”表示上述底层范围最终必须形成完整、可验证、可恢复的系统；执行方式是一项一项完成，每项同时补齐正常路径、错误路径、边界、测试和文档。

## 2. 证据等级

- `implemented`：当前代码或配置存在。
- `automated_verified`：相关自动测试在本次注明的环境通过。
- `manually_verified`：按运行手册完成真实操作验证。
- `unknown`：尚无足够证据。
- `gap`：已找到与当前契约冲突或缺少必要实现的证据。

## 3. 当前基础设施矩阵

| 能力 | 当前证据 | 当前裁决 | 完成还需要什么 |
|---|---|---|---|
| Auth / JWT | `internal/auth/`、JWT 中间件、持久化 refresh session、当前会话家族退出与全会话退出接口及测试存在 | `implemented` / `automated_verified`：HS256、正整数身份、一次性 refresh 轮换、重放家族撤销及 Owner 绑定撤销已验证 | 密钥轮换、生产迁移与真实客户端验证 |
| RBAC | `internal/rbac/`、服务端权限中间件及测试存在 | `implemented` / `automated_verified`：禁用角色不再授权，用户角色分配纳入高风险目录 | 全路由权限覆盖矩阵、生产角色数据与权限变更演练 |
| Approval | domain approval、HTTP/Command/ToolBridge gate、持久化 execution binding、PostgreSQL 状态机及全写路由策略清单存在 | `implemented` / `automated_verified` / 隔离 PostgreSQL `manually_verified`：动作/目标绑定、过期阻止、一次审批一次幂等执行、绑定不可修改、成功终态且记录不可删除；385 条写路由已静态与运行时双向核对 | 生产并发、崩溃点与审计故障演练；直接领域 Service 调用边界仍需随新增代码持续审计 |
| HTTP Audit | mutation 与敏感读取中间件、数据库 append-only trigger、SHA-256 前序哈希链与完整性校验存在 | `implemented` / `automated_verified` / 隔离 PostgreSQL `manually_verified`：UPDATE/DELETE 被拒绝、链重算无断点；同步 HTTP 审计仍非业务事务原子写 | 失败指标、关键业务事务边界、保留/归档、外部不可变 checkpoint；数据库超级管理员仍可绕过 |
| EventBus | worker、schema、outbox、DLQ、幂等与 MutationGuard 存在 | `implemented` / `automated_verified`：outbox 与幂等存储故障 fail-closed；pending outbox 启动恢复保留事件上下文；有界 drain 已验证；事件时间线以单调 outbox ID 确定性排序 | 生产 PostgreSQL 故障注入与真实停机排空证据 |
| Command Dispatcher | 模式、目录、审批、限流、审计回调存在；本轮新增持久化幂等执行记录 | `implemented` / `automated_verified`：原子 claim、并发拒绝、成功结果重放、失败重试、过期租约接管已通过测试 | PostgreSQL 迁移实测、崩溃恢复演练、无幂等键生产动作的准入策略 |
| Scheduler | 调度、重试、持久化 retry store 与 PostgreSQL leader lease 存在 | `implemented` / `automated_verified` / 本地 PostgreSQL `manually_verified`：失败队列跨进程恢复；恢复失败不 running；仅持 advisory lock 实例运行 | 生产双实例接管演练、长期任务状态保留策略 |
| ToolBridge | driver、模式、追踪、类型化执行、持久化幂等与 circuit breaker 存在 | `implemented` / `automated_verified`：生产 mutation 要求审批+本地 claim+外部幂等；有限重试、open/half-open、取消与指标已验证 | 真实外部 provider 幂等/熔断演练 |
| Config / Secrets | Viper、环境变量覆盖、统一 `Config.Validate` 与 JWT `kid` 轮换窗口存在 | `implemented` / `automated_verified`：新令牌仅由当前 key 签发，显式旧 key 可验证，撤下即失效；release JWT/CORS 安全门已测试 | 外部 secret manager、生产轮换演练与配置清单 |
| Migration / DB lifecycle | 97 对迁移、静态配对门、专用空库全 up/down/up、runbook 与 CI 门存在 | `implemented` / `manually_verified`（本地隔离 PostgreSQL）：2026-07-12 从空库升至 100、全量降至 0、再次升至 100 | 正式数据库账本、结构和迁移 checksum 仍为 `unknown`；正式迁移需 Owner 批准 |
| Observability | request ID、context correlation、审计 correlation、EventBus/outbox correlation、metrics、Sentry、日志与告警规则存在 | `implemented` / `automated_verified`：HTTP RequestID 可贯通 Go context 和 EventBus，非法外部 ID 会被替换；通知送达仍为 `unknown` | Command/ToolBridge 所有调用点强制继承、审计失败指标、生产告警演练 |
| Health / Readiness | `/api/health` liveness 与 `/api/ready` readiness 已分离 | `implemented` / `automated_verified`：readiness 检查数据库、EventBus、Scheduler 和流量排空状态 | 容器和部署环境实际故障注入、告警联动 |
| Graceful shutdown | App 持有 Bus/Scheduler/Cancel，显式 BeginDrain 与 EventBus StopWithContext | `implemented` / `automated_verified` / release 隔离环境 `manually_verified`：拒绝新事件、排空队列/handler、Scheduler retry loop 可取消；SIGTERM 1 秒内状态 0 退出 | 生产长外部调用和强制超时演练 |
| CI gates | Go/前端、迁移、恢复、Compose、告警配置、漏洞扫描与底层 race workflow 存在 | `implemented` / `automated_verified`：当前本地完整 Go、重点 race、前端测试/构建通过；GitHub main 已启用 strict required checks、管理员约束、禁止强推/删除 | 本工作区尚未推送，新增 CI 配置在 GitHub 的实际运行结果仍为 `unknown` |
| Backup / Recovery | 原子 custom archive、archive list、SHA-256、S3 fail-closed、Versioning/Object Lock/对象 retain-until 强制检查、隔离恢复脚本与 CI 门存在 | `implemented` / `automated_verified`；2026-07-12 本地隔离恢复为 `manually_verified` | 生产 timer、真实不可变 S3、IAM 隔离、Owner 告警与生产 RPO/RTO 仍为 `unknown` |
| Production network boundary | 生产 Compose 清除开发 ports/volumes/command，CI 渲染验证脚本存在 | `implemented` / `automated_verified`：本地最终配置只有 Caddy 发布 80/443 | 生产主机最终配置与外网 3000/5432/8080 不可达实测仍为 `unknown` |

## 4. 第一批已确认缺口

1. Command 幂等键此前未实现；本轮已加入持久化 claim、结果重放、失败重试和过期租约接管，生产迁移与崩溃演练仍待验证。
2. HTTP 审计此前是 fire-and-forget，测试依赖固定睡眠；本轮已改为请求内有界等待并新增取消测试。
3. 健康接口固定返回 `ok`，不能作为生产 readiness 证据。
4. 配置加载尚未形成按运行环境强制执行的安全校验。
5. 多个并发基础设施测试依赖 `time.Sleep`，可能形成慢且不稳定的 CI 信号。
6. 备份、恢复、生产迁移和外部告警只有文档或代码存在证据，尚无本轮真实演练证据。

## 5. 固定完成标准

每一项基础设施能力只有同时满足以下条件才可标记完成：

1. 契约明确，且不含具体经营逻辑；
2. 正常、失败、超时、取消、重复、并发和重启边界按适用范围实现；
3. 高风险动作经过 Auth、RBAC、Approval、Audit，且不能绕过；
4. 有确定性自动测试，异步测试优先使用信号而不是固定睡眠；
5. 有可观测状态，能回答触发、执行、结果和失败原因；
6. 有升级、回滚或恢复路径；
7. 在当前环境完成相关包测试，最终完成前执行全量 build、vet、test、race、迁移和运行手册演练；
8. 对没有连接生产环境的事项保持 `unknown`，不得写成生产可用。

## 6. 当前验证记录

- 2026-07-12：修改前 `go test ./internal/domain/experiment ./internal/httpx/middleware ./internal/domain/operationlog`，143 个测试通过。
- 2026-07-12：HTTP 审计同步化后 `go test ./internal/domain/operationlog ./internal/httpx/middleware`，132 个测试通过。
- 2026-07-12：新增审计取消测试后相同两个包 133 个测试通过。
- 2026-07-12：Command 持久化幂等实现后 `go test ./internal/platform/command ./internal/httpx ./internal/httpx/middleware ./internal/domain/operationlog`，205 个测试通过。
- 2026-07-12：统一配置安全门后 6 个相关包 218 个测试通过，定向 `go vet` 通过。
- 2026-07-12：liveness/readiness 与 EventBus/Scheduler 生命周期状态实现后 7 个相关包 294 个测试通过。
- 2026-07-12：全量 `go vet ./...` 与 `go build ./...` 通过。
- 2026-07-12：底层 6 个包执行 `go test -race`，修复一处 EventBus 集成测试 bool 数据竞争后，283 个测试通过。
- 2026-07-12：全量 `go test ./...` 为 2964 通过、5 失败、11 跳过；5 个失败全部位于本轮未修改的未提交 `sourcing1688` 工作，故不能声明仓库全绿。
- 2026-07-12：Auth 身份校验与 refresh session 轮换后，Auth/HTTP middleware 普通及 race 测试各 122 个通过。
- 2026-07-12：Auth、RBAC、Approval、Command 与 route catalog 安全收口后，6 个包普通测试 256 个通过，5 个高风险包 race 测试 226 个通过。
- 2026-07-12：EventBus outbox/idempotency fail-closed 与 DLQ 错误语义完成后，EventBus/Scheduler 普通及 race 测试各 72 个通过。
- 2026-07-12：ToolBridge 类型化审批与 context 执行接入后，6 个相关包普通测试 152 个通过，4 个底层包 race 测试 117 个通过。
- 2026-07-12：生产 Compose 最终配置完成自动渲染验证；db/backend/frontend 无 host ports、源码挂载和开发 command，只有 Caddy 发布 80/443。
- 2026-07-12：备份合同脚本与 production backup profile 验证通过；本地 `multisell` 生成约 12 MB archive，隔离恢复后验证 107 张 public 表及 `user`、`operation_log`、`schema_migrations`，临时库自动删除。Docker daemon 未运行，专用 backup 镜像构建仍为 `unknown`。
- 2026-07-12：迁移静态检查发现并补齐 041 down；全生命周期演练发现并修复 061 down 漂移。专用空库完成 version 1→91、91→0、0→91。当前本地 `multisell` 的账本为 version 1 dirty、缺 72 张参考表；其隔离副本补跑在 migration 2 安全失败，正式数据修复未获授权且保持 `unknown`。
- 2026-07-12：当前会话家族与全部 refresh session 撤销实现后，Auth/HTTP middleware 普通及 race 测试各 124 个通过。
- 2026-07-12：排空状态与停机顺序修复后，HTTP/Scheduler/EventBus 普通及 race 测试各 84 个通过；排空中的 readiness 503 已自动验证。
- 2026-07-12：CI 增加底层内核 race 门，7 个包本地 race 测试 317 个通过。Prometheus/Alertmanager 官方配置校验已加入 CI；本机 Docker 未运行，当前本地校验状态为 `unknown`。
- 2026-07-12：ToolBridge 双层幂等完成：本地 `tool_execution` 持久化 claim/结果重放，加外部驱动幂等键协议；相关普通测试 46 个、race 测试 34 个通过。
- 2026-07-12：Scheduler retry 持久化与启动恢复完成，恢复失败时 fail-closed；Scheduler 相关普通测试 32 个、race 测试 20 个通过。
- 2026-07-12：EventBus pending outbox 自动恢复与完整上下文保存完成，并修复 Bus/Scheduler 早于订阅注册启动的问题；相关普通测试 87 个、race 测试 75 个通过。
- 2026-07-12：隔离 PostgreSQL 完成 92 对迁移的 latest down/up、全 down、全 up，最终 version 95，临时库已删除。随后全量 `go build ./...`、`go vet ./...`、`go test ./...` 通过，测试为 3069 个、118 个包。
- 2026-07-12：operation_log 数据库 SHA-256 链与 append-only trigger 完成；隔离 PostgreSQL 两条链记录重算 `broken=0`，UPDATE/DELETE 均被拒绝，临时库已删除。
- 2026-07-12：JWT `kid` 轮换和 correlation context/审计贯通完成；相关普通及 race 测试各 224 个通过，Auth/Config 相关普通测试 139 个、Auth race 22 个通过。
- 2026-07-12：94 对迁移在隔离 PostgreSQL 完成 latest down/up、全 down、全 up，最终 version 97，临时库已删除。最终 `go build ./...`、`go vet ./...` 通过；底层 9 包 race 371 个通过；全量测试因同时变化的 sourcing1688 业务测试为 3060 通过、12 失败、11 跳过，故仓库当前不能标记全绿。
- 2026-07-12：Scheduler PostgreSQL advisory leader lease 完成；真实双连接演练为持锁期间第二连接 `false`、连接释放后 `true`。单元与 race 测试均通过。
- 2026-07-12：EventBus 有界 drain 完成，排空 8 个排队事件、拒绝 drain 中新发布并覆盖 deadline；普通及 race 测试各 79 个通过。
- 2026-07-12：审计写/完整性指标、每小时校验任务及 critical 告警完成；审计 HMAC checkpoint 在隔离 PostgreSQL 生成成功，异地必需但缺 URI 时非零退出；生产 systemd timer 与失败 webhook 已实现但未部署。
- 2026-07-12：ToolBridge 有限重试、熔断、half-open、取消及 Prometheus 指标完成；普通测试 48 个、race 36 个通过。
- 2026-07-12：release 二进制隔离 PostgreSQL SIGTERM 首次暴露 retry loop 造成 29 秒延迟；修复并加回归测试后再次演练 readiness=true、1 秒内退出、状态码 0，验证库删除。
- 2026-07-12：本阶段最终 `go build ./...`、`go vet ./...`、`go test ./...` 全绿，3085 个测试通过、118 个包；底层 9 包 race 378 个通过；范围内 `git diff --check` 通过。
- 该条历史缺口已由本文件后续验证记录取代；正式环境恢复仍保持 `unknown`。
- 2026-07-12：审批执行数据库状态机加入迁移 100；97 对迁移在隔离 PostgreSQL 完成 latest down/up 与全 down/up，最终 version 100。合法 `processing → failed → processing → succeeded` 成功，直接 SQL 修改 action/target 绑定、回退或修改 succeeded 终态、DELETE 均被 trigger 拒绝；独立 verifier 已接入 CI。正式数据库尚未迁移，保持 `unknown`。
- 2026-07-12：扩大 route coverage 枚举后确认现有 CI 只硬性检查 12 个选定领域的 101 条 mutation route；静态解析整个活跃后端得到约 345 条 mutation，其中约 233 条未进入 high-risk catalog。这不证明 233 条都应审批，但证明它们尚无逐路由的 risk/action/permission/approval/audit 明确裁决，因此“所有高风险写入不可绕过”仍为 `gap`，现有 coverage 脚本通过不得解释为全后端覆盖。
- 2026-07-12：上述 coverage gap 已形成静态与运行时双门：扫描范围扩大到全部非测试 Go 注册代码，当前 385 条源码 mutation route 全部写入不可隐式默认的策略清单（public 5、standard 198、high 182）；新增、删除、改路径或移动来源会使 CI 失败。Gin 实际路由表在后台服务启动前与清单双向核对，未分类 live route 或非可选 stale policy 会使进程 fail-fast；中间件对运行时未分类的受保护 mutation 返回 500 且 handler 不执行；全部 DELETE 强制 high。首次真实 release 启动校验发现静态正则遗漏 36 条 Feedback、AI action、1688 路由及 1 条条件 Prism 路由，修复扫描器、显式可选标记和策略后，隔离 PostgreSQL release 服务成功启动并干净退出。明显危险的 LLM 凭证、策略/Agent 规则、自治等级、执行/回滚、采购、合规抑制、账单审核、Feedback 删除/迁移和真实发布执行已提升为 high。公开 Feedback 写入口新增 IP 限流；其认证写组补上 approval middleware。standard 清单的领域语义仍需持续复核，正式服务器运行状态仍为 `unknown`。
- 2026-07-12：可观测性复核发现 Prometheus 原配置没有 `alerting.alertmanagers`，规则 firing 时不会送达；已连接 `alertmanager:9093`，生产 backend 默认启用 metrics，CI 从只校验 rules 扩为同时校验完整 Prometheus config。配置链路为 `implemented`，真实 Alertmanager firing/resolved 与 Owner 收到 Slack 仍为 `unknown`。
- 2026-07-12：生产备份与审计 checkpoint 从“上传后对象存在”升级为 fail-closed 不可变异地合同：生产默认要求 Versioning、Bucket Object Lock 默认 retention，且逐个验证 archive、checksum/checkpoint 对象实际具有 ObjectLockMode 和 retain-until；缺失任一项任务失败。备份、checkpoint 和 ops webhook 合同测试通过；Compose verifier 修复此前未带 `manual` profile、实际没检查 backup/audit-checkpoint 的盲区，并强制 metrics/offsite/immutable 配置。真实 S3 bucket、IAM 隔离和实际保留仍为 `unknown`。
- 2026-07-12：GitHub 只读核验发现 CI workflow 有真实 success/failure 记录，但 main required checks 为空；随后通过 GitHub branch protection API 将 `governance-checks`、`pr-compliance`、`go-lint`、`go-test`、`next-build`、`e2e-test` 设为 strict required checks，并复读确认 enforce_admins=true、禁止 force-push/delete、要求 linear history 与 conversation resolution。当前未提交改动尚未在 GitHub CI 执行，仍不能引用旧 run 证明本工作树。
- 2026-07-12：首次强制漏洞扫描发现可达的 Go 标准库、quic-go 和 pgx 漏洞；依赖升级至 quic-go 0.59.1、pgx 5.9.2，构建/CI 工具链固定到 Go 1.25.12，govulncheck 1.6.0 从 continue-on-error 改为阻断门。使用 Go 1.25.12 复扫为 0 个可达漏洞；模块依赖中仍有 1 个不可达漏洞信号，按 govulncheck 证据不构成当前调用路径漏洞。
- 2026-07-12：正式主机 `lingmirror` 只读取证据：Ubuntu 24.04，SSH 有效配置为公钥登录、密码和键盘交互关闭；但 Docker 未安装、`/opt/multisell` 不存在、UFW inactive、fail2ban 未安装/未运行、LingMirror backup/audit timer 不存在，主机监听仅 SSH 和两个 localhost OpenClaw 端口。外部 `nc` 对多个端口报告连接成功却与主机 `ss` 冲突，云安全组状态仍需在云控制面核验。正式部署、TLS、readiness、告警、备份和恢复均未发生，保持 `unknown/gap`；未经 Owner 批准未修改主机。
- 2026-07-12：直接外部写出口复核修复两处：listing task PublishHook 在 production/approval-required 模式下现在独立验证审批存在、状态、目标和有效期，并把 approval ID 写入 adapter context；Ozon HTTP handler 不再信任 body 中孤立的 approval ID，只接受统一 HTTP gate 注入且与 body 一致的 ID。1688 发布执行 route 新增通用 target type/target ID param 元数据，合法 `sourcing_1688_publish` 审批可绑定 attempt；reconcile 只核对超时后的外部结果，不重复消费执行审批。相关普通/race 测试通过。
- 2026-07-12：最终本地回归再次执行：Go 1.25.12 `go build ./...`、`go vet ./...`、`go test ./...` 全部通过；认证、配置、HTTP、审批、审计、集成发布、listing task、schema drift、route catalog、Command、EventBus、Scheduler、ToolBridge 的 `go test -race` 全部通过。前端 25 个测试文件、133 项测试通过，生产构建生成 91 个页面；lint 为 0 error、10 个既有 warning。`govulncheck` 为 0 个可达漏洞，另有 1 个仅模块依赖、当前代码不可达的信号。
- 2026-07-12：完整 Go 回归首次暴露 supplychain event timeline 的偶发顺序颠倒。根因为以 wall-clock `created_at` 排序不能提供稳定因果顺序；已改为数据库生成的单调 outbox ID 排序，同一测试重复 50 次及全包回归通过。该修复只涉及事件底层读取语义，不扩大业务范围。
- 2026-07-12：最终隔离 PostgreSQL 16 再次验证 97 对迁移：空库升至 version 100、latest down/up、全量 down 至 0、再次 up 至 100；审批状态机合法路径成功，绑定篡改、成功终态回退/修改与 DELETE 均被拒绝。验证库在结束后删除。
- 2026-07-12：最终静态门复核为 385 条 mutation routes（public 5、standard 198、high 182），生产 Compose 仍只有 Caddy 发布 80/443；备份、审计 checkpoint、运维失败通知合同测试通过。上述是本地/隔离环境证据，不证明正式主机、真实 S3、真实 Slack 或真实外部 provider 已工作。
- 2026-07-12：生产回滚脚本复核发现两个危险旧假设：使用 `git reset --hard` 会丢弃现场改动，且通过未公开的 `localhost:8080` 判断健康。现已要求明确 `--target`、拒绝有已跟踪改动的工作区、用 detached checkout 保留分支历史；切换旧代码前强制使用当前可信版本完成异地不可变备份，目标版本还必须通过生产 Compose 边界验证并在触碰数据库前完成镜像构建；选择数据库降级时先停止后端写入且不再忽略失败；回滚后通过生产域名同时验证 HTTPS liveness/readiness，60 秒失败则保留现场并返回非零。静态安全合同测试已接入 CI。正式回滚演练仍为 `unknown`，未经 Owner 批准不得执行。
- 2026-07-12：Owner 随后明确授权尝试部署。正式主机已实际安装 Docker Engine 29.6.1、Compose 5.3.1、PostgreSQL client 与 Fail2ban；Docker/Fail2ban 均 active，Fail2ban sshd jail 已观察到实际封禁；UFW 已启用 deny incoming/allow outgoing，仅允许 22/80/443，启用后第二次 SSH 连接成功；`/opt/multisell` 与 mode 700 的 backups 目录已初始化。应用未部署：本地 main 与远程 main 已分叉且工作区有 145 个已跟踪改动，不能上传为确定发布版本；生产 DOMAIN/ACME/S3/Slack 配置缺失；服务器到 GitHub 的 full clone 与 depth-1 clone 均在有限等待窗口内未完成，已终止进程并清理半成品目录。当前主机只监听 SSH（另有既存 localhost-only OpenClaw，不在公网）；80/443 尚无服务。不得把本次结果称为应用上线、TLS、备份或告警完成。
- 2026-07-12：数据库池新增 30m 最大连接寿命和 5m 最大空闲时间；HTTP server 新增 read-header/read/idle/shutdown timeout 与 1 MiB header 上限，SSE write timeout 保持 0 并依赖操作 context。schema drift 修复三类假信号：过程体内 CREATE 不再被误解析、无注册模型时跳过静态比较并明确降级、golang-migrate 单行 version/dirty 账本按真实语义计算且允许版本空号；隔离 release 复验 version=100、97 files applied、unapplied=0、missing=0、dirty=false。
