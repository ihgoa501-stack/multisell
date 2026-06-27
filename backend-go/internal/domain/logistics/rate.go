package logistics

import (
	"math"
	"sort"
)

// RateEngine calculates shipping rates from static rate tables.
type RateEngine struct {
	tables []RateTableEntry
}

// RateTableEntry represents a single rate rule in the pricing table.
type RateTableEntry struct {
	ID                 int64   `yaml:"id"`
	ChannelName        string  `yaml:"channel_name"`
	ProviderName       string  `yaml:"provider_name"`
	RuleType           string  `yaml:"rule_type"` // "first_additional" | "tiered" | "fixed" | "per_kg"
	Priority           int     `yaml:"priority"`
	MinWeightKg        float64 `yaml:"min_weight_kg"`
	MaxWeightKg        float64 `yaml:"max_weight_kg"`
	DestinationCountry string  `yaml:"destination_country"`
	CargoType          string  `yaml:"cargo_type"` // "normal" | "battery" | "liquid" | "magnet"

	// First+additional pricing
	FirstKg         float64 `yaml:"first_kg"`
	FirstPrice      float64 `yaml:"first_price"`
	AdditionalKg    float64 `yaml:"additional_kg"`
	AdditionalPrice float64 `yaml:"additional_price"`

	// Tiered pricing ([]Tier stored inline in YAML)
	Tiers []Tier `yaml:"tiers"`

	// Fixed fee
	FixedFee float64 `yaml:"fixed_fee"`

	// Per-kg pricing
	PerKgPrice float64 `yaml:"per_kg_price"`

	// Minimum charge & surcharges
	MinimumCharge    float64 `yaml:"minimum_charge"`
	FuelSurchargePct float64 `yaml:"fuel_surcharge_pct"`
	SurchargeFixed   float64 `yaml:"surcharge_fixed"`

	Currency              string `yaml:"currency"`
	EstimatedDeliveryMin  int    `yaml:"estimated_delivery_min"`
	EstimatedDeliveryMax  int    `yaml:"estimated_delivery_max"`
}

// Tier defines a weight bracket for tiered pricing.
type Tier struct {
	Min   float64 `json:"min" yaml:"min"`
	Max   float64 `json:"max" yaml:"max"`
	Price float64 `json:"price" yaml:"price"`
}

// Cargo represents a package to be shipped.
type Cargo struct {
	ActualWeightKg float64
	LengthCm       float64 // 0 if unknown
	WidthCm        float64 // 0 if unknown
	HeightCm       float64 // 0 if unknown
}

// QuoteResult is a single channel's quote.
type QuoteResult struct {
	ChannelName          string  `json:"channel_name"`
	ProviderName         string  `json:"provider_name"`
	ChargeableWeightKg   float64 `json:"chargeable_weight_kg"`
	BaseShippingFee      float64 `json:"base_shipping_fee"`
	SurchargeFee         float64 `json:"surcharge_fee"`
	FuelSurchargeFee     float64 `json:"fuel_surcharge_fee"`
	TotalShippingFee     float64 `json:"total_shipping_fee"`
	Currency             string  `json:"currency"`
	EstimatedDeliveryMin int     `json:"estimated_delivery_min"`
	EstimatedDeliveryMax int     `json:"estimated_delivery_max"`
}

// QuoteResponse aggregates results across all matching channels.
type QuoteResponse struct {
	Results []QuoteResult `json:"results"`
}

// NewRateEngine creates a rate engine with the given rate table entries.
func NewRateEngine(tables []RateTableEntry) *RateEngine {
	// Ensure consistent evaluation order by sorting on creation.
	sorted := make([]RateTableEntry, len(tables))
	copy(sorted, tables)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return &RateEngine{tables: sorted}
}

