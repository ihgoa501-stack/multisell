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
