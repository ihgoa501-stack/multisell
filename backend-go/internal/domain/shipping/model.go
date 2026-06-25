package shipping

import (
	"encoding/json"
	"time"
)

// ShippingProvider maps to "shipping_provider".
type ShippingProvider struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Code      string    `gorm:"column:code;uniqueIndex" json:"code"`
	Contact   string    `gorm:"column:contact" json:"contact"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	Remark    string    `gorm:"column:remark" json:"remark"`
	Status    int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ShippingProvider) TableName() string { return "shipping_provider" }

// ShippingChannel maps to "shipping_channel".
type ShippingChannel struct {
	ID                  int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProviderID         int64           `gorm:"column:provider_id;not null" json:"provider_id"`
	Name               string          `gorm:"column:name;not null" json:"name"`
	Code               string          `gorm:"column:code" json:"code"`
	VolumetricDivisor  int             `gorm:"column:volumetric_divisor;default:6000" json:"volumetric_divisor"`
	CargoTypes         json.RawMessage `gorm:"column:cargo_types;type:jsonb" json:"cargo_types,omitempty"`
	EstimatedDeliveryMin *int          `gorm:"column:estimated_delivery_min" json:"estimated_delivery_min,omitempty"`
	EstimatedDeliveryMax *int          `gorm:"column:estimated_delivery_max" json:"estimated_delivery_max,omitempty"`
	Currency           string          `gorm:"column:currency;default:CNY" json:"currency"`
	SortOrder          int             `gorm:"column:sort_order;default:0" json:"sort_order"`
	Status             int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ShippingChannel) TableName() string { return "shipping_channel" }

// ShippingZone maps to "shipping_zone".
type ShippingZone struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID      int64     `gorm:"column:channel_id;not null" json:"channel_id"`
	CountryCode    string    `gorm:"column:country_code;not null" json:"country_code"`
	PostalCodeFrom string    `gorm:"column:postal_code_from" json:"postal_code_from"`
	PostalCodeTo   string    `gorm:"column:postal_code_to" json:"postal_code_to"`
	Status         int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ShippingZone) TableName() string { return "shipping_zone" }

// ShippingQuoteRule maps to "shipping_quote_rule".
type ShippingQuoteRule struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID         int64           `gorm:"column:channel_id;not null" json:"channel_id"`
	ZoneID           *int64          `gorm:"column:zone_id" json:"zone_id,omitempty"`
	RuleType         string          `gorm:"column:rule_type;not null" json:"rule_type"`
	Priority         int             `gorm:"column:priority;default:0" json:"priority"`
	MinWeightKg      *float64        `gorm:"column:min_weight_kg" json:"min_weight_kg,omitempty"`
	MaxWeightKg      *float64        `gorm:"column:max_weight_kg" json:"max_weight_kg,omitempty"`
	FirstKg          *float64        `gorm:"column:first_kg" json:"first_kg,omitempty"`
	FirstPrice       *float64        `gorm:"column:first_price" json:"first_price,omitempty"`
	AdditionalKg     *float64        `gorm:"column:additional_kg" json:"additional_kg,omitempty"`
	AdditionalPrice  *float64        `gorm:"column:additional_price" json:"additional_price,omitempty"`
	FixedFee         *float64        `gorm:"column:fixed_fee" json:"fixed_fee,omitempty"`
	PerKgPrice       *float64        `gorm:"column:per_kg_price" json:"per_kg_price,omitempty"`
	MinimumCharge    *float64        `gorm:"column:minimum_charge" json:"minimum_charge,omitempty"`
	TierConfig       json.RawMessage `gorm:"column:tier_config;type:jsonb" json:"tier_config,omitempty"`
	SurchargeFixed   *float64        `gorm:"column:surcharge_fixed" json:"surcharge_fixed,omitempty"`
	FuelSurchargePct *float64        `gorm:"column:fuel_surcharge_pct" json:"fuel_surcharge_pct,omitempty"`
	RoundingIncrement *float64       `gorm:"column:rounding_increment" json:"rounding_increment,omitempty"`
	Remark           string          `gorm:"column:remark" json:"remark"`
	Status           int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ShippingQuoteRule) TableName() string { return "shipping_quote_rule" }

// ShippingBillBatch maps to "shipping_bill_batch".
type ShippingBillBatch struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProviderID    *int64    `gorm:"column:provider_id" json:"provider_id,omitempty"`
	SourceFilename string   `gorm:"column:source_filename;not null" json:"source_filename"`
	Currency      string    `gorm:"column:currency;default:CNY" json:"currency"`
	RowCount      int       `gorm:"column:row_count;default:0" json:"row_count"`
	MatchedCount  int       `gorm:"column:matched_count;default:0" json:"matched_count"`
	MismatchCount int       `gorm:"column:mismatch_count;default:0" json:"mismatch_count"`
	UnmatchedCount int      `gorm:"column:unmatched_count;default:0" json:"unmatched_count"`
	Status        string    `gorm:"column:status;default:imported" json:"status"`
	CreatedBy     string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ShippingBillBatch) TableName() string { return "shipping_bill_batch" }

// ShippingBillItem maps to "shipping_bill_item".
type ShippingBillItem struct {
	ID                 int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BatchID            int64           `gorm:"column:batch_id;not null" json:"batch_id"`
	RowNumber          int             `gorm:"column:row_number;not null" json:"row_number"`
	ReconciliationStatus string        `gorm:"column:reconciliation_status;default:unmatched_bill" json:"reconciliation_status"`
	TrackingNumber     string          `gorm:"column:tracking_number" json:"tracking_number"`
	OrderNo           string          `gorm:"column:order_no" json:"order_no"`
	ProviderName      string          `gorm:"column:provider_name" json:"provider_name"`
	ChannelName       string          `gorm:"column:channel_name" json:"channel_name"`
	DestinationCountry string         `gorm:"column:destination_country" json:"destination_country"`
	BilledWeightKg    *float64        `gorm:"column:billed_weight_kg" json:"billed_weight_kg,omitempty"`
	Currency          string          `gorm:"column:currency;default:CNY" json:"currency"`
	ActualShippingFee *float64        `gorm:"column:actual_shipping_fee" json:"actual_shipping_fee,omitempty"`
	SurchargeFee      *float64        `gorm:"column:surcharge_fee" json:"surcharge_fee,omitempty"`
	TotalActualFee    *float64        `gorm:"column:total_actual_fee" json:"total_actual_fee,omitempty"`
	BilledAt          *time.Time      `gorm:"column:billed_at" json:"billed_at,omitempty"`
	MatchedOrderID    *int64          `gorm:"column:matched_order_id" json:"matched_order_id,omitempty"`
	MatchedSnapshotID *int64          `gorm:"column:matched_snapshot_id" json:"matched_snapshot_id,omitempty"`
	SnapshotShippingFee *float64      `gorm:"column:snapshot_shipping_fee" json:"snapshot_shipping_fee,omitempty"`
	VarianceAmount    *float64        `gorm:"column:variance_amount" json:"variance_amount,omitempty"`
	RawPayload        json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	Note              string          `gorm:"column:note" json:"note"`
	ResolvedBy        string          `gorm:"column:resolved_by" json:"resolved_by"`
	ResolvedAt        *time.Time      `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ShippingBillItem) TableName() string { return "shipping_bill_item" }

