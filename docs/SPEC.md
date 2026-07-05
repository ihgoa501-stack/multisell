# 凌镜 LingMirror — 项目规格文档 (Spec)

> 技术名：MultiSell · 版本 v0.3.0.0
> 日期：2026-07-05

---

## 1. 目标

### 我们构建什么

凌镜是一个跨境电子商务 AI Agent 操作系统（AgentOS）。它不是传统的 ERP 或 SaaS 工具，而是一个 Owner（非技术业务决策者）可以信赖的电商经营平台，其核心价值是：

1. **让 Owner 看清经营全貌** — 商品、订单、库存、利润、风险一目了然
2. **让 AI Agent 辅助决策** — Agent 提供有数据支撑的建议，Owner 做最终决定
3. **安全门禁封装高风险操作** — 不擅自发布、改价、删数据除非 Owner 明确批准
4. **审计和追溯** — 每一个关键操作都有记录

### 用户

- **主要用户**：跨境电商从业者，非技术的 Owner（老板/运营管理者）
- **Agent 用户**：15 个内置 AI Agent（A1~A11 + G0~G3 + M1），各负责一个经营环节

### 成功标准

```
Owner 打开凌镜可以确信：
- 系统不会静默修改价格、库存、订单、资金或外部 Listing
- Agent 的建议清晰可见、可解释、可审核
- 高风险操作需要明确批准
- 已批准的操作可追溯至用户、原因、目标和审计记录
- 失败可见且有后续步骤
- Mock/Sandbox/Read-Only/Production 模式清晰分离
```

---

## 2. 技术栈

| 层 | 技术 | 版本 |
|---|------|------|
| 后端语言 | Go | 1.25 |
| Web 框架 | Gin | latest |
| ORM | GORM | latest |
| 数据库 | PostgreSQL | 15 |
| 前端框架 | Next.js | 16 |
| 前端 UI | React | 19 |
| UI 组件库 | Ant Design | 6 |
| 客户端状态 | Zustand | 5 |
| 服务端状态 | TanStack React Query | 5 |
| 类型安全 | TypeScript | latest |
| 容器化 | Docker + docker-compose | latest |
| 代理 | Nginx | latest |
| 监控 | Prometheus + Sentry | latest |

### 基础设施核心

| 组件 | 包 | 用途 |
|---|---|---|
| Event Bus | `internal/platform/eventbus/` | pub/sub 事件总线 |
| Command Dispatcher | `internal/platform/command/` | Agent 决策到领域服务桥接 |
| Scheduler | `internal/platform/scheduler/` | 定时任务（5min - 6hr） |
| ToolBridge | `internal/platform/toolbridge/` | 插件驱动工具执行桥接 |
| WebSocket Hub | `internal/realtime/` | 实时推送 AI 流式输出 |
| AI Orchestrator | `internal/ai/` | LLM 编排、聊天、追踪 |
| Agent Registry | `internal/agent/` | Agent 注册和执行入口 |
| AIOS | `internal/aios/` | AgentOS 基础设施（toolregistry/llmgateway/runtime 等） |

---

## 3. 命令

### 后端

```bash
# 基础设施
docker compose up -d db              # 启动 PostgreSQL
docker compose up -d                 # 启动全部服务

# 后端开发
cd backend-go
go run cmd/server/main.go            # 开发服务器
go test ./...                        # 全部测试
go test -v ./internal/domain/order/  # 单包测试
go vet ./...                         # 静态分析
go build -o bin/server cmd/server/main.go  # 编译

# 烟雾测试
./scripts/smoke_test_setup.sh
./scripts/smoke_test.sh

# 生产部署（macOS 交叉编译）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/server cmd/server/main.go
```

### 前端

```bash
cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000  # 开发模式
npm run build                                      # 生产构建
npm test                                           # Vitest 测试
npm run lint                                       # ESLint 检查
npx playwright test                                # E2E 测试
```

### API 端点

前缀: `/api/v1`，认证: `Authorization: Bearer <jwt>`
响应: `{"code":0,"message":"ok","data":...}`
健康检查: `GET /api/health`

---

## 4. 项目结构

