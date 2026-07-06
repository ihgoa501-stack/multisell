# 凌镜 LingMirror — 多Agent并行调研全景报告

> **日期**: 2026-07-06
> **说明**: 本报告由11个 Teammate Agent 从不同角色和维度并行调研凌镜 LingMirror（MultiSell）项目后合成的全景文档。每个Agent独立输出原始分析，此处原样收录。

---

## 第一部分：代码与架构调研

---

### 1.1 架构总览 — arch-agent

#### 技术栈

| 层 | 技术 | 版本 / 备注 |
|---|---|---|
| 后端语言 | Go | 1.25 |
| 框架 | Gin + GORM | 最新 |
| 数据库 | PostgreSQL | 15 |
| 前端 | Next.js 16 + React 19 + Ant Design 6 | TS |
| 状态管理 | Zustand + TanStack React Query | 5 |
| 监控 | Sentry + Prometheus | opt-in |
| 部署 | Docker Compose | dev + prod |
| CI | GitHub Actions + pre-commit | — |

#### 目录结构

```
multisell/
├── backend-go/
│   ├── cmd/server/main.go              # 入口
│   ├── internal/
│   │   ├── config/                     # YAML + env 覆盖
│   │   ├── database/                   # GORM 初始化
│   │   ├── auth/                       # JWT 登录/注册/刷新（公开）
│   │   ├── rbac/                       # 权限控制
│   │   ├── httpx/                      # 路由 + 中间件 + 事件订阅 + WS
│   │   │   ├── router.go              # 所有模块路由/事件订阅汇总
│   │   │   ├── middleware/            # CORS→RequestID→Metrics→Sentry→Audit→Auth
│   │   │   └── response/              # 统一响应
│   │   ├── common/                    # 分页、排序工具
│   │   ├── realtime/                  # WebSocket hub
│   │   ├── dbtest/                    # SQLite 内存测试辅助
│   │   ├── platform/                  # AI Agent 通信原语
│   │   │   ├── eventbus/              #   pub/sub + glob 匹配
│   │   │   ├── command/              #   类型化命令调度
│   │   │   ├── scheduler/            #   定时任务 5min-6hr
│   │   │   ├── toolbridge/           #   插件式工具执行
│   │   │   ├── statemachine/         #   状态机
│   │   │   └── actioncatalog/        #   操作目录
│   │   ├── agent/                    # Agent 注册 + 执行
│   │   ├── ai/                       # LLM 调度、流式输出
│   │   ├── agentos/                  # AIOS 驾驶舱
│   │   ├── aios/                     # AIOS 基础设施（成本/护栏/工具注册）
│   │   └── domain/                   # ~65 个领域模块（四文件模式）
│   │       ├── order/, shipping/, settlement/, finance/...
│   │       ├── producthub/, listing/, catalog/...
│   │       ├── sourcing/, supplier/, compliance/...
│   │       ├── agentrule/, trustscore/, entropy/...
│   │       ├── integrations/  # Ozon/Shopee 适配器
│   │       └── ... (共 ~65)
│   ├── migrations/                   # 50+ 迁移文件
│   └── scripts/smoke_test.sh         # 10 步 E2E
│
├── frontend-next/
│   ├── src/
│   │   ├── app/(auth)/login/         # 公开登录
│   │   ├── app/(main)/dashboard, products...  # 认证页
│   │   ├── components/  # AuthGuard, CrudListPage, layout, ui
│   │   ├── lib/  # api-client, auth, query-client
│   │   ├── stores/  # Zustand
│   │   └── config/menu.ts
│   └── e2e/  # Playwright
│
├── docker-compose.yml / .prod.yml
├── AGENTS.md + docs/governance/  # 宪法级文档
└── VERSION (0.4.0.0)
```

#### 模块边界与通信

每个模块四文件模式：`routes.go → handler.go → service.go → model.go`

跨模块通信四条路径：

1. **HTTP API** — Frontend → gin.HandlerFunc → service → GORM (同步)
2. **EventBus** — `publish("order.*")` → glob 匹配 → ~15 订阅者 (异步/进程内)
3. **Command Dispatch** — Agent `dispatch("stock_alert")` → registered handler (同步/类型匹配)
4. **Scheduler** — 定时 tick → publish `scheduler.tick.{agent_id}` → Agent 执行

**Agent 管道链**:
```
A5 stock_alert → G3 discount_risk_check → A6 profit_watch → A2 listing_optimize
G0 system_health(异常>3) → G1 dashboard_overview
```
定时触发: G0/A4/G1/A5/G3/A6/A3/G2/A7/M1/trustscore/entropy

**平台适配器**: `RegisterAdapter("ozon", &Adapter{})` — PlatformAdapter 接口

#### API 设计

- 基础路径: `/api/v1`，健康检查: `/api/health`
- Swagger: `GET /swagger/index.html` (44 端点)
- 认证: JWT Bearer (auth + RBAC)
- 统一响应: `{"code":0, "message":"ok", "data":...}` / 分页含 total/page/size
- 中间件栈: CORS → RequestID → Metrics → Sentry → Audit → Auth

#### 部署架构

```
Docker Compose 四容器:
Frontend(node:22, port 3000) → Backend(go:1.25, port 8080) → PostgreSQL 15
Migrations(一次性) + Backup(按需, pg_dump → S3)
验证: go test/vet + npm test/build + playwright + smoke_test.sh
CI: GitHub Actions + pre-commit
```

#### 综合评价

**优点**:
1. **模块纪律严格** — ~65 领域模块统一四文件模式，导航可预测
2. **事件驱动完整** — EventBus + Scheduler + Command 三原语覆盖 Agent 执行闭环
3. **文档完备** — governance 宪法、项目病历本、Swagger、INDEX、api-inventory
4. **测试到位** — SQLite dbtest (t.Parallel)、smoke_test E2E、Playwright
5. **监控完善** — Sentry + Prometheus + Audit + WebSocket

**架构隐患**:
1. **65 模块平铺无分层** — 商业核心/AI Agent/CRUD 混在 domain/ 根下。建议按 commerce/、agent/、platform/ 分组，当前规模已超过平铺舒适线
2. **EventBus 进程内内存实现** — 重启订阅丢失、多实例无法广播。水平扩展需换 NATS/RabbitMQ/Redis PubSub
3. **Go module path 不对齐** — `github.com/lingmirror/backend-go` 不在公开路径，每次 build 触发远程查询
4. **隐式依赖耦合** — ~15 条事件订阅收敛在 router.go import side-effect 中，无显式依赖图
5. **无 API 版本策略** — 硬编码 `/api/v1`，缺少向后兼容机制

---

### 1.2 AI/Agent 系统分析 — ai-agent

#### LLM Orchestration (`internal/ai/`)

**Architecture:** Clean `LLMProvider` interface with `Chat`, `ChatStream`, `Name`. Five backends: OpenAI-compatible (reusable struct handles openai/qwen/deepseek/azure via different base URLs), Anthropic (Messages API), and StubProvider for dev.

**Strengths:** Stub provider forbidden in production (`log.Fatal` guard). Prompt caching tracking via Prometheus counters. Clean env-based factory in `NewLLMProvider()`.

**Gaps:** Anthropic streaming falls back to non-streaming+single-chunk. No tool-use/function-calling in LLM interface. No structured output parsing. No multi-turn conversation management.

#### Agent Registry and Execution

