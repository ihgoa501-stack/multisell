// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"go.uber.org/zap"
)

// ---------- Required context fields ----------

var (
	carrierCompareRequiredFields = []string{"weight_kg", "destination"}
)

// carrierOption holds one quoted carrier result during comparison.
type carrierOption struct {
	ProviderName string
	ChannelName  string
	ChannelCode  string
	DeliveryMin  *int
	DeliveryMax  *int
	TotalCost    float64
	CostDetail   string
}

// ========================================================================
// carrier_compare — Compare carriers by cost, estimated speed,
// and suitability for the given shipment.
// ========================================================================

func (a *LogisticsOpsAgent) carrierCompare(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, carrierCompareRequiredFields); len(missing) > 0 {
		return insufficientData("carrier_compare", missing), 0.0, "low", nil
	}

	weightKg := safeFloat(ctx["weight_kg"])
	volumeCbm := safeFloat(ctx["volume_cbm"])
	destination := safeString(ctx["destination"])
	cargoValue := safeFloat(ctx["cargo_value"])

	if weightKg <= 0 {
		return insufficientData("carrier_compare", []string{"weight_kg"}), 0.0, "low", nil
	}

	// No DB — return sensible defaults with low confidence.
	if a.db == nil {
		out := carrierCompareDefaults(weightKg, volumeCbm, destination, cargoValue)
		return out, 0.45, "low", nil
	}

	// Query active providers.
	var providers []shipping.ShippingProvider
	if err := a.db.Where("status = ?", 1).Find(&providers).Error; err != nil {
		a.logger.Warn("carrier_compare: failed to query providers", zap.Error(err))
		out := carrierCompareDefaults(weightKg, volumeCbm, destination, cargoValue)
		return out, 0.45, "low", nil
	}
	if len(providers) == 0 {
		out := carrierCompareDefaults(weightKg, volumeCbm, destination, cargoValue)
		return out, 0.45, "low", nil
	}

	// Enumerate all channels + zones + rules per provider.
	var options []carrierOption
	for _, p := range providers {
		var channels []shipping.ShippingChannel
		if err := a.db.Where("provider_id = ? AND status = ?", p.ID, 1).
			Order("sort_order ASC").Find(&channels).Error; err != nil {
			continue
		}
		for _, ch := range channels {
			// Find zone matching destination country.
			var zones []shipping.ShippingZone
			if err := a.db.Where("channel_id = ? AND country_code = ? AND status = ?",
				ch.ID, destination, 1).Find(&zones).Error; err != nil {
				continue
			}
			// Fallback: zone with empty country_code (global zone).
			if len(zones) == 0 {
				if err := a.db.Where("channel_id = ? AND (country_code = '' OR country_code IS NULL) AND status = ?",
					ch.ID, 1).Limit(1).Find(&zones).Error; err != nil {
					continue
				}
			}
			if len(zones) == 0 {
				continue
			}

			for _, z := range zones {
				// Get quote rules for this zone and channel.
				var rules []shipping.ShippingQuoteRule
				q := a.db.Where("channel_id = ? AND status = ?", ch.ID, 1)
				if z.ID > 0 {
					q = q.Where("(zone_id = ? OR zone_id IS NULL)", z.ID)
				}
				if err := q.Order("priority ASC").Find(&rules).Error; err != nil {
					continue
				}
				if len(rules) == 0 {
					continue
				}

				// Pick the rule that yields the lowest cost.
				bestCost := math.MaxFloat64
				bestDetail := ""
				for _, rule := range rules {
					cost, detail := estimateShippingCost(weightKg, volumeCbm, ch, rule)
					if cost >= 0 && cost < bestCost {
						bestCost = cost
						bestDetail = detail
					}
				}
				if bestCost < math.MaxFloat64 {
					options = append(options, carrierOption{
						ProviderName: p.Name,
						ChannelName:  ch.Name,
						ChannelCode:  ch.Code,
						DeliveryMin:  ch.EstimatedDeliveryMin,
						DeliveryMax:  ch.EstimatedDeliveryMax,
						TotalCost:    bestCost,
						CostDetail:   bestDetail,
					})
				}
			}
		}
	}

	// No carriers found — defaults.
	if len(options) == 0 {
		out := carrierCompareDefaults(weightKg, volumeCbm, destination, cargoValue)
		return out, 0.45, "low", nil
	}

	// Sort by total cost ascending.
	sort.Slice(options, func(i, j int) bool {
		return options[i].TotalCost < options[j].TotalCost
	})
	// Build carrier list with suitability.
	minCost := options[0].TotalCost
	var carriers []map[string]interface{}
	for i, opt := range options {
		estimatedDays := "待确认"
		if opt.DeliveryMin != nil && opt.DeliveryMax != nil {
			estimatedDays = fmt.Sprintf("%d-%d天", *opt.DeliveryMin, *opt.DeliveryMax)
		} else if opt.DeliveryMin != nil {
			estimatedDays = fmt.Sprintf("约%d天", *opt.DeliveryMin)
		}

		suitability := "alternative"
		if i == 0 {
			suitability = "recommended"
		}

		costLevel := computeCostLevel(opt.TotalCost, minCost, options)
		carriers = append(carriers, map[string]interface{}{
			"provider":       opt.ProviderName,
			"channel":        opt.ChannelName,
			"estimated_days": estimatedDays,
			"cost_estimate":  costLevel,
			"total_cost":     opt.TotalCost,
			"cost_detail":    opt.CostDetail,
			"suitability":    suitability,
			"rank":           i + 1,
		})
	}

	confidence = 0.85
	riskLevel = "low"
	if len(carriers) < 3 {
		confidence = 0.70
		riskLevel = "low"
	}
	if len(carriers) == 0 {
		confidence = 0.45
		riskLevel = "medium"
	}

	recommendation := fmt.Sprintf("推荐使用 %s(%s)，预估运费 ¥%.2f",
		carriers[0]["provider"].(string),
		carriers[0]["channel"].(string),
		carriers[0]["total_cost"].(float64),
	)

	if len(carriers) > 0 {
		riskReason := ""
		if carriers[0]["cost_estimate"].(string) == "高" {
			riskReason = fmt.Sprintf("最低报价仍偏高(¥%.2f)，建议联系承运商议价", minCost)
		}
		output = map[string]interface{}{
			"decision_point": "carrier_compare",
			"weight_kg":      weightKg,
			"volume_cbm":     volumeCbm,
			"destination":    destination,
			"cargo_value":    cargoValue,
			"carriers":       carriers,
			"total_carriers": len(carriers),
			"recommendation": recommendation,
			"risk_reason":    riskReason,
			"confidence":     confidence,
		}
	}

	return output, confidence, riskLevel, nil
}

