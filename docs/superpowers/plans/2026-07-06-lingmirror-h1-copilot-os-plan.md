# 凌镜 H1 路线图 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> Workstream M1 must complete before M2/M3/M4 start. M5 requires at least one of M2/M3 to be complete first.
> M6 is fully parallel with all other workstreams.

**Goal:** 把凌镜从"可信 Copilot 原型"（v0.4.1.0）推进到 Owner 可信任并每天使用的跨境电商经营 Copilot。

**Architecture:** 维持现有 Go/Next.js 技术栈。六个工作流按 M1 → (M2 ‖ M3 ‖ M4 ‖ M6) → M5 依赖顺序推进，不引入新语言/框架。

**Tech Stack:** Go 1.25, Gin, GORM, PostgreSQL 15, Next.js 16, React 19, Ant Design 6, TypeScript, TanStack React Query 5, Zustand 5

## Global Constraints

- 新增 domain 模块遵循 `routes.go / handler.go / service.go / model.go` pattern
- 审批操作必须写审计日志 + 绑定登录用户 + RBAC 检查
- Agent 不能直接写数据库，只能通过审批门禁执行业务命令
- 所有 mutation API 必须走 RBAC + 审计中间件
- 错误处理：不静默吃掉 DB 错误
- 测试：Go `dbtest.NewDB(t, &Model{})` + Vitest 前端 + Playwright E2E
- 后端测试全绿通过再提交：`go test ./...`
- 前端构建通过再提交：`npm run build`

---

## 工作流依赖图

```
M1（可信基础）← 唯一前置，必须先完成
  ├──→ M2（商品闭环）   ← 可并行
  ├──→ M3（订单利润）   ← 可并行
  ├──→ M4（平台受控）   ← 可并行
  └──→ M6（运营化）     ← 可与全部并行
         ↓
M5（Workflow 平台）← 依赖至少一个闭环（M2/M3其一）跑通
```

## 文件改动总览

### 新建模块/文件

| 工作流 | 文件 | 用途 |
|--------|------|------|
| M1 | `backend-go/internal/domain/reliability/` | 可观测性/运行时状态 |
| M4 | 在 `integrations/` 内补齐 | 各平台 adapter |
| M5 | `frontend-next/src/app/(main)/workflows/` | Workflow 管理前端 |
| M5 | `backend-go/internal/domain/workflow/` (增强) | Workflow 引擎 |
| M6 | 在 `dashboard/` 内增强 | 运营仪表盘后端 |

### 修改已有模块

| 工作流 | 模块 | 改动 |
|--------|------|------|
| M1 | `operationlog/` | 审计覆盖补全 |
| M1 | `approval/` | 审批门禁全面接入 |
| M1 | 各 handler | 覆盖缺失的 RBAC/审计 |
| M2 | `completeness/` | 完整度引擎增强 |
| M2 | `profit/`, `listing/`, `listingtask/` | 商品闭环 |
| M3 | `orderimport/`, `order/`, `profit/` | 订单闭环 |
| M3 | `exceptions/`, `notification/` | 异常处理 |
| M4 | `integrations/types.go` | 平台适配器增强 |
| M4 | `integrations/service.go` | 写回门禁 |
| M5 | `workflow/`, `aios/` | 工作流引擎增强 |
| M6 | `dashboard/`, `owner/` | 仪表盘真实数据 |
| M6 | 前端 `(main)/` layout | 合并 Workbench→Dashboard |

---

## M1：可信基础收口

**依赖：** 无（最高优先级）
**并行度：** 内部子任务可并行

---

### Task M1.1: 修复测试/lint/build 已知问题

**Files:**
- Modify: `backend-go/internal/domain/supplier/handler_test.go`
- Test: 修复预存失败的测试

**Interfaces:** 无 API 变更

- [ ] **Step 1: 诊断失败的测试**

```bash
cd backend-go && go test ./internal/domain/supplier/ -run TestHandler_GetSupplierComparison -v
```

观察 500 vs 400 差异的原因——是 handler 层状态码变更还是测试期望值过时。

- [ ] **Step 2: 修复测试期望**

如果是 handler 行为变更（400 比 500 更合理），则更新测试期望：
```go
// 原：assert.Equal(t, 500, w.Code)
assert.Equal(t, 400, w.Code)
```

如果是 handler 逻辑问题，修复 handler 层逻辑。

- [ ] **Step 3: 全量验证**

```bash
cd backend-go && go test ./... 2>&1 | tail -5
cd backend-go && go vet ./... 2>&1 | tail -5
```

Expected: `ok` + 无 vet 输出

- [ ] **Step 4: 修复前端构建（若当前 node_modules 缺失）**

```bash
cd frontend-next && npm ci && npm run build 2>&1 | tail -10
```

如有失败，逐项修复。

**验收条件：** `go test ./...` 全绿，`go vet ./...` 无输出，`npm run build` 通过

---

### Task M1.2: 统一项目状态文档口径

**Files:**
- Modify: `docs/PROJECT_STATUS.md` — 与当前代码一致
- Modify: `docs/INDEX.md` — 验证所有引用有效
- Modify: 其它发现有死引用的文档

**Interfaces:** 纯文档变更

- [ ] **Step 1: 扫描死引用**

```bash
grep -rn '\.\./docs/' backend-go/ --include="*.go" | head -20
# 检查 docs/INDEX.md 中每个文件链接是否有效
```

- [ ] **Step 2: 更新 PROJECT_STATUS.md**

更新版本号、日期、当前完成模块清单、未完成事项。确保与代码一致。

- [ ] **Step 3: 提交文档修复**

```bash
git add docs/
git commit -m "docs: sync PROJECT_STATUS and INDEX to current code state"
```

**验收条件：** 文档无死引用，API 存量/模块清单准确，状态日期为当前

---

### Task M1.3: 生产可观测性