```
multisell/
├── backend-go/                      # Go 后端 ← 活跃栈
│   ├── cmd/server/main.go           # 入口
│   ├── internal/
│   │   ├── auth/                    # JWT 认证
│   │   ├── rbac/                    # 角色权限
│   │   ├── ai/                      # AI 编排
│   │   ├── agent/                   # Agent 注册 & 实装
│   │   ├── agentos/                 # AgentOS 驾驶舱
│   │   ├── aios/                    # AIOS 基础设施
│   │   ├── common/                  # 工具（分页/排序）
│   │   ├── config/                  # 配置管理
│   │   ├── domain/                  # 所有业务领域模块 ← 核心
│   │   ├── httpx/                   # 路由注册 + 中间件
│   │   ├── platform/                # 基础设施（eventbus/command/scheduler/toolbridge）
│   │   ├── realtime/                # WebSocket hub
│   │   └── response/                # 统一响应格式
│   ├── migrations/                  # SQL 迁移（+ 种子数据）
│   ├── scripts/                     # 烟雾测试
│   └── configs/config.yaml          # 配置
├── frontend-next/                   # Next.js 前端 ← 活跃栈
│   ├── src/
│   │   ├── app/                     # Next App Router
│   │   │   ├── (auth)/login/       # 登录页
│   │   │   └── (main)/             # 认证后页面
│   │   ├── components/              # 共享 UI 组件
│   │   ├── lib/                     # API client, auth, query
│   │   ├── stores/                  # Zustand 状态
│   │   └── config/menu.ts          # 侧边栏菜单
│   └── e2e/                         # Playwright E2E
├── deploy/                          # Docker + Nginx + Prometheus
├── chrome-extension/                # 凌镜选品助手扩展
├── docs/                            # 文档
│   ├── governance/                  # 治理文档（宪法/协议/契约）
│   │   ├── PLATFORM_CONSTITUTION.md
│   │   ├── OWNER_FIRST_PROTOCOL.md
│   │   ├── AGENT_DEVELOPMENT_PROTOCOL.md
│   │   └── KERNEL_CONTRACTS.md
│   ├── adr/                         # 架构决策记录（ADR-001~006）
│   └── ops/                         # 运维手册
├── VERSION                          # v0.3.0.0
├── AGENTS.md                        # 跨 Agent 项目指引
└── CLAUDE.md                        # Claude Code 指引
```

### 关键架构决策

- 每个领域模块 `internal/domain/xxx/` 遵循 `routes.go → handler.go → service.go → model.go` 模式
- 模块注册在 `internal/httpx/router.go` 的 JWT 保护组 `/api/v1`
- 响应通过 `response.Success/Error/Paginated/InternalError` 统一输出
- 所有非 CRUD 模块的外部操作通过 EventBus + Scheduler 调度
- Agent 通过 `internal/aios/toolregistry/` 注册和使用工具
- 前端 API 调用通过 `apiClient`（含 JWT 刷新 + 请求去重）

### 当前活跃领域模块（40+）

```
product sku category brand inventory price platform store
listing listingtask decision finance settlement platformfee
order orderimport shipping logistics aftersales
sourcing sourcing1688 supplier purchase allocation
exceptions support notification search report
dashboard settings imagegen importbatch operationlog
exchange-rate product-analysis integration platform-integration
candidate completeness profit loop mock owner
actionpolicy trustscore entropy evolution agentrule
```

---

## 5. 代码风格

### Go

```go
// 模块模式 template — internal/domain/example/routes.go
package example

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
    h := &Handler{service: svc}
    g := rg.Group("/examples")
    g.GET("", h.List)
    g.GET("/:id", h.Get)
    g.POST("", h.Create)
    g.PUT("/:id", h.Update)
    g.DELETE("/:id", h.Delete)
}

// service.go — 业务逻辑层
type Service struct {
    db  *gorm.DB
    log *slog.Logger
}

// Handler 层仅做请求响应映射，不做业务逻辑
// 分页: common.ParsePagination(c), common.ParseSort(c)
// 测试: dbtest.NewDB(t, &Model{}) — 内存 SQLite，支持 t.Parallel()
```

### TypeScript / React Next.js

```typescript
// API 调用通过 apiClient
import { apiClient } from '@/lib/api-client';

// 服务端状态用 React Query
const { data } = useQuery({
  queryKey: ['examples'],
  queryFn: () => apiClient.get('/v1/examples'),
});

// 全局状态用 Zustand
const useAppStore = create<AppState>()((set) => ({ ... }));

// 组件风格: Ant Design 函数组件 + PageContainer
export default function ExamplePage() {
  return (
    <PageContainer>
      <Table columns={columns} dataSource={data} />
    </PageContainer>
  );
}
```