// CalculateRate returns all matching quotes for the given cargo and destination.
// Pass empty cargo (zero values) for cases where weight is unknown — the engine
// returns available rates that match by destination and cargo type alone.
func (e *RateEngine) CalculateRate(cargo Cargo, destinationCountry string, cargoType string) (*QuoteResponse, error) {
	if cargoType == "" {
		cargoType = "normal"
	}

	// Step 1: volumetric weight = (L x W x H) / 6000.
	// If any dimension is 0 or negative, skip volumetric calculation.
	var volumetricWeight float64
	if cargo.LengthCm > 0 && cargo.WidthCm > 0 && cargo.HeightCm > 0 {
		volumetricWeight = (cargo.LengthCm * cargo.WidthCm * cargo.HeightCm) / 6000.0
	}

	// Step 2: chargeable weight = max(actual, volumetric)
	chargeableWeight := cargo.ActualWeightKg
	if volumetricWeight > chargeableWeight {
		chargeableWeight = volumetricWeight
	}

	// Step 3: filter matching rules by destination, cargo type, and weight range.
	var matching []RateTableEntry
	for _, entry := range e.tables {
		if entry.DestinationCountry != destinationCountry {
			continue
		}
		if entry.CargoType != "" && entry.CargoType != cargoType {
			continue
		}
		if chargeableWeight < entry.MinWeightKg-1e-10 {
			continue
		}
		if entry.MaxWeightKg > 0 && chargeableWeight > entry.MaxWeightKg+1e-10 {
			continue
		}
		matching = append(matching, entry)
	}

	// Step 4: calculate quote for each matching entry (already sorted by priority).
	results := make([]QuoteResult, 0, len(matching))
	for _, entry := range matching {
		result := e.calculateForEntry(entry, chargeableWeight)
		results = append(results, result)
	}

	return &QuoteResponse{Results: results}, nil
}

// calculateForEntry computes the full quote for one rate table entry.
func (e *RateEngine) calculateForEntry(entry RateTableEntry, chargeableWeight float64) QuoteResult {
	// Pricing mode (step 4)
	var baseFee float64
	switch entry.RuleType {
	case "first_additional":
		baseFee = applyFirstAdditional(entry, chargeableWeight)
	case "tiered":
		baseFee = applyTiered(entry, chargeableWeight)
	case "fixed":
		baseFee = applyFixed(entry)
	case "per_kg":
		baseFee = applyPerKg(entry, chargeableWeight)
	}

	// Surcharges (step 5)
	surchargeFee := entry.SurchargeFixed
	fuelSurchargeFee := baseFee * entry.FuelSurchargePct / 100.0

	// Total (step 5)
	totalFee := baseFee + surchargeFee + fuelSurchargeFee

	// Minimum charge enforcement (step 6)
	if entry.MinimumCharge > 0 && totalFee < entry.MinimumCharge {
		totalFee = entry.MinimumCharge
	}

	currency := entry.Currency
	if currency == "" {
		currency = "CNY"
	}

	return QuoteResult{
		ChannelName:          entry.ChannelName,
		ProviderName:         entry.ProviderName,
		ChargeableWeightKg:   roundTo(chargeableWeight, 4),
		BaseShippingFee:      roundTo(baseFee, 2),
		SurchargeFee:         roundTo(surchargeFee, 2),
		FuelSurchargeFee:     roundTo(fuelSurchargeFee, 2),
		TotalShippingFee:     roundTo(totalFee, 2),
		Currency:             currency,
		EstimatedDeliveryMin: entry.EstimatedDeliveryMin,
		EstimatedDeliveryMax: entry.EstimatedDeliveryMax,
	}
}

// applyFirstAdditional computes the base fee using first+additional pricing.
// firstPrice + ceil(max(0, weight - firstKg) / additionalKg) * additionalPrice.
func applyFirstAdditional(entry RateTableEntry, chargeableWeight float64) float64 {
	if chargeableWeight <= entry.FirstKg+1e-10 {
		return entry.FirstPrice
	}
	rawUnits := (chargeableWeight - entry.FirstKg) / entry.AdditionalKg
	additionalUnits := math.Ceil(rawUnits - 1e-10)
	return entry.FirstPrice + (additionalUnits * entry.AdditionalPrice)
}

// applyTiered finds the matching weight tier and returns its price.
func applyTiered(entry RateTableEntry, chargeableWeight float64) float64 {
	for _, tier := range entry.Tiers {
		if chargeableWeight >= tier.Min-1e-10 {
			if tier.Max <= 0 || chargeableWeight <= tier.Max+1e-10 {
				return tier.Price
			}
		}
	}
	// Fallback to the last tier's price.
	if n := len(entry.Tiers); n > 0 {
		return entry.Tiers[n-1].Price
	}
	return 0
}

// applyFixed returns the fixed shipping fee.
func applyFixed(entry RateTableEntry) float64 {
	return entry.FixedFee
}

// applyPerKg computes fee as chargeable weight times per-kilogram price.
func applyPerKg(entry RateTableEntry, chargeableWeight float64) float64 {
	return chargeableWeight * entry.PerKgPrice
}

// roundTo rounds a floating-point value to the specified number of decimal places.
func roundTo(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