// estimateShippingCost computes the estimated shipping cost for given
// weight/volume using the channel's volumetric divisor and the quote rule's
// pricing structure.
func estimateShippingCost(weightKg, volumeCbm float64, ch shipping.ShippingChannel, rule shipping.ShippingQuoteRule) (float64, string) {
	chargeWeight := weightKg
	if volumeCbm > 0 && ch.VolumetricDivisor > 0 {
		volWeight := volumeCbm * 1000000 / float64(ch.VolumetricDivisor)
		if volWeight > chargeWeight {
			chargeWeight = volWeight
		}
	}

	cost := 0.0
	ruleUsed := rule.RuleType

	switch rule.RuleType {
	case "per_kg":
		if rule.PerKgPrice != nil {
			cost = chargeWeight * *rule.PerKgPrice
		} else if rule.FixedFee != nil {
			cost = *rule.FixedFee
		} else {
			return -1, rule.RuleType
		}

	case "first_additional":
		if rule.FirstKg != nil && rule.FirstPrice != nil && rule.AdditionalKg != nil && rule.AdditionalPrice != nil {
			if chargeWeight <= *rule.FirstKg {
				cost = *rule.FirstPrice
			} else {
				extraKg := chargeWeight - *rule.FirstKg
				units := math.Ceil(extraKg / *rule.AdditionalKg)
				cost = *rule.FirstPrice + units**rule.AdditionalPrice
			}
		} else if rule.FixedFee != nil {
			cost = *rule.FixedFee
		} else {
			return -1, rule.RuleType
		}

	case "fixed":
		if rule.FixedFee != nil {
			cost = *rule.FixedFee
		} else {
			return -1, rule.RuleType
		}

	case "tiered":
		// Tiered rules use JSON tier_config; fall back to per_kg or fixed.
		if rule.PerKgPrice != nil {
			cost = chargeWeight * *rule.PerKgPrice
			ruleUsed = "per_kg"
		} else if rule.FixedFee != nil {
			cost = *rule.FixedFee
			ruleUsed = "fixed"
		} else {
			return -1, rule.RuleType
		}

	default:
		if rule.PerKgPrice != nil {
			cost = chargeWeight * *rule.PerKgPrice
		} else if rule.FixedFee != nil {
			cost = *rule.FixedFee
		} else {
			return -1, rule.RuleType
		}
	}

	// Add surcharges.
	if rule.SurchargeFixed != nil {
		cost += *rule.SurchargeFixed
	}
	if rule.FuelSurchargePct != nil {
		cost *= (1 + *rule.FuelSurchargePct/100)
	}
	if rule.MinimumCharge != nil && cost < *rule.MinimumCharge {
		cost = *rule.MinimumCharge
	}

	return round2(cost), ruleUsed
}

