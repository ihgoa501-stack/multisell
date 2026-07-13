# 凌镜本地运行验收事实审计

> 审计日期：2026-07-13
> 审计范围：ADR-001 Owner 自用平台的本地跨层运行链、迁移健康、Schema Drift 与图片服务契约
> 环境：本机隔离 PostgreSQL 数据库；临时 Go 后端与 Next.js 开发服务
> 证据限制：未连接正式服务器、真实渠道账号、真实供应商、真实订单、真实结算或银行数据

## 结论

本次把此前的自动测试证据推进到本地隔离环境的跨层人工运行验收，但不提升任何真实经营事实或生产可用性声明。

| 声明 | 证据等级 |
|---|---|
| 数据库从空库迁移至版本 145，136 个迁移文件全部应用，`dirty=false` | `manually_verified`（本地隔离 PostgreSQL） |
| 后端在版本 145 数据库上启动，迁移健康为 145/145，Schema Drift 报告 0 缺表、0 多表、0 缺列、0 类型不一致 | `manually_verified`（本地隔离 PostgreSQL） |
| JWT 登录、RBAC、跨 Owner 404、缺审批 403 且无决定副作用、一次性审批、成功决定、审批重放 409、成功/失败审计、页面刷新及 refresh token 恢复 | `manually_verified`：Playwright Chromium 1/1 通过 |
| 跨层验收只使用明确标记为 `mock` 的订单事实，没有执行渠道、供应商或其他外部写入 | `actual` |
| Schema Drift 解析器支持迁移中的列重命名、列类型变更、多词 PostgreSQL 类型，并避免根据不足的 ARRAY/UDT 元数据制造误报 | `implemented / automated_verified` |
| LingMirror 与 Image Service 的付费图片任务契约同步保存并校验成本、币种、地区、Provider 环境、沙箱、水印和不可发布边界 | `implemented / automated_verified` |
| 后端全量 122 包 3406 项测试、Go vet/build；Image Service 12 包 85 项测试与 Go vet；前端全量测试及生产构建 | `automated_verified` |

## 验收中发现并修复的问题

1. Schema Drift 只读取初始 `CREATE TABLE`，没有应用后续列重命名和类型变化；同时把 `VARCHAR(n)` 与 information_schema 的基础类型误判为不一致。修复后隔离版本 145 数据库报告无 live drift。
2. 图片领域已使用新的预算与沙箱字段，但 LingMirror 的 Image Service 客户端 DTO 未同步，导致后端无法构建。现已同步完整字段。
3. 图片服务持久化契约缺少 `region`，导致任务、执行授权和远端回执不能形成 exact 地区绑定。现已贯通内存存储、PostgreSQL schema、客户端 DTO 和执行授权校验。
4. 未配置的 typed-nil Image Service 客户端会在能力读取时 panic。现改为明确返回“未配置”，Owner 权限路由测试不再崩溃。

## 仍然未知

- 正式生产服务器的 HTTPS、域名、告警投递、外部备份恢复、生产迁移和回滚仍未执行。
- Docker Desktop 本次未运行，因此没有重跑 Compose 容器级验收。
- Schema Drift 的 live database 比较已通过，但启动时尚未注册 GORM models，因此静态 migration-vs-model 比较仍会明确跳过。
- 没有真实市场选择、真实 1688 页面、真实渠道发布、真实订单、收货、售后、结算、最终利润或现金对账。
- 本地 `mock` 验收只能证明工程链在注明环境中可运行，不能证明真实经营价值、因果关系或生产稳定性。

## 清理

临时后端、前端和隔离数据库在验收后停止并删除；没有修改现有 `multisell` 数据库。

## 架构收口补充核验（2026-07-13）

本节只记录针对权威经营读取、外部未知结果、关键审计与 Router 职责的工程收口，不提升真实经营或生产可用性等级。

