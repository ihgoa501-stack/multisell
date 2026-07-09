# Module: `sourcing1688`

Package: `backend-go/internal/domain/sourcing1688/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/sourcing-1688` | `h.List` |
| `POST` | `/api/v1/sourcing-1688` | `h.Create` |
| `DELETE` | `/api/v1/sourcing-1688/:id` | `h.Delete` |
| `GET` | `/api/v1/sourcing-1688/:id` | `h.Get` |
| `PUT` | `/api/v1/sourcing-1688/:id` | `h.Update` |
| `POST` | `/api/v1/sourcing-1688/:id/import` | `h.Import` |
| `POST` | `/api/v1/sourcing-1688/:id/reject` | `h.Reject` |
| `GET` | `/api/v1/sourcing-1688/summary` | `h.Summary` |

## Models

### `Sourcing1688Product`
**DB table:** `sourcing_1688_product`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SourceURL` | `string` | `source_url` | `source_url` | NOT NULL |
| `Title` | `*string` | `title,omitempty` | `title` |  |
| `Price` | `*float64` | `price,omitempty` | `price` |  |
| `MOQ` | `int` | `moq` | `moq` | default:1 |
| `SupplierName` | `string` | `supplier_name` | `supplier_name` |  |
| `ShopURL` | `*string` | `shop_url,omitempty` | `shop_url` |  |
| `ShopLocation` | `*string` | `shop_location,omitempty` | `shop_location` |  |
| `Images` | `*json.RawMessage` | `images,omitempty` | `images` |  |
| `Attributes` | `*json.RawMessage` | `attributes,omitempty` | `attributes` |  |
| `SkuVariants` | `*json.RawMessage` | `sku_variants,omitempty` | `sku_variants` |  |
| `Description` | `*string` | `description,omitempty` | `description` |  |
| `PackageLengthCm` | `*float64` | `package_length_cm,omitempty` | `package_length_cm` |  |
| `PackageWidthCm` | `*float64` | `package_width_cm,omitempty` | `package_width_cm` |  |
| `PackageHeightCm` | `*float64` | `package_height_cm,omitempty` | `package_height_cm` |  |
| `PackageWeightKg` | `*float64` | `package_weight_kg,omitempty` | `package_weight_kg` |  |
| `RawData` | `*json.RawMessage` | `raw_data,omitempty` | `raw_data` |  |
| `Status` | `string` | `status` | `status` | default:collected |
| `ProductID` | `*int64` | `product_id,omitempty` | `product_id` |  |
| `SupplierID` | `*int64` | `supplier_id,omitempty` | `supplier_id` |  |
| `CollectedBy` | `*string` | `collected_by,omitempty` | `collected_by` |  |
| `ImportedBy` | `*string` | `imported_by,omitempty` | `imported_by` |  |
| `ImportedAt` | `*time.Time` | `imported_at,omitempty` | `imported_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceURL` | `string` | `source_url` | `—` |  |
| `Title` | `*string` | `title` | `—` |  |
| `Price` | `*float64` | `price` | `—` |  |
| `MOQ` | `*int` | `moq` | `—` |  |
| `SupplierName` | `string` | `supplier_name` | `—` |  |
| `ShopURL` | `*string` | `shop_url` | `—` |  |
| `ShopLocation` | `*string` | `shop_location` | `—` |  |
| `Description` | `*string` | `description` | `—` |  |
| `ProductID` | `*int64` | `product_id` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `CollectedBy` | `*string` | `collected_by` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `RawData` | `*json.RawMessage` | `raw_data` | `—` |  |

### `UpdateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceURL` | `*string` | `source_url` | `—` |  |
| `Title` | `*string` | `title` | `—` |  |
| `Price` | `*float64` | `price` | `—` |  |
| `MOQ` | `*int` | `moq` | `—` |  |
| `SupplierName` | `*string` | `supplier_name` | `—` |  |
| `ShopURL` | `*string` | `shop_url` | `—` |  |
| `ShopLocation` | `*string` | `shop_location` | `—` |  |
| `Description` | `*string` | `description` | `—` |  |
| `ProductID` | `*int64` | `product_id` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `CollectedBy` | `*string` | `collected_by` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `RawData` | `*json.RawMessage` | `raw_data` | `—` |  |

### `ListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `ProductID` | `*int64` | `` | `—` |  |

### `ImportInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ImportedBy` | `string` | `imported_by` | `—` |  |

### `RejectInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RejectedBy` | `string` | `rejected_by` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |

### `Summary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
