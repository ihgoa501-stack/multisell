# v0.4：真实商品沙箱上架闭环

> 版本: v0.4
> 日期: 2026-07-10
> 状态: 设计确认，待实现

## 1. 目标

让 Owner 把 1 个真实商品录入系统，系统完成资料检查、费用利润测算、上架建议、审批、沙箱上架任务执行，Owner 能看到结果和审计记录。

**验收标准**：从一个界面跑通完整流程：
商品录入 → 补齐成本/物流/平台费 → 查看利润证据卡 → 系统给出上架建议 → Owner 审批 → 生成沙箱上架任务 → 执行沙箱任务 → 查看执行结果和审计记录

## 2. 界面结构

**新增 `/sandbox-listing` Wizard 页面**，6 步顺序流程：

```
Step 1: 录入真实商品   (POST /api/v1/candidates)
Step 2: 补齐字段       (PUT /api/v1/candidates/:id/fields)
Step 3: 利润证据卡     (GET /api/v1/profit/evidence-card/:id)   ← 新增 API
Step 4: 上架建议       (POST /api/v1/loop/evaluate/:id)
Step 5: 审批沙箱任务   (PUT /api/v1/approval/:id/review + HighRiskConfirmDialog)
Step 6: 执行与复盘     (POST /api/v1/listing-task/:task_id/execute + 结果展示)
```

**复用关系：**

| 步骤 | 复用模块 | 改动 |
|------|----------|------|
| Step 1 | `candidate` 已有 create API | 前端新表单 |
| Step 2 | `candidate.FillFields` 已有 | 前端补齐页面 |
| Step 3 | `profit` 模块扩展 | 新增 `EvidenceCard` 模型 + API + 服务 |
| Step 4 | `loop.Evaluate` 已有 | EvaluateResult 增加 `approval_id` 返回字段 |
| Step 5 | `approval` 模块 + `HighRiskConfirmDialog` 已有 | 确认 reviewer 由后端登录态绑定；EvaluateResult 返回 approval_id |
| Step 6 | `listingtask` 模块 | `/listing-tasks/[id]` 增强复盘；确认 task detail 返回 approval_id、audit 链接 |

**菜单与入口：**
- `/owner` 总控台增加入口卡片："沙箱上架 → /sandbox-listing"
- 侧边栏"Owner 总控台"下新增：`/sandbox-listing`，标签"沙箱上架"，状态 `sandbox`

## 3. 数据流

### Step 1 → Step 2（录入 → 补齐）
```
POST /api/v1/candidates   → 返回 { id, ... }
PUT /api/v1/candidates/:id/fields → 字段增量补齐
```
Step 2 展示完整度评分，缺失字段逐个补齐。允许标记"暂缺"(skip)。

### Step 3（利润证据卡）
```
GET /api/v1/profit/evidence-card/:id
→ ProfitEvidenceCard
```
纯查询，无副作用。见下文模型定义。

### Step 4（上架建议）
```
POST /api/v1/loop/evaluate/:id
→ EvaluateResult { decision, confidence, reason, risk_flags, listing_task_id, approval_id }
```
**副作用说明**：当 `decision == "list"` 时，Evaluate() 会事务性地创建：
- listing_task（status=blocked）
- approval（status=pending, risk_level=high）

UI 按钮文案："生成建议并创建沙箱审批任务"，不是"查看建议"。

**后端待改动**：`loop.EvaluateResult` 当前返回中有 `listing_task_id` 但没有 `approval_id`。v0.4 需要：
- `loop.Evaluate` 在事务中创建的 approval 的 ID 返回给调用者
- 在 `EvaluateResult` 结构体中增加 `approval_id` 字段
- 这是一个小改动，不是新功能，但如果不做则 Step 5 取不到审批 ID

