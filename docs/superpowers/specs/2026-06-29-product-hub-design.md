# Product Hub — 全链路产品中台设计

> 版本: v1（第一版可交付范围）
> 日期: 2026-06-29
> 状态: 设计评审通过，待实现

---

## 1. 背景

### 1.1 问题

LingMirror 现有 50+ domain modules，产品相关数据散布在 sku、producthub、brand、category、content、supplier、sourcing、purchase、listing 等模块中。没有一个统一的产品主数据来贯穿全生命周期：

- 产品创意/设计/打样过程不可追踪
- 供应商-产品关联断裂（不知道哪个供应商能做什么）
- 真实成本算不清楚（材料+加工+运费+平台费没有统一视图）
- 一个产品的完整信息需要在多个模块间跳转

### 1.2 业务模式

混合模式：
- **OEM/ODM**：自有设计，找工厂生产
- **选品采购（Catalog）**：从 1688 或供应商目录中选品采购

两种产品共用统一的 product_master 身份，但上游流程不同。

### 1.3 目标

构建 **Product Hub**——一个产品从概念到终端全链路的统一数据层：
- 每个产品有一个统一身份 ID（product_master）
- 所有上下游信息通过这个 ID 串联
- 一个页面看全：设计信息、供应商、成本、上架、销售
- 不替代现有业务模块（listing、purchase、supplier 等），做编排层

---

## 2. 架构设计

### 2.1 分层架构

```
┌──────────────────────────────────────────────────────┐
│                    Product Hub                       │
├──────────────────────────────────────────────────────┤
│                                                      │
│  展示层: 产品档案聚合页（一个页面看全所有信息）        │
│                                                      │
│  编排层: 聚合 API（从各模块拉取数据，统一返回）        │
│                                                      │
│  数据层: product_master + 新表 + 现有模块引用          │
│    ┌──────────────────────────────────────────┐       │
│    │  product_master（核心身份，轻量~12字段）    │       │
│    │    ├── product_variant                    │       │
│    │    ├── product_concept                    │       │
│    │    ├── supplier_offer                     │       │
│    │    ├── sample_request                     │       │
│    │    ├── cost_version                       │       │
│    │    └── 引用现有模块（sku/listing/purchase） │       │
│    └──────────────────────────────────────────┘       │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### 2.2 分层引用规则

```
product_master（身份层）
    ↕
product_variant（变体层，关联 sku.Product）
    ↕
各业务表分层引用（不是所有表 flat 指向 master）：
    ├── product_concept（创意层 → master）
    ├── design_spec_revision（设计层 → variant）
    ├── supplier_offer（供应层 → variant）
    ├── sample_request（打样层 → variant）
    ├── cost_version（成本层 → variant）
    ├── BOM / BOMItem（物料层 → variant）
    ├── channel_listing（渠道层 → variant，引用现有 listing）
    └── ...
```

**product_master_id 作为 denormalized 便利字段**保留在需要快速查询的表中，但逻辑主引用是 variant 或对应层级。

### 2.3 与现有模块的关系

| 现有模块 | Product Hub 的处理方式 |
|---------|----------------------|
| sku.Product | 通过 product_variant.sku_product_id 关联，不重复建表 |
| brand / category | 通过 ID 引用，不复制数据 |
| supplier | 通过 supplier_offer.supplier_id 引用 |
| listing | 通过 channel_listing 引用现有 listing 的 external_id |
| purchase / order | API 聚合时从现有模块拉取 |
| supplychain | 引用已有编排结果 |
| producthub（现有） | 版本管理、关系图谱保留，扩展适配 |

---

## 3. 数据模型（V1）

### 3.1 product_master

```go
type ProductMaster struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductCode     string     `gorm:"uniqueIndex;size:64" json:"product_code"`     // 产品编号 P20260001
    Name            string     `gorm:"size:256" json:"name"`                        // 产品名称
    BrandID         *int64     `json:"brand_id"`                                    // 关联 brand 模块
    CategoryID      *int64     `json:"category_id"`                                 // 关联 category 模块
    BusinessModel   string     `gorm:"size:32" json:"business_model"`               // oem / odm / catalog / private_label
    LifecycleStatus string     `gorm:"size:32;default:idea" json:"lifecycle_status"`
    // idea → researching → sampling → approved → costed → ready_to_list → listed → active → sunset → archived
    OwnerID         int64      `json:"owner_id"`         // 负责人
    TeamID          *int64     `json:"team_id"`          // 团队（多租户预留）
    Description     string     `gorm:"type:text" json:"description"`    // 产品描述
    TargetMarket    string     `gorm:"size:128" json:"target_market"`   // 目标市场 US/JP/SG
    TargetPrice     float64    `json:"target_price"`                    // 目标售价
    TargetMargin    float64    `json:"target_margin"`                   // 目标毛利率
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3.2 product_variant

