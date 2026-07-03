# Logistics Rate Engine（物流费率引擎）使用指南

> 相关模块：`internal/domain/logistics/` | Agent: A10 物流运费引擎

## 概述

Logistics 模块是一个独立的费率引擎，不依赖数据库，纯代码计算。它通过静态费率表（YAML 配置）计算跨境运费报价。与 `backend-go/internal/domain/shipping/` 不同，此模块是轻量的、数据库无关的，适合嵌入 Agent 和测试场景。

## 四种定价模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `first_additional` | 首重价格 + 续重每单位价格 | 快递标准定价 |
| `tiered` | 按重量段定价（如 0-0.5kg = ¥60，0.5-2kg = ¥80） | 邮政小包 |
| `fixed` | 一口价，不论重量 | 特殊渠道 |
| `per_kg` | 每公斤单价 × 重量 | 大货海运空运 |

## 使用方式

### 代码调用

```go
import "github.com/lingmirror/backend-go/internal/domain/logistics"

// 加载费率表
table, err := logistics.LoadRateTableFromYAML([]byte(yamlData))

// 创建服务
svc := logistics.NewService(table)

// 查询报价
cargo := logistics.Cargo{ActualWeightKg: 1.5, LengthCm: 20, WidthCm: 15, HeightCm: 10}
quote, err := svc.GetQuote(cargo, "RU", "normal")
// 或获取最便宜的
cheapest, err := svc.GetCheapestQuote(cargo, "RU", "normal")
```

### 费率表 YAML 格式

```yaml
rate_table:
  - id: 1
    channel_name: "燕文经济线"
    provider_name: "Yanwen"
    rule_type: "first_additional"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 2
    destination_country: "RU"
    cargo_type: "normal"
    first_kg: 1
    first_price: 60
    additional_kg: 0.5
    additional_price: 30
    minimum_charge: 60
    fuel_surcharge_pct: 5
    surcharge_fixed: 2
    currency: "CNY"
    estimated_delivery_min: 10
    estimated_delivery_max: 15

  - id: 2
    channel_name: "航空专线"
    provider_name: "Cainiao"
    rule_type: "per_kg"
    priority: 2
    min_weight_kg: 0
    max_weight_kg: 10
    destination_country: "RU"
    cargo_type: "normal"
    per_kg_price: 55
    minimum_charge: 55
    currency: "CNY"
    estimated_delivery_min: 7
    estimated_delivery_max: 12

  - id: 3
    channel_name: "经济小包"
    provider_name: "Cainiao"
    rule_type: "tiered"
    priority: 3
    min_weight_kg: 0
    max_weight_kg: 2
    destination_country: "KZ"
    cargo_type: "normal"
    tiers:
      - min: 0; max: 0.5; price: 50
      - min: 0; max: 1;   price: 85
      - min: 0; max: 2;   price: 150
    currency: "CNY"
    estimated_delivery_min: 10
    estimated_delivery_max: 18
```

## 报价结果

```go
type QuoteResult struct {
    ChannelName        string
    ProviderName       string
    BaseFee            float64 // 基础运费
    SurchargeFee       float64 // 附加费
    FuelSurchargeFee   float64 // 燃油附加费
    FuelSurchargePct   float64 // 燃油附加费比例
    TotalShippingFee   float64 // 总运费
    Currency            string
    EstimatedDeliveryMin int
    EstimatedDeliveryMax int
}
```

## A10 Agent 接线

A10 Agent (`NewLogisticsOpsAgent`) 支持四个决策点：

| 决策点 | 用途 |
|--------|------|
| `carrier_compare` | 比较承运商报价 |
| `shipping_bill_audit` | 运费账单审核 |
| `carrier_performance` | 承运商绩效分析 |
| `logistics_route_opt` | 物流路线优化 |

接线方式：在 `router.go` 中创建 `logistics.NewService` 并注入到 A10 Agent。

---

## Phase 1: 履约智能中枢（2026-07-03）

### 统一报价入口

所有报价计算统一走 `logistics.RateEngine`。Shipping 模块通过 `ToRateTableEntry()` 将DB渠道/规则转换为 `logistics.RateTableEntry`，然后调用 `logistics.NewRateEngine().CalculateRate()`。

```
Shipping Service.QuoteUnified()
  ├─ 从 DB 读取渠道/区域/规则
  ├─ ToRateTableEntry() → logistics.RateTableEntry
  └─ logistics.RateEngine.CalculateRate() → QuoteResult
```

调用方式：`POST /api/v1/shipping/quote-unified`（与现有 `/shipping/quote` 并存）

### 费率规则版本化

`ShippingQuoteRule` 新增版本字段：

| 字段 | 说明 |
|------|------|
| `effective_start_time` | 规则生效时间 |
| `effective_end_time` | 规则失效时间（空 = 长期有效） |
| `rule_version` | 版本号（自增） |
| `import_batch` | 导入批次标识 |

- `GetActiveRulesAtTime()` 获取指定时间点的有效规则
- `ListRuleVersions()` 列出某渠道的所有规则版本
- 历史订单运费快照记录 `rule_version_id`，可追溯到命中的规则版本

### 订单运费快照

`SalesOrderShippingSnapshot` 新增字段：

| 字段 | 说明 |
|------|------|
| `rule_version_id` | 命中的规则 ID |
| `rule_version` | 命中的规则版本号 |
| `quoted_by` | 报价人（system/user/agent_id） |
| `source_trigger` | 触发来源（manual/agent/auto） |

⚠️ 快照创建后不可变：模型已移除 `UpdatedAt` 自动更新。

### 账单对账

`ShippingBillItem` 新增对账字段：

| 字段 | 说明 |
|------|------|
| `variance_pct` | 差异百分比 |
| `anomaly_type` | 异常类型（overcharge/undercharge） |
| `review_status` | 复核状态（pending/resolved/confirmed） |

- `ReconcileBillBatch()` 按批次执行对账：匹配订单快照、计算差异、标记异常
- `ReviewBillItem()` 更新复核结果
- `ListBillAnomalies()` 列出某批次的异常账单

### A10 可解释建议

新增两个决策点（结构化输出）：

| 决策点 | 用途 | 输出字段 |
|--------|------|----------|
| `bill_discrepancy_advice` | 账单差异建议 | reason, data_basis, risk_level, suggested_action, needs_approval |
| `channel_performance_advice` | 渠道表现建议 | reason, data_basis, risk_level, suggested_action, needs_approval |

所有建议均包含：
- **reason**: 为什么发起这个建议
- **data_basis**: 基于哪些数据
- **risk_level**: low/medium/high
- **suggested_action**: 具体建议操作
- **needs_approval**: 是否需要审批（高风险操作需要）

⚠️ A10 建议仅为建议，不自动修改订单/物流/价格/库存。
