# Module: `listingtask`

Package: `backend-go/internal/domain/listingtask/`

**Base mount prefix:** `/api/v1`
**Required permission:** `listing.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/listing-task/:task_id/execute` | `h.Execute` |
| `POST` | `/api/v1/listing-task/:task_id/items/:item_id/retry` | `h.RetryItem` |
| `POST` | `/api/v1/listing-task/:task_id/retry-failed` | `h.RetryFailed` |
| `POST` | `/api/v1/listing-task/retry-all` | `h.RetryAll` |
| `GET` | `/api/v1/listing-task/stats` | `h.ListStats` |
| `GET` | `/api/v1/listing-tasks` | `h.List` |
| `POST` | `/api/v1/listing-tasks` | `h.Create` |
| `DELETE` | `/api/v1/listing-tasks/:id` | `h.Delete` |
| `GET` | `/api/v1/listing-tasks/:id` | `h.Get` |
| `PUT` | `/api/v1/listing-tasks/:id` | `h.Update` |
| `GET` | `/api/v1/listing-tasks/:id/items` | `h.ListItems` |
| `POST` | `/api/v1/listing-tasks/:id/items` | `h.CreateItem` |
| `DELETE` | `/api/v1/listing-tasks/:id/items/:item_id` | `h.DeleteItem` |
| `PUT` | `/api/v1/listing-tasks/:id/items/:item_id` | `h.UpdateItem` |
| `GET` | `/api/v1/listing-tasks/:id/review` | `h.Review` |
| `POST` | `/api/v1/listing-tasks/from-suggestion` | `h.CreateFromSuggestion` |

## Models

### `ListingTask`
**DB table:** `listing_task`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `PlatformID` | `int64` | `platform_id` | `platform_id` | NOT NULL |
| `SkuID` | `*int64` | `sku_id,omitempty` | `sku_id` |  |
| `ProductListingID` | `*int64` | `product_listing_id,omitempty` | `product_listing_id` |  |
| `SourceType` | `string` | `source_type` | `source_type` | default:decision |
| `SourceItemKey` | `string` | `source_item_key` | `source_item_key` |  |
| `Status` | `string` | `status` | `status` | default:blocked |
| `MissingRequirements` | `json.RawMessage` | `missing_requirements,omitempty` | `missing_requirements` |  |
| `DecisionSnapshot` | `json.RawMessage` | `decision_snapshot,omitempty` | `decision_snapshot` |  |
| `TargetSalePrice` | `*float64` | `target_sale_price,omitempty` | `target_sale_price` |  |
| `TargetProfitMargin` | `*float64` | `target_profit_margin,omitempty` | `target_profit_margin` |  |
| `DestinationCountry` | `string` | `destination_country` | `destination_country` |  |
| `ApprovalID` | `*int64` | `approval_id,omitempty` | `approval_id` |  |
| `LastError` | `string` | `last_error` | `last_error` |  |
| `DryRun` | `bool` | `dry_run` | `—` |  |
| `ExecutionMode` | `ExecutionMode` | `execution_mode` | `execution_mode` | default:0 |
| `ExternalReferenceID` | `string` | `external_reference_id,omitempty` | `external_reference_id` |  |
| `ExternalReferenceURL` | `string` | `external_reference_url,omitempty` | `external_reference_url` |  |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `UpdatedBy` | `string` | `updated_by` | `updated_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ListingTaskItem`
**DB table:** `listing_task_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `TaskID` | `int64` | `task_id` | `task_id` | NOT NULL |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `PlatformID` | `int64` | `platform_id` | `platform_id` | NOT NULL |
| `Status` | `string` | `status` | `status` | default:pending |
| `Result` | `json.RawMessage` | `result,omitempty` | `result` |  |
| `ErrorMessage` | `string` | `error_message` | `error_message` |  |
| `RetryCount` | `int` | `retry_count` | `retry_count` | default:0 |
| `ExecutedAt` | `*time.Time` | `executed_at,omitempty` | `executed_at` |  |

### `CreateTaskInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProductID` | `int64` | `product_id` | `—` |  |
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `SkuID` | `*int64` | `sku_id` | `—` |  |
| `ProductListingID` | `*int64` | `product_listing_id` | `—` |  |
| `SourceType` | `string` | `source_type` | `—` |  |
| `SourceItemKey` | `string` | `source_item_key` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `MissingRequirements` | `json.RawMessage` | `missing_requirements` | `—` |  |
| `DecisionSnapshot` | `json.RawMessage` | `decision_snapshot` | `—` |  |
| `TargetSalePrice` | `*float64` | `target_sale_price` | `—` |  |
| `TargetProfitMargin` | `*float64` | `target_profit_margin` | `—` |  |
| `DestinationCountry` | `string` | `destination_country` | `—` |  |
| `ApprovalID` | `*int64` | `approval_id` | `—` |  |
| `DryRun` | `*bool` | `dry_run` | `—` |  |
| `CreatedBy` | `string` | `created_by` | `—` |  |

### `UpdateTaskInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Status` | `*string` | `status` | `—` |  |
| `SourceItemKey` | `*string` | `source_item_key` | `—` |  |
| `MissingRequirements` | `*json.RawMessage` | `missing_requirements` | `—` |  |
| `DecisionSnapshot` | `*json.RawMessage` | `decision_snapshot` | `—` |  |
| `TargetSalePrice` | `*float64` | `target_sale_price` | `—` |  |
| `TargetProfitMargin` | `*float64` | `target_profit_margin` | `—` |  |
| `DestinationCountry` | `*string` | `destination_country` | `—` |  |
| `ApprovalID` | `*int64` | `approval_id` | `—` |  |
| `LastError` | `*string` | `last_error` | `—` |  |
| `ProductListingID` | `*int64` | `product_listing_id` | `—` |  |
| `UpdatedBy` | `*string` | `updated_by` | `—` |  |
| `ExecutionMode` | `*ExecutionMode` | `execution_mode` | `—` |  |
| `ExternalReferenceID` | `*string` | `external_reference_id` | `—` |  |
| `ExternalReferenceURL` | `*string` | `external_reference_url` | `—` |  |

### `CreateTaskItemInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TaskID` | `int64` | `task_id` | `—` |  |
| `ProductID` | `int64` | `product_id` | `—` |  |
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Result` | `json.RawMessage` | `result` | `—` |  |

### `CreateFromSuggestionInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CandidateID` | `uint` | `candidate_id` | `—` |  |

### `UpdateTaskItemInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Status` | `*string` | `status` | `—` |  |
| `Result` | `*json.RawMessage` | `result` | `—` |  |
| `ErrorMessage` | `*string` | `error_message` | `—` |  |
| `RetryCount` | `*int` | `retry_count` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