// SalesOrderShippingSnapshot maps to "sales_order_shipping_snapshot".
type SalesOrderShippingSnapshot struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID             int64     `gorm:"column:order_id;not null;uniqueIndex" json:"order_id"`
	SkuID               int64     `gorm:"column:sku_id;not null;index" json:"sku_id"`
	Quantity            int       `gorm:"column:quantity;not null" json:"quantity"`
	DestinationCountry  string    `gorm:"column:destination_country;not null" json:"destination_country"`
	PostalCode          string    `gorm:"column:postal_code" json:"postal_code"`
	CargoType           string    `gorm:"column:cargo_type;default:normal" json:"cargo_type"`
	PackageSource       string    `gorm:"column:package_source" json:"package_source"`
	PackageLengthCm     float64   `gorm:"column:package_length_cm;not null" json:"package_length_cm"`
	PackageWidthCm      float64   `gorm:"column:package_width_cm;not null" json:"package_width_cm"`
	PackageHeightCm     float64   `gorm:"column:package_height_cm;not null" json:"package_height_cm"`
	PackageWeightKg     float64   `gorm:"column:package_weight_kg;not null" json:"package_weight_kg"`
	ProviderID          int64     `gorm:"column:provider_id;not null;index" json:"provider_id"`
	ProviderName        string    `gorm:"column:provider_name;not null" json:"provider_name"`
	ChannelID           int64     `gorm:"column:channel_id;not null;index" json:"channel_id"`
	ChannelName         string    `gorm:"column:channel_name;not null" json:"channel_name"`
	Currency            string    `gorm:"column:currency;default:CNY" json:"currency"`
	ActualWeightKg      float64   `gorm:"column:actual_weight_kg;not null" json:"actual_weight_kg"`
	VolumetricWeightKg  float64   `gorm:"column:volumetric_weight_kg;not null" json:"volumetric_weight_kg"`
	ChargeableWeightKg  float64   `gorm:"column:chargeable_weight_kg;not null" json:"chargeable_weight_kg"`
	BaseShippingFee     float64   `gorm:"column:base_shipping_fee;not null" json:"base_shipping_fee"`
	SurchargeFee        float64   `gorm:"column:surcharge_fee;default:0" json:"surcharge_fee"`
	FuelSurchargeFee    float64   `gorm:"column:fuel_surcharge_fee;default:0" json:"fuel_surcharge_fee"`
	TotalShippingFee    float64   `gorm:"column:total_shipping_fee;not null" json:"total_shipping_fee"`
	CalculationDetail   string    `gorm:"column:calculation_detail" json:"calculation_detail"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SalesOrderShippingSnapshot) TableName() string { return "sales_order_shipping_snapshot" }

// ---------- Input / DTO structs ----------

type CreateProviderInput struct {
	Name    string `json:"name" binding:"required"`
	Code    string `json:"code"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Remark  string `json:"remark"`
	Status  *int16 `json:"status"`
}