```go
type ProductVariant struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductMasterID int64      `gorm:"index;not null" json:"product_master_id"`
    SKUProductID    int64      `gorm:"index" json:"sku_product_id"`     // 关联 sku.Product
    SKUCode         string     `gorm:"size:64" json:"sku_code"`
    VariantLabel    string     `gorm:"size:128" json:"variant_label"`   // 黑色-大号
    Barcode         string     `gorm:"size:64" json:"barcode"`
    Attributes      JSON       `gorm:"type:jsonb" json:"attributes"`    // 变体属性组合
    Weight          float64    `json:"weight"`                           // 重量(kg)
    Dimensions      string     `gorm:"size:64" json:"dimensions"`       // 包装尺寸
    CountryOfOrigin string     `gorm:"size:8" json:"country_of_origin"` // 原产国
    HSCode          string     `gorm:"size:32" json:"hs_code"`
    Status          string     `gorm:"size:32" json:"status"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3.3 product_concept

```go
type ProductConcept struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductMasterID int64      `gorm:"index;not null" json:"product_master_id"`
    Brief           string     `gorm:"type:text" json:"brief"`              // 产品创意简述
    TargetCustomer  string     `gorm:"type:text" json:"target_customer"`    // 目标客户
    PainPoint       string     `gorm:"type:text" json:"pain_point"`         // 解决痛点
    MarketResearch  string     `gorm:"type:text" json:"market_research"`    // 市场调研
    CompetitorInfo  string     `gorm:"type:text" json:"competitor_info"`    // 竞品分析
    DesignSource    string     `gorm:"size:32" json:"design_source"`        // internal / agency / supplier
    AttachmentURLs  []string   `gorm:"type:jsonb" json:"attachment_urls"`
    Status          string     `gorm:"size:32;default:draft" json:"status"` // draft / submitted / approved / rejected
    CreatedBy       int64      `json:"created_by"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3.4 design_spec_revision

```go
type DesignSpecRevision struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductMasterID int64      `gorm:"index;not null" json:"product_master_id"`
    ProductVariantID *int64    `json:"product_variant_id"`
    Version         string     `gorm:"size:16" json:"version"`               // V1.0, V1.1
    SpecData        JSON       `gorm:"type:jsonb" json:"spec_data"`          // 规格书（结构化JSON）
    TechPackURL     string     `gorm:"size:512" json:"tech_pack_url"`        // 工程文件
    DrawingsURL     string     `gorm:"size:512" json:"drawings_url"`         // 设计图纸
    ChangeReason    string     `gorm:"type:text" json:"change_reason"`
    ApprovedBy      *int64     `json:"approved_by"`
    ApprovedAt      *time.Time `json:"approved_at"`
    Status          string     `gorm:"size:32;default:draft" json:"status"`  // draft / pending / approved
    CreatedAt       time.Time  `json:"created_at"`
}
```

### 3.5 supplier_offer

