# AI 选品（Sourcing）使用指南

> 相关模块：`internal/domain/sourcing/` | Agent: A8 选品盈利分析 | 前端页面：`/sourcing`

## 概述

选品引擎是 A8 Agent 的核心能力，用于分析 1688 商品，估算利润和可行性。它由三部分组成：

- **利润计算器** (`profit.go`) — 纯代码计算，无 LLM 依赖。从 1688 价格、运费估算、平台费率算出净利润和毛利率
- **质量评估器** (`eval.go`) — 确定性评分引擎，从标题、图片、供应商、规格等维度打分（0-100）
- **Agent 决策** (`agent.go`) — A8 Agent 整合利润和质量数据，产出 `sourcing_recommend` 决策（viable / marginal / unviable）

## 使用方式

### 通过 Agent 调用

触发 A8 Agent 的 `sourcing_recommend` 决策点，传入以下上下文：

```json
{
  "source_url": "https://detail.1688.com/offer/xxx.html",
  "price_1688": 45.0,
  "weight_kg": 0.5,
  "destination": "US",
  "markup_pct": 250.0,
  "product_name": "蓝牙耳机",
  "supplier_name": "某某工厂"
}
```

返回结果包含 `profit_breakdown` 和状态判断。

### 通过 API（未接线）

⚠️ Routes 已定义但尚未在 `router.go` 中注册。前端 `/sourcing` 页面调用以下端点:

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/v1/sourcing/fetch` | 抓取并分析 1688 商品 |
| GET | `/api/v1/sourcing/recommendations` | 获取选品推荐列表 |

### 通过前端页面

浏览器访问 `/sourcing`，打开 AI 选品面板，输入 1688 商品 URL 即可触发分析。

## 利润计算器 API

```go
type ProfitInput struct {
    SourcePriceCNY float64 // 1688 价格（CNY）
    WeightKg       float64 // 预估包裹重量（kg）
    Destination    string  // 目标市场：US/EU/JP/RU/BR/AU
    MarkupPct      float64 // 期望加价率（如 250.0 = 2.5 倍）
}
```

运费和平台费率按目的地估算：

| 目的地 | 运费（CNY/kg） | 平台费率 |
|--------|--------------|---------|
| US | 45.0 | 15% |
| EU | 50.0 | 15% |
| JP | 35.0 | 10% |
| RU | 55.0 | 12% |
| BR | 70.0 | 18% |
| AU | 50.0 | 15% |

## 质量评分引擎

评分范围 0-100，各维度权重：

| 维度 | 满分 | 说明 |
|------|------|------|
| 标题 | 30 | ≥20 字符满分 |
| 图片 | 25 | ≥3 张图片满分 |
| 供应商 | 15 | 有供应商名称满分 |
| 描述 | 15 | ≥100 字符满分 |
| 规格丰富度 | 10 | 有规格信息满分 |
| 物流信息 | 5 | 有重量/体积满分 |
| 扣分项 | -20 | 每项警告扣 5 分 |
