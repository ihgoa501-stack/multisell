# Module: `report`

Package: `backend-go/internal/domain/report/`

**Base mount prefix:** `/api/v1`
**Required permission:** `report.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/report/daily` | `h.DailyReport` |
| `GET` | `/api/v1/report/inventory` | `h.Inventory` |
| `GET` | `/api/v1/report/platform-fee` | `h.PlatformFee` |
| `GET` | `/api/v1/report/profit` | `h.Profit` |
| `GET` | `/api/v1/report/sales` | `h.Sales` |
| `GET` | `/api/v1/report/settlement` | `h.Settlement` |
| `GET` | `/api/v1/report/weekly` | `h.WeeklyReport` |

## Models

### `SalesReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalOrders` | `int64` | `total_orders` | `—` |  |
| `TotalRevenue` | `float64` | `total_revenue` | `—` |  |
| `TotalProfit` | `float64` | `total_profit` | `—` |  |
| `ByPlatform` | `[]PlatformBreakdown` | `by_platform` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |

### `PlatformBreakdown`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `Count` | `int64` | `count` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |

### `ProfitReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalProfit` | `float64` | `total_profit` | `—` |  |
| `ProfitMargin` | `float64` | `profit_margin` | `—` |  |
| `ByPlatform` | `[]PlatformProfit` | `by_platform` | `—` |  |
| `ByCategory` | `[]CategoryProfit` | `by_category` | `—` |  |

### `PlatformProfit`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `ProfitMargin` | `float64` | `profit_margin` | `—` |  |

### `CategoryProfit`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CategoryID` | `int64` | `category_id` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `ProfitMargin` | `float64` | `profit_margin` | `—` |  |

### `InventoryReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuTotal` | `int64` | `sku_total` | `—` |  |
| `TotalStockValue` | `float64` | `total_stock_value` | `—` |  |
| `ByWarehouse` | `[]WarehouseStock` | `by_warehouse` | `—` |  |
| `LowStockTop20` | `[]LowStockRow` | `low_stock_top_20` | `—` |  |

### `WarehouseStock`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Warehouse` | `string` | `warehouse` | `—` |  |
| `SkuCount` | `int64` | `sku_count` | `—` |  |
| `StockValue` | `float64` | `stock_value` | `—` |  |

### `LowStockRow`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `ProductID` | `int64` | `product_id` | `—` |  |
| `Code` | `string` | `code` | `—` |  |
| `SpecDesc` | `string` | `spec_desc` | `—` |  |
| `Stock` | `int` | `stock` | `—` |  |
| `WarningStock` | `int` | `warning_stock` | `—` |  |
| `CostPrice` | `float64` | `cost_price` | `—` |  |

### `SettlementReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalSettlements` | `int64` | `total_settlements` | `—` |  |
| `TotalNet` | `float64` | `total_net` | `—` |  |
| `ReconciliationDist` | `map[string]int64` | `reconciliation_dist` | `—` |  |

### `PlatformFeeReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ByPlatform` | `[]PlatformFeeRow` | `by_platform` | `—` |  |
| `ByFeeType` | `[]FeeTypeRow` | `by_fee_type` | `—` |  |

### `PlatformFeeRow`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `TotalFee` | `float64` | `total_fee` | `—` |  |
| `Count` | `int64` | `count` | `—` |  |

### `FeeTypeRow`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `FeeType` | `string` | `fee_type` | `—` |  |
| `TotalFee` | `float64` | `total_fee` | `—` |  |
| `Count` | `int64` | `count` | `—` |  |

### `DailyReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Date` | `string` | `date` | `—` |  |
| `Sales` | `float64` | `sales` | `—` |  |
| `Orders` | `int64` | `orders` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |
| `NewListings` | `int64` | `new_listings` | `—` |  |
| `Anomalies` | `int64` | `anomalies` | `—` |  |
| `Approvals` | `int64` | `approvals` | `—` |  |
| `AgentProposals` | `int64` | `agent_proposals` | `—` |  |
| `LLMCost` | `float64` | `llm_cost` | `—` |  |

### `WeeklyReport`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `WeekStart` | `string` | `week_start` | `—` |  |
| `WeekEnd` | `string` | `week_end` | `—` |  |
| `DailyReports` | `[]DailyReport` | `daily_reports` | `—` |  |
| `SalesTotal` | `float64` | `sales_total` | `—` |  |
| `ProfitTotal` | `float64` | `profit_total` | `—` |  |
| `OrdersTotal` | `int64` | `orders_total` | `—` |  |
| `AnomaliesTotal` | `int64` | `anomalies_total` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