```go
type SupplierOffer struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    SupplierID      int64      `gorm:"index;not null" json:"supplier_id"`          // 关联 supplier 模块
    ProductMasterID int64      `gorm:"index;not null" json:"product_master_id"`
    ProductVariantID *int64    `json:"product_variant_id"`
    OfferType       string     `gorm:"size:32" json:"offer_type"`                  // catalog / oem_quote / odm_quote
    UnitCost        float64    `json:"unit_cost"`      // ponytail: float, switch to integer minor units when currency precision matters
    Currency        string     `gorm:"size:8;default:CNY" json:"currency"`
    MOQ             int        `json:"moq"`             // 最小起订量
    LeadTimeDays    int        `json:"lead_time_days"`  // 交期（天）
    Incoterm        string     `gorm:"size:32" json:"incoterm"`                    // FOB / CIF / EXW
    IsPreferred     bool       `gorm:"default:false" json:"is_preferred"`
    ValidFrom       time.Time  `json:"valid_from"`
    ValidTo         time.Time  `json:"valid_to"`
    Notes           string     `gorm:"type:text" json:"notes"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3.6 sample_request

```go
type SampleRequest struct {
    ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductMasterID int64      `gorm:"index;not null" json:"product_master_id"`
    ProductVariantID *int64    `json:"product_variant_id"`
    SupplierOfferID *int64     `json:"supplier_offer_id"`
    SpecRevisionID  *int64     `json:"spec_revision_id"`
    SupplierID      int64      `gorm:"not null" json:"supplier_id"`
    Quantity        int        `json:"quantity"`
    Requirements    string     `gorm:"type:text" json:"requirements"`
    RequestedAt     time.Time  `json:"requested_at"`
    DueAt           *time.Time `json:"due_at"`
    Status          string     `gorm:"size:32;default:pending" json:"status"`
    // pending → in_progress → received → evaluated → approved / rejected

    // V1 精简版：sample_iteration 合并到 sample_request 中
    IterationNo     int        `json:"iteration_no"`
    ReceivedAt      *time.Time `json:"received_at"`
    Evaluation      string     `gorm:"type:text" json:"evaluation"`
    QualityScore    float64    `json:"quality_score"`
    Decision        string     `gorm:"size:32" json:"decision"`    // pass / rework / reject
    ImageURLs       []string   `gorm:"type:jsonb" json:"image_urls"`

    CreatedBy       int64      `json:"created_by"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 3.7 cost_version

```go
type CostVersion struct {
    ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
    ProductMasterID   int64      `gorm:"index;not null" json:"product_master_id"`
    ProductVariantID  *int64     `json:"product_variant_id"`
    Version           string     `gorm:"size:16" json:"version"`       // C20260701-V1
    BaseCost          float64    `json:"base_cost"`                     // 出厂成本
    MaterialCost      float64    `json:"material_cost"`                 // 原材料成本
    PackagingCost     float64    `json:"packaging_cost"`                // 包装成本
    FreightCost       float64    `json:"freight_cost"`                  // 头程运费
    DutyCost          float64    `json:"duty_cost"`                     // 关税
    PlatformFeeRate   float64    `json:"platform_fee_rate"`             // 平台费率(%)
    AdCostEstimate    float64    `json:"ad_cost_estimate"`              // 广告成本预估
    FXRate            float64    `json:"fx_rate"`                       // 汇率
    FXRateDate        *time.Time `json:"fx_rate_date"`
    LandedCost        float64    `json:"landed_cost"`                   // 到仓成本（自动计算）
    RecommendedPrice  float64    `json:"recommended_price"`             // 建议售价
    GrossMargin       float64    `json:"gross_margin"`                  // 毛利率（自动计算）
    EffectiveFrom     time.Time  `json:"effective_from"`
    Status            string     `gorm:"size:16;default:draft" json:"status"` // draft / confirmed
    Notes             string     `gorm:"type:text" json:"notes"`
    CreatedBy         int64      `json:"created_by"`
    CreatedAt         time.Time  `json:"created_at"`
}
```

---

## 4. V1 范围（2 周）