**Registry:** 14 agents defined in `DefaultRegistry()` (A1-A7, A8-Sourcing, A9-BatchOps, A10-Logistics, A11-Aftersales, G0-Coordinator, G1-Dashboard, G2-Warehouse, G3-Discount, plus `content_ai`, `scheduler`). Static hardcoded list -- no dynamic registration.

**Execution Pipeline:** `Orchestrator.Run()` is the central 280-line pipeline:
1. Validate agent + decision point
2. Start DB-persisted trace
3. Emit stub prompt_start + tool_call events
4. Check concrete impl.Agent implementation
5. Fall back to LLM provider or deterministic stub
6. Guardrails chain (L1-L5 input/output check)
7. Budget/cost controller check
8. Personal rule evaluation
9. Unified action creation gated by autonomy level
10. Forbidden action check
11. Approval policy evaluation (auto-approve/block/escalate)
12. Publish agent.decided.* event for pipeline chaining
13. Async trust score recalculation + autonomy upgrade

**Agent Impls:** All 14 agents have concrete `impl.Agent` implementations via `All()`. Good separation: orchestrator handles infrastructure, impl agents handle domain logic.

#### AgentOS Kernel

**Entropy (`domain/entropy/`):** Self-cleansing with 5 defense lines executed in order: TTL sweeper, budget enforcer, decay scheduler, merge detection, shadow/override detection. Includes SPC controller for anomaly detection and `CheckAgentHealth()`. System entropy index as weighted combination of unhealthy ratio, shadow ratio, health score.

**Evolution (`domain/evolution/`):** Autonomy nudges -- when trust score hits 80% of next threshold, creates a `Nudge` record for Owner approval. Four-level ladder: advisory -> guided -> supervised -> autonomous.

**Trust Score (`domain/trustscore/`):** Score from action adoption rate, execution success, confidence. `Upgrader` handles automatic promotion. Async recalculation after every agent action.

**Action Policy (`domain/actionpolicy/`):** Configurable rule engine (block/escalate/auto_approve), priority-ordered, with high-risk gate preventing auto-approval of high-risk actions.

#### Event-Driven Pipeline

**Event Bus (`platform/eventbus/`):** Mature in-process pub/sub with:
- Glob topic matching (`order.*`)
- Per-priority QoS worker pools (normal=4w, high=2w, critical=2w)
- DB outbox persistence + DLQ with configurable retries (default 3)
- Idempotency state machine via `event_processed` table
- Schema validation registry + Prometheus metrics

**Scheduler (`platform/scheduler/`):** Per-task goroutines with `time.NewTicker`, publishes `scheduler.tick.{agent_id}` events. Per-task mutex prevents overlapping runs. Tracks cumulative ticks/skips/duration.

**Pipeline Engine (`agent/pipeline/`):** Declarative decision DAG with 5 edges:
- A5 stock_alert(red) -> G3 discount_risk_check
- G3 discount_risk_check(block) -> A6 profit_watch
- A6 profit_watch(is_loss/below_threshold) -> A2 listing_optimize
- G0 system_health(anomaly_count>3) -> G1 dashboard_overview

Per-target circuit breakers with degraded-skip on open.

**Wiring (`router.go`):** ~15 event bus subscriptions connecting scheduler ticks to agents, plus trustscore and entropy maintenance ticks.

#### Overall Assessment

**Strong:**
- Clean layered architecture (LLM -> Orchestrator(infrastructure) -> Agent impls(domain logic) -> Action(policy) -> Event bus(async chains))
- Well-designed autonomy progression with trust score gating
- Guardrails chain integrated at input, output, and execution levels
- Production-quality event bus with QoS, DLQ, idempotency
- Pipeline engine with circuit breakers superior to inline handlers

**Nascent:**
- Most orchestrator output is stub text, not real LLM calls
- Agent coordination limited to 5 pipeline edges and G0 supervisor -- no real-time agent-to-agent communication
- MOA coordinator is basic: sequential execution, naive conflict detection
- Agent capabilities are static -- no runtime discovery
- No agent memory or learning (agentlearning package is minimal)
- Tool calling is stubbed, not wired to real function-calling
- Evolution nudges exist but Owner approval UX is not observable here

**Verdict:** Foundation is robust. Coordination and safety layers (guardrails/policy/eventbus) are ahead of agent intelligence (LLM integration). Next step is wiring real LLM provider with tool use and structured output into the existing infrastructure.

---

### 1.3 业务领域分析 — biz-agent

#### 核心业务域总览（48+ 个模块）

**1.3.1 订单履约域**
- **order** — 销售订单生命周期, 状态机: pending -> confirmed -> shipped -> delivered -> completed, 可从3态取消
- **orderimport** — 从电商平台拉取/同步订单
- **aftersales** — 退货/退款/纠纷/RMA跟踪
- **shipping** — Carrier适配/费率/运输规则
- **supplychain** — 供应链编排/异常升级/追踪

**1.3.2 商品域（最厚重）**
- **producthub** (24文件) — 商品中枢: master/variant/version回滚/cost/relations关系图谱
- **listing** (9文件) — 上架发布+状态机+SKU适配
- **sourcing** (18文件) — 选品调研(BSR/关键词/市场趋势)+利润关税评估
- **candidate** — 候选商品+完整度引擎
- **compliance** (11文件) — 合规扫描+新鲜度
- **content** (8文件) — AI生成+本地化+校验
- **supplier/purchase/inventory/price** — 供应商/采购/库存/定价

**1.3.3 财务域（利润真相引擎）**
- **finance** — 利润核算: NetProfit = Revenue - PFee - LFee - AdCost - Purchase - Other (按SKU-订单粒度)
- **profit** — 利润汇总/亏损分析
- **settlement/exchangerate/platformfee/cost/landedcost/price** — 结算/汇率/费/成本/到岸成本/定价

#### 平台集成
`integrations/` 通过统一 PlatformAdapter 接口，已注册 3 个电商平台：
- **Ozon** (俄罗斯) · **Shopee** (东南亚) · **Shopify** (全球)
- 覆盖: 发布/库存同步/运单推送/订单拉取/退货拉取/结算拉取

#### AI Agent 赋能 (18+ Agent)

`internal/agent/impl/` 中:

| Agent | 核心决策点 |
|-------|-----------|
| A1 collection_agent | 候选商品发掘+供应商匹配 |
| A2 listing_optimizer | 标题/描述/关键词/定价建议 |
| A3 compliance_guard | 内容违规扫描+合规判断 (可block) |
| A5 inventory_alert (21KB) | 安全库存告警+补货 |
| A6 profit_watch | 亏损SKU检测+利润阈值 |
| A7 ad_advice | 广告策略建议 |
| A8 product_scout (35KB) | 市场分析+产品发掘 |
| A9 fulfillment_advice | 物流路由+履约时效 |
| A10 logistics_* (6文件) | 物流运营/路由/计费/绩效/发货 |
| A11 aftersales_* (6文件) | 退款/退货/纠纷/报告 |
| customer_service | 自动回复/工单 |
| G3 discount_risk | 折扣风控 |

**自动决策链**: A5 inventory_alert (red) → G3 discount_risk (可block) → A6 profit_watch → A2 listing_optimizer

#### 前端业务页面 (37个目录)
`frontend-next/src/app/(main)/` 下与后端domain一一映射：
- 订单类: orders/aftersales/order-import
- 商品类: products/product-hub/listings/brands/categories
- 供应链: sourcing/sourcing1688/suppliers/purchase/supplychain
- 财务: finance/settlement/platform-fees/shipping
- AI: ai/agents/agent-learning/agentos/decision
- 管理: settings/owner/approval/allocation/dashboard/reports/notifications...

