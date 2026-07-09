# Module: `dashboard`

Package: `backend-go/internal/domain/dashboard/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/dashboard/brief` | `h.DailyBrief` |
| `GET` | `/api/v1/dashboard/exceptions` | `h.Exceptions` |
| `GET` | `/api/v1/dashboard/inventory` | `h.Inventory` |
| `GET` | `/api/v1/dashboard/orders` | `h.Orders` |
| `GET` | `/api/v1/dashboard/overview` | `h.Overview` |
| `GET` | `/api/v1/dashboard/rejection-reasons` | `h.RejectionReasons` |

## Models

### `DashboardOverview`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderTotal` | `int64` | `order_total` | `—` |  |
| `OrderByStatus` | `map[string]int64` | `order_by_status` | `—` |  |
| `OrderRevenue` | `float64` | `order_revenue` | `—` |  |
| `OrderProfit` | `float64` | `order_profit` | `—` |  |
| `SkuTotal` | `int64` | `sku_total` | `—` |  |
| `LowStockCount` | `int64` | `low_stock_count` | `—` |  |
| `OutOfStockCount` | `int64` | `out_of_stock_count` | `—` |  |
| `ListingActiveCount` | `int64` | `listing_active_count` | `—` |  |
| `AftersalesPendingCount` | `int64` | `aftersales_pending_count` | `—` |  |
| `ExceptionOpenCount` | `int64` | `exception_open_count` | `—` |  |
| `MonthRevenue` | `float64` | `month_revenue` | `—` |  |
| `MonthCost` | `float64` | `month_cost` | `—` |  |
| `TodaySales` | `float64` | `today_sales` | `—` |  |
| `PendingApprovals` | `int64` | `pending_approvals` | `—` |  |
| `AnomalyCount` | `int64` | `anomaly_count` | `—` |  |
| `AgentSuggestions` | `int64` | `agent_suggestions` | `—` |  |
| `RecentAlerts` | `[]AlertBrief` | `recent_alerts` | `—` |  |
| `PlatformConnections` | `[]PlatformConnectionStatus` | `platform_connections` | `—` |  |
| `AgentStatuses` | `[]AgentStatusEntry` | `agent_statuses` | `—` |  |

### `PlatformConnectionStatus`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `PlatformCode` | `string` | `platform_code` | `—` |  |
| `PlatformName` | `string` | `platform_name` | `—` |  |
| `StoreName` | `string` | `store_name` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `SyncStatus` | `string` | `sync_status` | `—` |  |
| `LastSyncAt` | `*string` | `last_sync_at,omitempty` | `—` |  |
| `LastError` | `string` | `last_error,omitempty` | `—` |  |

### `AgentStatusEntry`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AgentID` | `string` | `agent_id` | `—` |  |
| `Name` | `string` | `name` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `LastActivity` | `*string` | `last_activity,omitempty` | `—` |  |

### `OrderTrendPoint`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Date` | `string` | `date` | `—` |  |
| `OrderCnt` | `int64` | `order_cnt` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |

### `LowStockSku`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `ProductID` | `int64` | `product_id` | `—` |  |
| `Code` | `string` | `code` | `—` |  |
| `SpecDesc` | `string` | `spec_desc` | `—` |  |
| `Stock` | `int` | `stock` | `—` |  |
| `WarningStock` | `int` | `warning_stock` | `—` |  |
| `Warehouse` | `string` | `warehouse` | `—` |  |

### `ExceptionDistribution`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Severity` | `string` | `severity` | `—` |  |
| `SourceModule` | `string` | `source_module` | `—` |  |
| `Cnt` | `int64` | `cnt` | `—` |  |

### `RejectionReasonStat`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AgentID` | `string` | `agent_id` | `—` |  |
| `RejectionReason` | `string` | `rejection_reason` | `—` |  |
| `Count` | `int64` | `count` | `—` |  |

### `DailyBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TodayProfit` | `float64` | `today_profit` | `—` |  |
| `TodayRevenue` | `float64` | `today_revenue` | `—` |  |
| `MonthProfit` | `float64` | `month_profit` | `—` |  |
| `MonthRevenue` | `float64` | `month_revenue` | `—` |  |
| `MonthCost` | `float64` | `month_cost` | `—` |  |
| `OpenExceptionCount` | `int64` | `open_exception_count` | `—` |  |
| `LowStockCount` | `int64` | `low_stock_count` | `—` |  |
| `OutOfStockCount` | `int64` | `out_of_stock_count` | `—` |  |
| `NegativeMarginCount` | `int64` | `negative_margin_count` | `—` |  |
| `PendingSupportCount` | `int64` | `pending_support_count` | `—` |  |
| `PendingAftersalesCount` | `int64` | `pending_aftersales_count` | `—` |  |
| `LowStockSkus` | `[]LowStockSkuBrief` | `low_stock_skus` | `—` |  |
| `NegativeMarginSkus` | `[]NegativeMarginSkuBrief` | `negative_margin_skus` | `—` |  |
| `RecentExceptions` | `[]ExceptionBrief` | `recent_exceptions` | `—` |  |
| `UrgentConversations` | `[]UrgentConversationBrief` | `urgent_conversations` | `—` |  |
| `PlatformConnections` | `[]PlatformConnectionStatus` | `platform_connections` | `—` |  |

### `LowStockSkuBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `ProductID` | `int64` | `product_id` | `—` |  |
| `Code` | `string` | `code` | `—` |  |
| `SpecDesc` | `string` | `spec_desc` | `—` |  |
| `Stock` | `int` | `stock` | `—` |  |
| `WarningStock` | `int` | `warning_stock` | `—` |  |

### `NegativeMarginSkuBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProductID` | `int64` | `product_id` | `—` |  |
| `SkuCode` | `string` | `sku_code` | `—` |  |
| `Title` | `string` | `title` | `—` |  |
| `ProfitMargin` | `float64` | `profit_margin` | `—` |  |
| `EstimatedProfit` | `float64` | `estimated_profit` | `—` |  |

### `ExceptionBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `Severity` | `string` | `severity` | `—` |  |
| `SourceModule` | `string` | `source_module` | `—` |  |
| `Message` | `string` | `message` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `CreatedAt` | `string` | `created_at` | `—` |  |

### `UrgentConversationBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `CustomerName` | `string` | `customer_name` | `—` |  |
| `Subject` | `string` | `subject` | `—` |  |
| `Priority` | `string` | `priority` | `—` |  |
| `Platform` | `string` | `platform` | `—` |  |
| `LastMessageAt` | `*string` | `last_message_at,omitempty` | `—` |  |

### `AlertBrief`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `Severity` | `string` | `severity` | `—` |  |
| `Title` | `string` | `title` | `—` |  |
| `CreatedAt` | `string` | `created_at` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
