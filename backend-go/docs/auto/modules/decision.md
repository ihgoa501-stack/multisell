# Module: `decision`

Package: `backend-go/internal/domain/decision/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/decision` | `h.List` |
| `POST` | `/api/v1/decision` | `h.Create` |
| `DELETE` | `/api/v1/decision/:id` | `h.Delete` |
| `GET` | `/api/v1/decision/:id` | `h.Get` |
| `PUT` | `/api/v1/decision/:id` | `h.Update` |
| `POST` | `/api/v1/decision/:id/approve` | `h.Approve` |
| `POST` | `/api/v1/decision/:id/reject` | `h.Reject` |
| `GET` | `/api/v1/decision/summary` | `h.Summary` |

## Models

### `PreListingDecision`
**DB table:** `pre_listing_decision`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `platform_id` |  |
| `CountryCode` | `string` | `country_code` | `country_code` |  |
| `DecisionPoint` | `string` | `decision_point` | `decision_point` | default:pre_listing |
| `EstimatedRevenue` | `float64` | `estimated_revenue` | `estimated_revenue` | default:0 |
| `EstimatedProductCost` | `float64` | `estimated_product_cost` | `estimated_product_cost` | default:0 |
| `EstimatedShippingCost` | `float64` | `estimated_shipping_cost` | `estimated_shipping_cost` | default:0 |
| `EstimatedPlatformFee` | `float64` | `estimated_platform_fee` | `estimated_platform_fee` | default:0 |
| `EstimatedPaymentFee` | `float64` | `estimated_payment_fee` | `estimated_payment_fee` | default:0 |
| `EstimatedOtherFee` | `float64` | `estimated_other_fee` | `estimated_other_fee` | default:0 |
| `EstimatedProfit` | `float64` | `estimated_profit` | `estimated_profit` | default:0 |
| `ProfitMargin` | `float64` | `profit_margin` | `profit_margin` | default:0 |
| `ConfidenceScore` | `float64` | `confidence_score` | `confidence_score` | default:0 |
| `RiskLevel` | `string` | `risk_level` | `risk_level` | default:medium |
| `Recommendation` | `string` | `recommendation` | `recommendation` |  |
| `Reasoning` | `string` | `reasoning` | `reasoning` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `DecidedBy` | `string` | `decided_by` | `decided_by` |  |
| `DecidedAt` | `*time.Time` | `decided_at,omitempty` | `decided_at` |  |
| `TraceID` | `string` | `trace_id` | `trace_id` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `CountryCode` | `string` | `country_code` | `—` |  |
| `DecisionPoint` | `string` | `decision_point` | `—` |  |
| `EstimatedRevenue` | `*float64` | `estimated_revenue` | `—` |  |
| `EstimatedProductCost` | `*float64` | `estimated_product_cost` | `—` |  |
| `EstimatedShippingCost` | `*float64` | `estimated_shipping_cost` | `—` |  |
| `EstimatedPlatformFee` | `*float64` | `estimated_platform_fee` | `—` |  |
| `EstimatedPaymentFee` | `*float64` | `estimated_payment_fee` | `—` |  |
| `EstimatedOtherFee` | `*float64` | `estimated_other_fee` | `—` |  |
| `EstimatedProfit` | `*float64` | `estimated_profit` | `—` |  |
| `ProfitMargin` | `*float64` | `profit_margin` | `—` |  |
| `ConfidenceScore` | `*float64` | `confidence_score` | `—` |  |
| `RiskLevel` | `string` | `risk_level` | `—` |  |
| `Recommendation` | `string` | `recommendation` | `—` |  |
| `Reasoning` | `string` | `reasoning` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `TraceID` | `string` | `trace_id` | `—` |  |

### `UpdateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `CountryCode` | `*string` | `country_code` | `—` |  |
| `EstimatedRevenue` | `*float64` | `estimated_revenue` | `—` |  |
| `EstimatedProductCost` | `*float64` | `estimated_product_cost` | `—` |  |
| `EstimatedShippingCost` | `*float64` | `estimated_shipping_cost` | `—` |  |
| `EstimatedPlatformFee` | `*float64` | `estimated_platform_fee` | `—` |  |
| `EstimatedPaymentFee` | `*float64` | `estimated_payment_fee` | `—` |  |
| `EstimatedOtherFee` | `*float64` | `estimated_other_fee` | `—` |  |
| `EstimatedProfit` | `*float64` | `estimated_profit` | `—` |  |
| `ProfitMargin` | `*float64` | `profit_margin` | `—` |  |
| `ConfidenceScore` | `*float64` | `confidence_score` | `—` |  |
| `RiskLevel` | `*string` | `risk_level` | `—` |  |
| `Recommendation` | `*string` | `recommendation` | `—` |  |
| `Reasoning` | `*string` | `reasoning` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `TraceID` | `*string` | `trace_id` | `—` |  |

### `ListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `SkuID` | `*int64` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `RiskLevel` | `string` | `` | `—` |  |

### `ApproveInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `DecidedBy` | `string` | `decided_by` | `—` |  |

### `RejectInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `DecidedBy` | `string` | `decided_by` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |

### `Summary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByRecommendation` | `map[string]int64` | `by_recommendation` | `—` |  |
| `ByRiskLevel` | `map[string]int64` | `by_risk_level` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
