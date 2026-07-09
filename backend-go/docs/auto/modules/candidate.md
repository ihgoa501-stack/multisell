# Module: `candidate`

Package: `backend-go/internal/domain/candidate/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/candidates` | `h.List` |
| `POST` | `/api/v1/candidates` | `h.Create` |
| `DELETE` | `/api/v1/candidates/:id` | `h.Delete` |
| `GET` | `/api/v1/candidates/:id` | `h.Get` |
| `PUT` | `/api/v1/candidates/:id` | `h.Update` |
| `PUT` | `/api/v1/candidates/:id/fields` | `h.FillFields` |
| `POST` | `/api/v1/candidates/:id/rescrape` | `h.Rescrape` |
| `POST` | `/api/v1/candidates/:id/skip-field` | `h.SkipField` |
| `GET` | `/api/v1/candidates/collect-leads` | `h.ListCollectLeads` |
| `GET` | `/api/v1/candidates/collect-leads/:id` | `h.GetCollectLead` |
| `GET` | `/api/v1/candidates/count` | `h.Count` |
| `GET` | `/api/v1/candidates/dedup` | `h.Dedup` |
| `POST` | `/api/v1/candidates/seed` | `h.Seed` |

## Models

### `CandidateProduct`
**DB table:** `candidate_product`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Title` | `string` | `title` | `title` | NOT NULL |
| `Description` | `string` | `description` | `description` |  |
| `MainImage` | `string` | `main_image` | `main_image` |  |
| `Images` | `json.RawMessage` | `images,omitempty` | `images` |  |
| `CategoryID` | `*int64` | `category_id,omitempty` | `category_id` |  |
| `BrandID` | `*int64` | `brand_id,omitempty` | `brand_id` |  |
| `SpecJSON` | `json.RawMessage` | `spec_json,omitempty` | `spec_json` |  |
| `SupplierID` | `*int64` | `supplier_id,omitempty` | `supplier_id` |  |
| `PurchasePrice` | `float64` | `purchase_price` | `purchase_price` | default:0 |
| `PurchaseCurrency` | `string` | `purchase_currency` | `purchase_currency` | default:CNY |
| `PackageWeightKg` | `float64` | `package_weight_kg` | `package_weight_kg` | default:0 |
| `PackageLengthCm` | `float64` | `package_length_cm` | `package_length_cm` | default:0 |
| `PackageWidthCm` | `float64` | `package_width_cm` | `package_width_cm` | default:0 |
| `PackageHeightCm` | `float64` | `package_height_cm` | `package_height_cm` | default:0 |
| `HSCode` | `string` | `hs_code` | `hs_code` |  |
| `OriginCountry` | `string` | `origin_country` | `origin_country` | default:CN |
| `TargetSalePrice` | `float64` | `target_sale_price` | `target_sale_price` | default:0 |
| `TargetCurrency` | `string` | `target_currency` | `target_currency` | default:USD |
| `TargetPlatformID` | `*int64` | `target_platform_id,omitempty` | `target_platform_id` |  |
| `DestinationCountry` | `string` | `destination_country` | `destination_country` | default:US |
| `Status` | `string` | `status` | `status` |  |
| `IsSeedData` | `bool` | `is_seed_data` | `is_seed_data` | default:false |
| `SourceURL` | `string` | `source_url` | `source_url` | default:'' |
| `SourcePlatform` | `string` | `source_platform` | `source_platform` | default:'' |
| `RawPayload` | `json.RawMessage` | `raw_payload,omitempty` | `raw_payload` |  |
| `CompletenessStatus` | `string` | `completeness_status` | `completeness_status` | default:incomplete |
| `SkippedFields` | `json.RawMessage` | `skipped_fields,omitempty` | `skipped_fields` |  |
| `CollectedAt` | `*time.Time` | `collected_at,omitempty` | `collected_at` |  |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `UpdatedBy` | `string` | `updated_by` | `updated_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CollectLead`
**DB table:** `collect_leads`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Title` | `string` | `title` | `title` | NOT NULL, default:'' |
| `PriceRange` | `string` | `price_range` | `price_range` | default:'' |
| `DetailURL` | `string` | `detail_url` | `detail_url` | default:'' |
| `ImageURL` | `string` | `image_url` | `image_url` | default:'' |
| `ShopHint` | `string` | `shop_hint` | `shop_hint` | default:'' |
| `SourcePageURL` | `string` | `source_page_url` | `source_page_url` | default:'' |
| `Status` | `string` | `status` | `status` | default:pending_detail_collect |
| `CollectedAt` | `*time.Time` | `collected_at,omitempty` | `collected_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CreateCandidateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Title` | `string` | `title` | `—` |  |
| `Description` | `string` | `description` | `—` |  |
| `MainImage` | `string` | `main_image` | `—` |  |
| `Images` | `json.RawMessage` | `images` | `—` |  |
| `CategoryID` | `*int64` | `category_id` | `—` |  |
| `BrandID` | `*int64` | `brand_id` | `—` |  |
| `SpecJSON` | `json.RawMessage` | `spec_json` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `PurchasePrice` | `*float64` | `purchase_price` | `—` |  |
| `PurchaseCurrency` | `string` | `purchase_currency` | `—` |  |
| `PackageWeightKg` | `*float64` | `package_weight_kg` | `—` |  |
| `PackageLengthCm` | `*float64` | `package_length_cm` | `—` |  |
| `PackageWidthCm` | `*float64` | `package_width_cm` | `—` |  |
| `PackageHeightCm` | `*float64` | `package_height_cm` | `—` |  |
| `HSCode` | `string` | `hs_code` | `—` |  |
| `OriginCountry` | `string` | `origin_country` | `—` |  |
| `TargetSalePrice` | `*float64` | `target_sale_price` | `—` |  |
| `TargetCurrency` | `string` | `target_currency` | `—` |  |
| `TargetPlatformID` | `*int64` | `target_platform_id` | `—` |  |
| `DestinationCountry` | `string` | `destination_country` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `CreatedBy` | `string` | `created_by` | `—` |  |
| `SourceURL` | `string` | `source_url` | `—` |  |
| `SourcePlatform` | `string` | `source_platform` | `—` |  |
| `RawPayload` | `json.RawMessage` | `raw_payload` | `—` |  |
| `CompletenessStatus` | `string` | `completeness_status` | `—` |  |
| `CollectedAt` | `*time.Time` | `collected_at` | `—` |  |