// computeCostLevel assigns a human-readable cost level label.
func computeCostLevel(cost, minCost float64, opts []carrierOption) string {
	if len(opts) == 0 || minCost <= 0 {
		return "中"
	}
	ratio := cost / minCost
	switch {
	case ratio <= 1.0:
		return "低"
	case ratio <= 1.25:
		return "中低"
	case ratio <= 1.5:
		return "中"
	case ratio <= 2.0:
		return "中高"
	default:
		return "高"
	}
}

// carrierCompareDefaults returns sensible defaults when no data is available.
func carrierCompareDefaults(weightKg, volumeCbm float64, destination string, cargoValue float64) map[string]interface{} {
	carriers := []map[string]interface{}{
		{
			"provider":       "海运",
			"channel":        "海运快船",
			"estimated_days": "10-15天",
			"cost_estimate":  "中",
			"suitability":    "recommended",
			"rank":           1,
		},
		{
			"provider":       "海运",
			"channel":        "海运普船",
			"estimated_days": "15-25天",
			"cost_estimate":  "低",
			"suitability":    "alternative",
			"rank":           2,
		},
		{
			"provider":       "DHL",
			"channel":        "DHL国际快递",
			"estimated_days": "3-7天",
			"cost_estimate":  "高",
			"suitability":    "alternative",
			"rank":           3,
		},
	}

	if strings.EqualFold(destination, "EU") {
		carriers = append(carriers, map[string]interface{}{
			"provider":       "中欧班列",
			"channel":        "中欧铁路",
			"estimated_days": "15-20天",
			"cost_estimate":  "中低",
			"suitability":    "alternative",
			"rank":           4,
		})
	}

	if cargoValue > 50 {
		carriers = append([]map[string]interface{}{
			{
				"provider":       "FedEx",
				"channel":        "FedEx优先",
				"estimated_days": "3-5天",
				"cost_estimate":  "高",
				"suitability":    "alternative",
				"rank":           4,
			},
		}, carriers...)
	}

	return map[string]interface{}{
		"decision_point": "carrier_compare",
		"weight_kg":      weightKg,
		"volume_cbm":     volumeCbm,
		"destination":    destination,
		"cargo_value":    cargoValue,
		"carriers":       carriers,
		"total_carriers": len(carriers),
		"recommendation": fmt.Sprintf("推荐使用 %s(%s)",
			carriers[0]["provider"].(string),
			carriers[0]["channel"].(string),
		),
		"data_source": "default",
	}
}