#### 关键发现
1. **利润真相引擎是中枢**: 所有业务最终汇入 finance.ProfitCalculation, 驱动定价/采购/广告决策
2. **ProductHub 是商品侧中枢**: 最厚重模块(24文件), master/variant/version/relations 全链路
3. **Agent 嵌入深度高**: 18+ Agent 覆盖 80% 业务域, G3 拥有block权限
4. **统一抽象层好**: PlatformAdapter 接口统一3家平台, 新增平台成本低
5. **当前缺 Lazada/TikTok Shop** 等东南亚主流平台

---

### 1.4 基础设施与运维分析 — ops-agent

#### Docker 编排

三层 docker-compose 叠加模式：基础定义 + 生产覆写 + 监控栈。

四个核心服务:

- **db**: PostgreSQL 15-alpine，pg_isready 健康检查，持久化卷
- **migrate**: migrate/migrate v4.18.1，db 健康后执行，完成后退出
- **backend**: Go 1.25，多阶段构建(builder->runtime alpine 3.21)，非 root 用户，HEALTHCHECK /api/health
- **frontend**: Node 22-alpine，三阶段构建(deps->builder->runner)，Next.js standalone，非 root 用户

生产环境用 Caddy 2 作为统一入口：自动 Let's Encrypt TLS、安全头(HSTS preload/CSP/X-Frame-Options)、/api/* 和 /ws 路由到 backend、/metrics 仅内网可达。

资源限制：backend 2核/1G、db 2核/2G、frontend 1核/512M、Caddy 128M。日志 json-file 10MB 轮转保留 3 个文件。

backup 服务通过 profile: manual 按需运行，宿主机 cron 定时触发。

#### 数据库

迁移体系: 62 个顺序文件(000001~000064)，格式 NNNNNN_description.up/down.sql，golang-migrate 执行，每个均有 down 回滚。

子目录问题: migrations/ 下有 5 个模块子目录(finance, order, settlement, inventory, sku)，各含 001_init.sql，但不会被 golang-migrate 的平铺扫描自动发现。

Schema 架构: 5 个独立 PostgreSQL schema(order_module, finance_module, inventory_module, sku_module, settlement_module)通过 search_path 串联，schema 级模块隔离。

连接池: max_idle=10, max_open=100

#### CI/CD

4 个 workflow 文件。ci.yml 主流水线 7 个 job: doc-links(引用检测), go-lint(golangci-lint v1.62), go-test(PG 15 容器 + migrate + govulncheck + 180s), next-build(npm ci + build + test + lint), e2e-test(Playwright), deploy(push main SSH 部署), push-images(ghcr.io), notify(Slack)。release.yml: 合并 main -> 读 VERSION -> git tag + Release。

关键问题: push-images 推送了 ghcr.io 镜像，但 deploy 仍在生产服务器 docker compose build。构建与部署未分离，回滚依赖 git 而非镜像标签。

#### 监控

Prometheus v2.53 + Grafana 11.1 + Alertmanager v0.27，auto-provisioning。

11 条告警规则覆盖基础设施 + AI 管道:

| 级别 | 告警 | 监控意图 |
|------|------|---------|
| critical | BackendDown | 服务不可用 |
| critical | HighErrorRate | 5xx >5% |
| warning | HighGCHeap | 内存泄漏 |
| warning | DLQNonEmpty | 死信队列积压 |
| warning | EventBusQueueBackpressure | Agent 管道阻塞 |
| warning | SchedulerPublishErrors | Agent 调度异常 |
| warning | AgentHandlerErrors | LLM 调用失败 |
| warning | AgentHeartbeatMissed | 代理失联 |
| critical | AgentInstanceDown | 代理退出 |
| critical | SentryErrorSpike | 异常脉冲 |

错误追踪: sentry-go v0.29.1 + `@sentry/nextjs`
审计日志: 所有写操作记录到 operationlog 表

#### 配置管理

Viper 驱动，三层覆盖: config.yaml < 环境变量(15 个显式绑定 + AutomaticEnv 兜底) < 生产容器 env(required 强制必填)

LLM API key 仅通过 env 传入，不在 yaml 暴露。

配置块覆盖: server(port/mode), database(连接池+5 schema), jwt(24h/168h 双 token), llm(daily_budget=0 不限), prism(strict 模式), schemadrift 检测, sentry, redis, cors。

#### 部署脚本

deploy.sh 5 步: git pull -> 迁移 -> build -> 零停机滚动(backend 60s healthcheck 超时回滚 -> frontend -> caddy) -> 监控栈+全链路验证

rollback.sh 4 步: git tag 回退 -> 可选迁移回滚 -> reset --hard + build -> 重启验证

backup.sh: pg_dump --format=custom + gzip -> 本地 7 天 -> 可选 S3

#### 改进优先级

| 优先级 | 问题 | 建议 |
|--------|------|------|
| P1 | deploy.sh 用 git reset --hard | 改为 git checkout 分支隔离或 ghcr.io 拉取模式 |
| P1 | ghcr.io 镜像已推送未被生产使用 | deploy.sh 改为 docker compose pull，回滚只需换 tag |
| P2 | 5 个迁移子目录不被自动执行 | 确认用途，删除或 embed 按需执行 |
| P3 | 进程内限流器重启清零 | 需强化时改 Redis 滑动窗口 |
| P4 | 无 pg_stat_activity exporter | 出问题前暂不需要 |
| P4 | CSP 宽松(unsafe-inline/eval) | 需前端配合改 nonce，低优先级 |
| INFO | CI lint 已知失败 | 不影响部署门禁 |

---

### 1.5 代码健康度与安全分析 — quality-agent

#### 测试覆盖

**Frontend**: 12 test files, 77 tests, all passing. Healthy.

**Backend**: 140 test files across ~100 packages. 38 packages pass, 64 fail (build failures, not test logic failures). Two root causes cascade to all dependent modules:

- `internal/common/types.go:145` -- UserIDFromCtx redeclared (duplicate at lines 106 and 145)
- `internal/ai/ai_test.go:160` + 5 sibling files -- Unresolved git merge conflict markers (`<<<<<<< HEAD`)

What passes today: Platform infrastructure (eventbus, scheduler, statemachine, toolbridge, dbtest) and AIOS core (guardrails, costcontrol, pipeline, runtime, memory, sdk, observability, ipc, knowledge) -- all green. Auth, RBAC, and commerce domain modules have test files written but are blocked by the build cascade. Middleware tests: 1492 lines with rate limit and auth middleware coverage.

#### 安全机制