### 4.1 核心逻辑

| 模块 | 说明 |
|------|------|
| product_master | 产品身份管理，生命周期状态流转 |
| product_variant | 变体管理，关联 sku.Product |
| product_concept | 产品创意/立项 |
| supplier_offer | 供应商产品报价 |
| sample_request | 打样申请（含精简版样品评价） |
| cost_version | 成本快照（含汇率、假设条件） |
| 聚合 API | 一个端点返回产品完整信息 |
| 生命周期状态流转 | idea→researching→sampling→...→sunset |

### 4.2 前端

- 产品档案详情页（聚合展示）
- 产品列表页（含生命周期状态筛选）
- 产品创建表单（含概念信息和变体）
- 供应商报价和打样记录区块
- 成本快照展示

### 4.3 明确不做（后续版本）

- BOM / BOMItem
- production_batch / qc_inspection
- sales_snapshot / after_sales_signal
- 完整版 sample_iteration（独立表）
- supplier_capability
- universal lifecycle_event 表
- channel_listing 新表（引用现有 listing）
- 合规/文档管理
- 审批工作流引擎
- 供应商能力匹配

### 4.4 工作量估算

- 新表：6 张
- API 端点：~20（CRUD + 聚合 + 状态流转）
- 后端：~10 天
- 前端：~7 天
- 总预估：~17 天（含测试、迁移、QA）

---

## 5. 后续版本路线

### V2 —— 研发深化

- product_concept/design_spec_revision 完整版
- 审批流程（概念 → 设计 → 打样 → 成本各节点的审批）
- 产品资产表（图片、图纸、视频）
- 合规文档跟踪（证书、检测报告）

### V3 —— 生产深化

- BOM / BOMItem
- production_batch
- qc_inspection
- 生产批次 → BOM → 质检的链路

### V4 —— 数据闭环

- sales_snapshot（从 order 模块聚合）
- after_sales_signal（从 aftersales 模块导入）
- 产品分析面板（生命周期利润、异常检测）
- lifecycle_event timeline

### V5 —— 智能匹配

- supplier_capability 标签体系
- 供应商-产品智能匹配
- 成本优化建议

---

## 6. 实现建议

### 6.1 项目结构（Go）

```
internal/domain/producthub/
├── model.go          // V1 所有表模型
├── handler.go        // CRUD + 聚合端点
├── service.go        // 业务逻辑
├── routes.go         // 路由注册
├── aggregation.go    // 产品档案聚合查询
├── lifecycle.go      // 生命周期状态机
└── migration.go      // 数据库迁移
```

### 6.2 技术约束

- 金额字段使用 `decimal` / integer minor units，不用 float
- JSONB 字段只在非核心查询场景使用，核心查询字段加 GIN 索引
- 聚合查询使用 sectioned 方式（不是一次性 `SELECT *`），支持懒加载
- 生命周期状态用 string 枚举，后端做校验
- 附件不嵌 JSON，V2 抽象为 assets 表；V1 先存 URL

### 6.3 核心聚合 API 设计

```
GET /api/v1/products/:id/hub

返回：
{
  "master": { ... },              // product_master
  "variants": [ ... ],            // product_variant 列表
  "concept": { ... },             // product_concept（如有）
  "latest_spec": { ... },         // 最新 design_spec_revision（如有）
  "supplier_offers": [ ... ],     // 活跃报价
  "current_sample": { ... },      // 最新打样（如有）
  "latest_cost": { ... },         // 已确认的最新成本
  "listings": [ ... ],            // 从现有 listing 模块聚合
  "orders_summary": { ... },      // 从 order 模块聚合（销量/金额）
  "lifecycle_events": [ ... ]     // 生命周期事件
}
```

---

## 7. 参考

- Codex 审查建议（2026-06-29）：分层引用、两条产品线分开、不做第二套 ERP
- Akeneo PIM 产品模型
- ERPNext 的物料/BOM/采购流转
- Saleor 的 product/variant/channel 分离模式
