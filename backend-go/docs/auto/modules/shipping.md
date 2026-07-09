# Module: `shipping`

Package: `backend-go/internal/domain/shipping/`

**Base mount prefix:** `/api/v1`
**Required permission:** `shipping.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/shipping/bill-batches` | `h.ListBillBatches` |
| `POST` | `/api/v1/shipping/bill-batches` | `h.CreateBillBatch` |
| `DELETE` | `/api/v1/shipping/bill-batches/:id` | `h.DeleteBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id` | `h.GetBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id/anomalies` | `h.ListBillAnomalies` |
| `GET` | `/api/v1/shipping/bill-batches/:id/items` | `h.ListBillItems` |
| `POST` | `/api/v1/shipping/bill-batches/:id/reconcile` | `h.ReconcileBillBatch` |
| `POST` | `/api/v1/shipping/bill-batches/import` | `h.ImportBill` |
| `PUT` | `/api/v1/shipping/bill-items/:id/review` | `h.ReviewBillItem` |
| `GET` | `/api/v1/shipping/carrier-performance` | `h.GetCarrierPerformance` |
| `GET` | `/api/v1/shipping/carriers` | `h.ListCarriers` |
| `POST` | `/api/v1/shipping/carriers/:code/quote` | `h.CarrierQuote` |
| `GET` | `/api/v1/shipping/channels` | `h.ListChannels` |
| `POST` | `/api/v1/shipping/channels` | `h.CreateChannel` |
| `DELETE` | `/api/v1/shipping/channels/:id` | `h.DeleteChannel` |
| `GET` | `/api/v1/shipping/channels/:id` | `h.GetChannel` |
| `PUT` | `/api/v1/shipping/channels/:id` | `h.UpdateChannel` |
| `GET` | `/api/v1/shipping/providers` | `h.ListProviders` |
| `POST` | `/api/v1/shipping/providers` | `h.CreateProvider` |
| `DELETE` | `/api/v1/shipping/providers/:id` | `h.DeleteProvider` |
| `GET` | `/api/v1/shipping/providers/:id` | `h.GetProvider` |
| `PUT` | `/api/v1/shipping/providers/:id` | `h.UpdateProvider` |
| `POST` | `/api/v1/shipping/quote` | `h.Quote` |
| `POST` | `/api/v1/shipping/quote-unified` | `h.QuoteUnified` |
| `GET` | `/api/v1/shipping/rules` | `h.ListRules` |
| `POST` | `/api/v1/shipping/rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/shipping/rules/:id` | `h.DeleteRule` |
| `GET` | `/api/v1/shipping/rules/:id/versions` | `h.ListRuleVersions` |
| `GET` | `/api/v1/shipping/snapshots` | `h.ListSnapshots` |
| `POST` | `/api/v1/shipping/snapshots` | `h.CreateSnapshot` |
| `GET` | `/api/v1/shipping/snapshots/:orderId` | `h.GetSnapshot` |
| `GET` | `/api/v1/shipping/tracking` | `h.ListTracking` |
| `POST` | `/api/v1/shipping/tracking` | `h.CreateTracking` |
| `PUT` | `/api/v1/shipping/tracking/:id/event` | `h.UpdateTrackingEvent` |
| `PUT` | `/api/v1/shipping/tracking/:id/exception` | `h.MarkTrackingException` |
| `GET` | `/api/v1/shipping/tracking/:orderId` | `h.GetTracking` |
| `GET` | `/api/v1/shipping/zones` | `h.ListZones` |
| `POST` | `/api/v1/shipping/zones` | `h.CreateZone` |
| `DELETE` | `/api/v1/shipping/zones/:id` | `h.DeleteZone` |

## Models