**Files:**
- Create: `backend-go/internal/domain/reliability/routes.go`
- Create: `backend-go/internal/domain/reliability/handler.go`
- Create: `backend-go/internal/domain/reliability/service.go`
- Create: `backend-go/internal/domain/reliability/model.go`
- Modify: `backend-go/internal/httpx/router.go` — 注册可靠性路由

**Interfaces:**
- `GET /api/v1/reliability/agent-status` — 所有 Agent 运行状态（最后心跳、是否暂停、失败数）
- `GET /api/v1/reliability/llm-cost` — LLM token 消耗聚合（今日/本周/本月）
- `GET /api/v1/reliability/failures` — 失败操作列表（带原因、重试状态）

- [ ] **Step 1: 创建 reliability 模块**

```go
// backend-go/internal/domain/reliability/model.go
package reliability

import (
    "time"
    "gorm.io/gorm"
)

// AgentStatus 表示 Agent 运行时状态
type AgentStatus struct {
    ID           uint      `gorm:"primaryKey"`
    AgentID      string    `gorm:"uniqueIndex;size:100"`
    AgentName    string    `gorm:"size:255"`
    Squad        string    `gorm:"size:100"`
    Status       string    `gorm:"size:50"` // running, paused, stopped, error
    LastHeartbeat time.Time
    FailedCount  int
    ErrorReason  string
    IsPaused     bool
}

// LLMCostRecord token 消耗记录
type LLMCostRecord struct {
    ID         uint      `gorm:"primaryKey"`
    AgentID    string    `gorm:"index;size:100"`
    ModelName  string    `gorm:"size:100"`
    InputTokens  int
    OutputTokens int
    CostUSD    float64
    CreatedAt  time.Time
}
```

- [ ] **Step 2: 实现 service + handler 层**

```go
// handler.go
func (h *Handler) GetAgentStatus(c *gin.Context) {
    statuses, err := h.svc.GetAgentStatus(c.Request.Context())
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.Success(c, statuses)
}

func (h *Handler) GetLLMCost(c *gin.Context) {
    period := c.DefaultQuery("period", "today") // today, week, month
    cost, err := h.svc.GetLLMCost(c.Request.Context(), period)
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.Success(c, cost)
}
```

- [ ] **Step 3: 注册路由**

在 `router.go` 中 `/api/v1` 受保护组下添加：
```go
r := domain.NewReliabilityRouter(reliabilitySvc)
r.Register(v1)
```

- [ ] **Step 4: 集成 Agent 心跳上报**

在 `internal/agent/` 的执行入口处，每次 Agent 执行结束时更新 `AgentStatus` 最后心跳和失败计数。

- [ ] **Step 5: 测试**

```bash
cd backend-go && go test ./internal/domain/reliability/... -v
```

**验收条件：** Owner 能看到所有 Agent 运行状态、LLM token 消耗聚合、失败操作及原因

---

### Task M1.4: HighRiskConfirmDialog 全接入

**Files:**
- Modify:  `frontend-next/src/app/(main)/owner/page.tsx` — Owner 工作台 approve 按钮接确认弹窗
- Modify:  `frontend-next/src/app/(main)/actions/page.tsx` — AI action 页面 execute 按钮接确认弹窗
- Modify:  `frontend-next/src/app/(main)/agentos/page.tsx` — AgentOS 审批接确认弹窗
- Test: 现有 `HighRiskConfirmDialog.test.tsx` 验证通过

**Interfaces:**
- `HighRiskConfirmDialog` 已存在组件，需在三个页面集成
- Props: `{ action: ActionProposal, onConfirm, onCancel, mode: 'sandbox'|'production' }`

- [ ] **Step 1: 确认现有组件 API**

```bash
grep -rn "HighRiskConfirmDialog" frontend-next/src/ --include="*.tsx" | head -10
```

- [ ] **Step 2: Owner 工作台接入**

```tsx
// owner/page.tsx — 在审批操作位置
import { HighRiskConfirmDialog } from '@/components/ui/HighRiskConfirmDialog'

// 在审批按钮点击时渲染弹窗
const [confirmAction, setConfirmAction] = useState<ActionProposal | null>(null)

{confirmAction && (
  <HighRiskConfirmDialog
    action={confirmAction}
    mode={env}
    onConfirm={() => { /* 执行审批 */ setConfirmAction(null) }}
    onCancel={() => setConfirmAction(null)}
  />
)}
```

- [ ] **Step 3: AI action 页面 + AgentOS 页面同样接入**

重复 Step 2 的模式在 `actions/page.tsx` 和 `agentos/page.tsx` 中接入。

- [ ] **Step 4: 运行前端测试**

```bash
cd frontend-next && npm test -- --run 2>&1 | tail -20
```

**验收条件：** 价格/库存/发布/订单状态变更在三个页面都走确认弹窗，弹窗显示风险等级/目标/前后值/环境模式/审计去向

---

### Task M1.5: 审批+审计全覆盖

**Files:**
- Modify: `backend-go/internal/domain/price/routes.go` — 价格变更加审批
- Modify: `backend-go/internal/domain/inventory/handler.go` — 库存变更加审批
- Modify: `backend-go/internal/domain/platform/handler.go` — 发布操作加审批
- Modify: `backend-go/internal/domain/order/handler.go` — 订单状态变更加审批
- Modify: `backend-go/internal/httpx/middleware/audit.go` — 审计覆盖率检查
- Test: 新增审计覆盖测试

**Interfaces:**
- 已有 `approval.Service` + `operationlog.Service`
- 每个 mutation handler 需调用 `approvalSvc.RequireApproval(ctx, action)` + `oplogSvc.Log(ctx, record)`

- [ ] **Step 1: 审计所有 mutation API 的现有覆盖率**

```bash
grep -rn "response\.Success\|response\.Paginated" backend-go/internal/domain/ --include="*.go" | grep -v "_test.go" | wc -l
# 与登录审计日志的数量对比
```

- [ ] **Step 2: 补齐缺失审计**