### 命名惯例

| 层 | 惯例 | 示例 |
|----|------|------|
| 路由文件 | `routes.go` | `internal/domain/order/routes.go` |
| 处理器 | `handler.go` | `internal/domain/order/handler.go` |
| 服务 | `service.go` | `internal/domain/order/service.go` |
| 模型 | `model.go` | `internal/domain/order/model.go` |
| 前端路由目录 | kebab-case | `src/app/(main)/product-analysis/` |
| API 路径 | kebab-case plural | `/api/v1/product-analysis` |
| 数据库表 | snake_case | `product_analysis` |
| Go 类型 | CamelCase | `ProductAnalysisService` |

---

## 6. 测试策略

### 框架

| 层 | 框架 | 位置 |
|----|------|------|
| Go 单元测试 | `testing` 标准库 | 同目录 `*_test.go` |
| 前端单元测试 | Vitest | `frontend-next/src/` |
| E2E | Playwright | `frontend-next/e2e/` |
| 烟雾测试 | Bash | `backend-go/scripts/smoke_test.sh` |

### 测试原则

1. **核心+钱路径**（AI/Agent/AgentOS + Order/Finance/Shipping/Settlement）— 先用 `/plan`，再写测试，再实现
2. **CRUD 模块**（Product/Category/Brand 等）— 按现有 pattern 快速实现
3. **新加测试用 `dbtest.NewDB(t, &Model{})`** — 内存 SQLite，支持并行（`t.Parallel()`）
4. **非平凡逻辑必须留一个可执行检查**（`demo()` 自检或一个 `*_test.go`）
5. **烟雾测试覆盖端到端管道**（10 步骤: 健康检查 → Agent 执行 → 数据库验证 → 审计追溯）

### 基准测试命令

```bash
# 后端 — 必须在推前通过
go test ./...        # 全绿
go vet ./...         # 无 vet 输出

# 前端 — 必须在推前通过
npm test             # 全绿
npm run build        # 构建成功

# 已知不通过（不阻塞）
npm run lint         # 已知 ~34 problems（12 errors + 22 warnings），非阻塞
```

---

## 7. 边界

### ✅ 始终要做

- 修改代码前检查 `git status --short`
- 运行受改动影响的最小测试集
- 新领域模块遵循 `routes → handler → service → model` 模式
- 所有 API 受 JWT 保护（非 auth 端点）
- API 路径使用 `/api/v1/` 前缀
- 响应使用 `response.Success/Error/Paginated/InternalError` 统一格式
- 高风险函数添加 `// ponytail:` 注释说明取舍和升级路径
- 遵循 Owner-First 开发协议：用业务语言汇报
- 修改前确认是活跃栈（Go/Next.js），非遗留栈（Python/Vue）

### ⚠️ 要先问

- **数据库 schema 变更** — 加表、加列、改迁移文件
- **新依赖** — 加 Go module 或 npm 包
- **CI/CD 变更** — 改 workflow、deploy config
- **认证/授权变更** — 改 JWT 逻辑、RBAC 规则
- **AI 模型变更** — 切换提供商、改 prompt 架构
- **平台集成写回** — 向 Ozon/Shopee 等外部平台发布、改价、改库存
- **审计日志结构变更** — 改 operationlog schema 或脱敏规则
- **跨越系统层**（Kernel ⇄ Domain ⇄ Agent ⇄ Integration ⇄ UI ⇄ Docs）

### ❌ 永远不要做

- 提交密钥、Token、API Key 到 git
- 修改 `vendor/`、`node_modules/`、`.git/` 目录
- 未经审批向生产环境外部平台写数据
- 删除未通过审核的测试（修复而不是删除）
- 在遗留栈（`backend/` Python、`frontend/` Vue）中做新功能
- 直接使用 `http.Request.Body` 而不脱敏就写入审计日志
- 在 CI 输出或日志中明文输出密钥

---

## 8. 北极星与当前优先级

### 北极星指标

```
商品能不能卖，系统能说清楚；
订单和履约会不会亏，系统能说清楚；
高风险动作是否可执行，Owner 能看懂并审批。
```

### 执行节奏

按改动模块选择：