### `ShippingProvider`
**DB table:** `shipping_provider`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Code` | `string` | `code` | `code` |  |
| `Contact` | `string` | `contact` | `contact` |  |
| `Phone` | `string` | `phone` | `phone` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ShippingChannel`
**DB table:** `shipping_channel`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProviderID` | `int64` | `provider_id` | `provider_id` | NOT NULL |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Code` | `string` | `code` | `code` |  |
| `VolumetricDivisor` | `int` | `volumetric_divisor` | `volumetric_divisor` | default:6000 |
| `CargoTypes` | `json.RawMessage` | `cargo_types,omitempty` | `cargo_types` |  |
| `EstimatedDeliveryMin` | `*int` | `estimated_delivery_min,omitempty` | `estimated_delivery_min` |  |
| `EstimatedDeliveryMax` | `*int` | `estimated_delivery_max,omitempty` | `estimated_delivery_max` |  |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `SortOrder` | `int` | `sort_order` | `sort_order` | default:0 |
| `Status` | `int16` | `status` | `status` | default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ShippingZone`
**DB table:** `shipping_zone`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ChannelID` | `int64` | `channel_id` | `channel_id` | NOT NULL |
| `CountryCode` | `string` | `country_code` | `country_code` | NOT NULL |
| `PostalCodeFrom` | `string` | `postal_code_from` | `postal_code_from` |  |
| `PostalCodeTo` | `string` | `postal_code_to` | `postal_code_to` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ShippingQuoteRule`
**DB table:** `shipping_quote_rule`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ChannelID` | `int64` | `channel_id` | `channel_id` | NOT NULL |
| `ZoneID` | `*int64` | `zone_id,omitempty` | `zone_id` |  |
| `RuleType` | `string` | `rule_type` | `rule_type` | NOT NULL |
| `Priority` | `int` | `priority` | `priority` | default:0 |
| `MinWeightKg` | `*float64` | `min_weight_kg,omitempty` | `min_weight_kg` |  |
| `MaxWeightKg` | `*float64` | `max_weight_kg,omitempty` | `max_weight_kg` |  |
| `FirstKg` | `*float64` | `first_kg,omitempty` | `first_kg` |  |
| `FirstPrice` | `*float64` | `first_price,omitempty` | `first_price` |  |
| `AdditionalKg` | `*float64` | `additional_kg,omitempty` | `additional_kg` |  |
| `AdditionalPrice` | `*float64` | `additional_price,omitempty` | `additional_price` |  |
| `FixedFee` | `*float64` | `fixed_fee,omitempty` | `fixed_fee` |  |
| `PerKgPrice` | `*float64` | `per_kg_price,omitempty` | `per_kg_price` |  |
| `MinimumCharge` | `*float64` | `minimum_charge,omitempty` | `minimum_charge` |  |
| `TierConfig` | `json.RawMessage` | `tier_config,omitempty` | `tier_config` |  |
| `SurchargeFixed` | `*float64` | `surcharge_fixed,omitempty` | `surcharge_fixed` |  |
| `FuelSurchargePct` | `*float64` | `fuel_surcharge_pct,omitempty` | `fuel_surcharge_pct` |  |
| `RoundingIncrement` | `*float64` | `rounding_increment,omitempty` | `rounding_increment` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `Status` | `int16` | `status` | `status` | default:1 |
| `EffectiveStartTime` | `*time.Time` | `effective_start_time,omitempty` | `effective_start_time` |  |
| `EffectiveEndTime` | `*time.Time` | `effective_end_time,omitempty` | `effective_end_time` |  |
| `RuleVersion` | `int` | `rule_version` | `rule_version` | default:1 |
| `ImportBatch` | `string` | `import_batch,omitempty` | `import_batch` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ShippingBillBatch`
**DB table:** `shipping_bill_batch`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProviderID` | `*int64` | `provider_id,omitempty` | `provider_id` |  |
| `SourceFilename` | `string` | `source_filename` | `source_filename` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `RowCount` | `int` | `row_count` | `row_count` | default:0 |
| `MatchedCount` | `int` | `matched_count` | `matched_count` | default:0 |
| `MismatchCount` | `int` | `mismatch_count` | `mismatch_count` | default:0 |
| `UnmatchedCount` | `int` | `unmatched_count` | `unmatched_count` | default:0 |
| `Status` | `string` | `status` | `status` | default:imported |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `ShippingBillItem`
**DB table:** `shipping_bill_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `BatchID` | `int64` | `batch_id` | `batch_id` | NOT NULL |
| `RowNumber` | `int` | `row_number` | `row_number` | NOT NULL |
| `ReconciliationStatus` | `string` | `reconciliation_status` | `reconciliation_status` | default:unmatched_bill |
| `TrackingNumber` | `string` | `tracking_number` | `tracking_number` |  |
| `OrderNo` | `string` | `order_no` | `order_no` |  |
| `ProviderName` | `string` | `provider_name` | `provider_name` |  |
| `ChannelName` | `string` | `channel_name` | `channel_name` |  |
| `DestinationCountry` | `string` | `destination_country` | `destination_country` |  |
| `BilledWeightKg` | `*float64` | `billed_weight_kg,omitempty` | `billed_weight_kg` |  |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `ActualShippingFee` | `*float64` | `actual_shipping_fee,omitempty` | `actual_shipping_fee` |  |
| `SurchargeFee` | `*float64` | `surcharge_fee,omitempty` | `surcharge_fee` |  |
| `TotalActualFee` | `*float64` | `total_actual_fee,omitempty` | `total_actual_fee` |  |
| `BilledAt` | `*time.Time` | `billed_at,omitempty` | `billed_at` |  |
| `MatchedOrderID` | `*int64` | `matched_order_id,omitempty` | `matched_order_id` |  |
| `MatchedSnapshotID` | `*int64` | `matched_snapshot_id,omitempty` | `matched_snapshot_id` |  |
| `SnapshotShippingFee` | `*float64` | `snapshot_shipping_fee,omitempty` | `snapshot_shipping_fee` |  |
| `VarianceAmount` | `*float64` | `variance_amount,omitempty` | `variance_amount` |  |
| `VariancePct` | `*float64` | `variance_pct,omitempty` | `variance_pct` |  |
| `AnomalyType` | `string` | `anomaly_type,omitempty` | `anomaly_type` |  |
| `ReviewStatus` | `string` | `review_status,omitempty` | `review_status` | default:pending |
| `RawPayload` | `json.RawMessage` | `raw_payload,omitempty` | `raw_payload` |  |
| `Note` | `string` | `note` | `note` |  |
| `ResolvedBy` | `string` | `resolved_by` | `resolved_by` |  |
| `ResolvedAt` | `*time.Time` | `resolved_at,omitempty` | `resolved_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `SalesOrderShippingSnapshot`
**DB table:** `sales_order_shipping_snapshot`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `int64` | `order_id` | `order_id` | NOT NULL |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `Quantity` | `int` | `quantity` | `quantity` | NOT NULL |
| `DestinationCountry` | `string` | `destination_country` | `destination_country` | NOT NULL |
| `PostalCode` | `string` | `postal_code` | `postal_code` |  |
| `CargoType` | `string` | `cargo_type` | `cargo_type` | default:normal |
| `PackageSource` | `string` | `package_source` | `package_source` |  |
| `PackageLengthCm` | `float64` | `package_length_cm` | `package_length_cm` | NOT NULL |
| `PackageWidthCm` | `float64` | `package_width_cm` | `package_width_cm` | NOT NULL |
| `PackageHeightCm` | `float64` | `package_height_cm` | `package_height_cm` | NOT NULL |
| `PackageWeightKg` | `float64` | `package_weight_kg` | `package_weight_kg` | NOT NULL |
| `ProviderID` | `int64` | `provider_id` | `provider_id` | NOT NULL |
| `ProviderName` | `string` | `provider_name` | `provider_name` | NOT NULL |
| `ChannelID` | `int64` | `channel_id` | `channel_id` | NOT NULL |
| `ChannelName` | `string` | `channel_name` | `channel_name` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `ActualWeightKg` | `float64` | `actual_weight_kg` | `actual_weight_kg` | NOT NULL |
| `VolumetricWeightKg` | `float64` | `volumetric_weight_kg` | `volumetric_weight_kg` | NOT NULL |
| `ChargeableWeightKg` | `float64` | `chargeable_weight_kg` | `chargeable_weight_kg` | NOT NULL |
| `BaseShippingFee` | `float64` | `base_shipping_fee` | `base_shipping_fee` | NOT NULL |
| `SurchargeFee` | `float64` | `surcharge_fee` | `surcharge_fee` | default:0 |
| `FuelSurchargeFee` | `float64` | `fuel_surcharge_fee` | `fuel_surcharge_fee` | default:0 |
| `TotalShippingFee` | `float64` | `total_shipping_fee` | `total_shipping_fee` | NOT NULL |
| `CalculationDetail` | `string` | `calculation_detail` | `calculation_detail` |  |
| `RuleVersionID` | `*int64` | `rule_version_id,omitempty` | `rule_version_id` |  |
| `RuleVersion` | `int` | `rule_version` | `rule_version` | default:1 |
| `QuotedBy` | `string` | `quoted_by,omitempty` | `quoted_by` |  |
| `SourceTrigger` | `string` | `source_trigger,omitempty` | `source_trigger` | default:manual |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CreateProviderInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `string` | `name` | `—` |  |
| `Code` | `string` | `code` | `—` |  |
| `Contact` | `string` | `contact` | `—` |  |
| `Phone` | `string` | `phone` | `—` |  |
| `Remark` | `string` | `remark` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `UpdateProviderInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `*string` | `name` | `—` |  |
| `Code` | `*string` | `code` | `—` |  |
| `Contact` | `*string` | `contact` | `—` |  |
| `Phone` | `*string` | `phone` | `—` |  |
| `Remark` | `*string` | `remark` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `CreateChannelInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProviderID` | `int64` | `provider_id` | `—` |  |
| `Name` | `string` | `name` | `—` |  |
| `Code` | `string` | `code` | `—` |  |
| `VolumetricDivisor` | `*int` | `volumetric_divisor` | `—` |  |
| `CargoTypes` | `json.RawMessage` | `cargo_types` | `—` |  |
| `EstimatedDeliveryMin` | `*int` | `estimated_delivery_min` | `—` |  |
| `EstimatedDeliveryMax` | `*int` | `estimated_delivery_max` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `SortOrder` | `*int` | `sort_order` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `UpdateChannelInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `*string` | `name` | `—` |  |
| `Code` | `*string` | `code` | `—` |  |
| `VolumetricDivisor` | `*int` | `volumetric_divisor` | `—` |  |
| `CargoTypes` | `*json.RawMessage` | `cargo_types` | `—` |  |
| `EstimatedDeliveryMin` | `*int` | `estimated_delivery_min` | `—` |  |
| `EstimatedDeliveryMax` | `*int` | `estimated_delivery_max` | `—` |  |
| `Currency` | `*string` | `currency` | `—` |  |
| `SortOrder` | `*int` | `sort_order` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `CreateZoneInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ChannelID` | `int64` | `channel_id` | `—` |  |
| `CountryCode` | `string` | `country_code` | `—` |  |
| `PostalCodeFrom` | `string` | `postal_code_from` | `—` |  |
| `PostalCodeTo` | `string` | `postal_code_to` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `CreateQuoteRuleInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ChannelID` | `int64` | `channel_id` | `—` |  |
| `ZoneID` | `*int64` | `zone_id` | `—` |  |
| `RuleType` | `string` | `rule_type` | `—` |  |
| `Priority` | `*int` | `priority` | `—` |  |
| `MinWeightKg` | `*float64` | `min_weight_kg` | `—` |  |
| `MaxWeightKg` | `*float64` | `max_weight_kg` | `—` |  |
| `FirstKg` | `*float64` | `first_kg` | `—` |  |
| `FirstPrice` | `*float64` | `first_price` | `—` |  |
| `AdditionalKg` | `*float64` | `additional_kg` | `—` |  |
| `AdditionalPrice` | `*float64` | `additional_price` | `—` |  |
| `FixedFee` | `*float64` | `fixed_fee` | `—` |  |
| `PerKgPrice` | `*float64` | `per_kg_price` | `—` |  |
| `MinimumCharge` | `*float64` | `minimum_charge` | `—` |  |
| `TierConfig` | `json.RawMessage` | `tier_config` | `—` |  |
| `SurchargeFixed` | `*float64` | `surcharge_fixed` | `—` |  |
| `FuelSurchargePct` | `*float64` | `fuel_surcharge_pct` | `—` |  |
| `RoundingIncrement` | `*float64` | `rounding_increment` | `—` |  |
| `Remark` | `string` | `remark` | `—` |  |
| `Status` | `*int16` | `status` | `—` |  |