| Mechanism | Status | Detail |
|---|---|---|
| JWT Authentication | Implemented | HMAC-SHA256, algorithm check, access/refresh type enforcement |
| RBAC | Implemented, blocked by build | Permission checks exist on routes |
| WebSocket CheckOrigin | Protected | makeOriginCheck(allowedOrigins) in both ws.go and extension_handler.go |
| Webhook HMAC | Implemented | WebhookVerifier interface, per-platform signature verification |
| Rate Limiting | Partial | Per-IP limiter works but **CleanupPeriodic never started from cmd/server/** -- memory leak |

Gaps:
1. Rate limiter cleanup goroutine missing
2. Password policy -- min 8 chars only, no complexity requirement
3. `golang.org/x/crypto v0.53.0` is outdated
4. No CSRF protection visible

#### 代码风格

Go: Consistent routes.go/handler.go/service.go/model.go pattern across all ~70 domain modules. Standard response envelope. Go vet fails from the UserIDFromCtx redeclaration only.

TypeScript: ESLint flat config with eslint-config-next/core-web-vitals + TypeScript. 4 errors (useEffect setState), 15 warnings (unused vars). No automated format gate in CI.

#### 依赖健康

Go: Clean tree. One concern -- `golang.org/x/crypto v0.53.0` is outdated (latest v0.35.0+). Used for bcrypt. All other deps current (gin 1.12, gorm 1.31, jwt v5, zaps 1.28).

TypeScript: Modern stack (Next.js 16.2.9, React 19.2.4, Antd 6.5.0, TanStack Query 5, Zustand 5, Vitest 4). All current. Good health.

#### 凭证管理

No hardcoded credentials. All via environment variables (JWT_SECRET, DB_*, REDIS_*, SENTRY_DSN). PasswordHash has `json:"-"` tag. Config uses mapstructure + BindEnv pattern. Clean.

#### 总体评分：7/10

| Dimension | Score |
|---|---|
| Test Coverage | 5/10 (38 pass, 64 fail from 2 fixable build errors) |
| Security | 6/10 (core mechanisms present, 4 gaps) |
| Code Style | 7/10 (consistent patterns, lint debt) |
| Dependency Health | 8/10 (one stale crypto dep) |
| Secrets Management | 9/10 |

**Critical fixes** (unblocks CI + 64 tests):
1. Remove duplicate UserIDFromCtx in internal/common/types.go (lines 143-149)
2. Resolve merge conflicts in all 6 internal/ai/ files

**Medium priority**:
3. Start rateLimiter.CleanupPeriodic(5min) from server boot
4. Update golang.org/x/crypto
5. Add password complexity validation in register handler
6. Fix 4 eslint errors + 15 warnings

---

## 第二部分：产品战略与商业模式

---

### 2.1 CEO 战略思考 — ceo-thinker

#### 5年愿景

凌镜 LingMirror 5年后应该是 **一人跨境电商公司的操作系统**。

不是"一个工具"，不是"一个 SaaS"，而是：一个人注册凌镜，告诉它"我要卖什么品类、什么市场、预算多少"，然后凌镜自动完成选品→供货→上架→广告→客服→履约→财务的全部环节。创始人每天花15分钟看仪表盘做决策，剩下的事 AI Agent 去做。

在跨境电商SaaS中的角色：**低端颠覆（Low-end Disruption）**。现有玩家（Shopify生态、ChannelAdvisor、JungleScout）服务的是"有团队的公司"，价格高、复杂度高。凌镜切的是"一个人的公司"——这个群体原本根本买不起这些工具，凌镜用 AI 把服务成本降到原来的 1/10，自然打开一个新市场。

#### 从第一天开始的"可变现设计"

"不能按个人项目做" = 你现在写的每一行代码，要么在给收费铺路，要么在制造收费障碍。具体：

**架构层面必须做的：**
- **多租户数据隔离** — 现在哪怕只有一个用户，表结构也得带 `tenant_id`。不这么干，以后分租户就是全量重构。
- **功能门控（Feature Flags）** — 每个功能有 built-in 的 kill switch。免费版用 flag A，付费版用 flag A+B+C。现在不埋点，以后加付费层要改代码。
- **用量计量** — 架构预留"这个用户跑了多少个 agent 任务？上了多少个 listing？处理了多少个订单？"的计数点。不需要现在算账，但数据要进数据库。

**产品层面不能做的：**
- 不要给用户无限的东西。有限 API 调用、有限 SKU 数量、有限平台数。限制本身就是收费线。
- 不要做"全免费到一定程度再收费"。**从第一天就有限制**，以后就是"加钱解锁更多"而不是"突然开始收费"。

**数据层面：**
- 用户的操作日志、agent 决策日志天然就是定价依据。谁用得多谁付得多。现在存好这些数据，以后定价模型随便换。

简而言之：可变现设计 = 让架构有"收费面"可以附着，而不是收费时推倒重来。

#### 核心竞争力与护城河

**差异化（为什么用户选你不选别的）：**

传统跨境电商 SaaS 的逻辑：给你工具，你自己操作。
凌镜的逻辑：给你 AI 员工，你当老板。

这是本质区别。传统工具帮人"做得更快"，凌镜帮人"不用做"。对于一个想一个人运营跨境电商的人来说，后者是完全不同的价值主张。

**护城河（别人为什么抄不了）：**

三个层面，从薄到厚：

1. **平台覆盖**（最薄）— 接的渠道越多越难替代。但不是不可复制，纯堆人力的事。
2. **Agent 行为数据**（中等）— 每个 agent 跑过 1000 个真实订单后，决策质量和对平台规则的适应度是新玩家花6个月才能追上的。这是 time-based moat。
3. **从"自动化"到"自主化"的信任飞轮**（最厚）— 用户让凌镜赚到第一笔钱→给更多权限→凌镜赚更多钱→用户信任更强→给更多平台和资金。这个循环一旦启动，用户迁移成本极高，因为迁移的不是"一个工具"，而是"你的生意运营大脑"。

#### 先做什么，后做什么（6个月，2人团队）

**这是唯一正确的顺序，不要被任何东西分心：**

**Month 1-2：找到第一个真实用户（不是朋友）**
- 不是写代码。是去社交媒体、跨境电商社群找到1-2个正在做一人跨境的卖家。
- 问他们最大的痛点是什么。免费给他们用。
- 目的不是收入，是确认"有人愿意为这事花时间"。

**Month 3-4：跑通一个平台的一个完整闭环**
- 把精力集中在一个平台上（建议 Shopify，API 最成熟开发最快）。
- 闭环：上架一个商品 → 有人下单 → 通知发货 → 收到钱 → 算清楚利润。
- 这期间不接新平台、不做泛功能。就这一个闭环。

**Month 5-6：让第一个用户用起来并实现第一笔收入**
- 让那个真实用户用凌镜上架他的真实商品、跑他的真实订单。
- 过程中修的 bug、加的 feature 全部来自"他卡在哪里了"。
- 收入：要么订阅（$29/月），要么按订单抽成（1%），选一个，定下来就开始收。哪怕只收一个人。

**不做的事（YAGNI）：** 多语言、多币种、聊天机器人、移动端、AI 生成的广告创意、大数据看板（超过10个指标的都是噪音）。

#### 关键里程碑（从自用到收费）

```
M0  ✗  创始人在用（当前状态）
    ↓
M1  ✅ 第一个非创始人用户在真实使用
    ↓
M2  ✅ 完成第一次真实销售（用户通过凌镜接到并处理了一个订单）
    ↓
M3  ✅ 用户第2周还在用（留存信号）
    ↓
M4  ✅ 用户主动问"怎么付费"或"免费额度不够了"
    ↓   ← 这时候开始收费
M5  ✅ 第一个付费用户
    ↓
M6  ✅ 有了第2个付费用户（非推荐）
    ↓   ← 这时候才算 PMF 候选
M7  ✅ 月流失率 < 10%
```

**M4 之前不要想"定价策略"，M5 之前不要想"增长"。** 只有一个目标：让一个人通过凌镜运营他的生意，并且愿意为这个能力付钱。

---

### 2.2 商业模式分析 — cfo-thinker

#### 定价模型（3种）

**方案A：基础月费 + 单Agent订阅（推荐）**
- Base $29/mo（平台管理 + 看板 + 基础Agent可运行）
- 每个额外Agent $15/mo（A1选品、A8定价、A2优化等）
- 14个Agent全开 ≈ $29 + 13×$15 = **$224/mo**
- 理由：用户感知"按需付费"，实际用满的和全包价格接近

**方案B：纯按GMV抽佣**
- 0.5%-1% of GMV
- 单人$1万GMV → $50-100/mo
- 单人$10万GMV → $500-1,000/mo
- 问题：用户抗拒"抽成"，且你要验证真实GMV，管理成本高

**方案C：分层坐席制**
- Starter $49/mo（5个Agent，单平台）
- Pro $149/mo（10个Agent，3平台）
- Enterprise $399/mo（全部Agent + 自定义规则 + 白标）

**推荐：Phase 0-1 走A，Phase 3-4 在A基础上加C入口。** $29起步无门槛，单卖家平均$100-150/mo。

#### 一人公司付费意愿

**愿付钱（痛点直接相关的）：**
- 自动选品（找爆款等于直接收入）→ P0
- 定价/利润分析（多赚的钱 > 软件费）→ P0
- 商品图生成（外包一张$15-30，自己生成不要钱）→ P1
- 自动上架（省掉电商运营的时薪）→ P1

**觉得"不需要"（早期喊不动）：**
- 数据看板大屏
- 系统健康监控
- Agent行为审计
- 跨平台迁移工具

**核心洞见：** 单人卖家的沉默成本 = 自己的时间。任何能帮他从每天8小时筛品到每小时批处理的，他都愿意付。

#### 从零到收费的节奏

| 阶段 | 用户量 | 收费策略 | 交付物 |
|------|--------|---------|--------|
| Phase 0：纯自用 | 1人（自己） | 不收费 | 核心Agent跑通。现在 |
| Phase 1：内测 | 5-10个朋友卖家 | 免费 + 访谈 | 收集付费意愿 + NPS |
| Phase 2：创始会员 | 50人 | $29/mo 锁价 | 限时"创始价"永不涨价 |
| Phase 3：正式上线 | 200人 | $49-149/mo | A/B测试两档定价 |
| Phase 4：扩张 | 1000+ | 方案C分层 | 按需调整 |

Phase 1 → 2 的关键动作：**找5个愿意付$100的卖家做"付费访谈"**，比他嘴上说"我会买"有价值100倍。

#### 单位经济

假设单人卖家中位数 GMV $3万/月：
- 净利15%: $4,500
- SaaS承受力（5%净利）：$225/mo 或（0.3% GMV）：$90/mo

**你的利润空间：**
- 当前单用户边际成本 ≈ 服务器 + LLM API ≈ $10-20/mo
- 按$100/mo ARPU：毛利率80-90%
- 关键变量：LLM token消耗 —— 必须用量限制（Pro plan 10万token/月免费，超额$0.01/千token）

**ARR模型：**
- 200用户 × $100 ARPU × 12 = **$240k ARR**
- 1000用户 = **$1.2M ARR**
- 10人团队 + 云成本 ≈ $500k/yr → 盈亏平衡在~400用户

#### 竞争壁垒（定价保护伞）

**保护伞不是技术，是——**

1. **数据飞轮**：用户越用，选品模型越准。老用户的Agent决策质量 > 新用户。
2. **绑定深度**：14个Agent之间共享上下文。一个选品决策自动影响定价→上架→广告。
3. **迁移成本**：用户把整个生意跑在你上面，迁移 = 重建一个运营团队。
4. **场景锁定**：A8定价Agent直接对接物流费率、平台费率、汇率三个数据源。竞品要同时对接三套API才能拉到同一起跑线。

**定价保护伞策略：** 早期定价低于竞品总和（Helium10 $80 + Photoroom $15 + 虚拟助手 $500 = $595 → 你卖$100），等用户绑定后每年涨15-20%。

---

### 2.3 用户需求分析 — pm-thinker（真实卖家视角）

#### 一人卖家的真实一天

**早上7:30** — 第一件事不是刷牙，是摸手机看出单。三个平台各刷一遍：Ozon 昨晚俄罗斯时间出了多少、Shopee 台湾站凌晨的单、新开的 TikTok Shop 有没有新消息。确认没问题才去洗漱。

**9:00-10:30 客服噩梦** — 每天第一个硬障碍。三个平台、每个平台两三个店铺的 buyer message、退货申请、物流纠纷。70%的问题一模一样（"什么时候发货"、"跟踪号多少"），但必须一个一个点进去回复。最折磨人的是每个平台的消息系统不同，你要切四五个 tab。这个环节你愿意花钱让别人做，但目前没有工具能解决——客服外包贵且不安全，自己搞又没有聚合入口。

**10:30-12:00 发货+库存** — 看哪些订单没发出，生成面单，通知供应商。有的平台自动打印面单方便，有的要手动填发货信息。库存从 Excel 或脑袋里查，经常发生系统显示有货但供应商说没了的情况——这时候你得临时找替代品或取消单。这个环节的焦虑来自"你永远不确定你的库存数据是不是真的"。

**下午 2:00-4:00 选品上架** — 唯一"创造价值"的环节，但80%时间花在翻译和重复劳动上。AliExpress/1688 找到的产品要改三遍才能上到不同平台：一个平台限60字标题、一个限120字、一个要求图片白底、一个要求正方形。你在做搬运工的工作，不是在经营生意。

**4:00-5:30 广告调整** — 看广告 ROI。哪个关键词烧钱没转化、哪个品曝光不够。坦白说基本靠感觉，因为平台官方的广告工具给的是它想让你看到的数据，不是真相。

**晚上 10:00-12:00** — 真正的"战略时间"：趁安静的时候看同行上了什么品、研究趋势、学新平台规则。但这只是因为白天没时间思考。

**最痛苦的：不是任何一个单环节，而是永远切屏。** 一天在 8-12 个页面之间来回切，每个任务都要重新登录、重新加载、重新消化上下文。心智负担高到宁愿推迟也不愿开始。跨境电商一人卖家的本质困境不是"不会做"，而是"被切碎的事压垮"。

#### AI 怎么真正帮到他

**最不放心，必须自己看：广告支出和定价。** 钱是自己的。AI 说"这个品 ROI 低建议暂停"，你会先看数据再决定。广告预算是每个月最大可变成本，在这上面不信任任何黑箱。同样，定价——定低了亏钱、定高了不出单——不敢全交给 AI。

**不在乎谁在做，只求快点做完：客服消息和重复运营。** "什么时候发货"就是一个模板回答，一键发送就行。不需要 AI 替卖家思考，需要它替卖家点鼠标。同样：面单生成、订单状态批量更新、平台规则变更通知。

**愿意全自动：数据汇总和异常监控。** 跨平台的销售、利润、费用、库存，每天更新一次，月报一次。出错了找 AI 问原因，但不需要每天在后台等审批。库存不足、价格异常、平台政策变化这类 alert 可以全自动推。

**AI 不该出现的地方：每个决策都需要确认。** 最大的信任杀手是 AI 替卖家做了决定然后等着确认——每天打开后台发现 20 个待审批事项，比原来更累。需要的是"出了错再找我"模式，不是"每一步都需要你点头"模式。

#### 产品该砍掉/推迟什么

**"智能选品推荐" — 最大的废话功能。** 每个 SaaS 都做这个："AI 为您推荐 100 个爆品"。没有人靠这功能找到过好产品。选品靠的是你的经验和供应链人脉，不是算法从公开数据爬到的。如果你不具备某个品类的供应链优势，AI 推荐 100 个也没用。

**"全自动竞价优化"** — 听着好但卖家自己平台内的竞价都不信 AI，跨平台的自动竞价更是空中楼阁。

**"AI 生成详情页"** — 有用但不是核心价值。发布前卖家还是会改三遍。可以做成附带功能，不需要单独宣传。

**"竞品分析报告"** — 花 20 分钟看完，结论是"知道了"，然后什么都不会改变。看完产生焦虑但没有行动点。

**"社交电商内容批量生成"** — TikTok 做起来之前，这只是一个没人在乎的功能。

#### 隐藏需求

**需求一：决策疲劳** — 一天 50 个小决定，每个消耗同样的脑力。卖家不缺数据，缺的是"这个数据到底意味着什么"。本质需求：不要告诉我有什么问题，告诉我现在该做什么。Dashboard 上应该只有三件事：今天必须你亲自做的、AI 已经做了你只需知道的、可以忽略的。

**需求二：跨平台利润真相** — 最大的真痛点。每个平台算自己的账永远好看，但把采购成本、头程、尾程、平台佣金、广告费、退货率、汇率损失全算进去，很多"爆品"其实是亏钱的。这是每个卖家睡前都在想、但白天没时间算的东西。多数人的"利润"只是"售价减采购价"，真正的利润率可能只有一半。这个功能不会每天用，但每月用一次的价值超过所有 AI 选品功能加起来。

**需求三：出错的代价是真实的** — AI 一次失误能毁掉一个月利润。填错价格亏本、违反规则被封店、发错货亏运费。卖家不信任 AI 不是因为保守，是吃过亏。所以 AI 犯错容忍度必须极低，或者出错后补救成本极低。

**需求四：它是个生意，不是个系统** — 一人卖家不需要完美的代码架构。他需要每天早上花 3 分钟知道"今天能不能赚钱，有没有需要立刻处理的事情"。产品验证做错了：大部分团队在验证"这个功能能不能用"，没有人在验证"这个功能能不能让卖家今晚早点睡"。

#### 多平台策略

一人卖家一般做 2-3 个平台，最多 4 个，再多管不过来。

**伪需求**：跨平台同步上架。每个平台需要不同的图片尺寸、不同的标题格式、不同的关键词策略、不同的定价结构（含佣含运费不同）。强行同步的结果是两边都不对。

**真需求**：统一看板 + 统一客服 + 统一利润核算。真正想要的是"一个地方看所有平台的生意，在一个地方处理所有平台的日常运营"。顺序很重要：先被动聚合（看数据），再主动聚合（做操作）。

**启动策略建议：**
1. **MVP = Dashboard + 客服聚合 + 利润引擎**。不是又一个 AI 选品工具，而是运营中枢。
2. **不要做"全平台全能"**。先服务 2-3 个平台上一人卖家，做到极致再扩展。
3. **Platform adapter 的设计是对的，但前期优先做 read 和 reply，write 最后做。**
4. **托管模式优于 SaaS**：一人卖家不需要另一个需要学习的管理系统。

#### 对现有架构的影响

现有产品的 modular 架构是对的，但**优先级需要重新排序**：

- **P0：客服聚合（Support Mate）** — 最高频最痛点，MVP 可以不做 AI 自动回复，只做统一收件箱
- **P0：真实利润引擎（Profit Watch）** — 睡前焦虑的解药，Dashboard 的一级入口
- **P1：跨平台数据聚合** — 不只是 Dashboard，要变成每天早上的"生意日报"
- **P2：AI 自动上架辅助** — 当用户说到这个环节时有帮助，但不是主动推送的功能
- **P3：选品推荐 — 暂缓**
- **P3：竞品分析 — 暂缓**

**一句话给创始人**：你现在有一个还不错的 OS 骨架。但首先需要的是减轻用户白天压力的功能（客服），而不是增加晚上思考时间的功能（选品分析）。"帮卖家早点睡"比"帮卖家卖更多"更容易让人付费。

---

### 2.4 市场分析 — cmo-thinker

#### 竞争格局

跨境电商工具市场按"覆盖范围 + AI程度"分成四个象限：

| 象限 | 代表 | 能力 | 年付费 | 人 vs AI |
|------|------|------|--------|----------|
| L1 传统ERP | 店小秘、马帮、领星、船长BI | 订单/库存/采购/财务全链路，人工操作 | ¥3000-20000 | 人操作系统 |
| L2 数据选品 | Helium 10, Jungle Scout, Sorftime | 市场分析、关键词、竞品追踪 | $360-960 | 人看数据决策 |
| L3 AI垂直 | Photoroom(图), Teikametrics(广告), Gorgias(客服) | AI替代单一人力环节 | $120-2400 | AI提效人执行 |
| L4 AI全栈 | 凌镜 LingMirror | 14个Agent端到端自主运营 | TBD | AI替代人执行 |

L1-L3互不打通，卖家买4-5个工具手动对数据。L4是市场空白。凌镜的对手不是店小秘，是"卖家觉得Excel就够了"。

#### 市场规模（估算）

| 指标 | 数值 |
|------|------|
| 全球活跃跨境电商卖家 | 500-800万 |
| 1-3人团队占比 | 60-70% |
| 潜在用户池 | 300-500万 |
| 每人年工具支出 | $500-2000 |
| 可触达市场 (SAM) | $1.5B/年 |

ARR爬坡路径（$30/月定价）：

| 阶段 | 用户 | ARR | 前提 |
|------|------|-----|------|
| Year 1 | ~800 | ~$288K | PMF验证 |
| Year 3 | ~8,000 | ~$2.88M | 3应用市场+社群口碑 |
| Year 5 | ~40,000 | ~$14.4M | 品类定义权+搜索 |
| 天花板 | 100-150K | $36-54M | 一人公司赛道极限 |

结论：非千亿级，但$30-50M ARR足够做成健康的SaaS公司。天花板之后增长靠纵向（物流金融/融资聚合）和横向（平台扩展/品牌升级）。

#### Go-to-Market（零预算冷启动）

**第一梯队（立即执行，零现金成本）：**

1. **Shopify/Shopee 应用市场上架**
   - 卖家主动搜索，意图最强，转化率最高
   - 两周内上架，主图亮ROI："省一个运营的钱"
   - 产品入口改：安装→绑店→1分钟跑第一个Agent

2. **跨境电商社群渗透**
   - 雨果网 / AMZ123 / 卖家微信群
   - 创始人每天30分钟回答问题，不是发广告
   - 对话中自然植入："我帮朋友用AI管店，运营时间从3小时降到15分钟"
   - 关键节点：有人问"有没有自动回客服的"时凌镜正好有

3. **YouTube实操（可选，看创始人意愿）**
   - "我用AI运营Shopee，30天利润实盘"系列
   - 公开真实后台数据 > 任何广告文案

**第二梯队：** SEO + 内容营销 + 大卖推荐背书

**第三梯队（PMF之后再烧钱）：** 付费广告

**种子留存机制：**
- 前100用户免费但每周15分钟访谈，痛点=路线图
- Onboarding硬门槛：第一周必须完成绑店→跑3个Agent→看dashboard
- 7天内"哇"时刻：凌晨A5自动调价挽回单，次日用户登录看到通知
- 数据粘性：绑得越深迁移成本越高

#### 一句话定位

> "你的第一个跨境电商员工——AI，24小时在线，不要工资。"

| 片段 | 打中什么 |
|------|---------|
| "你的第一个员工" | 一人公司最大痛点：没人 |
| "AI，24小时在线" | 卖家焦虑：半夜差评 |
| "不要工资" | ROI钩子，让人算账 |

不写的："AgentOS"（卖家不懂）、"全链路"（听吐了）、"让AI帮你管店"（太弱）

SEO备选： "一个人，一台电脑，一个AI——管理你的整个跨境生意。"

#### 给CEO的五个行动项

1. 两周上Shopify/Shopee应用市场
2. 首页做ROI计算器
3. 前100用户当产品经理
4. 定价$30-49/月
5. 100天看自然传播率

---

### 2.5 风险盲点分析 — risk-thinker

#### 风险一：市场风险 — "一人公司"是伪标签（致命）

跨境电商真实画像分三类：兼职代购/scalper（不付费）、独立站 DTC 小老板（忙到没时间调 AI）、试水工厂/贸易商（10-50人小团队而非一人）。真实付费群体是"3-10人小团队"，按"一人"定价和设计会同时失去两类客户：一人付不起，小团队嫌太浅。

**为什么创始人没看到**：技术背景容易把"我自己用着爽"等价为"别人也需要"——但创始人是能搭 Go+Next.js+Postgres 全栈的人，他的"一人"不是真的一人。

#### 风险二：产品风险 — "AI 自主运营"是伪命题（致命）

电商核心决策——选品、定价、广告预算——天然需要人负责。不是技术上限，是责任上限：AI 选品侵权了谁坐牢？AI 调价导致亏本谁赔？卖家要的是 Copilot（更快做对决策），不是 Autopilot（替你决策）。

**为什么创始人没看到**：技术人相信"自动化就是价值"，但电商决策结果导向——挂一次就清退，没有迭代优化空间。

#### 风险三：商业模式风险 — LTV < CAC 概率极高（重要）

最可能定价 $100-300/月，电商 SaaS 平均 retention 6-12 个月，LTV ~$1500-3000。但获客成本极高（卖家分散在 FB 群/微信群/论坛，CPC $5-10，转化率 1-2%，CAC=$250-1000）。LTV/CAC < 3 几乎确定。

**为什么创始人没看到**："先做功能再想变现"在公司模式下是反向的——代码写完了发现客户付不起获客成本。

#### 风险四：竞争风险 — 真正对手是平台巨头（致命）

真正的对手不是 Photoroom/JungleScout，而是 Shopify、Amazon、Airwallex。他们把 AI 功能叠进现有套餐不另收费时，凌镜的存在理由直接归零。防御不是"做得更好"，而是"他们不会做"——但这个假设经不起推敲。

**为什么创始人没看到**：技术人习惯比技术产品，忘了平台巨头的吞并能力。

#### 风险五：技术风险 — AI 准确率是法律问题（致命）

AI 翻译错误 → 产品下架；价格推荐错了 → 每单亏 $20；合规判断遗漏 → 整店被封押金不退。责任归谁？电商圈子极小，出事一次毁灭口碑——一个微信群一转发全部人知道了。

**为什么创始人没看到**：工程师觉得"prompt 优化就好了"，但 AI 只有"低错误率"没有"不出错"版本，出不出事是概率问题，不承认这个概率才是问题。

#### 风险六：运营风险 — 一个人做 SaaS 的元悖论（重要）

产品帮一人公司运营，但公司本身也一人——凌晨出 Bug 用户流失、对接新平台停掉所有开发、客户要定制功能被拒走人、DDoS 防护一个人搞不定。SaaS 60% 成本在交付后，这不会因为产品是 AI 公司就消失。

**为什么创始人没看到**：技术人觉得产品做好就完了，不知道交付后的运营成本等同于另一个全栈工程师。

#### 风险七：法律/合规风险 — 跨境数据三不管（值得关注）

涉及中国《个人信息保护法》《数据安全法》(数据出境限制)、欧盟 GDPR、美国 CCPA。可能连创始人都不知道自己处理了什么数据属于哪个管辖范围。合规成本是固定的——不分 1 个还是 1000 个客户。

#### 风险八：时机风险 — 窗口逻辑变了（重要）

2026 年中，市场已挤满融资 5000 万+ 的竞品、内嵌 AI 的平台巨头、成熟的 LangChain 生态。VC 估值逻辑从 PS 倍数变成 "Profitable or Dead"。创始人的节奏停在"先做好功能再想变现"，但市场已经不给了。

#### 最重要的一条建议

**立刻从 Auto-pilot 转向 Copilot，"AI 自主运营" 转向 "AI 辅助决策"，定价从 $300/mo 提到 $1000-3000/mo，客户从"一人公司"转到"3-10 人跨境小团队"。**

三个原因：
1. Copilot 的法律信任风险比 Autopilot 低两个数量级——AI 给建议人确认，责任在人
2. 3-10 人团队付费意愿和续费率是一人公司的 5-10 倍
3. Autopilot 需要完美才能卖，Copilot 只需要有用就能卖——以当前完成度更适合 Copilot

现有产品大部分可以直接复用，改的是话术、定价、目标客户定义。

---

### 2.6 技术战略分析 — cto-thinker

#### 商业运营的技术准备

**1. 多租户：共享数据库 + 行级 tenant_id**

一人团队管理不了多数据库。选共享 PG 单库，每条记录带 `tenant_id`。

**必须现在就做的事**：给所有业务 model 加 `TenantID string`，在 GORM 层用 scope 自动注入 `WHERE tenant_id = ?`，避免任何人肉忘加。这是唯一不可逆的技术债——越晚加 migration 越痛苦。

隔离靠 app 层 RBAC。运营后台管理员有跨租户视图。

**不做的**：schema-per-tenant、独立数据库实例。100 个付费用户的数据量在 PG 单表百万行级别，远不到瓶颈。

**2. 计费：Stripe + 3-tier 月付**

一人公司搞计量计费是坑——需要实时计数器、余额检查、熔断，至少 2 周。

**推荐**：3 个 tier（基础 $19 / 专业 $79 / 企业 $199），Stripe Checkout 配 Webhook，2 天搞定。

流程：注册 → 14 天免费试用 → 到期弹窗 → Stripe 付款 → Webhook 更新 tier。

Stripe Webhook 只需要处理两个事件：`checkout.session.completed`（激活）和 `customer.subscription.deleted`（降级冻结）。

**3. 安全（够用版）**

- HTTPS：Cloudflare，30 分钟
- JWT：已经有了，确认 refresh token rotation
- Rate limit：改成 per-tenant 粒度
- 数据导出：每个用户能下载全部数据（减少"怕数据锁死"的心理阻力）

**不做的**：SOC2、ISO27001、私有部署。前 100 个付费用户不在乎这些。

#### 架构取舍

**单体够用多久？**

**够用到 50 个活跃租户（500-1000 日活）。**

现在的 Go 单体架构很健康：一个 binary 部署、进程内 EventBus 零网络开销、Agent 链 < 5ms、同一个连接池。

**什么时候拆**：只有一个可靠信号——某个模块的 bug 把整个系统拖垮了（比如 A2 OOM 导致 G0 也挂了）。另一个信号：编译太慢。Go 编译 3-5 秒，所以不会很快出现。

**EventBus 什么时候换？**

现在：进程内 channel-based pub/sub。

**换到 Redis Stream 的信号**：Agent 链需要可靠投递（进程重启不丢事件）或 Agent 链跨多台机器。这两件事在 10 个付费用户阶段都不成立。不换。

**现在不做好以后会后悔的事**

| 要做的事 | 不做的后果 |
|----------|-----------|
| **所有 model 加 tenant_id** | 分租户要全表扫描，可能丢数据 |
| 关键操作审计日志（谁改了什么） | 用户投诉时无法自证 |
| 结构化日志 + 请求 ID | 出问题找不到是哪个租户 |

**API 版本化（`/api/v1/`）和 RequestID middleware 你们已经有了**，继续保持。

#### 多平台策略

**PlatformAdapter 高效扩展**

每个新平台实现 N 个方法，80% 的代码重复。

**解决方案**：把平台差异抽象成配置，不是代码。

```
platform-adapter/
├── base/           # 通用 HTTP 调用、重试、限速、OAuth 刷新、字段映射引擎
├── ozon/adapter.go + config.yaml
├── shopee/...
├── amazon/...
└── tiktok/...
```

```yaml
# config.yaml 示例
platform: amazon
auth_type: aws_sigv4
endpoints:
  list_orders: /orders/v0/orders
field_mappings:
  product.title: "ItemName"
  order.status.shipped: "Shipped"
```

新平台接入：写 config.yaml（1-2 小时）+ base adapter 跑通 CRUD（4-8 小时）。只有特殊逻辑（Amazon SP-API 签名、TikTok 内容审核）才写定制代码。

**平台优先级**

| 平台 | 差异 | 优先级 |
|------|------|--------|
| Shopee/Ozon/Lazada | 中等 | 已有 |
| **Amazon** | **高（SP-API 复杂）** | **最高** |
| TikTok Shop | 高（内容审核） | 中 |
| Temu/AliExpress | 中 | 低 |

**Amazon 必须第一个做**。找跨境 SaaS 的用户大概率在用 Amazon，没有 Amazon 适配器获客转化率折半。

**维护策略**

规则：月活 < 3 用户 → 不维护。API 连续失败超阈值 → 自动标记"不健康"。30 天无用户且 80% 失败 → 发弃用公告。

CI 写一个 `go test` 用沙箱调一轮 CRUD，挂了就知道。

#### AI 去幻想策略

**核心原则**

不要试图消除幻觉（做不到）。让 AI 给建议，人做决策，系统执行并被审计。

**决策可信度三级**

**Level 1：全自动（仅记录）**
- 图片裁切/背景替换、属性补全、描述润色
- 库存预警通知（不自动下单）
- 竞品价格收集（不自动改价）
- 不经过人工，直接执行

**Level 2：人工复核（AI 出建议 + 理由，用户点确认）**
- 定价调价（显示：当前价、建议价、竞品均价）
- 广告预算调整（显示：ROI、历史数据、预估效果）
- 采购补货下单（显示：库存、报价、到货时间）
- Listing 发布（显示：改动 diff、理由）
- 弹窗确认，7 天无响应自动忽略（不是自动执行）

**Level 3：手动参考（纯展示）**
- 选品推荐、供应商推荐
- 市场趋势分析
- Agent 周报
- 无门禁，纯展示

**划分逻辑**：跟钱直接相关的（定价、采购、广告）必须人确认。有风险但可回滚的（描述修改）展示 diff 后 AI 可执行。信息类随意。

**验证层（不靠"让 AI 更准"）**

1. **数值约束**：AI 输出的价格在（成本价 x 1.1, 成本价 x 5）外，直接拒
2. **格式约束**：JSON 输出必须符合 schema，parse 失败不执行
3. **上下文一致性**：同一商品不同 Agent 的描述不一致时标记警告
4. **用户反馈闭环**：用户点"不合理" → 记录 → 定期分析 hallucination pattern

**兜底**

所有 Level 2 操作执行前：快照当前状态。操作后 30 分钟内用户可一键回滚。

#### 运维方案（一人团队版）

**推荐托管方案**

| 组件 | 方案 | 月费 |
|------|------|------|
| 应用 | 一个 VPS 8C16G，Docker Compose | ~$40 |
| 数据库 | Supabase / Render Postgres 托管 | ~$20 |
| 对象存储 | AWS S3 / 阿里 OSS | ~$5 |
| Redis | Upstash 免费版 | $0 |
| CDN/HTTPS | Cloudflare 免费版 | $0 |
| 监控 | 现有 Grafana + UptimeRobot | $0 |
| 日志 | 文件 + journalctl，疼了买 Grafana Cloud | $0 |
| CI/CD | GitHub Actions 免费额度 | $0 |

**不要碰 K8s**。K8s 本身就是一个全职运维工作。单机扛不住时先升级 VPS 到 16C32G（~$80/月），又可以撑一年。

**不用 Serverless**：Agent 链可能执行几十秒到几分钟，不适合函数计算的超时限制和冷启动。

**监控到什么程度**

够用标准：
1. **存活性**：UptimeRobot 每 5 分钟 `/api/health`，挂了发邮件
2. **错误率**：Sentry 捕获 500 错误，按频率报警
3. **磁盘**：使用率 > 85% 报警（日志写爆是常见死法）
4. **业务 KPI**：每日新增用户、活跃用户、Agent 执行成功率

不做的：APM tracing、慢查询分析、CPU 内存曲线（偶尔看一眼就行）。

**可以外包的**

| 自己做 | 买现成 |
|--------|--------|
| 客服系统 | Intercom / Crisp |
| 帮助文档 | Notion / GitBook 公开页面 |
| 官网 Landing | Framer / Webflow |
| 支付合规 | Stripe / Lemon Squeezy |
| LLM 代理/缓存 | Portkey / Helicone |
| SEO | Agent 写博客，Cloudflare Web Analytics |

#### 总结路线图

```
Now → 1个月             1-3个月             3-6个月             6-12个月
────────────────────────────────────────────────────────────────────────
加 tenant_id           上线 Stripe 计费      Amazon 适配器       Redis Stream
执行门禁 Level 2-3     开放免费试用          更多平台（按用户）  根据需要拆 Agent OS
Docker 部署优化        第一个付费用户         用户反馈驱动优化    VPS 升配
```

**一句话**：你们的代码基础已经很好了。现在唯一需要的是：加 tenant_id，接 Stripe，然后去卖。技术欠的债都可以用钱还，但没用户欠的债没法还。

---

## 附录：核心分歧汇总

### A. 关于目标用户

| 观点 | 来源 |
|------|------|
| 一人公司是正确的赛道，这是低端颠覆的机会 | CEO |
| 一人公司不是问题，但付费意愿低，真正付费的是3-10人小团队 | 风险官 |
| 一人公司是卖家真实状态，$29起可接受 | CFO |
| 一人卖家时间被切碎，最需要聚合型工具 | PM |

### B. 关于 AI 角色

| 观点 | 来源 |
|------|------|
| 自主运营是最终愿景，逐步放开 | CEO |
| 必须从 Copilot 开始，这是法律和信任问题不是技术问题 | 风险官 |
| AI 辅助决策 + 三级门禁是目前最务实的方案 | CTO |
| 卖家最不放心的就是 AI 碰钱（定价/广告），最放心的反而是客服 | PM |

### C. 关于产品方向

| 观点 | 来源 |
|------|------|
| 先打磨 14 个 Agent 的智力，实现更深的自动化 | CEO（隐含） |
| 先做客服聚合 + 利润引擎 + Dashboard，Agent 智力可以迭代 | PM |
| 技术基础很好，先加 tenant_id 再卖，不要过度设计 | CTO |
| 从 Autopilot 转向 Copilot，一句话重定位 | 风险官 |
| 两周上应用市场，前100用户做产品经理 | 市场 |

---

*报告完*
