# Module: `settlement`

Package: `backend-go/internal/domain/settlement/`

**Base mount prefix:** `/api/v1`
**Required permission:** `settlement.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/settlement` | `h.List` |
| `POST` | `/api/v1/settlement` | `h.Create` |
| `DELETE` | `/api/v1/settlement/:id` | `h.Delete` |
| `GET` | `/api/v1/settlement/:id` | `h.Get` |
| `PUT` | `/api/v1/settlement/:id` | `h.Update` |
| `GET` | `/api/v1/settlement/:id/items` | `h.ListItems` |
| `POST` | `/api/v1/settlement/:id/items` | `h.AddItem` |
| `POST` | `/api/v1/settlement/:id/reconcile` | `h.Reconcile` |
| `PUT` | `/api/v1/settlement/items/:item_id/reconciliation` | `h.UpdateItemReconciliation` |
| `POST` | `/api/v1/settlement/recalculate` | `h.RecalculateAll` |
| `GET` | `/api/v1/settlement/summary` | `h.Summary` |

## Models

### `Settlement`
**DB table:** `settlement`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `PlatformID` | `int64` | `platform_id` | `platform_id` | NOT NULL |
| `SettlementNo` | `string` | `settlement_no` | `settlement_no` |  |
| `PeriodStart` | `*time.Time` | `period_start,omitempty` | `period_start` |  |
| `PeriodEnd` | `*time.Time` | `period_end,omitempty` | `period_end` |  |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `TotalRevenue` | `float64` | `total_revenue` | `total_revenue` | default:0 |
| `TotalFee` | `float64` | `total_fee` | `total_fee` | default:0 |
| `TotalRefund` | `float64` | `total_refund` | `total_refund` | default:0 |
| `TotalNet` | `float64` | `total_net` | `total_net` | default:0 |
| `Status` | `string` | `status` | `status` | default:pending |
| `RawData` | `json.RawMessage` | `raw_data,omitempty` | `raw_data` |  |
| `ImportedAt` | `*time.Time` | `imported_at,omitempty` | `imported_at` |  |
| `SourceType` | `string` | `source_type` | `source_type` | default:manual |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `SettlementItem`
**DB table:** `settlement_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SettlementID` | `int64` | `settlement_id` | `settlement_id` | NOT NULL |
| `TransactionType` | `string` | `transaction_type` | `transaction_type` |  |
| `TransactionID` | `string` | `transaction_id` | `transaction_id` |  |
| `OrderNo` | `string` | `order_no` | `order_no` |  |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `SkuID` | `*int64` | `sku_id,omitempty` | `sku_id` |  |
| `Amount` | `float64` | `amount` | `amount` | default:0 |
| `Fee` | `float64` | `fee` | `fee` | default:0 |
| `Net` | `float64` | `net` | `net` | default:0 |
| `Quantity` | `int` | `quantity` | `quantity` | default:0 |
| `OccurredAt` | `*time.Time` | `occurred_at,omitempty` | `occurred_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `ReconciliationStatus` | `string` | `reconciliation_status` | `reconciliation_status` | default:pending |
| `ReconciliationNote` | `string` | `reconciliation_note` | `reconciliation_note` |  |
| `ReconciledAt` | `*time.Time` | `reconciled_at,omitempty` | `reconciled_at` |  |
| `ReconciledBy` | `string` | `reconciled_by` | `reconciled_by` |  |

### `PlatformSettlementBatch`
**DB table:** `platform_settlement_batch`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `PlatformName` | `string` | `platform_name` | `platform_name` |  |
| `Filename` | `string` | `filename` | `filename` | NOT NULL |
| `RowCount` | `int` | `row_count` | `row_count` | default:0 |
| `MatchedCount` | `int` | `matched_count` | `matched_count` | default:0 |
| `UnmatchedCount` | `int` | `unmatched_count` | `unmatched_count` | default:0 |
| `ImportStatus` | `string` | `import_status` | `import_status` | default:imported |
| `Status` | `string` | `status` | `status` | default:imported |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `PlatformSettlementItem`
**DB table:** `platform_settlement_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `BatchID` | `int64` | `batch_id` | `batch_id` | NOT NULL |
| `RowNumber` | `int` | `row_number` | `row_number` | NOT NULL |
| `Platform` | `string` | `platform` | `platform` |  |
| `StoreName` | `string` | `store_name` | `store_name` |  |
| `PlatformOrderNo` | `string` | `platform_order_no` | `platform_order_no` |  |
| `OrderNo` | `string` | `order_no` | `order_no` |  |
| `TransactionType` | `string` | `transaction_type` | `transaction_type` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `Amount` | `float64` | `amount` | `amount` | default:0 |
| `SettledAt` | `*time.Time` | `settled_at,omitempty` | `settled_at` |  |
| `Description` | `string` | `description` | `description` |  |
| `MatchStatus` | `string` | `match_status` | `match_status` | default:unmatched |
| `MatchedOrderID` | `*int64` | `matched_order_id,omitempty` | `matched_order_id` |  |
| `RawPayload` | `json.RawMessage` | `raw_payload,omitempty` | `raw_payload` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `SettlementDetail`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Settlement` | `Settlement` | `settlement` | `—` |  |
| `Items` | `[]SettlementItem` | `items` | `—` |  |

### `CreateSettlementInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `SettlementNo` | `string` | `settlement_no` | `—` |  |
| `PeriodStart` | `*time.Time` | `period_start` | `—` |  |
| `PeriodEnd` | `*time.Time` | `period_end` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `TotalRevenue` | `*float64` | `total_revenue` | `—` |  |
| `TotalFee` | `*float64` | `total_fee` | `—` |  |
| `TotalRefund` | `*float64` | `total_refund` | `—` |  |
| `TotalNet` | `*float64` | `total_net` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `RawData` | `json.RawMessage` | `raw_data` | `—` |  |
| `ImportedAt` | `*time.Time` | `imported_at` | `—` |  |
| `SourceType` | `string` | `source_type` | `—` |  |
| `Items` | `[]SettlementItemInput` | `items` | `—` |  |