### `UpdateCandidateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Title` | `*string` | `title` | `—` |  |
| `Description` | `*string` | `description` | `—` |  |
| `MainImage` | `*string` | `main_image` | `—` |  |
| `Images` | `*json.RawMessage` | `images` | `—` |  |
| `CategoryID` | `*int64` | `category_id` | `—` |  |
| `BrandID` | `*int64` | `brand_id` | `—` |  |
| `SpecJSON` | `*json.RawMessage` | `spec_json` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `PurchasePrice` | `*float64` | `purchase_price` | `—` |  |
| `PurchaseCurrency` | `*string` | `purchase_currency` | `—` |  |
| `PackageWeightKg` | `*float64` | `package_weight_kg` | `—` |  |
| `PackageLengthCm` | `*float64` | `package_length_cm` | `—` |  |
| `PackageWidthCm` | `*float64` | `package_width_cm` | `—` |  |
| `PackageHeightCm` | `*float64` | `package_height_cm` | `—` |  |
| `HSCode` | `*string` | `hs_code` | `—` |  |
| `OriginCountry` | `*string` | `origin_country` | `—` |  |
| `TargetSalePrice` | `*float64` | `target_sale_price` | `—` |  |
| `TargetCurrency` | `*string` | `target_currency` | `—` |  |
| `TargetPlatformID` | `*int64` | `target_platform_id` | `—` |  |
| `DestinationCountry` | `*string` | `destination_country` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `UpdatedBy` | `*string` | `updated_by` | `—` |  |

### `CandidateDetail`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `MissingFields` | `[]string` | `missing_fields` | `—` |  |

### `ListCandidateFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Status` | `string` | `` | `—` |  |
| `Search` | `string` | `` | `—` |  |
| `CompletenessStatus` | `string` | `` | `—` |  |

### `FillField`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Field` | `string` | `field` | `—` |  |
| `Value` | `interface{}` | `value` | `—` |  |

### `FillFieldsInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Fields` | `[]FillField` | `fields` | `—` |  |

### `SkipFieldInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Field` | `string` | `field` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