| 声明 | 证据等级 |
|---|---|
| Owner 经营行动在派发前被权限/审批明确拒绝时记录 `failed`；进入命令处理后未取得成功回执时进入 `reconcile_required`，中断的 `executing` 状态会在再次读取执行入口时恢复为待对账且不会重新派发 | `implemented / automated_verified`；businessfeedback 聚焦测试通过 |
| Owner 经营行动在派发前必须与 `pending` 审计同事务落库，结果状态与终局审计同事务落库；审计不可写时外部命令不会执行 | `implemented / automated_verified`；审计失败关闭回归测试通过 |
| Listing 审批状态机、状态更新和权威审计已收回 listingtask 领域并在同一事务完成；Router 只解析事件身份并调用领域方法 | `implemented / automated_verified`；listingtask 与 httpx 聚焦测试通过 |
| 小Q Owner 默认入口不再提供基于 `experiment_id` 的旧经营终局读取，只保留按 exact `order_id` 的权威经营事实；experiment 页面不再展示或推进旧利润/现金终局，历史报表不再展示利润/结算终局标签或日报/周报旧利润值 | `implemented / automated_verified`；真相边界与小Q聚焦测试、定向 ESLint 与 Next.js 生产构建通过 |
| 142 对迁移可在空白隔离 PostgreSQL 完成全上行、最新回退/上行、全回退和再次全上行 | `manually_verified`；当前工作树最终版本 151，临时数据库 `lingmirror_migration_verify_final_20260713` 已删除 |
| 商品图片付费派发在外呼前进入待对账；精确回执才恢复排队，响应未知禁止再次付费执行；可信证据可结案为追回输出、未扣费或已扣费但无可恢复输出，Provider request ID 在成功及可取得的失败路径保存 | `implemented / automated_verified`；productimage、image-service Worker/OpenAI Provider 聚焦测试通过；真实 Provider 调用仍为 `unknown` |
| 后端全量测试、静态检查和构建 | `automated_verified`；122 包 3449 项测试、`go vet ./...`、`go build ./...` 通过 |
| 真实外部命令对账、真实 Owner 订单终局读取和真实经营价值 | `unknown`；本次没有连接渠道、供应商、订单、结算或银行数据 |

## 旧 Multi-Agent 运行面退役补充核验（2026-07-13）

本节记录小Q成为唯一 Owner Agent 后的运行面收口，不代表旧源码已全部物理删除，也不提升真实 Provider 验收等级。

| 声明 | 证据等级 |
|---|---|
| 旧 A1—A12/G0—G3 Orchestrator 在共享生产入口失败关闭；AIOS 初始化不再注册旧 Agent 名册 | `implemented / automated_verified`；Orchestrator 失败关闭与 AIOS 空名册聚焦测试通过 |
| 旧 Agent 周期任务、`agent.decided.*` DAG、MoA、信任分自治升级、AgentOS/熵/进化/代谢/旧编排生产路由不再由 Router 注册 | `implemented / automated_verified`；Router 编译与聚焦测试通过，生产注册源码复核无残留入口 |
| `/api/v1/ai` 仅保留历史 Trace 与 Action 的 GET 读取；旧 chat/run/MoA/action mutation 路由不再注册 | `implemented / automated_verified`；只读路由清单测试通过 |
| Owner 导航及对应页面移除 AI 指挥中心、Agent 列表、Action 中心、信任自治度、AgentOS 工作队列、Agent 学习/升级、熵/进化/代谢和旧编排；失效的旧内容生成、旧 A1/A8 mock 选品与伪 LLM 配置入口同步移除 | `implemented / automated_verified`；44 个前端测试文件 235 项测试及 79 页生产构建通过，已删除路由不再出现在构建清单 |
| 全局 Copilot 不再调用旧 `/ai/chat`，只说明小Q边界并进入 `/xiaoq`；WebSocket 不再绑定旧通用 AI Chat handler | `implemented / automated_verified`；源码回归扫描与 Router 测试通过 |
| 旧 Trace/Action 历史读取要求 `audit.read`，并按 JWT 中当前 Owner 过滤；跨 Owner 详情返回 404 | `implemented / automated_verified`；列表、详情及跨 Owner 回归测试通过 |
| 小Q产品运行时在所有环境都拒绝 `stub` 与缺失 API Key，身份接口明确展示真实 Provider 是否可用；当前本机四项 Provider 环境变量均未配置 | `implemented / automated_verified`（失败关闭测试）；当前真实 Provider 状态为 `unknown / unavailable`，未执行付费调用 |
| 真实 Provider 人工验收入口要求显式付费确认，最多 2 次模型调用且每次最多输出 400 tokens；结果必须包含真实工具读取、证据和 Trace | `implemented / automated_verified`（未确认时明确 skip）；真实调用仍为 `unknown` |
| 本轮最终工程回归 | `automated_verified`；后端 122 包 3504 项测试、Go vet/build，前端 44 文件 235 项测试与 79 页生产构建，462 条 mutation 审计分类及 269 个文档引用检查通过 |
| 旧领域规则、公式和查询实现 | `implemented / frozen`；暂不物理删除，需在逐项迁入小Q Capability或确认无价值后删除 |
| 小Q真实 Provider受限人工验收 | `unknown`；本次没有授权或执行付费模型调用 |