### `SettlementItemInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TransactionType` | `string` | `transaction_type` | `—` |  |
| `TransactionID` | `string` | `transaction_id` | `—` |  |
| `OrderNo` | `string` | `order_no` | `—` |  |
| `OrderID` | `*int64` | `order_id` | `—` |  |
| `SkuID` | `*int64` | `sku_id` | `—` |  |
| `Amount` | `*float64` | `amount` | `—` |  |
| `Fee` | `*float64` | `fee` | `—` |  |
| `Net` | `*float64` | `net` | `—` |  |
| `Quantity` | `*int` | `quantity` | `—` |  |
| `OccurredAt` | `*time.Time` | `occurred_at` | `—` |  |

### `UpdateSettlementInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `PeriodStart` | `*time.Time` | `period_start` | `—` |  |
| `PeriodEnd` | `*time.Time` | `period_end` | `—` |  |
| `Currency` | `*string` | `currency` | `—` |  |
| `TotalRevenue` | `*float64` | `total_revenue` | `—` |  |
| `TotalFee` | `*float64` | `total_fee` | `—` |  |
| `TotalRefund` | `*float64` | `total_refund` | `—` |  |
| `TotalNet` | `*float64` | `total_net` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `RawData` | `*json.RawMessage` | `raw_data` | `—` |  |

### `SettlementListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |

### `ReconcileInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ItemID` | `*int64` | `item_id` | `—` |  |
| `ReconciliationStatus` | `string` | `reconciliation_status` | `—` |  |
| `ReconciliationNote` | `string` | `reconciliation_note` | `—` |  |
| `ReconciledBy` | `string` | `reconciled_by` | `—` |  |

### `SettlementSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |
| `NetByPlatform` | `[]PlatformNetTotal` | `net_by_platform` | `—` |  |

### `PlatformNetTotal`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `TotalNet` | `float64` | `total_net` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
