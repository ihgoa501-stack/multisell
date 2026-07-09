# Module: `inventory`

Package: `backend-go/internal/domain/inventory/`

**Base mount prefix:** `/api/v1`
**Required permission:** `inventory.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/inventory` | `h.List` |
| `GET` | `/api/v1/inventory/:id` | `h.Get` |
| `PUT` | `/api/v1/inventory/:id` | `h.Update` |
| `POST` | `/api/v1/inventory/:id/lock` | `h.Lock` |
| `POST` | `/api/v1/inventory/:id/unlock` | `h.Unlock` |
| `GET` | `/api/v1/inventory/allocate/:sku_id` | `h.AllocateStock` |
| `POST` | `/api/v1/inventory/dead-stock/analyze` | `h.IdentifyDeadStock` |
| `GET` | `/api/v1/inventory/dead-stock/logs` | `h.ListDeadStockLogs` |
| `GET` | `/api/v1/inventory/locations` | `h.ListLocations` |
| `GET` | `/api/v1/inventory/logs` | `h.ListLogs` |
| `GET` | `/api/v1/inventory/oversell-report` | `h.OversellReport` |
| `GET` | `/api/v1/inventory/safety-config/:sku_id` | `h.GetSafetyConfig` |
| `PUT` | `/api/v1/inventory/safety-config/:sku_id` | `h.UpsertSafetyConfig` |
| `GET` | `/api/v1/inventory/safety-configs` | `h.ListSafetyConfigs` |
| `GET` | `/api/v1/inventory/sku/:sku_id/warehouses` | `h.ListInventoryBySku` |
| `POST` | `/api/v1/inventory/sync-cross-platform/:productId` | `h.SyncCrossPlatform` |
| `GET` | `/api/v1/inventory/transfers` | `h.ListTransfers` |

## Models

### `Inventory`
**DB table:** `inventory`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `Warehouse` | `string` | `warehouse` | `warehouse` | default:默认仓库 |
| `Location` | `string` | `location` | `location` |  |
| `Quantity` | `int` | `quantity` | `quantity` | default:0 |
| `LockedQuantity` | `int` | `locked_quantity` | `locked_quantity` | default:0 |
| `SafetyStock` | `int` | `safety_stock` | `safety_stock` | default:0 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `InventoryLog`
**DB table:** `inventory_log`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `ChangeType` | `string` | `change_type` | `change_type` | NOT NULL |
| `ChangeQty` | `int` | `change_qty` | `change_qty` | NOT NULL |
| `BeforeQty` | `int` | `before_qty` | `before_qty` |  |
| `AfterQty` | `int` | `after_qty` | `after_qty` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `Operator` | `string` | `operator` | `operator` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `InventoryWarehouse`
**DB table:** `inventory_warehouse`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `WarehouseID` | `int64` | `warehouse_id` | `warehouse_id` | NOT NULL |
| `Quantity` | `int` | `quantity` | `quantity` | default:0 |
| `LockedQuantity` | `int` | `locked_quantity` | `locked_quantity` | default:0 |
| `SafetyStock` | `int` | `safety_stock` | `safety_stock` | default:0 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CrossPlatformSyncResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProductID` | `int64` | `product_id` | `—` |  |
| `AvailableInventory` | `int` | `available_inventory` | `—` |  |
| `TotalCommitted` | `int` | `total_committed` | `—` |  |
| `OversellDetected` | `bool` | `oversell_detected` | `—` |  |
| `OversellBy` | `int` | `oversell_by,omitempty` | `—` |  |
| `PlatformBreakdown` | `[]PlatformCommitment` | `platform_breakdown` | `—` |  |
| `AlertGenerated` | `bool` | `alert_generated` | `—` |  |

### `PlatformCommitment`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Committed` | `int` | `committed` | `—` |  |
| `MaxAllowed` | `int` | `max_allowed` | `—` |  |

### `InventoryOversellLog`
**DB table:** `inventory_oversell_log`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `AvailableStock` | `int` | `available_stock` | `available_stock` | NOT NULL |
| `TotalCommitted` | `int` | `total_committed` | `total_committed` | NOT NULL |
| `OversellBy` | `int` | `oversell_by` | `oversell_by` | default:0 |
| `DetectedAt` | `time.Time` | `detected_at` | `detected_at` |  |
| `ResolvedAt` | `*time.Time` | `resolved_at,omitempty` | `resolved_at` |  |
| `Status` | `string` | `status` | `status` | default:open |

### `InventorySafetyConfig`
**DB table:** `inventory_safety_config`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `MinStockLevel` | `int` | `min_stock_level` | `min_stock_level` | default:0 |
| `MaxStockLevel` | `int` | `max_stock_level` | `max_stock_level` | default:0 |
| `LeadTimeDays` | `int` | `lead_time_days` | `lead_time_days` | default:7 |
| `SafetyDays` | `int` | `safety_days` | `safety_days` | default:7 |
| `DailyAvgSales` | `float64` | `daily_avg_sales` | `daily_avg_sales` | default:0 |
| `AutoReorder` | `bool` | `auto_reorder` | `auto_reorder` | default:false |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `AllocationRecommendation`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `TotalAvailable` | `int` | `total_available` | `—` |  |
| `ReservedTotal` | `int` | `reserved_total` | `—` |  |
| `Recommendations` | `[]PlatformAllocation` | `recommendations` | `—` |  |
| `Unallocated` | `int` | `unallocated` | `—` |  |

### `PlatformAllocation`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `PlatformName` | `string` | `platform_name` | `—` |  |
| `SalesShare` | `float64` | `sales_share` | `—` |  |
| `CurrentStock` | `int` | `current_stock` | `—` |  |
| `Recommended` | `int` | `recommended` | `—` |  |
| `Priority` | `int` | `priority` | `—` |  |

### `DeadStockRecord`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `SkuCode` | `string` | `sku_code` | `—` |  |
| `ProductName` | `string` | `product_name` | `—` |  |
| `Warehouse` | `string` | `warehouse` | `—` |  |
| `CurrentQty` | `int` | `current_qty` | `—` |  |
| `LastMovedAt` | `*time.Time` | `last_moved_at,omitempty` | `—` |  |
| `DaysSinceMove` | `int` | `days_since_move` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Suggestion` | `string` | `suggestion` | `—` |  |

### `DeadStockLog`
**DB table:** `dead_stock_log`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `Quantity` | `int` | `quantity` | `quantity` | default:0 |
| `DaysSinceMove` | `int` | `days_since_move` | `days_since_move` | default:0 |
| `Status` | `string` | `status` | `status` | default:normal |
| `Notes` | `string` | `notes` | `notes` |  |
| `DetectedAt` | `time.Time` | `detected_at` | `detected_at` |  |
| `ResolvedAt` | `*time.Time` | `resolved_at,omitempty` | `resolved_at` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