### Step 5（审批）
```
Owner 查看审批摘要
→ HighRiskConfirmDialog 确认审批
→ PUT /api/v1/approval/:id/review
    body: { "action": "approve", "review_note": "同意沙箱上架" }
→ 展示执行预览
→ Owner 手动点击"执行沙箱任务"
```

注意：
- 审批端点使用现有 `PUT /api/v1/approval/:id/review`，不是独立的 approve endpoint。
- `reviewer` 不由前端传入，后端从 JWT 登录态绑定（已确认：handler 使用 `common.ReviewerFromCtx(c)` + `common.UserIDFromCtx(c)` 覆写请求中的身份字段）。
- 审批和执行是两个独立操作。执行前必须已有有效审批。

### Step 6（结果复盘）
- 任务成功 → 展示 `external_reference_id`、approval_id、审计记录链接
- 任务失败 → 展示 `last_error`、重试按钮
- 底部："查看完整任务详情" → 跳转 `/listing-tasks/:taskId`
- 执行端点：`POST /api/v1/listing-task/:task_id/execute`

## 4. ProfitEvidenceCard（新增后端模块）

### 文件位置
`backend-go/internal/domain/profit/evidence_card.go`

### 模型

```go
type EvidenceCard struct {
    ProductID int64  `json:"product_id"`
    Title     string `json:"title"`
    Currency  string `json:"currency"`

    Revenue       MoneyRow    `json:"revenue"`
    CostItems     []CostItem  `json:"cost_items"`
    TotalFixedCost float64 `json:"total_fixed_cost"`

    TotalVariableFeeRate float64 `json:"total_variable_fee_rate"`
    EstimatedVariableFee  float64 `json:"estimated_variable_fee"`
    TotalCostAtTargetPrice float64 `json:"total_cost_at_target_price"`

    EstimatedProfit float64 `json:"estimated_profit"`
    ProfitMargin    float64 `json:"profit_margin"`
    Status          string  `json:"status"` // profitable / marginal / unprofitable / unknown

    ConfidenceLevel string `json:"confidence_level"` // high / medium / low / insufficient_data
    CanEvaluate     bool   `json:"can_evaluate"`
    ConfirmedItems  []DataField `json:"confirmed_items"`
    EstimatedItems  []DataField `json:"estimated_items"`
    MissingItems    []string `json:"missing_items"`
    BlockingReasons []string `json:"blocking_reasons"`

    BreakEvenPrice      float64 `json:"break_even_price"`
    RecommendedMinPrice float64 `json:"recommended_min_price"`
    TargetMargin        float64 `json:"target_margin"`
    BufferRate          float64 `json:"buffer_rate"`
}

type CostItem struct {
    Category       string  `json:"category"`
    Label          string  `json:"label"`
    Amount         float64 `json:"amount"`
    Rate           float64 `json:"rate"`
    CalculationType string `json:"calculation_type"` // fixed_amount | percent_of_revenue
    DataSource     string  `json:"data_source"`      // confirmed | estimated | template_default | missing
    SourceNote     string  `json:"source_note"`
    Required       bool    `json:"required"`
}

type MoneyRow struct {
    Amount float64 `json:"amount"`
    Label  string  `json:"label"`
}

type DataField struct {
    FieldName string `json:"field_name"`
    Label     string `json:"label"`
    Value     string `json:"value"`
    Source    string `json:"source"` // confirmed | estimated | missing
}
```

### 成本分类

| 分类 | 类型 | 必须阻断 |
|------|------|----------|
| purchase_cost | fixed_amount | ✅ |
| domestic_shipping | fixed_amount | ❌ |
| international_shipping | fixed_amount | ✅ |
| platform_commission | percent_of_revenue | ✅ |
| payment_fee | percent_of_revenue | ❌ |
| packaging_fee | fixed_amount | ❌ |
| tariff | fixed_amount | ❌ |
| exchange_rate_buffer | fixed_amount | ❌ |
| loss_buffer | fixed_amount | ❌ |

