# Module: `tariff`

Package: `backend-go/internal/domain/tariff/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/tariff` | `h.List` |
| `POST` | `/api/v1/tariff` | `h.Create` |
| `DELETE` | `/api/v1/tariff/:id` | `h.Delete` |
| `GET` | `/api/v1/tariff/:id` | `h.Get` |
| `PUT` | `/api/v1/tariff/:id` | `h.Update` |
| `POST` | `/api/v1/tariff/decide` | `h.Decide` |

## Models

### `TariffRule`
**DB table:** `tariff_rule`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `CountryCode` | `string` | `country_code` | `country_code` | NOT NULL |
| `HSCode` | `string` | `hs_code,omitempty` | `hs_code` |  |
| `HSCodePrefix` | `string` | `hs_code_prefix,omitempty` | `hs_code_prefix` |  |
| `DutyRatePct` | `float64` | `duty_rate_pct` | `duty_rate_pct` | default:0 |
| `VatRatePct` | `float64` | `vat_rate_pct` | `vat_rate_pct` | default:0 |
| `OtherTaxRatePct` | `float64` | `other_tax_rate_pct` | `other_tax_rate_pct` | default:0 |
| `MinThresholdUSD` | `float64` | `min_threshold_usd` | `min_threshold_usd` | default:0 |
| `MaxThresholdUSD` | `float64` | `max_threshold_usd` | `max_threshold_usd` | default:0 |
| `Incoterm` | `string` | `incoterm` | `incoterm` | NOT NULL, default:DDU |
| `Priority` | `int` | `priority` | `priority` | default:0 |
| `EffectiveFrom` | `*time.Time` | `effective_from,omitempty` | `effective_from` |  |
| `EffectiveTo` | `*time.Time` | `effective_to,omitempty` | `effective_to` |  |
| `Status` | `string` | `status` | `status` | default:active |
| `Remark` | `string` | `remark` | `remark` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateRuleInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CountryCode` | `string` | `country_code` | `—` |  |
| `HSCode` | `string` | `hs_code` | `—` |  |
| `HSCodePrefix` | `string` | `hs_code_prefix` | `—` |  |
| `DutyRatePct` | `*float64` | `duty_rate_pct` | `—` |  |
| `VatRatePct` | `*float64` | `vat_rate_pct` | `—` |  |
| `OtherTaxRatePct` | `*float64` | `other_tax_rate_pct` | `—` |  |
| `MinThresholdUSD` | `*float64` | `min_threshold_usd` | `—` |  |
| `MaxThresholdUSD` | `*float64` | `max_threshold_usd` | `—` |  |
| `Incoterm` | `string` | `incoterm` | `—` |  |
| `Priority` | `*int` | `priority` | `—` |  |
| `EffectiveFrom` | `*time.Time` | `effective_from` | `—` |  |
| `EffectiveTo` | `*time.Time` | `effective_to` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Remark` | `string` | `remark` | `—` |  |

### `UpdateRuleInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CountryCode` | `*string` | `country_code` | `—` |  |
| `HSCode` | `*string` | `hs_code` | `—` |  |
| `HSCodePrefix` | `*string` | `hs_code_prefix` | `—` |  |
| `DutyRatePct` | `*float64` | `duty_rate_pct` | `—` |  |
| `VatRatePct` | `*float64` | `vat_rate_pct` | `—` |  |
| `OtherTaxRatePct` | `*float64` | `other_tax_rate_pct` | `—` |  |
| `MinThresholdUSD` | `*float64` | `min_threshold_usd` | `—` |  |
| `MaxThresholdUSD` | `*float64` | `max_threshold_usd` | `—` |  |
| `Incoterm` | `*string` | `incoterm` | `—` |  |
| `Priority` | `*int` | `priority` | `—` |  |
| `EffectiveFrom` | `*time.Time` | `effective_from` | `—` |  |
| `EffectiveTo` | `*time.Time` | `effective_to` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `Remark` | `*string` | `remark` | `—` |  |

### `RuleListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CountryCode` | `string` | `` | `—` |  |
| `HSCode` | `string` | `` | `—` |  |
| `Incoterm` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |

### `DecisionRequest`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `DestinationCountry` | `string` | `destination_country` | `—` |  |
| `ProductValueUSD` | `float64` | `product_value_usd` | `—` |  |
| `HSCode` | `string` | `hs_code` | `—` |  |
| `Quantity` | `int` | `quantity` | `—` |  |
| `CargoType` | `string` | `cargo_type` | `—` |  |

### `DecisionResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Incoterm` | `string` | `incoterm` | `—` |  |
| `TotalDutyTaxUSD` | `float64` | `total_duty_tax_usd` | `—` |  |
| `DutyAmountUSD` | `float64` | `duty_amount_usd` | `—` |  |
| `VatAmountUSD` | `float64` | `vat_amount_usd` | `—` |  |
| `OtherTaxAmountUSD` | `float64` | `other_tax_amount_usd` | `—` |  |
| `RulesMatched` | `[]RuleMatchItem` | `rules_matched` | `—` |  |
| `IncotermReason` | `string` | `incoterm_reason` | `—` |  |
| `TotalValueUSD` | `float64` | `total_value_usd` | `—` |  |

### `RuleMatchItem`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RuleID` | `int64` | `rule_id` | `—` |  |
| `CountryCode` | `string` | `country_code` | `—` |  |
| `HSCode` | `string` | `hs_code,omitempty` | `—` |  |
| `HSCodePrefix` | `string` | `hs_code_prefix,omitempty` | `—` |  |
| `DutyRatePct` | `float64` | `duty_rate_pct` | `—` |  |
| `VatRatePct` | `float64` | `vat_rate_pct` | `—` |  |
| `OtherTaxRatePct` | `float64` | `other_tax_rate_pct` | `—` |  |
| `Incoterm` | `string` | `incoterm` | `—` |  |
| `Priority` | `int` | `priority` | `—` |  |
| `DutyAmountUSD` | `float64` | `duty_amount_usd` | `—` |  |
| `VatAmountUSD` | `float64` | `vat_amount_usd` | `—` |  |
| `OtherTaxAmount` | `float64` | `other_tax_amount_usd` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
