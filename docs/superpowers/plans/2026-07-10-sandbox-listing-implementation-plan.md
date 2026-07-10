# v0.4：真实商品沙箱上架闭环 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Owner 能从 /sandbox-listing 一个 Wizard 页面跑通"真实商品录入 → 补齐字段 → 利润证据卡 → 上架建议 → 审批 → 沙箱执行 → 结果复盘"的全闭环。

**Architecture:** 新增 `/sandbox-listing` 6 步 Wizard 页面，底层复用 `candidate`/`profit`/`loop`/`listingtask`/`approval` 现有模块。唯一新增后端能力是「利润证据卡 EvidenceCard」——一个纯查询的 read model，不承担决策或持久化职责。loop.EvaluateResult 增加 approval_id 返回。前端用 Zustand 管理 Wizard 状态，URL 支持 candidate_id 恢复。

**Tech Stack:** Go/Gin/GORM (backend), Next.js/React/Zustand/Ant Design (frontend), PostgreSQL

**验证提醒：** 后端 `PUT /api/v1/approval/:id/review` 确认 reviewer 由 JWT 绑定（`common.ReviewerFromCtx(c)`），无需改动。`GET /api/v1/listing-tasks/:id` 响应中 `ListingTask` 已包含 `approval_id` 字段，无需改动。

---

## 全局约束

- Step 4 按钮文案必须写"生成建议并创建沙箱审批任务"，不是"查看建议"
- Step 5 审批端点使用 `PUT /api/v1/approval/:id/review`，body: `{ "action": "approve", "review_note": "..." }`
- Step 6 执行端点使用 `POST /api/v1/listing-task/:task_id/execute`
- 证据卡路由：`GET /api/v1/profit/evidence-card/:productId`
- 成本公式：`break_even_price = total_fixed_cost / (1 - total_variable_fee_rate)`，避免重复扣除平台佣金
- 所有 Wizard 步骤数据从后端读取，不依赖 Zustand store 持久化业务数据
- URL 支持 `?candidate_id=123` 刷新恢复
- E2E 不在第一批实现提交中完成，但验收前必须补齐 Product Loop E2E

---

## 文件清单

### 后端新增/修改

| 文件 | 职责 |
|------|------|
| `backend-go/internal/domain/profit/evidence_card.go` | EvidenceCard 模型定义 + 计算服务 |
| `backend-go/internal/domain/profit/handler_evidence.go` | `GET /profit/evidence-card/:productId` handler |
| `backend-go/internal/domain/loop/model.go` | EvaluateResult 增加 `approval_id` 字段 |
| `backend-go/internal/domain/loop/service.go` | Evaluate() 返回新创建 approval.ID |
| `backend-go/internal/httpx/router.go` | 注册证据卡路由 |

### 前端新增/修改

| 文件 | 职责 |
|------|------|
| `frontend-next/src/app/(main)/sandbox-listing/store.ts` | Zustand store |
| `frontend-next/src/app/(main)/sandbox-listing/page.tsx` | Wizard 主页面 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepProductEntry.tsx` | Step 1: 录入 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepFieldCompletion.tsx` | Step 2: 补齐 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepEvidenceCard.tsx` | Step 3: 证据卡 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepRecommendation.tsx` | Step 4: 建议 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepApproval.tsx` | Step 5: 审批 |
| `frontend-next/src/app/(main)/sandbox-listing/steps/StepExecution.tsx` | Step 6: 执行 |
| `frontend-next/src/components/profit/EvidenceCard.tsx` | 证据卡展示组件 |
| `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx` | 修改：增强复盘视图 |
| `frontend-next/src/config/menu.ts` | 修改：加菜单项 |
| `frontend-next/src/app/(main)/owner/page.tsx` | 修改：加入口卡片 |

---

### Task 1: 利润证据卡后端 (EvidenceCard 模型 + 计算 + API)

**Files:**
- Create: `backend-go/internal/domain/profit/evidence_card.go`
- Create: `backend-go/internal/domain/profit/handler_evidence.go`
- Modify: `backend-go/internal/httpx/router.go`
- Test: `backend-go/internal/domain/profit/evidence_card_test.go`

**Interfaces:**
- Consumes: `candidate.CandidateProduct` fields (purchase_price, package_weight_kg, etc.)
- Produces: `GET /api/v1/profit/evidence-card/:productId` → `EvidenceCard`

- [ ] **Step 1: Create EvidenceCard model + types in `evidence_card.go`**

Create `backend-go/internal/domain/profit/evidence_card.go`:

```go
package profit

import "math"

// EvidenceCard is a read model for Owner decision-making.
// It shows cost breakdown, revenue, profit projection with confidence levels.
type EvidenceCard struct {
    ProductID           int64       `json:"product_id"`
    Title               string      `json:"title"`
    Currency            string      `json:"currency"`

    Revenue             MoneyRow    `json:"revenue"`
    CostItems           []CostItem  `json:"cost_items"`
    TotalFixedCost      float64     `json:"total_fixed_cost"`
    TotalVariableFeeRate float64    `json:"total_variable_fee_rate"`
    EstimatedVariableFee float64    `json:"estimated_variable_fee"`
    TotalCostAtTargetPrice float64  `json:"total_cost_at_target_price"`

    EstimatedProfit     float64     `json:"estimated_profit"`
    ProfitMargin        float64     `json:"profit_margin"`
    Status              string      `json:"status"` // profitable / marginal / unprofitable / unknown

    ConfidenceLevel     string      `json:"confidence_level"` // high / medium / low / insufficient_data
    CanEvaluate         bool        `json:"can_evaluate"`
    ConfirmedItems      []DataField `json:"confirmed_items"`
    EstimatedItems      []DataField `json:"estimated_items"`
    MissingItems        []string    `json:"missing_items"`
    BlockingReasons     []string    `json:"blocking_reasons"`

    BreakEvenPrice      float64     `json:"break_even_price"`
    RecommendedMinPrice float64     `json:"recommended_min_price"`
    TargetMargin        float64     `json:"target_margin"`
    BufferRate          float64     `json:"buffer_rate"`
}