### `CreateBillBatchInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProviderID` | `*int64` | `provider_id` | `—` |  |
| `SourceFilename` | `string` | `source_filename` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `CreatedBy` | `string` | `created_by` | `—` |  |

### `FulfillmentTracking`
**DB table:** `fulfillment_tracking`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `int64` | `order_id` | `order_id` | NOT NULL |
| `TrackingNumber` | `string` | `tracking_number` | `tracking_number` | NOT NULL |
| `CarrierCode` | `string` | `carrier_code,omitempty` | `carrier_code` |  |
| `CarrierName` | `string` | `carrier_name,omitempty` | `carrier_name` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `TrackingEvents` | `json.RawMessage` | `tracking_events,omitempty` | `tracking_events` | default:'[]' |
| `EstimatedDelivery` | `*time.Time` | `estimated_delivery,omitempty` | `estimated_delivery` |  |
| `DeliveredAt` | `*time.Time` | `delivered_at,omitempty` | `delivered_at` |  |
| `IsLost` | `bool` | `is_lost` | `is_lost` | default:false |
| `IsReturned` | `bool` | `is_returned` | `is_returned` | default:false |
| `IsDamaged` | `bool` | `is_damaged` | `is_damaged` | default:false |
| `Note` | `string` | `note` | `note` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateTrackingInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `int64` | `order_id` | `—` |  |
| `TrackingNumber` | `string` | `tracking_number` | `—` |  |
| `CarrierCode` | `string` | `carrier_code` | `—` |  |
| `CarrierName` | `string` | `carrier_name` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Note` | `string` | `note` | `—` |  |

### `TrackingEvent`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Timestamp` | `string` | `timestamp` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Location` | `string` | `location,omitempty` | `—` |  |
| `Message` | `string` | `message,omitempty` | `—` |  |

### `QuoteRequest`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Mode` | `string` | `mode` | `—` |  |
| `SkuID` | `*int64` | `sku_id` | `—` |  |
| `Quantity` | `int` | `quantity` | `—` |  |
| `DestinationCountry` | `string` | `destination_country` | `—` |  |
| `PostalCode` | `string` | `postal_code` | `—` |  |
| `CargoType` | `string` | `cargo_type` | `—` |  |
| `ManualWeightKg` | `*float64` | `manual_weight_kg` | `—` |  |
| `ManualLengthCM` | `*float64` | `manual_length_cm` | `—` |  |
| `ManualWidthCM` | `*float64` | `manual_width_cm` | `—` |  |
| `ManualHeightCM` | `*float64` | `manual_height_cm` | `—` |  |

