# Module: `orderimport`

Package: `backend-go/internal/domain/orderimport/`

**Base mount prefix:** `/api/v1`
**Required permission:** `order.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/order-import` | `h.List` |
| `POST` | `/api/v1/order-import` | `h.Create` |
| `DELETE` | `/api/v1/order-import/:id` | `h.Delete` |
| `GET` | `/api/v1/order-import/:id` | `h.Get` |
| `PUT` | `/api/v1/order-import/:id` | `h.Update` |
| `POST` | `/api/v1/order-import/:id/complete` | `h.Complete` |
| `POST` | `/api/v1/order-import/:id/process` | `h.Process` |
| `GET` | `/api/v1/order-import/summary` | `h.Summary` |

## Models

### `OrderImport`
**DB table:** `order_import`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `platform_id` |  |
| `SourceType` | `string` | `source_type` | `source_type` | default:manual |
| `FileName` | `string` | `file_name` | `file_name` |  |
| `TotalRows` | `int` | `total_rows` | `total_rows` | default:0 |
| `SuccessCount` | `int` | `success_count` | `success_count` | default:0 |
| `ErrorCount` | `int` | `error_count` | `error_count` | default:0 |
| `ErrorDetail` | `json.RawMessage` | `error_detail,omitempty` | `error_detail` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `OrderImportBatch`
**DB table:** `order_import_batch`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `AdapterCode` | `string` | `adapter_code` | `adapter_code` | NOT NULL |
| `Platform` | `string` | `platform` | `platform` |  |
| `StoreName` | `string` | `store_name` | `store_name` |  |
| `SourceFilename` | `string` | `source_filename` | `source_filename` | NOT NULL |
| `RowCount` | `int` | `row_count` | `row_count` | default:0 |
| `CreatedOrderCount` | `int` | `created_order_count` | `created_order_count` | default:0 |
| `SkippedDuplicateCount` | `int` | `skipped_duplicate_count` | `skipped_duplicate_count` | default:0 |
| `FailedCount` | `int` | `failed_count` | `failed_count` | default:0 |
| `ImportedBy` | `string` | `imported_by` | `imported_by` |  |
| `ChainStatus` | `string` | `chain_status` | `chain_status` | default:chain_pending |
| `LedgerRebuiltCount` | `int` | `ledger_rebuilt_count` | `ledger_rebuilt_count` | default:0 |
| `ExceptionGeneratedCount` | `int` | `exception_generated_count` | `exception_generated_count` | default:0 |
| `ChainFailureCount` | `int` | `chain_failure_count` | `chain_failure_count` | default:0 |
| `ProcessedAt` | `*time.Time` | `processed_at,omitempty` | `processed_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `OrderImportItem`
**DB table:** `order_import_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `BatchID` | `int64` | `batch_id` | `batch_id` | NOT NULL |
| `RowNumber` | `int` | `row_number` | `row_number` | NOT NULL |
| `Platform` | `string` | `platform` | `platform` |  |
| `StoreName` | `string` | `store_name` | `store_name` |  |
| `PlatformOrderNo` | `string` | `platform_order_no` | `platform_order_no` |  |
| `OrderNo` | `string` | `order_no` | `order_no` |  |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `SkuCode` | `string` | `sku_code` | `sku_code` | NOT NULL |
| `Quantity` | `int` | `quantity` | `quantity` | NOT NULL |
| `UnitPrice` | `*float64` | `unit_price,omitempty` | `unit_price` |  |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `RecipientName` | `string` | `recipient_name` | `recipient_name` |  |
| `RecipientPhone` | `string` | `recipient_phone` | `recipient_phone` |  |
| `CountryCode` | `string` | `country_code` | `country_code` |  |
| `ShippingAddress` | `string` | `shipping_address` | `shipping_address` |  |
| `ShippingFee` | `*float64` | `shipping_fee,omitempty` | `shipping_fee` |  |
| `TrackingNumber` | `string` | `tracking_number` | `tracking_number` |  |
| `PaidAt` | `string` | `paid_at` | `paid_at` |  |
| `Status` | `string` | `status` | `status` | NOT NULL |
| `FailureReason` | `string` | `failure_reason` | `failure_reason` |  |
| `ChainStatus` | `string` | `chain_status` | `chain_status` | default:chain_pending |
| `ChainFailureReason` | `string` | `chain_failure_reason` | `chain_failure_reason` |  |
| `RawPayload` | `json.RawMessage` | `raw_payload,omitempty` | `raw_payload` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CreateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `SourceType` | `string` | `source_type` | `—` |  |
| `FileName` | `string` | `file_name` | `—` |  |
| `TotalRows` | `*int` | `total_rows` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `CreatedBy` | `string` | `created_by` | `—` |  |

### `UpdateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `SourceType` | `*string` | `source_type` | `—` |  |
| `FileName` | `*string` | `file_name` | `—` |  |
| `TotalRows` | `*int` | `total_rows` | `—` |  |
| `SuccessCount` | `*int` | `success_count` | `—` |  |
| `ErrorCount` | `*int` | `error_count` | `—` |  |
| `ErrorDetail` | `*json.RawMessage` | `error_detail` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |

### `ListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |
| `SourceType` | `string` | `` | `—` |  |

### `CompleteInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SuccessCount` | `int` | `success_count` | `—` |  |
| `ErrorCount` | `int` | `error_count` | `—` |  |
| `ErrorDetail` | `json.RawMessage` | `error_detail` | `—` |  |
| `Status` | `string` | `status` | `—` |  |

### `Summary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |
| `TotalRows` | `int64` | `total_rows` | `—` |  |
| `SuccessCount` | `int64` | `success_count` | `—` |  |
| `ErrorCount` | `int64` | `error_count` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
