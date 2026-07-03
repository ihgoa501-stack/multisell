package shipping

import (
	"encoding/json"
	"fmt"

	"github.com/lingmirror/backend-go/internal/domain/logistics"
)

// ── Conversion: Shipping → Logistics Rate Engine ───────────────────────
// Phase 1 of Fulfillment Intelligence OS.
// These functions convert DB-backed shipping models into the
// logistics.RateTableEntry format so that ALL quote calculation goes
// through the unified logistics.RateEngine.

// shippingRuleTypeToLogistics maps shipping rule type names to logistics names.
// shipping uses "fixed_plus_per_kg", "first_weight_plus_increment", "tiered_weight"
// logistics uses "first_additional", "tiered", "fixed", "per_kg"
func shippingRuleTypeToLogistics(rulesType string) string {
	switch rulesType {
	case "first_weight_plus_increment":
		return "first_additional"
	case "tiered_weight":
		return "tiered"
	case "fixed_plus_per_kg":
		return "per_kg"
	default:
		return rulesType // pass through unknown types as-is
	}
}

// ToRateTableEntry converts a ShippingChannel + ShippingQuoteRule pair
// into a logistics.RateTableEntry for use with the unified RateEngine.
func ToRateTableEntry(ch *ShippingChannel, rule *ShippingQuoteRule, zone *ShippingZone) logistics.RateTableEntry {
	e := logistics.RateTableEntry{
		ID:                   rule.ID,
		ChannelName:          ch.Name,
		ProviderName:         "", // resolved below
		RuleType:             shippingRuleTypeToLogistics(rule.RuleType),
		Priority:             rule.Priority,
		DestinationCountry:   zone.CountryCode,
		FirstKg:              safeFloatVal(rule.FirstKg),
		FirstPrice:           safeFloatVal(rule.FirstPrice),
		AdditionalKg:         safeFloatVal(rule.AdditionalKg),
		AdditionalPrice:      safeFloatVal(rule.AdditionalPrice),
		FixedFee:             safeFloatVal(rule.FixedFee),
		PerKgPrice:           safeFloatVal(rule.PerKgPrice),
		MinimumCharge:        safeFloatVal(rule.MinimumCharge),
		FuelSurchargePct:     safeFloatVal(rule.FuelSurchargePct),
		SurchargeFixed:       safeFloatVal(rule.SurchargeFixed),
		Currency:             ch.Currency,
		EstimatedDeliveryMin: safeIntVal(ch.EstimatedDeliveryMin),
		EstimatedDeliveryMax: safeIntVal(ch.EstimatedDeliveryMax),
	}
	if e.Currency == "" {
		e.Currency = "CNY"
	}

	// Map min/max weight from rule
	if rule.MinWeightKg != nil {
		e.MinWeightKg = *rule.MinWeightKg
	}
	if rule.MaxWeightKg != nil {
		e.MaxWeightKg = *rule.MaxWeightKg
	}

	// Map tiered pricing config (JSONB → logistics.Tier)
	if rule.RuleType == "tiered_weight" && len(rule.TierConfig) > 0 {
		var tiers []struct {
			MinKg *float64 `json:"min_kg"`
			MaxKg *float64 `json:"max_kg"`
			Price *float64 `json:"price"`
		}
		if err := json.Unmarshal(rule.TierConfig, &tiers); err == nil {
			for _, t := range tiers {
				logisticTier := logistics.Tier{
					Price: safeFloatVal(t.Price),
				}
				if t.MinKg != nil {
					logisticTier.Min = *t.MinKg
				}
				if t.MaxKg != nil {
					logisticTier.Max = *t.MaxKg
				}
				e.Tiers = append(e.Tiers, logisticTier)
			}
		}
	}

	// Map cargo types from channel
	if ch.CargoTypes != nil && string(ch.CargoTypes) != "" {
		e.CargoType = string(ch.CargoTypes)
		// Simplification: use first cargo type
		var types []string
		if err := json.Unmarshal(ch.CargoTypes, &types); err == nil && len(types) > 0 {
			e.CargoType = types[0]
		}
	} else {
		e.CargoType = "normal"
	}

	// Handle fixed_plus_per_kg: add fixed fee as surcharge (always added to total).
	if rule.RuleType == "fixed_plus_per_kg" {
		if fixed := safeFloatVal(rule.FixedFee); fixed > 0 {
			e.SurchargeFixed += fixed
		}
	}

	return e
}

// ToQuoteResult converts a logistics.QuoteResult to a shipping.QuoteResult.
func ToQuoteResult(r logistics.QuoteResult) QuoteResult {
	return QuoteResult{
		ChannelName:        r.ChannelName,
		ProviderName:       r.ProviderName,
		ChargeableWeightKg: r.ChargeableWeightKg,
		BaseShippingFee:    r.BaseShippingFee,
		SurchargeFee:       r.SurchargeFee,
		FuelSurchargeFee:   r.FuelSurchargeFee,
		TotalShippingFee:   r.TotalShippingFee,
		Currency:           r.Currency,
	}
}

// ── Helpers ────────────────────────────────────────────────────────────

func safeFloatVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func safeIntVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// QuoteItemDetail is a human-readable calculation detail.
func (r *QuoteResult) Detail() string {
	return fmt.Sprintf("base=%.2f surcharge=%.2f fuel=%.2f total=%.2f %s",
		r.BaseShippingFee, r.SurchargeFee, r.FuelSurchargeFee, r.TotalShippingFee, r.Currency)
}