### `QuoteResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ChannelID` | `int64` | `channel_id` | `—` |  |
| `ChannelName` | `string` | `channel_name` | `—` |  |
| `ProviderName` | `string` | `provider_name` | `—` |  |
| `ActualWeightKg` | `float64` | `actual_weight_kg` | `—` |  |
| `VolumetricWeightKg` | `float64` | `volumetric_weight_kg` | `—` |  |
| `ChargeableWeightKg` | `float64` | `chargeable_weight_kg` | `—` |  |
| `BaseShippingFee` | `float64` | `base_shipping_fee` | `—` |  |
| `SurchargeFee` | `float64` | `surcharge_fee` | `—` |  |
| `FuelSurchargeFee` | `float64` | `fuel_surcharge_fee` | `—` |  |
| `TotalShippingFee` | `float64` | `total_shipping_fee` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `CalculationDetail` | `string` | `calculation_detail` | `—` |  |

### `QuoteResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Results` | `[]QuoteResult` | `results` | `—` |  |

### `CreateRuleInputV2`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `EffectiveStartTime` | `*time.Time` | `effective_start_time` | `—` |  |
| `EffectiveEndTime` | `*time.Time` | `effective_end_time` | `—` |  |
| `RuleVersion` | `*int` | `rule_version` | `—` |  |
| `ImportBatch` | `string` | `import_batch` | `—` |  |