对每个发现缺失 mutation 审计的 handler，添加：
```go
_ = oplogSvc.Log(c.Request.Context(), operationlog.Log{
    Action:      op, // 已有的动作枚举
    ResourceType: "price", // 具体资源
    ResourceID:  strconv.FormatUint(req.PriceID, 10),
    Detail:      fmt.Sprintf("价格变更: %.2f → %.2f (SKU: %d)", oldP, newP, skuID),
})
```

- [ ] **Step 3: 补齐缺失审批门禁**

对需要审批的 mutation（价格/库存/发布/订单状态），添加审批检查：
```go
if err := approvalSvc.RequireApproval(c.Request.Context(), approval.Request{
    Action:      "price_change",
    RiskLevel:   "high",
    ResourceID:  strconv.FormatUint(priceID, 10),
    RequestorID: auth.GetUserID(c),
}); err != nil {
    response.Error(c, http.StatusForbidden, "此操作需要审批: "+err.Error())
    return
}
```

- [ ] **Step 4: 运行测试验证**

```bash
cd backend-go && go test ./... 2>&1 | tail -10
```

**验收条件：** 所有价格/库存/发布/订单状态变更有审计记录、有身份绑定、有审批检查；测试全绿

---

## M2：商品经营闭环

**依赖：** M1.4 + M1.5 已完成
**并行度：** 与 M3、M4、M6 完全并行

---

### Task M2.1: 候选商品完整度引擎增强

**Files:**
- Modify: `backend-go/internal/domain/completeness/service.go` — 增强计算
- Modify: `backend-go/internal/domain/completeness/model.go` — 增加维度
- Modify: `backend-go/internal/domain/profit/service.go` — 利润测算集成
- Modify: `backend-go/internal/domain/logistics/service.go` — 物流费模板
- Modify: `backend-go/internal/domain/platformfee/service.go` — 平台费模板
- Test: 对应 `*_test.go`

**Interfaces:**
- `func (s *Service) CalculateFullCompleteness(candidateID uint) (*CompletenessReport, error)`
- `CompletenessReport` 扩展至含：毛利预估、物流费预估、平台费预估

- [ ] **Step 1: 定义完整度报告结构**

```go
// model.go
type CompletenessReport struct {
    CandidateID       uint
    BaseInfoScore     float64 // 基础信息完整度 0-1
    CostScore         float64 // 成本完整度
    LogisticsScore    float64 // 物流完整度
    PlatformFeeScore  float64 // 平台费完整度
    ProfitScore       float64 // 利润可测算
    OverallScore      float64 // 综合
    MissingFields     []string
    EstimatedProfit   *float64 // 预估毛利（如果有完整数据）
    EstimatedLogistics *float64
    EstimatedFees     *float64
}
```

- [ ] **Step 2: 集成利润估算到完整度引擎**

```go
// service.go
func (s *Service) CalculateFullCompleteness(ctx context.Context, candidateID uint) (*CompletenessReport, error) {
    base, err := s.calcBase(ctx, candidateID)
    if err != nil { return nil, err }
    report := &CompletenessReport{...}

    // 尝试利润测算
    profit, err := s.profitSvc.EstimatePreListing(ctx, candidateID)
    if err == nil {
        report.EstimatedProfit = &profit.TotalProfit
        report.ProfitScore = 1.0
    }
    return report, nil
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/completeness/... -v
cd backend-go && go test ./internal/domain/profit/... -v
```

**验收条件：** 候选商品完整度报告包含成本/物流费/平台费/毛利估算；缺失字段明确列出

---

### Task M2.2: Listing 建议生成

**Files:**
- Modify: `backend-go/internal/domain/listing/service.go` — 建议生成逻辑
- Modify: `backend-go/internal/domain/listing/handler.go` — 建议 API
- Create: `backend-go/internal/domain/listing/model_suggestion.go` — 建议模型
- Test: 对应 `*_test.go`

**Interfaces:**
- `POST /api/v1/listings/suggest` — 为候选商品生成 listing 建议
  - Request: `{ candidate_id: uint }`
  - Response: `{ title, category, price, stock, platform_adaptations: [...] }`

- [ ] **Step 1: 创建 listing suggestion 数据模型**

```go
// model_suggestion.go
type ListingSuggestion struct {
    CandidateID        uint   `json:"candidate_id"`
    Title              string `json:"title"`
    CategoryPath       string `json:"category_path"`
    SuggestedPrice     float64 `json:"suggested_price"`
    SuggestedStock     int    `json:"suggested_stock"`
    PlatformFields     []PlatformField `json:"platform_fields"`
    RiskLevel          string `json:"risk_level"`
}

type PlatformField struct {
    Platform  string `json:"platform"`
    FieldName string `json:"field_name"`
    Value     string `json:"value"`
}
```

- [ ] **Step 2: 实现建议生成服务的接口和路由**

```go
// service.go
func (s *Service) GenerateSuggestion(ctx context.Context, candidateID uint) (*ListingSuggestion, error) {
    // 1. 从 candidate 模块获取商品基础信息
    // 2. 从 profit 模块获取利润测算
    // 3. 从 platform 模块获取平台字段模板
    // 4. 组合成建议（标题、类目、价格、库存、各平台适配字段）
    // 5. 返回结构化的 ListingSuggestion
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/listing/... -v
```

**验收条件：** 对候选商品可生成结构化建议（标题、类目、价格、库存、多平台适配字段）

---

### Task M2.3: Owner 审批 → 受控上架任务

**Files:**
- Modify: `backend-go/internal/domain/listingtask/service.go` — 审批后生成受控任务
- Modify: `backend-go/internal/domain/listingtask/handler.go` — 任务 API
- Modify: `frontend-next/src/app/(main)/candidates/page.tsx` — 审批 UI
- Test: 对应 `*_test.go`

**Interfaces:**
- `POST /api/v1/listing-tasks/create-from-suggestion` — 从建议创建受控上架任务
  - 必须有审批门禁 + 审计