| 模块 | 节奏 |
|------|------|
| **AI/Agent/AgentOS + Commerce（钱）** | `/plan` → 写测试 → 小心实现 |
| **所有其他 CRUD/UI** | 按现有 pattern 快速实现 |

### 当前 P0（执行门禁收口）— 已全部完成（2026-07-05）

| # | 项目 | 状态 | 交付物 |
|---|------|------|--------|
| P0 | EventBus/Scheduler 生命周期验证 | ✅ 已完成 | 测试覆盖 start→publish→receive→stop→no-more-deliveries |
| P0 | 统一执行门禁 `/ai/actions/:id/execute` | ✅ 已完成 | 审计日志 + 幂等守卫 + RBAC 权限路由 |
| P0 | 审批/执行绑定登录用户 + RBAC | ✅ 已完成 | ActionDecisionInput 移除 operator；approve/execute 需 `ai.action` 权限 |
| P1 | 外部平台写安全（dry-run/sandbox/approval） | ✅ 已完成 | ExecutionMode 类型 + PublishToOzon dry-run 守卫 |
| P1 | 审计日志敏感字段脱敏 | ✅ 已完成 | `operationlog.RedactSensitive` — Log 和 LogStructured 自动脱敏 |
| P1 | 前端统一高风险动作确认 UX | ✅ 已完成 | HighRiskConfirmDialog 组件（已集成到 Owner 工作台）|

---

## 9. Agent 系统速览

| ID | 名称 | Squad | 关键决策点 | 自主性 | 周期 |
|----|------|-------|-----------|--------|------|
| A1 | 选品助理 | insight | product_scout, supplier_discovery | advisory | 按需 |
| A2 | 商品优化师 | insight | listing_optimize | advisory | 按需 |
| A3 | 广告分析师 | insight | acos_analysis | advisory | 1h |
| A4 | 客服助理 | ops | auto_reply | guided | 5min |
| A5 | 库存助理 | ops | stock_alert | guided | 15min |
| A6 | 利润看护 | ops | profit_watch | supervised | 1h |
| A7 | 合规专员 | ops | compliance_check | supervised | 2h |
| A8 | 选品盈利分析 | insight | sourcing_recommend | advisory | 按需 |
| A9 | 批量运维 | ops | batch_price_update | guided | 按需 |
| A10 | 物流运费引擎 | ops | carrier_compare | guided | 按需 |
| A11 | 售后管理 | ops | return_analysis | guided | 按需 |
| G0 | 系统健康 | governance | system_health | supervised | 5min |
| G1 | 驾驶舱 | governance | dashboard_overview | advisory | 5min |
| G3 | 折扣风控 | governance | discount_risk_check | supervised | 30min |
| M1 | 代谢评分 | governance | excretion_scoring | supervised | 1h |

### 管道链

```
A5 stock_alert (red)     → G3 discount_risk_check
G3 discount_risk_check (block) → A6 profit_watch
A6 profit_watch (loss/threshold) → A2 listing_optimize
G0 system_health (anomaly > 3) → G1 dashboard_overview
```

### 动作风险等级

| 等级 | 示例 | 需审批 |
|------|------|--------|
| low | 选品调研、关键词研究 | 否 |
| medium | 库存预警、利润检查 | 是 |
| high | 利润看护、折扣检查 | 是 |

---

## 10. 三个业务闭环

### 闭环一：选品→上架

```
候选商品 → 完善度检查 → 成本/物流/平台费/利润计算 → Listing 建议
→ Owner 审批 → 受控上架任务 → 结果回顾
```

### 闭环二：履约→结算

```
订单 → 库存和物流选择 → 运费快照 → 结算和利润检查
→ 异常检测 → Agent 建议 → Owner 审批或手动处理
```

### 闭环三：Agent OS 自运营

```
Agent 定时检查 → 事件链通知 → Owner 驾驶舱总览
→ 高风险动作审批 → 审计记录 → 复盘改进
```

---

## 11. 开放问题

1. ~~**当前 P0 执行门禁收口的第一迭代范围** — 三件全做还是一门禁一迭代？~~ ✅ 一期全部完成
2. ~~**前端展示优先还是后端逻辑优先？** — 后端已有部分骨架，前端高风险确认 UX 接近空白。~~ ✅ 后端全做完后，前端组件已创建并集成到 Owner 工作台
3. ~~**烟雾测试的实际执行环境** — 需要真实数据库还是可以用内存 SQLite 跑简化版？~~ 尚未验证
