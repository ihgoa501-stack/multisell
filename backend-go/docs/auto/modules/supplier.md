# Module: `supplier`

Package: `backend-go/internal/domain/supplier/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/product-suppliers` | `h.ListProductSuppliers` |
| `POST` | `/api/v1/product-suppliers` | `h.CreateProductSupplier` |
| `DELETE` | `/api/v1/product-suppliers/:id` | `h.DeleteProductSupplier` |
| `PUT` | `/api/v1/product-suppliers/:id` | `h.UpdateProductSupplier` |
| `GET` | `/api/v1/product-suppliers/comparison` | `h.GetSupplierComparison` |
| `GET` | `/api/v1/suppliers` | `h.List` |
| `POST` | `/api/v1/suppliers` | `h.Create` |
| `DELETE` | `/api/v1/suppliers/:id` | `h.Delete` |
| `GET` | `/api/v1/suppliers/:id` | `h.Get` |
| `PUT` | `/api/v1/suppliers/:id` | `h.Update` |
| `PUT` | `/api/v1/suppliers/:id/kpi-score` | `h.UpdateScoreManual` |
| `POST` | `/api/v1/suppliers/:id/recalculate` | `h.RecalculateScore` |
| `GET` | `/api/v1/suppliers/:id/score` | `h.GetScore` |
| `GET` | `/api/v1/suppliers/:id/score-history` | `h.GetScoreHistory` |
| `POST` | `/api/v1/suppliers/:id/score-snapshot` | `h.RecordScoreSnapshot` |
| `GET` | `/api/v1/suppliers/scoreboard` | `h.ListScoreboard` |

## Models

### `Supplier`
**DB table:** `supplier`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `ContactPerson` | `string` | `contact_person` | `contact_person` |  |
| `ContactPhone` | `string` | `contact_phone` | `contact_phone` |  |
| `Email` | `string` | `email` | `email` |  |
| `Address` | `string` | `address` | `address` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `Remark` | `string` | `remark` | `remark` |  |
| `KpiScore` | `float64` | `kpi_score` | `kpi_score` | default:0 |
| `PriceHistory` | `json.RawMessage` | `price_history,omitempty` | `price_history` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ProductSupplier`
**DB table:** `product_supplier`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `SupplierID` | `int64` | `supplier_id` | `supplier_id` | NOT NULL |
| `SupplyPrice` | `*decimal.Decimal` | `supply_price,omitempty` | `supply_price` |  |
| `MinOrderQty` | `int` | `min_order_qty` | `min_order_qty` | default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `SupplierScore`
**DB table:** `supplier_score`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SupplierID` | `int64` | `supplier_id` | `supplier_id` | NOT NULL |
| `OnTimeDeliveryRate` | `float64` | `on_time_delivery_rate` | `on_time_delivery_rate` | default:0 |
| `QualityPassRate` | `float64` | `quality_pass_rate` | `quality_pass_rate` | default:0 |
| `CommunicationScore` | `float64` | `communication_score` | `communication_score` | default:0 |
| `OrderFulfillmentPct` | `float64` | `order_fulfillment_pct` | `order_fulfillment_pct` | default:0 |
| `AvgLeadTimeDays` | `float64` | `avg_lead_time_days` | `avg_lead_time_days` | default:0 |
| `ReliabilityScore` | `float64` | `reliability_score` | `reliability_score` | default:0 |
| `DataFreshness` | `int` | `data_freshness` | `data_freshness` | default:0 |
| `LastOrderDate` | `*time.Time` | `last_order_date,omitempty` | `last_order_date` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `SupplierScoreHistory`
**DB table:** `supplier_score_history`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SupplierID` | `int64` | `supplier_id` | `supplier_id` | NOT NULL |
| `OnTimeDeliveryRate` | `float64` | `on_time_delivery_rate` | `on_time_delivery_rate` | default:0 |
| `QualityPassRate` | `float64` | `quality_pass_rate` | `quality_pass_rate` | default:0 |
| `CommunicationScore` | `float64` | `communication_score` | `communication_score` | default:0 |
| `OrderFulfillmentPct` | `float64` | `order_fulfillment_pct` | `order_fulfillment_pct` | default:0 |
| `AvgLeadTimeDays` | `float64` | `avg_lead_time_days` | `avg_lead_time_days` | default:0 |
| `ReliabilityScore` | `float64` | `reliability_score` | `reliability_score` | default:0 |
| `Trigger` | `string` | `trigger` | `trigger` | default:auto |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `SupplierComparisonResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProductID` | `int64` | `product_id` | `—` |  |
| `ProductName` | `string` | `product_name` | `—` |  |
| `Suppliers` | `[]SupplierRow` | `suppliers` | `—` |  |
| `SpecNames` | `map[string]string` | `spec_names,omitempty` | `—` |  |

### `SupplierRow`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SupplierID` | `int64` | `supplier_id` | `—` |  |
| `SupplierName` | `string` | `supplier_name` | `—` |  |
| `SupplyPrice` | `*decimal.Decimal` | `supply_price,omitempty` | `—` |  |
| `MinOrderQty` | `int` | `min_order_qty` | `—` |  |
| `SpecSummary` | `string` | `spec_summary` | `—` |  |
| `PackageLength` | `*decimal.Decimal` | `package_length_cm,omitempty` | `—` |  |
| `PackageWidth` | `*decimal.Decimal` | `package_width_cm,omitempty` | `—` |  |
| `PackageHeight` | `*decimal.Decimal` | `package_height_cm,omitempty` | `—` |  |
| `PackageWeight` | `*decimal.Decimal` | `package_weight_kg,omitempty` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