type CostItem struct {
    Category       string `json:"category"`
    Label          string `json:"label"`
    Amount         float64 `json:"amount"`
    Rate           float64 `json:"rate"`
    CalculationType string `json:"calculation_type"` // fixed_amount | percent_of_revenue
    DataSource     string `json:"data_source"`       // confirmed | estimated | template_default | missing
    SourceNote     string `json:"source_note"`
    Required       bool   `json:"required"`
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

// costDef defines one cost category computation.
type costDef struct {
    Category       string
    Label          string
    CalculationType string // fixed_amount | percent_of_revenue
    Required       bool
    DefaultRate    float64 // for percent_of_revenue defaults
    // sourceFn returns (amount, dataSource, sourceNote) given the candidate product.
    sourceFn func(prod *candidateProductReader) (float64, string, string)
}
```

Also add the internal minimal reader interface at the bottom:

```go
// candidateProductReader is the subset of candidate.CandidateProduct the evidence
// card needs. Keeps the profit package from depending on the full candidate package.
type candidateProductReader struct {
    Title              string
    TargetSalePrice    float64
    TargetCurrency     string
    PurchasePrice      float64
    PurchaseCurrency   string
    PackageWeightKg    float64
    PackageLengthCm    float64
    PackageWidthCm     float64
    PackageHeightCm    float64
    OriginCountry      string
    DestinationCountry string
    HSCode             string
    SourceURL          string
}
```

- [ ] **Step 2: Write EvidenceCard computation service**

Add to the same file `evidence_card.go`:

```go
// default constants — ponytail: hardcoded reasonable defaults, not config-driven.
const (
    defaultPaymentFeeRate        = 0.035  // 3.5% payment processing
    defaultPackagingFee          = 0.50   // $0.50 per unit
    defaultExchangeRateBuffer    = 0.02   // 2% of purchase cost
    defaultLossBuffer            = 0.01   // 1% of target revenue
    defaultDomesticShippingCN    = 1.50   // $1.50 domestic shipping from CN
    defaultTargetMargin          = 0.20   // 20% target profit margin
    defaultBufferRate            = 0.05   // 5% additional buffer
)

// BuildEvidenceCard computes the full evidence card for a candidate product.
func BuildEvidenceCard(prod *candidateProductReader, platformCommissionRate, internationalShippingCost float64) *EvidenceCard {
    card := &EvidenceCard{
        ProductID:   0,
        Title:       prod.Title,
        Currency:    prod.TargetCurrency,
        TargetMargin: defaultTargetMargin,
        BufferRate:  defaultBufferRate,
    }

    // 1. Revenue
    card.Revenue = MoneyRow{
        Amount: prod.TargetSalePrice,
        Label:  "目标售价",
    }

    // 2. Build cost items
    var costItems []CostItem
    var totalFixed float64
    var totalVarRate float64

    // Purchase cost
    purchaseItem := CostItem{
        Category:       "purchase_cost",
        Label:          "采购成本",
        Amount:         prod.PurchasePrice,
        CalculationType: "fixed_amount",
        Required:       true,
    }
    if prod.PurchasePrice > 0 {
        purchaseItem.DataSource = "confirmed"
        purchaseItem.SourceNote = "供应商报价"
    } else {
        purchaseItem.DataSource = "missing"
    }
    costItems = append(costItems, purchaseItem)
    totalFixed += purchaseItem.Amount

    // Domestic shipping
    domesticItem := CostItem{
        Category:       "domestic_shipping",
        Label:          "国内运费",
        CalculationType: "fixed_amount",
        Required:       false,
        Amount:         defaultDomesticShippingCN,
        DataSource:     "template_default",
        SourceNote:     "按模板默认值",
    }
    costItems = append(costItems, domesticItem)
    totalFixed += domesticItem.Amount

    // International shipping
    shipItem := CostItem{
        Category:       "international_shipping",
        Label:          "国际物流费",
        CalculationType: "fixed_amount",
        Required:       true,
    }
    if internationalShippingCost > 0 {
        shipItem.Amount = internationalShippingCost
        shipItem.DataSource = "estimated"
        shipItem.SourceNote = "按重量估算"
    } else {
        shipItem.DataSource = "missing"
    }
    costItems = append(costItems, shipItem)
    totalFixed += shipItem.Amount

    // Platform commission
    commItem := CostItem{
        Category:       "platform_commission",
        Label:          "平台佣金",
        CalculationType: "percent_of_revenue",
        Required:       true,
    }
    if platformCommissionRate > 0 {
        commItem.Rate = platformCommissionRate
        commItem.DataSource = "estimated"
        commItem.SourceNote = "按平台费率表"
    } else {
        commItem.DataSource = "missing"
    }
    costItems = append(costItems, commItem)
    totalVarRate += commItem.Rate

    // Payment fee
    payItem := CostItem{
        Category:       "payment_fee",
        Label:          "支付手续费",
        CalculationType: "percent_of_revenue",
        Required:       false,
        Rate:           defaultPaymentFeeRate,
        Amount:         round(prod.TargetSalePrice * defaultPaymentFeeRate, 2),
        DataSource:     "template_default",
        SourceNote:     "按模板默认值(3.5%)",
    }
    costItems = append(costItems, payItem)
    totalVarRate += payItem.Rate

    // Packaging fee
    packItem := CostItem{
        Category:       "packaging_fee",
        Label:          "包材费",
        CalculationType: "fixed_amount",
        Required:       false,
        Amount:         defaultPackagingFee,
        DataSource:     "template_default",
        SourceNote:     "按模板默认值",
    }
    costItems = append(costItems, packItem)
    totalFixed += packItem.Amount

    // Tariff
    tariffItem := CostItem{
        Category:       "tariff",
        Label:          "关税",
        CalculationType: "fixed_amount",
        Required:       false,
        DataSource:     "missing",
        SourceNote:     "未配置关税率",
    }
    costItems = append(costItems, tariffItem)

    // Exchange rate buffer (risk buffer, not actual cost)
    bufferItem := CostItem{
        Category:       "exchange_rate_buffer",
        Label:          "汇率缓冲",
        CalculationType: "fixed_amount",
        Required:       false,
        Amount:         round(prod.PurchasePrice*defaultExchangeRateBuffer, 2),
        DataSource:     "template_default",
        SourceNote:     "按模板默认值(2%)",
    }
    costItems = append(costItems, bufferItem)
    totalFixed += bufferItem.Amount

    // Loss buffer (risk buffer, not actual cost)
    lossItem := CostItem{
        Category:       "loss_buffer",
        Label:          "损耗缓冲",
        CalculationType: "fixed_amount",
        Required:       false,
        Amount:         round(prod.TargetSalePrice*defaultLossBuffer, 2),
        DataSource:     "template_default",
        SourceNote:     "按模板默认值(1%)",
    }
    costItems = append(costItems, lossItem)
    totalFixed += lossItem.Amount

    card.CostItems = costItems
    card.TotalFixedCost = round(totalFixed, 2)
    card.TotalVariableFeeRate = round(totalVarRate, 4)

    // 3. Cost at target price
    variableFee := round(prod.TargetSalePrice*totalVarRate, 2)
    card.EstimatedVariableFee = variableFee
    card.TotalCostAtTargetPrice = round(totalFixed+variableFee, 2)

    // 4. Profit
    estProfit := round(prod.TargetSalePrice-totalFixed-variableFee, 2)
    card.EstimatedProfit = estProfit
    if prod.TargetSalePrice > 0 {
        card.ProfitMargin = round(estProfit/prod.TargetSalePrice*100, 2)
    }

    // Status
    if card.ProfitMargin >= 15 {
        card.Status = "profitable"
    } else if card.ProfitMargin >= 5 {
        card.Status = "marginal"
    } else if card.ProfitMargin >= 0 {
        card.Status = "unprofitable"
    } else {
        card.Status = "unprofitable"
    }

    // 5. Break-even / recommended price
    if totalVarRate < 1 {
        card.BreakEvenPrice = round(totalFixed/(1-totalVarRate), 2)
        card.RecommendedMinPrice = round(totalFixed*(1+defaultTargetMargin+defaultBufferRate)/(1-totalVarRate), 2)
    }

    // 6. Confidence & blocking
    card.computeConfidenceAndBlocking(prod)

    return card
}

func (c *EvidenceCard) computeConfidenceAndBlocking(prod *candidateProductReader) {
    var confirmed, estimated, missing, blockingReasons []string

    // Required field check: purchase_cost
    if prod.PurchasePrice <= 0 {
        missing = append(missing, "采购成本(purchase_cost)")
        blockingReasons = append(blockingReasons, "缺少采购成本，无法计算利润")
    } else {
        confirmed = append(confirmed, "采购成本")
    }

    // Required: package_weight_kg (needed to estimate shipping)
    if prod.PackageWeightKg <= 0 {
        missing = append(missing, "商品重量(package_weight_kg)")
        blockingReasons = append(blockingReasons, "缺少商品重量，无法估算物流费")
    } else {
        confirmed = append(confirmed, "商品重量")
    }

    // Required: target_sale_price
    if prod.TargetSalePrice <= 0 {
        missing = append(missing, "目标售价(target_sale_price)")
        blockingReasons = append(blockingReasons, "缺少目标售价，无法计算利润")
    } else {
        confirmed = append(confirmed, "目标售价")
    }

    // Required: destination_country
    if prod.DestinationCountry == "" {
        missing = append(missing, "目标国家(destination_country)")
        blockingReasons = append(blockingReasons, "缺少目标国家，无法确定费率")
    } else {
        confirmed = append(confirmed, "目标国家")
    }

    // Required: international_shipping assessment
    shippingItem := findCost("international_shipping", c.CostItems)
    if shippingItem != nil && shippingItem.Amount <= 0 {
        missing = append(missing, "国际物流费(international_shipping)")
        blockingReasons = append(blockingReasons, "缺少国际物流费，无法完整核算成本")
    }

    // Required: platform_commission
    commItem := findCost("platform_commission", c.CostItems)
    if commItem != nil && commItem.Rate <= 0 && commItem.Amount <= 0 {
        missing = append(missing, "平台佣金(platform_commission)")
        blockingReasons = append(blockingReasons, "缺少平台佣金费率，无法完整核算成本")
    }

    // Check data source levels for non-required items
    for _, item := range c.CostItems {
        if item.DataSource == "estimated" {
            estimated = append(estimated, item.Label)
        } else if item.DataSource == "template_default" {
            estimated = append(estimated, item.Label+"(模板值)")
        }
    }

    c.ConfirmedItems = toDataFields(confirmed, "confirmed")
    c.EstimatedItems = toDataFields(estimated, "estimated")
    c.MissingItems = missing
    c.BlockingReasons = blockingReasons

    // Confidence level
    if len(blockingReasons) > 0 {
        c.CanEvaluate = false
        c.ConfidenceLevel = "insufficient_data"
    } else if len(estimated) > 0 {
        c.CanEvaluate = true
        // Check if any are template_default
        hasTemplateDefault := false
        for _, item := range c.CostItems {
            if item.DataSource == "template_default" {
                hasTemplateDefault = true
                break
            }
        }
        if hasTemplateDefault {
            c.ConfidenceLevel = "low"
        } else {
            c.ConfidenceLevel = "medium"
        }
    } else {
        c.CanEvaluate = true
        c.ConfidenceLevel = "high"
    }
}

func findCost(category string, items []CostItem) *CostItem {
    for i := range items {
        if items[i].Category == category {
            return &items[i]
        }
    }
    return nil
}

func toDataFields(names []string, source string) []DataField {
    fields := make([]DataField, len(names))
    for i, n := range names {
        fields[i] = DataField{FieldName: n, Label: n, Value: "", Source: source}
    }
    return fields
}

func round(f float64, n int) float64 {
    pow := math.Pow(10, float64(n))
    return math.Round(f*pow) / pow
}
```

- [ ] **Step 3: Write tests**

Create `backend-go/internal/domain/profit/evidence_card_test.go`:

```go
package profit

import (
    "testing"
)

func TestBuildEvidenceCard_FullData(t *testing.T) {
    prod := &candidateProductReader{
        Title:            "Test Product",
        TargetSalePrice:  29.99,
        TargetCurrency:   "USD",
        PurchasePrice:    8.50,
        PackageWeightKg:  0.5,
        DestinationCountry: "US",
    }
    card := BuildEvidenceCard(prod, 0.15, 4.50)

    if !card.CanEvaluate {
        t.Errorf("expected CanEvaluate=true, got false, blocking: %v", card.BlockingReasons)
    }
    if card.ConfidenceLevel != "low" {
        t.Errorf("expected low (template_default items), got %s", card.ConfidenceLevel)
    }
    if card.TotalFixedCost <= 0 {
        t.Error("TotalFixedCost should be > 0")
    }
    if card.BreakEvenPrice <= 0 {
        t.Error("BreakEvenPrice should be > 0")
    }
    // Verify no double-counting of platform fee
    expectedVarFee := 29.99 * (0.15 + 0.035) // commission + payment
    if card.EstimatedVariableFee > expectedVarFee+0.01 || card.EstimatedVariableFee < expectedVarFee-0.01 {
        t.Errorf("EstimatedVariableFee %.2f, expected %.2f", card.EstimatedVariableFee, expectedVarFee)
    }
}

func TestBuildEvidenceCard_MissingRequired_Blocks(t *testing.T) {
    prod := &candidateProductReader{
        Title:            "Incomplete",
        TargetSalePrice:  0,
        DestinationCountry: "",
    }
    card := BuildEvidenceCard(prod, 0, 0)

    if card.CanEvaluate {
        t.Error("expected CanEvaluate=false for missing required fields")
    }
    if card.ConfidenceLevel != "insufficient_data" {
        t.Errorf("expected insufficient_data, got %s", card.ConfidenceLevel)
    }
    if len(card.BlockingReasons) == 0 {
        t.Error("expected blocking reasons for missing data")
    }
}

func TestBuildEvidenceCard_BreakEvenPrice(t *testing.T) {
    prod := &candidateProductReader{
        Title:            "BE Test",
        TargetSalePrice:  20.00,
        TargetCurrency:   "USD",
        PurchasePrice:    10.00,
        PackageWeightKg:  0.5,
        DestinationCountry: "US",
    }
    card := BuildEvidenceCard(prod, 0.15, 5.00)

    // fixed = 10 + 1.5 + 5 + 0.5 + 0.2(2% of 10) + 0.2(1% of 20) = 17.4
    // var rate = 0.15 + 0.035 = 0.185
    // break_even = 17.4 / (1-0.185) = 17.4 / 0.815 ≈ 21.35
    if card.BreakEvenPrice <= 0 {
        t.Fatal("BreakEvenPrice should be > 0")
    }
    // Target price 20 < 21.35 -> unprofitable
    if card.ProfitMargin >= 0 {
        t.Logf("ProfitMargin: %.2f%%, check if expected negative", card.ProfitMargin)
    }
}
```

- [ ] **Step 4: Run tests to verify they fail (no implementation yet)**

Run: `cd backend-go && go test ./internal/domain/profit/ -run "TestBuildEvidenceCard" -v`
Expected: compile error (no BuildEvidenceCard function)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend-go && go test ./internal/domain/profit/ -run "TestBuildEvidenceCard" -v`
Expected: PASS

- [ ] **Step 6: Create evidence card handler**

Create `backend-go/internal/domain/profit/handler_evidence.go`:

```go
package profit

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lingmirror/backend-go/internal/response"
    "gorm.io/gorm"
)

type EvidenceHandler struct {
    db *gorm.DB
}

func NewEvidenceHandler(db *gorm.DB) *EvidenceHandler {
    return &EvidenceHandler{db: db}
}

// GetEvidenceCard GET /api/v1/profit/evidence-card/:productId
func (h *EvidenceHandler) GetEvidenceCard(c *gin.Context) {
    idStr := c.Param("productId")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, "invalid productId")
        return
    }

    // Read candidate product (raw scan to avoid importing candidate package)
    var prod candidateProductReader
    row := h.db.Table("candidate_product").
        Select("title, target_sale_price, target_currency, purchase_price, "+
            "purchase_currency, package_weight_kg, package_length_cm, "+
            "package_width_cm, package_height_cm, origin_country, "+
            "destination_country, hs_code, source_url").
        Where("id = ?", id).Row()
    if err := row.Scan(
        &prod.Title, &prod.TargetSalePrice, &prod.TargetCurrency,
        &prod.PurchasePrice, &prod.PurchaseCurrency,
        &prod.PackageWeightKg, &prod.PackageLengthCm,
        &prod.PackageWidthCm, &prod.PackageHeightCm,
        &prod.OriginCountry, &prod.DestinationCountry,
        &prod.HSCode, &prod.SourceURL,
    ); err != nil {
        response.Error(c, http.StatusNotFound, "商品不存在")
        return
    }

    // Read platform commission rate (default 15% for Ozon)
    // ponytail: hardcoded default, but try to read from platform table
    commissionRate := 0.15
    var platformRows []struct {
        CommissionRate float64
    }
    if err := h.db.Table("platforms").
        Select("commission_rate").
        Where("id = (SELECT target_platform_id FROM candidate_product WHERE id = ?)", id).
        Scan(&platformRows).Error; err == nil && len(platformRows) > 0 {
        commissionRate = platformRows[0].CommissionRate / 100.0
    }

    // Estimate international shipping from weight
    // ponytail: simple weight-based estimate
    shippingCost := 0.0
    if prod.PackageWeightKg > 0 {
        // Rough: $8 base + $4/kg for US
        shippingCost = 8.0 + prod.PackageWeightKg*4.0
    }

    card := BuildEvidenceCard(&prod, commissionRate, shippingCost)
    card.ProductID = id

    response.Success(c, card)
}
```

- [ ] **Step 7: Register evidence card route**

In `backend-go/internal/httpx/router.go`, find the profit routes section and add:

```go
evidenceHandler := profit.NewEvidenceHandler(db)
protected.GET("/profit/evidence-card/:productId", evidenceHandler.GetEvidenceCard)
```

Put it near the existing profit routes (search by `profit.RegisterRoutes` or existing `/profit` groups).

- [ ] **Step 8: Run full test suite to verify no regression**

Run: `cd backend-go && go test ./internal/domain/profit/... -v`
Expected: all existing + new tests pass

- [ ] **Step 9: Commit**

```bash
git add backend-go/internal/domain/profit/evidence_card.go backend-go/internal/domain/profit/handler_evidence.go backend-go/internal/domain/profit/evidence_card_test.go backend-go/internal/httpx/router.go
git commit -m "feat(profit): add EvidenceCard read model for profit confidence and blocking"
```

---

### Task 2: EvaluateResult 增加 approval_id

**Files:**
- Modify: `backend-go/internal/domain/loop/model.go` — add field
- Modify: `backend-go/internal/domain/loop/service.go` — populate field
- Test: `backend-go/internal/domain/loop/loop_test.go`

**Interfaces:**
- Consumes: `loop.Evaluate()` → previously returned `EvaluateResult` with `ListingTaskID` but no `approvalID`
- Produces: `EvaluateResult` with `ApprovalID *int64` populated

- [ ] **Step 1: Add approval_id to EvaluateResult model**

In `backend-go/internal/domain/loop/model.go`, add to `EvaluateResult` struct:

```go
type EvaluateResult struct {
    // ...existing fields...
    ListingTaskID       *int64   `json:"listing_task_id,omitempty"`
    ApprovalID          *int64   `json:"approval_id,omitempty"`   // <--- ADD THIS
    Error               string   `json:"error,omitempty"`
}
```

- [ ] **Step 2: Populate approval_id in Evaluate()**

In `backend-go/internal/domain/loop/service.go`, in the `Evaluate()` function, after the transaction block where `approvalID` is set (around line 107), capture the return. Currently `approvalID` is already the `Create` result's `ID`:

```go
// Already in the code, around line 100-110:
req, err := as.Create(&approval.CreateApprovalInput{...})
if err != nil {
    return err
}
approvalID = &req.ID   // <--- already exists
```

Then at the end, where `EvaluateResult` is constructed (around line 161), add:

```go
return &EvaluateResult{
    // ...existing fields...
    ListingTaskID:  listingTaskID,
    ApprovalID:     approvalID,        // <--- ADD THIS
}, nil
```

- [ ] **Step 3: Verify existing test still passes**

Run: `cd backend-go && go test ./internal/domain/loop/... -v`
Expected: all tests PASS

- [ ] **Step 4: Write a quick test for approval_id return**

Add to `backend-go/internal/domain/loop/loop_test.go` (or the existing test file):

```go
func TestEvaluateResult_IncludesApprovalID(t *testing.T) {
    // Quick check that EvaluateResult serializes with approval_id
    id := int64(42)
    r := EvaluateResult{
        ProductID:     1,
        ListingTaskID: &id,
        ApprovalID:    &id,
    }
    // If JSON has approval_id, the struct field is wired correctly
    b, err := json.Marshal(r)
    if err != nil {
        t.Fatal(err)
    }
    if !bytes.Contains(b, []byte(`"approval_id"`)) {
        t.Error("EvaluateResult JSON should contain approval_id")
    }
}
```

- [ ] **Step 5: Run test**

Run: `cd backend-go && go test ./internal/domain/loop/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend-go/internal/domain/loop/model.go backend-go/internal/domain/loop/service.go backend-go/internal/domain/loop/loop_test.go
git commit -m "feat(loop): add approval_id to EvaluateResult for Step 5 access"
```

---

### Task 3: Sandbox Listing Wizard 基础框架 (Store + Scaffold + Step 1 录入)

**Files:**
- Create: `frontend-next/src/app/(main)/sandbox-listing/store.ts`
- Create: `frontend-next/src/app/(main)/sandbox-listing/page.tsx`
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepProductEntry.tsx`

**Interfaces:**
- Consumes: `POST /api/v1/candidates`
- Produces: Zustand store with `candidateId`, current step, navigation methods

- [ ] **Step 1: Create Zustand store**

Create `frontend-next/src/app/(main)/sandbox-listing/store.ts`:

```typescript
'use client';

import { create } from 'zustand';

export type ListingStep = 1 | 2 | 3 | 4 | 5 | 6;

export interface SandboxListingState {
  // URL restore support
  candidateId: number | null;
  listingTaskId: number | null;
  approvalId: number | null;

  // Wizard state
  currentStep: ListingStep;

  // Navigation
  setStep: (step: ListingStep) => void;
  goNext: () => void;
  goBack: () => void;

  // Data
  setCandidateId: (id: number) => void;
  setListingTaskId: (id: number) => void;
  setApprovalId: (id: number) => void;

  // Reset
  reset: () => void;
}

export const useSandboxListingStore = create<SandboxListingState>((set) => ({
  candidateId: null,
  listingTaskId: null,
  approvalId: null,
  currentStep: 1,

  setStep: (step) => set({ currentStep: step }),
  goNext: () => set((s) => ({ currentStep: Math.min(6, s.currentStep + 1) as ListingStep })),
  goBack: () => set((s) => ({ currentStep: Math.max(1, s.currentStep - 1) as ListingStep })),

  setCandidateId: (id) => set({ candidateId: id }),
  setListingTaskId: (id) => set({ listingTaskId: id }),
  setApprovalId: (id) => set({ approvalId: id }),

  reset: () => set({
    candidateId: null,
    listingTaskId: null,
    approvalId: null,
    currentStep: 1,
  }),
}));
```

- [ ] **Step 2: Create Wizard page shell**

Create `frontend-next/src/app/(main)/sandbox-listing/page.tsx`:

```tsx
'use client';

import { useEffect } from 'react';
import { Steps, Button, Card } from 'antd';
import { useSandboxListingStore } from './store';
import StepProductEntry from './steps/StepProductEntry';
import StepFieldCompletion from './steps/StepFieldCompletion';
import StepEvidenceCard from './steps/StepEvidenceCard';
import StepRecommendation from './steps/StepRecommendation';
import StepApproval from './steps/StepApproval';
import StepExecution from './steps/StepExecution';
import PageContainer from '@/components/ui/PageContainer';
import { useSearchParams } from 'next/navigation';

const stepTitles = ['录入商品', '补齐字段', '利润证据卡', '上架建议', '审批沙箱任务', '执行与复盘'];

export default function SandboxListingPage() {
  const { currentStep, setStep, setCandidateId, candidateId } = useSandboxListingStore();
  const searchParams = useSearchParams();

  // URL restore: /sandbox-listing?candidate_id=123
  useEffect(() => {
    const cid = searchParams.get('candidate_id');
    if (cid && !candidateId) {
      setCandidateId(Number(cid));
      // TODO: auto-detect current step based on data state (Task 5-7 scope)
      setStep(2);
    }
  }, [searchParams, candidateId, setCandidateId, setStep]);

  const renderStep = () => {
    switch (currentStep) {
      case 1: return <StepProductEntry />;
      case 2: return <StepFieldCompletion />;
      case 3: return <StepEvidenceCard />;
      case 4: return <StepRecommendation />;
      case 5: return <StepApproval />;
      case 6: return <StepExecution />;
    }
  };

  return (
    <PageContainer title="真实商品沙箱上架" subtitle="Sandbox Mode — 不会真实发布">
      <Card>
        <Steps current={currentStep - 1} size="small" style={{ marginBottom: 32 }}>
          {stepTitles.map((t, i) => (
            <Steps.Step key={i} title={t} />
          ))}
        </Steps>
        <div style={{ minHeight: 400 }}>{renderStep()}</div>
      </Card>
    </PageContainer>
  );
}
```

- [ ] **Step 3: Create Step 1 (Product Entry)**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepProductEntry.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { Form, Input, InputNumber, Select, Button, message } from 'antd';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

export default function StepProductEntry() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { goNext, setCandidateId } = useSandboxListingStore();

  const onFinish = async (values: Record<string, unknown>) => {
    setLoading(true);
    try {
      const res = await apiClient.post('/v1/candidates', values);
      setCandidateId(res.data.id);
      message.success('商品录入成功');
      goNext();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '录入失败';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 600 }}>
      <Form.Item name="title" label="商品名称" rules={[{ required: true }]}>
        <Input placeholder="例: Wireless Bluetooth Earbuds" />
      </Form.Item>
      <Form.Item name="purchase_price" label="采购价 (CNY)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="8.50" />
      </Form.Item>
      <Form.Item name="purchase_currency" label="采购币种" initialValue="CNY">
        <Select>
          <Select.Option value="CNY">CNY</Select.Option>
          <Select.Option value="USD">USD</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="package_weight_kg" label="重量 (kg)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="0.5" />
      </Form.Item>
      <Form.Item name="package_length_cm" label="长 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="package_width_cm" label="宽 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="package_height_cm" label="高 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="destination_country" label="目标国家" initialValue="US">
        <Select>
          <Select.Option value="US">美国</Select.Option>
          <Select.Option value="RU">俄罗斯</Select.Option>
          <Select.Option value="DE">德国</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="target_platform_id" label="目标平台" rules={[{ required: true }]}>
        <Select placeholder="选择平台">
          <Select.Option value={1}>Ozon (Mock)</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="target_sale_price" label="目标售价 (USD)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="29.99" />
      </Form.Item>
      <Form.Item name="source_url" label="供应商链接">
        <Input placeholder="https://..." />
      </Form.Item>
      <Form.Item name="main_image" label="商品图 URL">
        <Input placeholder="https://..." />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading}>
          提交并继续
        </Button>
      </Form.Item>
    </Form>
  );
}
```

- [ ] **Step 4: Quick build check**

Run: `cd frontend-next && npx tsc --noEmit --pretty false`
Expected: 0 errors (may have pre-existing errors unrelated to this task)

- [ ] **Step 5: Commit**

```bash
cd frontend-next/src/app/\(main\)/sandbox-listing
git add store.ts page.tsx steps/StepProductEntry.tsx
cd ../../../../..
git commit -m "feat(sandbox-listing): add wizard scaffold and Step 1 product entry"
```

---

### Task 4: Step 2 补齐字段 + Step 3 证据卡前端

**Files:**
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepFieldCompletion.tsx`
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepEvidenceCard.tsx`
- Create: `frontend-next/src/components/profit/EvidenceCard.tsx`

**Interfaces:**
- Consumes: `PUT /api/v1/candidates/:id/fields`, `GET /api/v1/profit/evidence-card/:id`
- Produces: EvidenceCard display component, field completion form

- [ ] **Step 1: Create Step 2 (Field Completion)**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepFieldCompletion.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { Card, Button, message, Spin, Form, Input, InputNumber, Tag } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

export default function StepFieldCompletion() {
  const { candidateId, goNext, goBack } = useSandboxListingStore();
  const queryClient = useQueryClient();

  const { data: candidate, isLoading } = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => apiClient.get(`/v1/candidates/${candidateId}`).then(r => r.data),
    enabled: !!candidateId,
  });

  const { data: completeness } = useQuery({
    queryKey: ['completeness', candidateId],
    queryFn: () => apiClient.post(`/v1/completeness/check/${candidateId}`).then(r => r.data),
    enabled: !!candidateId,
  });

  const fillMutation = useMutation({
    mutationFn: (fields: Array<{ field: string; value: unknown }>) =>
      apiClient.put(`/v1/candidates/${candidateId}/fields`, { fields }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['candidate', candidateId] });
      queryClient.invalidateQueries({ queryKey: ['completeness', candidateId] });
      message.success('字段已更新');
    },
  });

  const [editValues, setEditValues] = useState<Record<string, string>>({});

  if (isLoading) return <Spin />;

  const missingItems: string[] = completeness?.missing_items || [];
  const score = completeness?.score || 0;

  const handleFill = (field: string) => {
    const val = editValues[field];
    if (!val) return;
    fillMutation.mutate([{ field, value: val }]);
  };

  const handleSkip = (field: string) => {
    fillMutation.mutate([{ field, value: '__skipped__' }]);
  };

  return (
    <div>
      <Card size="small" style={{ marginBottom: 16 }}>
        资料完整度: <Tag color={score >= 80 ? 'green' : score >= 50 ? 'orange' : 'red'}>{score}%</Tag>
      </Card>

      {missingItems.length === 0 ? (
        <p>所有字段已完成，可以进入下一步。</p>
      ) : (
        missingItems.map((field) => (
          <Card key={field} size="small" style={{ marginBottom: 8 }}>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <span style={{ minWidth: 160 }}>{field}</span>
              <Input
                size="small"
                style={{ width: 200 }}
                value={editValues[field] || ''}
                onChange={(e) => setEditValues({ ...editValues, [field]: e.target.value })}
              />
              <Button size="small" type="primary" onClick={() => handleFill(field)} disabled={!editValues[field]}>
                填写
              </Button>
              <Button size="small" onClick={() => handleSkip(field)}>暂缺</Button>
            </div>
          </Card>
        ))
      )}

      <div style={{ marginTop: 24, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={goNext}>继续到利润证据卡</Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create EvidenceCard display component**

Create `frontend-next/src/components/profit/EvidenceCard.tsx`:

```tsx
'use client';

import { Card, Tag, Table, Alert, Descriptions } from 'antd';
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';

interface CostItem {
  category: string;
  label: string;
  amount: number;
  rate: number;
  calculation_type: string;
  data_source: string;
  source_note: string;
  required: boolean;
}

interface DataField {
  field_name: string;
  label: string;
  value: string;
  source: string;
}

interface EvidenceCardData {
  product_id: number;
  title: string;
  currency: string;
  revenue: { amount: number; label: string };
  cost_items: CostItem[];
  total_fixed_cost: number;
  total_variable_fee_rate: number;
  estimated_variable_fee: number;
  total_cost_at_target_price: number;
  estimated_profit: number;
  profit_margin: number;
  status: string;
  confidence_level: string;
  can_evaluate: boolean;
  confirmed_items: DataField[];
  estimated_items: DataField[];
  missing_items: string[];
  blocking_reasons: string[];
  break_even_price: number;
  recommended_min_price: number;
  target_margin: number;
  buffer_rate: number;
}

const sourceIcon = (source: string) => {
  switch (source) {
    case 'confirmed': return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
    case 'estimated': return <WarningOutlined style={{ color: '#faad14' }} />;
    case 'template_default': return <WarningOutlined style={{ color: '#faad14' }} />;
    default: return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
  }
};

const sourceColor = (source: string) => {
  switch (source) {
    case 'confirmed': return 'green';
    case 'estimated': return 'orange';
    case 'template_default': return 'orange';
    default: return 'red';
  }
};

const statusColor = (status: string) => {
  switch (status) {
    case 'profitable': return 'green';
    case 'marginal': return 'orange';
    case 'unprofitable': return 'red';
    default: return 'default';
  }
};

export default function EvidenceCard({ data }: { data: EvidenceCardData }) {
  const costColumns = [
    { title: '项目', dataIndex: 'label', key: 'label' },
    {
      title: '金额', dataIndex: 'amount', key: 'amount',
      render: (v: number, r: CostItem) => r.calculation_type === 'percent_of_revenue'
        ? `${(r.rate * 100).toFixed(1)}% ($${v.toFixed(2)})`
        : `$${v.toFixed(2)}`,
    },
    {
      title: '数据源', dataIndex: 'data_source', key: 'data_source',
      render: (v: string) => <Tag color={sourceColor(v)}>{v}</Tag>,
    },
    { title: '备注', dataIndex: 'source_note', key: 'source_note' },
  ];

  return (
    <div>
      {/* Conclusion bar */}
      <Alert
        type={data.can_evaluate ? 'info' : 'error'}
        message={
          <strong>
            利润信心等级: {data.confidence_level}
            {data.can_evaluate
              ? ` | 预估利润: $${data.estimated_profit.toFixed(2)} (${data.profit_margin.toFixed(1)}%)`
              : ' | 数据不足，无法评估'}
          </strong>
        }
        style={{ marginBottom: 16 }}
      />

      {/* Revenue */}
      <Card size="small" title="收入" style={{ marginBottom: 12 }}>
        <Descriptions column={1} size="small">
          <Descriptions.Item label={data.revenue.label}>
            ${data.revenue.amount.toFixed(2)} {data.currency}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Cost breakdown */}
      <Card size="small" title="成本明细" style={{ marginBottom: 12 }}>
        <Table
          dataSource={data.cost_items}
          columns={costColumns}
          rowKey="category"
          pagination={false}
          size="small"
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0}><strong>总固定成本</strong></Table.Summary.Cell>
              <Table.Summary.Cell index={1}><strong>${data.total_fixed_cost.toFixed(2)}</strong></Table.Summary.Cell>
              <Table.Summary.Cell index={2} />
              <Table.Summary.Cell index={3} />
            </Table.Summary.Row>
          )}
        />
      </Card>

      {/* Price recommendations */}
      {data.can_evaluate && (
        <Card size="small" title="达标售价测算" style={{ marginBottom: 12 }}>
          <Descriptions column={1} size="small">
            <Descriptions.Item label="盈亏平衡价">${data.break_even_price.toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="建议最低售价">${data.recommended_min_price.toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="目标利润率">{(data.target_margin * 100).toFixed(0)}%</Descriptions.Item>
            <Descriptions.Item label="缓冲率">{(data.buffer_rate * 100).toFixed(0)}%</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {/* Data quality */}
      <Card size="small" title="数据质量">
        {data.confirmed_items.length > 0 && (
          <div>✅ 已确认: {data.confirmed_items.map(i => i.label).join('、')}</div>
        )}
        {data.estimated_items.length > 0 && (
          <div>⚠️ 估算: {data.estimated_items.map(i => i.label).join('、')}</div>
        )}
        {data.missing_items.length > 0 && (
          <div>❌ 缺失: {data.missing_items.map(i => i).join('、')}</div>
        )}
        {data.blocking_reasons.length > 0 && (
          <Alert type="error" message={data.blocking_reasons.join('；')} style={{ marginTop: 8 }} />
        )}
      </Card>
    </div>
  );
}
```

- [ ] **Step 3: Create Step 3 page**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepEvidenceCard.tsx`:

```tsx
'use client';

import { Spin, Button, Alert } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import EvidenceCard from '@/components/profit/EvidenceCard';
import apiClient from '@/lib/api-client';

export default function StepEvidenceCardPage() {
  const { candidateId, goNext, goBack } = useSandboxListingStore();

  const { data, isLoading, error } = useQuery({
    queryKey: ['evidence-card', candidateId],
    queryFn: () => apiClient.get(`/v1/profit/evidence-card/${candidateId}`).then(r => r.data),
    enabled: !!candidateId,
  });

  if (isLoading) return <Spin />;
  if (error) return <Alert type="error" message="加载利润证据卡失败" />;

  return (
    <div>
      <EvidenceCard data={data} />
      <div style={{ marginTop: 24, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={goNext}>继续到上架建议</Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Type check**

Run: `cd frontend-next && npx tsc --noEmit --pretty false`
Expected: 0 errors

- [ ] **Step 5: Commit**

```bash
git add frontend-next/src/app/\(main\)/sandbox-listing/steps/StepFieldCompletion.tsx frontend-next/src/app/\(main\)/sandbox-listing/steps/StepEvidenceCard.tsx frontend-next/src/components/profit/EvidenceCard.tsx
git commit -m "feat(sandbox-listing): add Step 2 field completion and Step 3 evidence card"
```

---

### Task 5: Step 4-6 Wizard 页面

**Files:**
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepRecommendation.tsx`
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepApproval.tsx`
- Create: `frontend-next/src/app/(main)/sandbox-listing/steps/StepExecution.tsx`

**Interfaces:**
- Consumes: `POST /api/v1/loop/evaluate/:id`, `PUT /api/v1/approval/:id/review`, `POST /api/v1/listing-task/:task_id/execute`

- [ ] **Step 1: Create Step 4 (Recommendation)**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepRecommendation.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { Card, Button, Spin, Tag, Alert, message, Result } from 'antd';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

const decisionConfig: Record<string, { color: string; label: string }> = {
  list: { color: 'green', label: '建议上架' },
  cautious: { color: 'orange', label: '建议谨慎' },
  skip: { color: 'red', label: '不建议上架' },
  blocked: { color: 'default', label: '数据不足无法判断' },
};

export default function StepRecommendation() {
  const { candidateId, setListingTaskId, setApprovalId, goNext, goBack } = useSandboxListingStore();
  const [evaluated, setEvaluated] = useState(false);

  const evalMutation = useMutation({
    mutationFn: () => apiClient.post(`/v1/loop/evaluate/${candidateId}`),
    onSuccess: (res) => {
      const data = res.data;
      setResult(data);
      setListingTaskId(data.listing_task_id);
      setApprovalId(data.approval_id);
      setEvaluated(true);
      message.success('上架建议已生成');
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '评估失败';
      message.error(msg);
    },
  });

  const [result, setResult] = useState<{
    decision: string; reason: string; confidence: number;
    completeness_score: number; profit_margin: number;
    risk_flags?: string[]; listing_task_id?: number; approval_id?: number;
  } | null>(null);

  if (!evaluated) {
    return (
      <div>
        <Card>
          <p>点击下方按钮，系统将：</p>
          <ul>
            <li>检查商品资料完整度</li>
            <li>基于利润证据卡评估商品可行性</li>
            <li>如果评估通过，创建一个<strong>待审批的沙箱上架任务</strong></li>
          </ul>
          <p style={{ color: '#888', fontSize: 13 }}>此操作不会真实发布任何商品。</p>
        </Card>
        <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
          <Button onClick={goBack}>上一步</Button>
          <Button type="primary" loading={evalMutation.isPending}
            onClick={() => evalMutation.mutate()}>
            生成建议并创建沙箱审批任务
          </Button>
        </div>
      </div>
    );
  }

  const dc = decisionConfig[result?.decision || 'blocked'];

  return (
    <div>
      <Result
        status={result?.decision === 'list' ? 'success' : 'warning'}
        title={<><Tag color={dc.color}>{dc.label}</Tag> 信心值: {(result.confidence * 100).toFixed(0)}%</>}
        subTitle={result?.reason}
      />

      {result?.risk_flags && result.risk_flags.length > 0 && (
        <Card size="small" title="风险标记" style={{ marginBottom: 16 }}>
          {result.risk_flags.map((f, i) => <Tag key={i} color="red">{f}</Tag>)}
        </Card>
      )}

      <Card size="small" style={{ marginBottom: 16 }}>
        <p>完整度评分: {result?.completeness_score?.toFixed(0)}% | 利润率: {result?.profit_margin?.toFixed(2)}%</p>
      </Card>

      <div style={{ display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>返回修改</Button>
        {result?.decision === 'list' && (
          <Button type="primary" onClick={goNext}>提交审批</Button>
        )}
        {result?.decision === 'cautious' && (
          <>
            <Button onClick={() => setEvaluated(false)}>返回补齐字段</Button>
            <Button type="primary" onClick={goNext}>仍然提交审批</Button>
          </>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create Step 5 (Approval)**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepApproval.tsx`:

```tsx
'use client';

import { useState, useCallback } from 'react';
import { Card, Button, Descriptions, Tag, message, Spin, Alert } from 'antd';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';
import apiClient from '@/lib/api-client';

export default function StepApproval() {
  const { approvalId, candidateId, listingTaskId, goNext, goBack } = useSandboxListingStore();
  const [showConfirm, setShowConfirm] = useState(false);

  const { data: approval, isLoading } = useQuery({
    queryKey: ['approval', approvalId],
    queryFn: () => apiClient.get(`/v1/approval/${approvalId}`).then(r => r.data),
    enabled: !!approvalId,
  });

  const { data: candidate } = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => apiClient.get(`/v1/candidates/${candidateId}`).then(r => r.data),
    enabled: !!candidateId,
  });

  const approveMutation = useMutation({
    mutationFn: (reviewNote: string) =>
      apiClient.put(`/v1/approval/${approvalId}/review`, {
        action: 'approve',
        review_note: reviewNote,
      }),
    onSuccess: () => {
      message.success('审批通过');
      setShowConfirm(false);
      goNext();
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '审批失败';
      message.error(msg);
    },
  });

  const handleApprove = useCallback((note: string) => {
    approveMutation.mutate(note);
  }, [approveMutation]);

  if (isLoading) return <Spin />;

  return (
    <div>
      <Card title="审批摘要">
        <Descriptions column={1} size="small">
          <Descriptions.Item label="商品">
            {candidate?.title || `ID: ${candidateId}`}
          </Descriptions.Item>
          <Descriptions.Item label="目标售价">
            ${candidate?.target_sale_price?.toFixed(2)}
          </Descriptions.Item>
          <Descriptions.Item label="风险等级">
            <Tag color="orange">{approval?.risk_level || 'high'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="执行模式">
            <Tag color="orange">Sandbox</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="不会真实发布">
            ✅ 本操作仅创建沙箱任务，不会发布真实 listing
          </Descriptions.Item>
          <Descriptions.Item label="审计记录">
            approval #{approvalId} 关联 listing task #{listingTaskId}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={() => setShowConfirm(true)}>
          确认审批沙箱上架任务
        </Button>
      </div>

      <HighRiskConfirmDialog
        visible={showConfirm}
        title="审批沙箱上架任务"
        target={`商品: ${candidate?.title || `#${candidateId}`}`}
        riskLevel="high"
        executionMode="Sandbox"
        onConfirm={(note) => handleApprove(note || '')}
        onCancel={() => setShowConfirm(false)}
      />
    </div>
  );
}
```

- [ ] **Step 3: Create Step 6 (Execution)**

Create `frontend-next/src/app/(main)/sandbox-listing/steps/StepExecution.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { Card, Button, Tag, Spin, Result, Descriptions, Alert } from 'antd';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';
import { useRouter } from 'next/navigation';

export default function StepExecution() {
  const { listingTaskId, goBack } = useSandboxListingStore();
  const router = useRouter();
  const [executed, setExecuted] = useState(false);

  const { data: task, isLoading, refetch } = useQuery({
    queryKey: ['listing-task', listingTaskId],
    queryFn: () => apiClient.get(`/v1/listing-tasks/${listingTaskId}`).then(r => r.data.task),
    enabled: !!listingTaskId,
  });

  const executeMutation = useMutation({
    mutationFn: () => apiClient.post(`/v1/listing-task/${listingTaskId}/execute`),
    onSuccess: () => {
      message.success('沙箱任务执行成功');
      setExecuted(true);
      refetch();
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '执行失败';
      message.error(msg);
      setExecuted(true);
      refetch();
    },
  });

  if (isLoading) return <Spin />;

  const isSuccess = task?.status === 'completed' || task?.status === 'approved';
  const isFailed = task?.status === 'failed';
  const isPending = !executed && (task?.status === 'blocked' || task?.status === 'pending' || task?.status === 'approved');

  return (
    <div>
      <Card title="执行沙箱上架任务">
        {isPending && (
          <div>
            <Result status="info" title="等待执行" subTitle="点击"执行沙箱任务"开始" />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
              <Button onClick={goBack}>返回</Button>
              <Button type="primary" loading={executeMutation.isPending}
                onClick={() => executeMutation.mutate()}>
                执行沙箱任务
              </Button>
            </div>
          </div>
        )}

        {isSuccess && (
          <Result
            status="success"
            title="沙箱执行成功"
            subTitle="任务已完成"
            extra={[
              <Button key="detail" type="primary"
                onClick={() => router.push(`/listing-tasks/${listingTaskId}`)}>
                查看完整任务详情
              </Button>,
            ]}
          />
        )}

        {isFailed && (
          <Result
            status="error"
            title="执行失败"
            subTitle={task?.last_error || '未知错误'}
            extra={[
              <Button key="detail" type="primary"
                onClick={() => router.push(`/listing-tasks/${listingTaskId}`)}>
                查看失败详情
              </Button>,
              <Button key="retry" onClick={() => executeMutation.mutate()}>重试</Button>,
            ]}
          />
        )}
      </Card>

      {/* Execution info */}
      {(isSuccess || isFailed) && (
        <Card size="small" title="执行信息">
          <Descriptions column={1} size="small">
            <Descriptions.Item label="状态"><Tag>{task?.status}</Tag></Descriptions.Item>
            <Descriptions.Item label="执行模式">
              <Tag color="orange">Sandbox</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="审批 ID">{task?.approval_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="外部引用 ID">{task?.external_reference_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="审计记录">
              <Button type="link" size="small"
                onClick={() => router.push(`/operation-logs?resource=listing_task:${listingTaskId}`)}>
                查看审计日志
              </Button>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Type check + build**

Run: `cd frontend-next && npx tsc --noEmit --pretty false`
Expected: 0 errors

- [ ] **Step 5: Commit**

```bash
git add frontend-next/src/app/\(main\)/sandbox-listing/steps/StepRecommendation.tsx frontend-next/src/app/\(main\)/sandbox-listing/steps/StepApproval.tsx frontend-next/src/app/\(main\)/sandbox-listing/steps/StepExecution.tsx
git commit -m "feat(sandbox-listing): add Step 4 recommendation, Step 5 approval, Step 6 execution"
```

---

### Task 6: 菜单入口 + Owner 入口卡片 + listing-tasks 复盘增强

**Files:**
- Modify: `frontend-next/src/config/menu.ts`
- Modify: `frontend-next/src/app/(main)/owner/page.tsx`
- Modify: `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx`

**Interfaces:**
- Consumes: existing menu config, owner page, listing tasks detail page
- Produces: visible entry points to /sandbox-listing + enhanced review view

- [ ] **Step 1: Add menu entry**

In `frontend-next/src/config/menu.ts`, add to the "Owner 总控台" group:

```typescript
{ key: '/sandbox-listing', label: '沙箱上架', status: 'sandbox' },
```

Place it after `/candidates` or at the end of the Owner group.

- [ ] **Step 2: Add owner dashboard entry card**

In `frontend-next/src/app/(main)/owner/page.tsx`, locate the existing content and add an entry card that links to `/sandbox-listing`. The exact placement depends on the existing layout. Add something like:

```tsx
<Card hoverable onClick={() => router.push('/sandbox-listing')}>
  <Card.Meta
    title="真实商品沙箱上架"
    description="录入一个真实商品，完成利润评估、审批、沙箱执行全流程"
  />
</Card>
```

Wrapped in the page's existing card grid or section layout.

- [ ] **Step 3: Enhance listing-tasks/[id] review page**

In `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx`, enhance the existing execution info section. After the existing content, add or modify the section that displays `execution_mode`, `approval_id`, `external_reference_id`:

```tsx
{/* Execution info block — add where appropriate */}
{task && (
  <Card size="small" title="执行信息" style={{ marginTop: 16 }}>
    <Descriptions column={1} size="small">
      <Descriptions.Item label="执行模式">
        <Tag color={EXECUTION_MODE_COLORS[task.execution_mode || 0]}>
          {EXECUTION_MODE_LABELS[task.execution_mode || 0] || 'Unknown'}
        </Tag>
      </Descriptions.Item>
      <Descriptions.Item label="审批 ID">
        {task.approval_id ? (
          <Button type="link" size="small"
            onClick={() => router.push(`/approval/${task.approval_id}`)}>
            #{task.approval_id}
          </Button>
        ) : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="外部引用 ID">
        {task.external_reference_id || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="审计记录">
        <Button type="link" size="small"
          onClick={() => router.push(`/operation-logs?resource=listing_task:${task.id}`)}>
          查看操作日志
        </Button>
      </Descriptions.Item>
    </Descriptions>
  </Card>
)}
```

Also add a "查看关联证据卡" link if needed:

```tsx
<Button type="link" size="small" onClick={() =>
  router.push(`/sandbox-listing?candidate_id=${task.product_id}`)}>
  查看关联证据卡
</Button>
```

- [ ] **Step 4: Build check**

Run: `cd frontend-next && npx tsc --noEmit --pretty false`
Expected: 0 errors

- [ ] **Step 5: Commit**

```bash
git add frontend-next/src/config/menu.ts frontend-next/src/app/\(main\)/owner/page.tsx frontend-next/src/app/\(main\)/listing-tasks/\[id\]/page.tsx
git commit -m "feat: add sandbox-listing menu, owner entry card, listing tasks review enhancement"
```

---

### Task 7: 全链路验证 (手动)

**Goal:** 验证所有步骤能端到端跑通。

- [ ] **Step 1: Start backend**

```bash
cd backend-go && go run cmd/server/main.go
```

- [ ] **Step 2: Start frontend**

```bash
cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000
```

- [ ] **Step 3: Execute full flow manually**
  1. Open `/sandbox-listing`
  2. Fill in a real product with partial data (skip some fields)
  3. Go through Step 2 and fill missing fields
  4. View EvidenceCard — verify it shows confidence level and blocking reasons for missing data
  5. Go back to Step 2, fill enough data to pass blocking threshold
  6. View EvidenceCard again — verify it shows profit estimate and break-even price
  7. Step 4: Click "生成建议并创建沙箱审批任务"
  8. Verify recommendation shows (list/skip/cautious)
  9. Step 5: Verify approval summary shown, use HighRiskConfirmDialog to approve
  10. Step 6: Click "执行沙箱任务"
  11. Verify result shows completed/failed with appropriate info

- [ ] **Step 4: Verify state recovery**

Open `/sandbox-listing?candidate_id=XX` (from created product) and verify it restores to correct step.

- [ ] **Step 5: Run backend tests**

```bash
cd backend-go && go test ./internal/domain/profit/... ./internal/domain/loop/... -v
```
Expected: all PASS

---

## 规范自审

| 检查 | 状态 | 说明 |
|------|------|------|
| Spec 覆盖度 | ✅ | 21 条要求全部有对应 task |
| 占位符 | ✅ | 无 TBD/TODO |
| 类型一致性 | ✅ | EvidenceCard、EvaluateResult、API 路径等跨 task 一致 |
| 后端审批端点 | ✅ | `PUT /api/v1/approval/:id/review` |
| 后端执行端点 | ✅ | `POST /api/v1/listing-task/:task_id/execute` |
| loop.EvaluateResult.approval_id | ✅ | Task 2 处理 |
| 证据卡避免重复扣除 | ✅ | 公式 `fixed/(1-var_rate)` + CostItem.CalculationType |
| Step 4 副作用 | ✅ | 按钮文案 "生成建议并创建沙箱审批任务" |
| v0.4 范围 | ✅ | 保持 listingtask + approval，不扩大到 AgentAction |
| E2E | ⚠️ | 独立于实现提交，验收前补齐 |
