# Module: `sku`

Package: `backend-go/internal/domain/sku/`

**Base mount prefix:** `/api/v1`
**Required permission:** `product.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/product-master` | `h.ListProducts` |
| `POST` | `/api/v1/product-master` | `h.CreateProduct` |
| `DELETE` | `/api/v1/product-master/:id` | `h.DeleteProduct` |
| `GET` | `/api/v1/product-master/:id` | `h.GetProduct` |
| `PUT` | `/api/v1/product-master/:id` | `h.UpdateProduct` |
| `GET` | `/api/v1/product-master/:id/skus` | `h.ListSkusByProduct` |
| `GET` | `/api/v1/product-master/:id/specs` | `h.ListSpecs` |
| `POST` | `/api/v1/product-master/:id/specs` | `h.CreateSpec` |
| `DELETE` | `/api/v1/product-master/:id/specs/:spec_id` | `h.DeleteSpec` |
| `PUT` | `/api/v1/product-master/:id/specs/:spec_id` | `h.UpdateSpec` |
| `POST` | `/api/v1/product-master/:id/specs/:spec_id/values` | `h.CreateSpecValue` |
| `GET` | `/api/v1/skus` | `h.ListSkus` |
| `POST` | `/api/v1/skus` | `h.CreateSku` |
| `DELETE` | `/api/v1/skus/:id` | `h.DeleteSku` |
| `GET` | `/api/v1/skus/:id` | `h.GetSku` |
| `PUT` | `/api/v1/skus/:id` | `h.UpdateSku` |
| `DELETE` | `/api/v1/spec-values/:id` | `h.DeleteSpecValue` |
| `PUT` | `/api/v1/spec-values/:id` | `h.UpdateSpecValue` |

## Models

### `Product`
**DB table:** `product`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Subtitle` | `string` | `subtitle` | `subtitle` |  |
| `Description` | `string` | `description` | `description` |  |
| `BrandID` | `int64` | `brand_id` | `brand_id` | default:0 |
| `CategoryID` | `int64` | `category_id` | `category_id` |  |
| `Unit` | `string` | `unit` | `unit` | default:件 |
| `Status` | `int16` | `status` | `status` | default:0 |
| `MainImage` | `string` | `main_image` | `main_image` |  |
| `Images` | `json.RawMessage` | `images` | `images` |  |
| `ProductLengthCm` | `decimal.Decimal` | `product_length_cm` | `product_length_cm` |  |
| `ProductWidthCm` | `decimal.Decimal` | `product_width_cm` | `product_width_cm` |  |
| `ProductHeightCm` | `decimal.Decimal` | `product_height_cm` | `product_height_cm` |  |
| `ProductWeightKg` | `decimal.Decimal` | `product_weight_kg` | `product_weight_kg` |  |
| `PackageLengthCm` | `decimal.Decimal` | `package_length_cm` | `package_length_cm` |  |
| `PackageWidthCm` | `decimal.Decimal` | `package_width_cm` | `package_width_cm` |  |
| `PackageHeightCm` | `decimal.Decimal` | `package_height_cm` | `package_height_cm` |  |
| `PackageWeightKg` | `decimal.Decimal` | `package_weight_kg` | `package_weight_kg` |  |
| `CargoType` | `string` | `cargo_type` | `cargo_type` | default:normal |
| `AiTitle` | `string` | `ai_title` | `ai_title` |  |
| `AiDescription` | `string` | `ai_description` | `ai_description` |  |
| `SeoKeywords` | `json.RawMessage` | `seo_keywords` | `seo_keywords` |  |
| `AiStatus` | `string` | `ai_status` | `ai_status` | default:pending |
| `PlatformStatuses` | `json.RawMessage` | `platform_statuses` | `platform_statuses` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `SpecName`
**DB table:** `spec_name`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `SortOrder` | `int` | `sort_order` | `sort_order` | default:0 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `SpecValue`
**DB table:** `spec_value`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SpecNameID` | `int64` | `spec_name_id` | `spec_name_id` | NOT NULL |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `Value` | `string` | `value` | `value` | NOT NULL |
| `SortOrder` | `int` | `sort_order` | `sort_order` | default:0 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `Sku`
**DB table:** `sku`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `Code` | `string` | `code` | `code` |  |
| `Barcode` | `string` | `barcode` | `barcode` |  |
| `SpecDesc` | `string` | `spec_desc` | `spec_desc` |  |
| `SpecValues` | `json.RawMessage` | `spec_values` | `spec_values` |  |
| `Price` | `decimal.Decimal` | `price` | `price` | default:0 |
| `CostPrice` | `decimal.Decimal` | `cost_price` | `cost_price` | default:0 |
| `MarketPrice` | `decimal.Decimal` | `market_price` | `market_price` | default:0 |
| `Stock` | `int` | `stock` | `stock` | default:0 |
| `LockStock` | `int` | `lock_stock` | `lock_stock` | default:0 |
| `WarningStock` | `int` | `warning_stock` | `warning_stock` | default:0 |
| `Weight` | `decimal.Decimal` | `weight` | `weight` | default:0 |
| `SkuLengthCm` | `decimal.Decimal` | `sku_length_cm` | `sku_length_cm` |  |
| `SkuWidthCm` | `decimal.Decimal` | `sku_width_cm` | `sku_width_cm` |  |
| `SkuHeightCm` | `decimal.Decimal` | `sku_height_cm` | `sku_height_cm` |  |
| `SkuWeightKg` | `decimal.Decimal` | `sku_weight_kg` | `sku_weight_kg` |  |
| `Image` | `string` | `image` | `image` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `ComplianceStatus` | `string` | `compliance_status` | `compliance_status` | default:'' |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