注：exchange_rate_buffer 和 loss_buffer 是风险缓冲成本，不是实际已发生费用，用于避免利润被高估。

### 计算规则

```
break_even_price = total_fixed_cost / (1 - total_variable_fee_rate)
recommended_min_price = total_fixed_cost * (1 + target_margin + buffer_rate) / (1 - total_variable_fee_rate)
total_cost_at_target_price = total_fixed_cost + (target_sale_price * total_variable_fee_rate)
```

TotalVariableFeeRate 是 platform_commission + payment_fee 等按售价比例计费项的费率合计。公式不重复扣除平台佣金。

### 可信度规则

| 条件 | ConfidenceLevel |
|------|----------------|
| 必须阻断项全部 confirmed | high |
| 必须阻断项齐全，有 estimated | medium |
| 必须阻断项齐全，有 template_default | low |
| 必须阻断项缺失 | insufficient_data |

`can_evaluate` = `confidence_level != "insufficient_data"`
EvidenceCard 只判断数据质量，不生成 listing task（决策在 loop.Evaluate 中做）。

### 后端新增文件清单

- `profit/evidence_card.go` — 模型定义 + 计算服务
- `profit/handler_evidence.go` — `GET /profit/evidence-card/:id`
- `httpx/router.go` — 注册路由

## 5. 状态管理

- **Zustand store**：`useSandboxListingStore` — wizard step、candidate_id、listing_task_id、approval_id 等
- **后端持久化**：所有业务数据
- **URL 恢复**：`/sandbox-listing?candidate_id=123` 支持刷新恢复，自动推断当前 step

## 6. 后端变更清单

| 文件 | 操作 |
|------|------|
| `profit/evidence_card.go` | 新增 — 模型定义 + 计算服务 |
| `profit/handler_evidence.go` | 新增 — `GET /api/v1/profit/evidence-card/:productId` |
| `loop/model.go` | 修改 — `EvaluateResult` 增加 `approval_id` 字段 |
| `loop/service.go` | 修改 — `Evaluate()` 返回新创建的 `approval.ID` |
| `httpx/router.go` | 修改 — 注册证据卡路由 |
| `approval/` (handler/routes) | 无需改动 — 已确认 `PUT /approval/:id/review` 使用 `common.ReviewerFromCtx(c)` + `common.UserIDFromCtx(c)` 绑定 JWT 身份，计划中只复用 |
| `listingtask/` (handler) | 核查 — 确认 `GET /listing-tasks/:id` 返回中包含 `approval_id`，供复盘页展示 |

## 7. 前端变更清单

| 文件 | 操作 |
|------|------|
| `sandbox-listing/page.tsx` | 新增 — Wizard 入口 |
| `sandbox-listing/steps/` | 新增 — 6 个 Step 组件 |
| `sandbox-listing/store.ts` | 新增 — Zustand |
| `components/profit/EvidenceCard.tsx` | 新增 — 证据卡展示 |
| `listing-tasks/[id]/page.tsx` | 修改 — 增强复盘视图 |
| `config/menu.ts` | 修改 — 加菜单项 |
| `owner/page.tsx` | 修改 — 加入口卡片 |

## 8. 边界条件

- 阻断由 loop.Evaluate 在 decision 中体现，前端不需要额外判断
- Step 4 的三种结果直接指导 Step 5-6 的可进入性
- 执行模式固定在 sandbox，不触发真实外部发布
- listingtask execute 已有幂等守卫
- URL 支持 candidate_id 恢复

## 9. 非目标（v0.4 不做）

- ❌ loop.Evaluate 拆 preview + commit（保持现状，UI 文案说明副作用）
- ❌ AgentAction 统一门禁改造（保留 listingtask + approval 现有路由）
- ❌ 真实外部平台发布
- ❌ 多商品批量上架
- ❌ E2E 不在第一批实现提交中完成，但 v0.4 验收前必须补齐 Product Loop E2E（没有 E2E 很难证明跑通）
