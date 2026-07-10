# Module: `price`

Package: `backend-go/internal/domain/price/`

**Base mount prefix:** `/api/v1`
**Required permission:** `finance.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/competitor-prices` | `h.ListCompetitorPrices` |
| `POST` | `/api/v1/competitor-prices` | `h.CreateCompetitorPrice` |
| `DELETE` | `/api/v1/competitor-prices/:id` | `h.DeleteCompetitorPrice` |
| `GET` | `/api/v1/competitor-prices/:id` | `h.GetCompetitorPrice` |
| `GET` | `/api/v1/prices` | `h.ListPrices` |
| `POST` | `/api/v1/prices` | `h.SetPrice` |
| `DELETE` | `/api/v1/prices/:id` | `h.DeletePrice` |
| `GET` | `/api/v1/prices/:id` | `h.GetPrice` |
| `PUT` | `/api/v1/prices/:id` | `h.UpdatePrice` |
| `GET` | `/api/v1/pricing-recommendations` | `h.ListRecommendations` |
| `POST` | `/api/v1/pricing-recommendations/:id/apply` | `h.ApplyRecommendation` |
| `POST` | `/api/v1/pricing-recommendations/generate` | `h.GenerateRecommendation` |
| `GET` | `/api/v1/pricing-strategies` | `h.ListPricingStrategies` |
| `POST` | `/api/v1/pricing-strategies` | `h.SavePricingStrategy` |
| `DELETE` | `/api/v1/pricing-strategies/:id` | `h.DeletePricingStrategy` |
| `GET` | `/api/v1/pricing-strategies/:id` | `h.GetPricingStrategy` |
| `PUT` | `/api/v1/pricing-strategies/:id` | `h.UpdatePricingStrategy` |
| `GET` | `/api/v1/skus/:id/current-price` | `h.GetCurrentPrice` |
| `GET` | `/api/v1/skus/:id/price-history` | `h.PriceHistory` |
| `GET` | `/api/v1/skus/:id/prices` | `h.ListPricesBySKU` |

## Models

### `Price`
**DB table:** `price`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `PriceType` | `string` | `price_type` | `price_type` | NOT NULL |
| `Price` | `decimal.Decimal` | `price` | `price` | NOT NULL |
| `StartTime` | `*time.Time` | `start_time,omitempty` | `start_time` |  |
| `EndTime` | `*time.Time` | `end_time,omitempty` | `end_time` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `PriceChangeLog`
**DB table:** `price_change_log`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `OldPrice` | `*decimal.Decimal` | `old_price,omitempty` | `old_price` |  |
| `NewPrice` | `*decimal.Decimal` | `new_price,omitempty` | `new_price` |  |
| `PriceType` | `string` | `price_type` | `price_type` |  |
| `ChangeType` | `string` | `change_type` | `change_type` |  |
| `Operator` | `string` | `operator` | `operator` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CompetitorPrice`
**DB table:** `competitor_prices`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `Platform` | `string` | `platform` | `platform` |  |
| `CompetitorName` | `string` | `competitor_name` | `competitor_name` | NOT NULL |
| `Price` | `decimal.Decimal` | `price` | `price` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:'USD' |
| `CapturedAt` | `time.Time` | `captured_at` | `captured_at` | NOT NULL |
| `SourceURL` | `string` | `source_url,omitempty` | `source_url` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `StrategyParameters`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `BuyBoxDiscount` | `float64` | `buy_box_discount,omitempty` | `—` |  |
| `MinProfitMargin` | `float64` | `min_profit_margin,omitempty` | `—` |  |
| `MinPrice` | `float64` | `min_price,omitempty` | `—` |  |
| `MaxPrice` | `float64` | `max_price,omitempty` | `—` |  |

### `PricingStrategy`
**DB table:** `pricing_strategies`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `*int64` | `sku_id,omitempty` | `sku_id` |  |
| `StrategyType` | `string` | `strategy_type` | `strategy_type` | NOT NULL |
| `Parameters` | `string` | `parameters` | `parameters` | default:'{}' |
| `Active` | `bool` | `active` | `active` | default:true |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `PricingRecommendation`
**DB table:** `pricing_recommendations`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `CurrentPrice` | `decimal.Decimal` | `current_price` | `current_price` | NOT NULL |
| `RecommendedPrice` | `decimal.Decimal` | `recommended_price` | `recommended_price` | NOT NULL |
| `StrategyUsed` | `string` | `strategy_used` | `strategy_used` |  |
| `Reason` | `string` | `reason` | `reason` |  |
| `RiskLevel` | `string` | `risk_level` | `risk_level` |  |
| `CompetitorCount` | `int` | `competitor_count` | `competitor_count` | default:0 |
| `Applied` | `bool` | `applied` | `applied` | default:false |
| `AppliedAt` | `*time.Time` | `applied_at,omitempty` | `applied_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `GenerateRecommendationInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `StrategyType` | `string` | `strategy_type` | `—` |  |
| `Cost` | `decimal.Decimal` | `cost,omitempty` | `—` |  |
| `PlatformFeeRate` | `float64` | `platform_fee_rate,omitempty` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