type UpdateProviderInput struct {
	Name    *string `json:"name"`
	Code    *string `json:"code"`
	Contact *string `json:"contact"`
	Phone   *string `json:"phone"`
	Remark  *string `json:"remark"`
	Status  *int16  `json:"status"`
}

type CreateChannelInput struct {
	ProviderID            int64           `json:"provider_id" binding:"required"`
	Name                  string          `json:"name" binding:"required"`
	Code                  string          `json:"code"`
	VolumetricDivisor     *int            `json:"volumetric_divisor"`
	CargoTypes            json.RawMessage `json:"cargo_types"`
	EstimatedDeliveryMin  *int           `json:"estimated_delivery_min"`
	EstimatedDeliveryMax  *int           `json:"estimated_delivery_max"`
	Currency              string          `json:"currency"`
	SortOrder             *int            `json:"sort_order"`
	Status                *int16          `json:"status"`
}

type UpdateChannelInput struct {
	Name                  *string          `json:"name"`
	Code                  *string          `json:"code"`
	VolumetricDivisor     *int             `json:"volumetric_divisor"`
	CargoTypes            *json.RawMessage `json:"cargo_types"`
	EstimatedDeliveryMin  *int             `json:"estimated_delivery_min"`
	EstimatedDeliveryMax  *int             `json:"estimated_delivery_max"`
	Currency              *string          `json:"currency"`
	SortOrder             *int             `json:"sort_order"`
	Status                *int16           `json:"status"`
}

type CreateZoneInput struct {
	ChannelID      int64  `json:"channel_id" binding:"required"`
	CountryCode    string `json:"country_code" binding:"required"`
	PostalCodeFrom string `json:"postal_code_from"`
	PostalCodeTo   string `json:"postal_code_to"`
	Status         *int16 `json:"status"`
}

type CreateQuoteRuleInput struct {
	ChannelID          int64           `json:"channel_id" binding:"required"`
	ZoneID            *int64           `json:"zone_id"`
	RuleType          string          `json:"rule_type" binding:"required"`
	Priority          *int            `json:"priority"`
	MinWeightKg       *float64        `json:"min_weight_kg"`
	MaxWeightKg       *float64        `json:"max_weight_kg"`
	FirstKg           *float64        `json:"first_kg"`
	FirstPrice        *float64        `json:"first_price"`
	AdditionalKg      *float64        `json:"additional_kg"`
	AdditionalPrice   *float64        `json:"additional_price"`
	FixedFee          *float64        `json:"fixed_fee"`
	PerKgPrice        *float64        `json:"per_kg_price"`
	MinimumCharge     *float64        `json:"minimum_charge"`
	TierConfig        json.RawMessage `json:"tier_config"`
	SurchargeFixed    *float64        `json:"surcharge_fixed"`
	FuelSurchargePct  *float64        `json:"fuel_surcharge_pct"`
	RoundingIncrement *float64        `json:"rounding_increment"`
	Remark            string          `json:"remark"`
	Status            *int16          `json:"status"`
}

type CreateBillBatchInput struct {
	ProviderID     *int64 `json:"provider_id"`
	SourceFilename string `json:"source_filename" binding:"required"`
	Currency       string `json:"currency"`
	CreatedBy      string `json:"created_by"`
}

// QuoteRequest is the payload for POST /shipping/quote.
type QuoteRequest struct {
	Mode                string  `json:"mode"`             // "sku" | "manual"
	SkuID               *int64  `json:"sku_id"`
	Quantity            int     `json:"quantity"`
	DestinationCountry  string  `json:"destination_country" binding:"required"`
	PostalCode          string  `json:"postal_code"`
	CargoType           string  `json:"cargo_type"`
	ManualWeightKg      *float64 `json:"manual_weight_kg"`
	ManualLengthCM      *float64 `json:"manual_length_cm"`
	ManualWidthCM       *float64 `json:"manual_width_cm"`
	ManualHeightCM      *float64 `json:"manual_height_cm"`
}

// QuoteResult is a single channel's quote.
type QuoteResult struct {
	ChannelID         int64   `json:"channel_id"`
	ChannelName       string  `json:"channel_name"`
	ProviderName      string  `json:"provider_name"`
	ActualWeightKg    float64 `json:"actual_weight_kg"`
	VolumetricWeightKg float64 `json:"volumetric_weight_kg"`
	ChargeableWeightKg float64 `json:"chargeable_weight_kg"`
	BaseShippingFee   float64 `json:"base_shipping_fee"`
	SurchargeFee      float64 `json:"surcharge_fee"`
	FuelSurchargeFee  float64 `json:"fuel_surcharge_fee"`
	TotalShippingFee  float64 `json:"total_shipping_fee"`
	Currency          string  `json:"currency"`
	CalculationDetail string  `json:"calculation_detail"`
}

// QuoteResponse aggregates results across channels.
type QuoteResponse struct {
	Results []QuoteResult `json:"results"`
}