- [ ] **Step 1: 实现"建议→任务"转换**

```go
// listingtask/service.go
func (s *Service) CreateFromSuggestion(ctx context.Context, suggestionID uint) (*ListingTask, error) {
    // 1. 获取建议
    // 2. 校验候选商品未被删除/重复
    // 3. 创建受控上架任务（status: pending_approval）
    // 4. 写入审计日志
    // 5. 返回任务
}
```

- [ ] **Step 2: 审批门禁集成**

使用 M1.5 的审批服务，确保创建上架任务走审批：
```go
if err := s.approvalSvc.RequireApproval(ctx, approval.Request{
    Action:    "create_listing_task",
    RiskLevel: "high",
}); err != nil {
    return nil, fmt.Errorf("需要审批: %w", err)
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/listingtask/... -v
```

**验收条件：** Owner 审批后生成受控上架任务，走审计链路，状态可追踪

---

### Task M2.4: 上架结果复盘

**Files:**
- Create: `backend-go/internal/domain/listingtask/model_review.go` — 复盘模型
- Modify: `backend-go/internal/domain/listingtask/service.go` — 复盘逻辑
- Modify: `backend-go/internal/domain/listingtask/handler.go` — 复盘 API
- Modify: `frontend-next/src/app/(main)/listing-tasks/page.tsx` — 复盘前端展示
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/listing-tasks/:id/review` — 获取复盘结果
- `POST /api/v1/listing-tasks/:id/review` — 触发复盘

- [ ] **Step 1: 复盘逻辑**

```go
func (s *Service) ReviewTask(ctx context.Context, taskID uint) (*TaskReview, error) {
    task, err := s.Get(ctx, taskID)
    // 对比：
    //   - 预期利润 vs 实际（如果已有订单）
    //   - 是否发布成功
    //   - 是否有平台返回的错误
    //   - 是否有异常
    return &TaskReview{
        TaskID:         taskID,
        Published:      task.Status == "published",
        PlatformErrors: task.Errors,
        ProfitExpected: task.ExpectedProfit,
        ProfitActual:   actualProfit, // 从订单数据获取
    }, nil
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/listingtask/... -v
```

**验收条件：** 上架后可查看复盘：是否发布成功、异常信息、预期利润 vs 实际利润

---

## M3：订单与履约利润闭环

**依赖：** M1.5 已完成
**并行度：** 与 M2、M4、M6 完全并行

---

### Task M3.1: 订单导入/同步稳定化

**Files:**
- Modify: `backend-go/internal/domain/orderimport/service.go` — 稳定化增强
- Modify: `backend-go/internal/domain/orderimport/handler.go` — 监控 API
- Modify: `backend-go/internal/domain/orderimport/model.go` — 增加同步状态
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/order-imports/status` — 各平台订单同步状态
- `POST /api/v1/order-imports/:id/retry` — 手动重试失败同步

- [ ] **Step 1: 添加同步状态追踪**

```go
// model.go 扩展
type ImportStatus struct {
    Platform       string
    LastSyncAt     *time.Time
    LastSyncResult string // success, failed, partial
    OrderCount     int
    ErrorMessage   string
    PendingCount   int
}
```

- [ ] **Step 2: 幂等防护 + 失败重试**

确保订单导入不会重复处理：
```go
// 已存在的 order_import_id 做去重
if existing := db.Where("platform_order_id = ?", platformOrderID).First(&model); existing.RowsAffected > 0 {
    return nil // skip
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/orderimport/... -v
```

**验收条件：** 平台订单持续同步不丢不重复；Owner 可查看同步状态；失败可重试

---

### Task M3.2: 成本核算链路

**Files:**
- Modify: `backend-go/internal/domain/profit/service.go` — 订单级利润计算
- Modify: `backend-go/internal/domain/shipping/service.go` — 运费快照
- Modify: `backend-go/internal/domain/platformfee/service.go` — 平台费计算
- Modify: `backend-go/internal/domain/finance/service.go` — 财务结算
- Modify: `backend-go/internal/domain/inventory/service.go` — 库存匹配
- Test: 对应 `*_test.go`

**Interfaces:**
- `POST /api/v1/profit/order/:orderId/calculate` — 计算订单利润

- [ ] **Step 1: 订单级利润计算**

```go
// profit/service.go
func (s *Service) CalculateOrderProfit(ctx context.Context, orderID uint) (*OrderProfit, error) {
    // 1. 订单收入（平台售价 - 平台佣金 - 支付费）
    // 2. 采购成本（商品成本 × 数量）
    // 3. 物流成本（运费 + 关税 + 保险）
    // 4. 平台费用（佣金、广告费、仓储费）
    // 5. 毛利率 = (收入 - 成本) / 收入
    // 6. 快照关键数据以备审计
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/profit/... -v
```

**验收条件：** 任意订单可计算利润：收入-采购成本-物流-平台费=毛利；含运费快照；结果可审计

---

### Task M3.3: 异常识别

**Files:**
- Modify: `backend-go/internal/domain/exceptions/service.go` — 自动异常识别逻辑
- Modify: `backend-go/internal/domain/exceptions/handler.go` — 异常列表 API
- Modify: `backend-go/internal/domain/exceptions/model.go` — 增加异常类型
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/exceptions/auto-detect` — 自动扫描异常
- 异常类型：`loss_order`, `out_of_stock`, `logistics_abnormal`, `fee_abnormal`

- [ ] **Step 1: 自动异常扫描**

```go
func (s *Service) AutoDetect(ctx context.Context) ([]Exception, error) {
    var results []Exception
    // 1. 亏损单：利润为负的订单
    // 2. 缺货：库存不足的 SKU
    // 3. 物流异常：超时未送达、运费异常波动
    // 4. 费用异常：平台费/物流费明显偏离预期
    // 每个异常自动创建 Exception 记录并通知
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/exceptions/... -v
```

**验收条件：** 亏损单、缺货、物流异常、费用异常四种类型可被系统自动识别并创建异常记录

---

### Task M3.4: Agent 处理建议 + Owner 审批

**Files:**
- Modify: `backend-go/internal/domain/exceptions/service.go` — Agent 建议集成
- Modify: `backend-go/internal/ai/service.go` — 生成处理建议
- Modify: `frontend-next/src/app/(main)/exceptions/page.tsx` — 异常处理前端
- Test: 对应 `*_test.go`

**Interfaces:**
- `POST /api/v1/exceptions/:id/suggest` — Agent 生成处理建议
- `POST /api/v1/exceptions/:id/resolve` — Owner 审批后处理

- [ ] **Step 1: Agent 建议接口**

```go
// ai/service.go 或 exceptions/service.go
func (s *Service) GenerateSuggestion(ctx context.Context, exceptionID uint) (*ResolutionSuggestion, error) {
    exc, _ := s.Get(ctx, exceptionID)
    prompt := fmt.Sprintf("异常类型: %s, 详情: %s, 建议如何处理?", exc.Type, exc.Detail)
    // 调用 LLM 生成处理方案
    suggestion, err := s.aiSvc.Generate(ctx, prompt)
    return &ResolutionSuggestion{
        ExceptionID: exceptionID,
        Suggestion:  suggestion,
        RiskLevel:   exc.RiskLevel(),
    }, nil
}
```

- [ ] **Step 2: Owner 审批处理**

Owner 查看建议后选择批准/拒绝/修改。批准后执行对应动作并写审计。

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/exceptions/... -v
```

**验收条件：** Agent 给出结构化处理建议；Owner 可审批/拒绝/修改；处理结果写审计

---

## M4：平台集成受控生产

**依赖：** M1.4 + M1.5 已完成
**并行度：** 与 M2、M3、M6 完全并行

---

### Task M4.1: 平台适配器核心流程补齐

**Files:**
- Modify: `backend-go/internal/domain/integrations/adapter.go` — 确保统一接口
- Modify: `backend-go/internal/domain/integrations/ozon.go` — 补齐 Ozon 流程
- Modify: `backend-go/internal/domain/integrations/shopee.go` — 补齐 Shopee 流程
- Create: `backend-go/internal/domain/integrations/shopify.go` — Shopify 适配器
- Test: 对应 `*_test.go`

**Interfaces:**
- `PlatformAdapter` 必须实现：Publish, SyncStatus, ValidateCredentials, SyncInventory, PushTracking, FetchOrders

- [ ] **Step 1: 审计现有适配器实现覆盖度**

```bash
grep -n "func.*Adapter.*Publish\|func.*Adapter.*SyncStatus\|func.*Adapter.*PushTracking" backend-go/internal/domain/integrations/*.go
```

- [ ] **Step 2: 补齐缺失方法**

对每个适配器，补齐缺失的平台方法。每个 adapter 至少实现：
```go
func (a *OzonAdapter) Publish(ctx context.Context, product *PlatformProduct) (*PublishResult, error)
func (a *OzonAdapter) SyncStatus(ctx context.Context, platformID string) (*SyncResult, error)
func (a *OzonAdapter) ValidateCredentials(ctx context.Context) error
func (a *OzonAdapter) SyncInventory(ctx context.Context, skuID string, qty int) error
func (a *OzonAdapter) PushTracking(ctx context.Context, orderID, tracking string) error
func (a *OzonAdapter) FetchOrders(ctx context.Context, since time.Time) ([]Order, error)
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/integrations/... -v
```

**验收条件：** 每个适配器实现全部平台核心流程（发布、同步状态、库存、订单、物流）。缺失方法补齐后测试通过。

---

### Task M4.2: 环境控制（dry-run/sandbox/production）

**Files:**
- Modify: `backend-go/internal/domain/integrations/types.go` — ExecutionMode 已有，补充字段
- Modify: `backend-go/internal/domain/integrations/service.go` — 写回门禁逻辑
- Modify: `backend-go/internal/domain/integrations/handler.go` — 环境管理 API
- Modify: `frontend-next/src/app/(main)/platform-integrations/page.tsx` — 环境切换 UI
- Test: 对应 `*_test.go`

**Interfaces:**
- `PUT /api/v1/integrations/:id/mode` — 切换平台环境模式
- `GET /api/v1/integrations/:id/mode` — 查看当前环境
- 已有 ExecutionMode: `dry_run, sandbox, production`

- [ ] **Step 1: 环境管理 API**

```go
// types.go
type EnvironmentConfig struct {
    Mode           ExecutionMode
    AllowedModes   []ExecutionMode // dry_run, sandbox, production
    RequiresApproval bool          // production 默认 true
}
```

- [ ] **Step 2: 写回门禁**

在 service.go 中添加写前检查：
```go
func (s *Service) CheckWriteMode(ctx context.Context, adapterID string, mode ExecutionMode) error {
    // 1. 获取平台配置
    // 2. production 模式必须检查审批
    // 3. 记录检查结果到审计
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/integrations/... -v
```

**验收条件：** 分环境 dry-run/sandbox/production；production 写回必须审批；环境切换有审计

---

### Task M4.3: 生产写回门禁

**Files:**
- Modify: `backend-go/internal/domain/integrations/service.go` — 增强写回门禁
- Modify: `backend-go/internal/domain/integrations/handler.go` — write-back endpoint
- Test: `*_test.go`

**Interfaces:**
- `POST /api/v1/integrations/:id/write-back` — 受控写回
  - 需要 approval + 生成外部 reference ID + 记录失败 + 可重试

- [ ] **Step 1: 受控写回逻辑**

```go
func (s *Service) WriteBack(ctx context.Context, req WriteBackRequest) (*WriteBackResult, error) {
    // 1. 审批检查（引用 M1.5）
    // 2. 生成外部 reference ID (uuid)
    // 3. 执行写回
    // 4. 记录结果（成功/失败 + reference ID）
    // 5. 失败时可重试（提供同 reference ID 重入）
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/integrations/... -v
```

**验收条件：** 生产写回有审批 + 外部 reference ID + 失败可见 + 可重试；测试覆盖写回全路径

---

### Task M4.4: 真实店铺生产试点

**Files:**
- Modify: `backend-go/internal/domain/integrations/handler.go` — 试点状态 API
- Modify: `frontend-next/src/app/(main)/platform-integrations/page.tsx` — 试点仪表盘
- 无代码改动，配置和操作层面的任务

**Interfaces:** 无新增 API

- [ ] **Step 1: 试点状态仪表盘**

在平台集成页面添加试点状态展示：平台连接状态、当前模式、上次写回结果、审批要求。

- [ ] **Step 2: 连接真实店铺并验证**

```bash
# 1. 配置平台 API 凭证 (通过 UI 或环境变量)
# 2. 在 sandbox 模式验证只读操作（获取订单/商品）
# 3. 在受控模式下执行一次受控写回
# 4. 验证审批流程、审计记录、重试机制
```

**验收条件：** 凌镜能安全连接真实平台；只读操作正常；受控写回走审批流程

---

## M5：AgentOS 工作流平台

**依赖：** 至少一个闭环 (M2/M3 其一) 已跑通
**并行度：** 与 M6 可并行

---

### Task M5.1: Workflow 管理页面

**Files:**
- Modify: `frontend-next/src/app/(main)/workflows/page.tsx` — Workflow 列表页
- Create: `frontend-next/src/app/(main)/workflows/[id]/page.tsx` — 详情页
- Modify: `frontend-next/src/config/menu.ts` — 菜单项
- Modify: `backend-go/internal/domain/workflow/handler.go` — 补齐列表/详情 API
- Test: 前端 Vitest + 后端 test

**Interfaces:**
- `GET /api/v1/workflows` — 列表
- `GET /api/v1/workflows/:id` — 详情

- [ ] **Step 1: 后端 Workflow 列表/详情 API**

```go
// workflow/handler.go
func (h *Handler) List(c *gin.Context) {
    workflows, total, err := h.svc.List(c, common.ParsePagination(c))
    response.Paginated(c, workflows, total, page, size)
}

func (h *Handler) Get(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
    wf, err := h.svc.Get(c.Request.Context(), uint(id))
    response.Success(c, wf)
}
```

- [ ] **Step 2: 前端 Workflow 管理页面**

```tsx
// workflows/page.tsx
// 使用 Ant Design Table 展示 workflows
// 列：名称、状态、触发器、最后运行、创建时间
// 点击进入详情
```

- [ ] **Step 3: 添加菜单项**

```typescript
// menu.ts
{
  key: '/workflows',
  label: '工作流',
  icon: 'NodeIndexOutlined',
}
```

- [ ] **Step 4: 测试**

```bash
cd backend-go && go test ./internal/domain/workflow/... -v
cd frontend-next && npm test -- --run
```

**验收条件：** 可查看工作流列表和详情；菜单有入口；前后端测试通过

---

### Task M5.2: 条件分支 + 事件触发 + 审批节点

**Files:**
- Modify: `backend-go/internal/domain/workflow/model.go` — 增加节点类型
- Modify: `backend-go/internal/domain/workflow/service.go` — 执行引擎
- Modify: `backend-go/internal/platform/eventbus/bus.go` — 事件触发扩展
- Test: `*_test.go`

**Interfaces:**
- `POST /api/v1/workflows/:id/trigger` — 手动触发
- 事件触发：EventBus 订阅 `workflow.trigger.*`

- [ ] **Step 1: 节点模型扩展**

```go
// workflow/model.go
const (
    NodeTypeCondition  = "condition"
    NodeTypeApproval   = "approval"
    NodeTypeAction     = "action"
    NodeTypeEvent      = "event"
)

type WorkflowNode struct {
    ID         uint   `gorm:"primaryKey"`
    WorkflowID uint   `gorm:"index"`
    Type       string // condition, approval, action, event
    Config     JSON   // 节点配置（条件表达式、审批人、Action 类型等）
    OrderIndex int
}
```

- [ ] **Step 2: 事件触发**

基于现有 EventBus 的 glob topic 订阅：
```go
// 订阅 workflow.trigger.{workflow_id} 事件
bus.Subscribe("workflow.trigger.*", func(ctx context.Context, event eventbus.Event) {
    workflowID := parseWorkflowID(event.Topic)
    svc.Execute(ctx, workflowID, event.Data)
})
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/workflow/... -v
cd backend-go && go test ./internal/platform/eventbus/... -v
```

**验收条件：** 工作流支持条件分支、事件触发、人工审批节点；测试覆盖状态流转

---

### Task M5.3: 工作流监控面板

**Files:**
- Modify: `frontend-next/src/app/(main)/agentos/page.tsx` — 增加监控面板组件
- Modify: `backend-go/internal/domain/aios/handler.go` — 监控数据 API
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/workflows/monitor` — 实时监控数据

- [ ] **Step 1: 监控数据 API**

```go
// aios/handler.go
type WorkflowMonitorData struct {
    Running     int    `json:"running"`
    Pending     int    `json:"pending"`
    Blocked     int    `json:"blocked"`  // 在审批节点阻塞
    Failed      int    `json:"failed"`
    Completed24h int   `json:"completed_24h"`
}
```

- [ ] **Step 2: 前端监控面板**

在 AgentOS 页面添加监控卡片，显示：
- 当前运行中工作流数
- 阻塞在审批节点的工作流
- 最近 24h 完成数
- 失败数

**验收条件：** Owner 可按工作流维度看到 Agent 正在做什么、卡在哪里、需要自己批准什么

---

### Task M5.4: 固化标准工作流

**Files:**
- Create: `backend-go/internal/domain/workflow/templates.go` — 标准工作流模板
- Modify: `backend-go/internal/domain/workflow/service.go` — 模板初始化
- Test: `*_test.go`

- [ ] **Step 1: 商品闭环标准工作流**

```go
// templates.go — 执行时生成模板实例
func ProductListingWorkflowTemplate() WorkflowTemplate {
    return WorkflowTemplate{
        Name: "商品上架审批流程",
        Nodes: []WorkflowNodeDef{
            {Type: "event", Config: `{"trigger": "product.ready_for_listing"}`},
            {Type: "action", Config: `{"action": "generate_listing_suggestion"}`},
            {Type: "approval", Config: `{"approvers": ["owner"]}`},
            {Type: "action", Config: `{"action": "create_publish_task"}`},
            {Type: "action", Config: `{"action": "review_publish_result"}`},
        },
    }
}
```

- [ ] **Step 2: 订单利润闭环标准工作流**

```go
func OrderProfitWorkflowTemplate() WorkflowTemplate {
    return WorkflowTemplate{
        Name: "订单利润复盘流程",
        Nodes: []WorkflowNodeDef{
            {Type: "event", Config: `{"trigger": "order.imported"}`},
            {Type: "action", Config: `{"action": "calculate_order_profit"}`},
            {Type: "condition", Config: `{"if": "profit < 0", "then": "create_exception"}`},
            {Type: "approval", Config: `{"approvers": ["owner"], "if": "is_anomaly"}`},
        },
    }
}
```

- [ ] **Step 3: 测试**

```bash
cd backend-go && go test ./internal/domain/workflow/... -v
```

**验收条件：** 商品闭环和订单闭环可固化为标准工作流模板；新创建时自动加载

---

### Task M5.5: 任务队列 + 失败重试 + 运行历史

**Files:**
- Modify: `backend-go/internal/domain/workflow/model.go` — 运行历史模型
- Create: `backend-go/internal/domain/workflow/history.go` — 历史记录
- Create: `backend-go/internal/domain/workflow/retry.go` — 重试逻辑
- Test: `*_test.go`

**Interfaces:**
- `GET /api/v1/workflows/:id/history` — 运行历史
- `POST /api/v1/workflows/:history-id/retry` — 重试失败节点

- [ ] **Step 1: 运行历史模型**

```go
type WorkflowRun struct {
    ID            uint      `gorm:"primaryKey"`
    WorkflowID    uint      `gorm:"index"`
    Status        string    // running, completed, failed, cancelled
    StartedAt     time.Time
    CompletedAt   *time.Time
    CurrentNodeID uint
    ErrorMessage  string
    RetryCount    int       `gorm:"default:0"`
    MaxRetries    int       `gorm:"default:3"`
}

type WorkflowRunLog struct {
    ID         uint      `gorm:"primaryKey"`
    RunID      uint      `gorm:"index"`
    NodeID     uint
    Status     string    // pending, running, completed, failed
    Input      JSON
    Output     JSON
    Error      string
    StartedAt  time.Time
    CompletedAt *time.Time
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/workflow/... -v
```

**验收条件：** 工作流运行有完整历史记录；失败节点可重试（≤3 次）；测试覆盖重试逻辑

---

## M6：运营化与 Beta 可用

**依赖：** 无（可与全部并行）
**并行度：** 与全部工作流可并行

---

### Task M6.1: Owner 运营仪表盘（合并 Workbench）

**Files:**
- Modify: `frontend-next/src/app/(main)/layout.tsx` — 路由重定向调整
- Modify: `frontend-next/src/app/(main)/dashboard/page.tsx` — 增强为统一仪表盘
- Modify: `backend-go/internal/domain/dashboard/handler.go` — 仪表盘数据 API
- Modify: `frontend-next/src/config/menu.ts` — 调整菜单项
- Delete 或 Redirect: `(main)/workbench` 路由到 `/dashboard`

**Interfaces:**
- `GET /api/v1/dashboard/overview` — 合并概览数据

- [ ] **Step 1: 后端合并数据 API**

```go
// dashboard/handler.go — 增强 Overview
type DashboardOverview struct {
    TodaySales      float64            `json:"today_sales"`
    PendingApprovals int               `json:"pending_approvals"`
    AnomalyCount    int                `json:"anomaly_count"`
    AgentSuggestions int               `json:"agent_suggestions"`
    OrderProfit     *ProfitSummary     `json:"order_profit"`
    ProductStatus   *ProductSummary    `json:"product_status"`
    RecentAlerts    []AlertItem        `json:"recent_alerts"`
    AgentStatuses   []AgentStatusCard  `json:"agent_statuses"`
}
```

- [ ] **Step 2: 前端统一仪表盘**

在 `dashboard/page.tsx` 布局：
```
顶部：销售 KPI + 待审批/异常/建议卡片
中部：待处理事项列表
底部：趋势图表 + Agent 状态
```

- [ ] **Step 3: 路由迁移**

在 `layout.tsx` 或 `page.tsx` 添加 `/` 重定向到 `/dashboard`。原有 `/workbench` 路由重定向到 `/dashboard`。

- [ ] **Step 4: 测试**

```bash
cd frontend-next && npm test -- --run
```

**验收条件：** 统一仪表盘页面显示销售/利润/异常/Agent 建议/审批；原 workbench 路由自动跳转

---

### Task M6.2: 日报/周报增强

**Files:**
- Modify: `backend-go/internal/domain/report/service.go` — 报告生成逻辑
- Modify: `frontend-next/src/app/(main)/reports/page.tsx` — 报告页面
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/reports/daily` — 日报
- `GET /api/v1/reports/weekly` — 周报

- [ ] **Step 1: 报告数据聚合**

日报和周报聚合：
```go
// report/service.go
type DailyReport struct {
    Date           string        `json:"date"`
    Sales          float64       `json:"sales"`
    Orders         int           `json:"orders"`
    Profit         float64       `json:"profit"`
    NewListings    int           `json:"new_listings"`
    Anomalies      int           `json:"anomalies"`
    Approvals      int           `json:"approvals"`
    AgentProposals int           `json:"agent_proposals"`
    LLMCost        float64       `json:"llm_cost"`
}
```

- [ ] **Step 2: 测试**

```bash
cd backend-go && go test ./internal/domain/report/... -v
```

**验收条件：** 日报/周报包含销售/利润/异常/Agent 建议/审批/LLM 成本数据

---

### Task M6.3: 运维基建

**Files:**
- Modify: `backend-go/internal/domain/notification/service.go` — 告警通知
- Modify: 运维配置：备份策略、告警规则文档
- Create: `docs/ops/BACKUP_POLICY.md` — 备份策略
- Create: `docs/ops/ALERT_RULES.md` — 告警规则

- [ ] **Step 1: 备份策略文档**

编写基本备份策略：每日 DB 备份 + 7 天保留 + 手动快照

- [ ] **Step 2: 告警规则**

定义并在系统内注册：
- Agent 连续失败 > 3 → 告警
- 订单同步中断 > 30min → 告警
- 利润率 < -10% → 异常标记
- LLM 月度预算 > 80% → 警告，> 100% → 停 Agent

- [ ] **Step 3: 成本控制面板**

在后端增加成本聚合 API（引用 M1.3 的数据），前端展示。

**验收条件：** 备份策略确定；告警规则文档化；成本控制面板展示数据

---

### Task M6.4: LLM 月度预算硬上限

**Files:**
- Modify: `backend-go/internal/domain/reliability/service.go` — 预算检查逻辑
- Modify: `backend-go/internal/domain/reliability/model.go` — 预算模型
- Modify: `backend-go/internal/domain/reliability/handler.go` — 预算 API
- Modify: `backend-go/internal/ai/service.go` — 调用前检查预算
- Test: 对应 `*_test.go`

**Interfaces:**
- `GET /api/v1/reliability/budget` — 查看预算
- `PUT /api/v1/reliability/budget` — 设置预算

- [ ] **Step 1: 预算模型**

```go
type LLMBudget struct {
    ID            uint      `gorm:"primaryKey"`
    MonthlyLimitUSD float64 `gorm:"not null"`
    CurrentMonthUSD float64 `gorm:"default:0"`
    BudgetMonth   string    `gorm:"size:7"` // "2026-07"
    IsPaused      bool      `gorm:"default:false"`
    UpdatedAt     time.Time
}
```

- [ ] **Step 2: AI 调用前预算检查**

在 `internal/ai/service.go` 的 LLM 调用入口添加：
```go
func (s *Service) checkBudget(ctx context.Context) error {
    budget, err := s.reliabilitySvc.GetCurrentBudget(ctx)
    if err != nil { return err }
    if budget.CurrentMonthUSD >= budget.MonthlyLimitUSD {
        return fmt.Errorf("月度 LLM 预算已超限 ($%.2f / $%.2f)", budget.CurrentMonthUSD, budget.MonthlyLimitUSD)
    }
    return nil
}
```

- [ ] **Step 3: 超限处理**

超限时停 Agent + 通知 Owner：
```go
if budget.CurrentMonthUSD >= budget.MonthlyLimitUSD {
    s.agentSvc.PauseAll(ctx, "LLM budget exceeded")
    s.notificationSvc.Send(ctx, "LLM budget exceeded — all agents paused")
}
```

- [ ] **Step 4: 测试**

```bash
cd backend-go && go test ./internal/domain/reliability/... -v
cd backend-go && go test ./internal/ai/... -v
```

**验收条件：** 可配置月度 LLM 预算硬上限；超限后自动停 Agent 并通知 Owner；测试覆盖预算检查和超限处理

---

### Task M6.5: Beta 验收

**Files:** 无代码改动
**依赖：** 所有 M1-M6 任务完成后

- [ ] **Step 1: 端到端验收**

```bash
# 后端测试
cd backend-go && go test ./... && go vet ./...

# 前端构建
cd frontend-next && npm run build

# 烟雾测试
cd backend-go && ./scripts/smoke_test.sh

# E2E 测试
cd frontend-next/e2e && npx playwright test
```

- [ ] **Step 2: 运行 2-3 个业务 Demo 场景**

1. **商品上架闭环**：创建候选商品 → 完整度检查 → 利润测算 → Listing 建议 → 审批 → 受控上架 → 复盘
2. **订单利润闭环**：导入订单 → 成本核算 → 利润计算 → 异常识别 → Agent 建议 → Owner 处理
3. **平台写回受控**：在 sandbox 模式发布 → 审批 → 真实写回 → 失败重试 → 审计追踪

- [ ] **Step 3: 连续运行验证**

确保系统在真实数据、真实平台上能稳定运行 24h+。

**验收条件：** 所有测试全绿；2-3 个业务场景可跑通；连续运行 > 24h 正常

---

## 执行顺序总纲

```
Phase 1: M1（可信基础收口）→ 全部 M1 任务完成
  │
Phase 2: 并行派发
  ├── M2（商品闭环）  ← 可并行
  ├── M3（订单利润）  ← 可并行
  ├── M4（平台受控）  ← 可并行
  └── M6（运营化）    ← 可与全部并行
  │
Phase 3: 依赖等待
  └── M5（Workflow 平台）← 等 M2 或 M3 至少一个完成
```

## 执行顺序建议

按并行工蜂最优分配：

| 轮次 | 工作流 | 工蜂数量建议 |
|------|--------|-------------|
| 1 | M1 | 全部工蜂（M1 完成后释放）|
| 2a | M2 + M6 | 2-3 工蜂 |
| 2b | M3 + M6 | 2-3 工蜂 |
| 2c | M4 + M6 | 2-3 工蜂 |
| 3 | M5 | 2-3 工蜂（M2 或 M3 完成后）|

每个工蜂独立处理一个 Task，按依赖顺序串行。工蜂之间无共享状态（隔离 worktree）。