### `CreateSnapshotInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `int64` | `order_id` | `—` |  |
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `Quantity` | `int` | `quantity` | `—` |  |
| `DestinationCountry` | `string` | `destination_country` | `—` |  |
| `PostalCode` | `string` | `postal_code` | `—` |  |
| `CargoType` | `string` | `cargo_type` | `—` |  |
| `PackageSource` | `string` | `package_source` | `—` |  |
| `PackageLengthCm` | `float64` | `package_length_cm` | `—` |  |
| `PackageWidthCm` | `float64` | `package_width_cm` | `—` |  |
| `PackageHeightCm` | `float64` | `package_height_cm` | `—` |  |
| `PackageWeightKg` | `float64` | `package_weight_kg` | `—` |  |
| `ProviderID` | `int64` | `provider_id` | `—` |  |
| `ProviderName` | `string` | `provider_name` | `—` |  |
| `ChannelID` | `int64` | `channel_id` | `—` |  |
| `ChannelName` | `string` | `channel_name` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `ActualWeightKg` | `float64` | `actual_weight_kg` | `—` |  |
| `VolumetricWeightKg` | `float64` | `volumetric_weight_kg` | `—` |  |
| `ChargeableWeightKg` | `float64` | `chargeable_weight_kg` | `—` |  |
| `BaseShippingFee` | `float64` | `base_shipping_fee` | `—` |  |
| `SurchargeFee` | `float64` | `surcharge_fee` | `—` |  |
| `FuelSurchargeFee` | `float64` | `fuel_surcharge_fee` | `—` |  |
| `TotalShippingFee` | `float64` | `total_shipping_fee` | `—` |  |
| `CalculationDetail` | `string` | `calculation_detail` | `—` |  |
| `RuleVersionID` | `*int64` | `rule_version_id` | `—` |  |
| `RuleVersion` | `int` | `rule_version` | `—` |  |
| `QuotedBy` | `string` | `quoted_by` | `—` |  |
| `SourceTrigger` | `string` | `source_trigger` | `—` |  |

### `ReconcileBatchInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `BatchID` | `int64` | `batch_id` | `—` |  |

### `BillReconciliationResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalItems` | `int` | `total_items` | `—` |  |
| `MatchedItems` | `int` | `matched_items` | `—` |  |
| `UnmatchedItems` | `int` | `unmatched_items` | `—` |  |
| `AnomalousItems` | `int` | `anomalous_items` | `—` |  |
| `TotalVariance` | `float64` | `total_variance` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |

### `A10Advice`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AdviceType` | `string` | `advice_type` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |
| `DataBasis` | `string` | `data_basis` | `—` |  |
| `RiskLevel` | `string` | `risk_level` | `—` |  |
| `SuggestedAction` | `string` | `suggested_action` | `—` |  |
| `NeedsApproval` | `bool` | `needs_approval` | `—` |  |
| `OrderID` | `*int64` | `order_id,omitempty` | `—` |  |
| `ChannelID` | `*int64` | `channel_id,omitempty` | `—` |  |
| `ProviderID` | `*int64` | `provider_id,omitempty` | `—` |  |
| `Confidence` | `float64` | `confidence` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
