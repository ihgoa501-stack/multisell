// Package impl provides concrete agent implementations.
//
// LogisticsOpsAgent implements A10 Logistics Operations business logic,
// covering carrier comparison, shipping bill auditing, carrier performance
// scoring, and logistics route optimization.
//
// Decision points:
//   - carrier_compare        — Compare carriers by cost, speed, and suitability
//   - shipping_bill_audit    — Audit shipping bills for overcharges
//   - carrier_performance    — Score carrier performance across dimensions
//   - logistics_route_opt    — Recommend logistics route mix optimization
//
// All outputs are in Chinese with confidence-adaptive graceful degradation.
package impl

import (
	"fmt"
	"math"
	"strings"
	"sort"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Package-level constants for business logic thresholds.
const (
	carrierOnTimeWeight    = 40.0 // on-time performance weight (%)
	carrierDamageWeight    = 30.0 // damage rate weight (%)
	carrierCostWeight      = 30.0 // cost weight (%)
	overchargeThresholdPct = 5.0  // overcharge percentage threshold
	billAuditLookbackDays  = 90   // days to look back for bill audit
	costComparisonRatio    = 1.3  // max acceptable cost ratio vs best
	onTimeComparisonRatio  = 0.9  // min acceptable on-time ratio vs best
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

// carrierScore holds performance metrics for one carrier.
type carrierScore struct {
	ProviderName    string
	TotalShipments  int
	OnTimeRate      float64
	AvgDeliveryDays float64
	AvgCost         float64
	DamageRate      float64
	Score           float64
}

// ---------- LogisticsOpsAgent ----------

// LogisticsOpsAgent implements A10 logistics operations logic.
type LogisticsOpsAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewLogisticsOpsAgent creates a new LogisticsOpsAgent with DB access.
func NewLogisticsOpsAgent(db *gorm.DB, logger *zap.Logger) *LogisticsOpsAgent {
	return &LogisticsOpsAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - carrier_compare        — compare carriers by cost/speed
//   - shipping_bill_audit    — audit recent shipping bills
//   - carrier_performance    — score carrier performance
//   - logistics_route_opt    — recommend logistics route mix
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *LogisticsOpsAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "carrier_compare":
		return a.carrierCompare(ctx)
	case "shipping_bill_audit":
		return a.shippingBillAudit(ctx)
	case "carrier_performance":
		return a.carrierPerformance(ctx)
	case "logistics_route_opt":
		return a.logisticsRouteOpt(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ========================================================================
// 1. carrier_compare — Compare carriers by cost, estimated speed,
//    and suitability for the given shipment.
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

// ========================================================================
// 2. shipping_bill_audit — Audit shipping bills against expected costs
//    and flag discrepancies >5%.
// ========================================================================

func (a *LogisticsOpsAgent) shippingBillAudit(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return map[string]interface{}{
			"decision_point":    "shipping_bill_audit",
			"status":            "no_database",
			"message":           "数据库不可用，无法执行账单审计",
			"total_audited":     0,
			"discrepancies":     []map[string]interface{}{},
			"total_overcharge":  0.0,
			"confidence":        0.0,
		}, 0.0, "high", nil
	}

	// Determine which bills to audit.
	var items []shipping.ShippingBillItem
	lookupInterval := billAuditLookbackDays // days

	if billID, ok := ctx["bill_id"]; ok {
		if id := safeFloat(billID); id > 0 {
			if err := a.db.First(&items, int64(id)).Error; err != nil {
				a.logger.Warn("shipping_bill_audit: bill not found", zap.Float64("bill_id", id), zap.Error(err))
			}
		}
	}

	// No specific bill_id: scan recent unmatched/imported items.
	if len(items) == 0 {
		since := time.Now().AddDate(0, 0, -lookupInterval)
		q := a.db.Where("created_at >= ?", since)
		// Focus on items that haven't been reconciled as matched.
		if statuses, ok := ctx["reconciliation_statuses"]; ok {
			if statusList, ok := statuses.([]interface{}); ok {
				strs := make([]string, 0, len(statusList))
				for _, s := range statusList {
					strs = append(strs, safeString(s))
				}
				q = q.Where("reconciliation_status IN ?", strs)
			}
		} else {
			q = q.Where("reconciliation_status IN ?", []string{"unmatched_bill", "imported"})
		}
		if err := a.db.Order("created_at DESC").Limit(50).Find(&items).Error; err != nil {
			a.logger.Warn("shipping_bill_audit: query failed", zap.Error(err))
			return map[string]interface{}{
				"decision_point":    "shipping_bill_audit",
				"status":            "query_failed",
				"message":           "查询运单账单失败",
				"total_audited":     0,
				"discrepancies":     []map[string]interface{}{},
				"total_overcharge":  0.0,
				"confidence":        0.0,
			}, 0.0, "high", nil
		}
	}

	if len(items) == 0 {
		return map[string]interface{}{
			"decision_point":    "shipping_bill_audit",
			"status":            "no_bills",
			"message":           "近90天无待审计账单",
			"total_audited":     0,
			"discrepancies":     []map[string]interface{}{},
			"total_overcharge":  0.0,
			"confidence":        0.85,
		}, 0.85, "low", nil
	}

	// Audit each bill item.
	var discrepancies []map[string]interface{}
	totalOvercharge := 0.0
	auditedCount := 0
	matchedCount := 0

	for _, item := range items {
		if item.TotalActualFee == nil {
			continue
		}
		auditedCount++

		billedAmount := *item.TotalActualFee
		expectedAmount := 0.0
		expectedSource := ""

		// Prefer matched snapshot fee.
		if item.SnapshotShippingFee != nil && *item.SnapshotShippingFee > 0 {
			expectedAmount = *item.SnapshotShippingFee
			expectedSource = "snapshot"
		} else {
			// Try to estimate from quote rules.
			expectedAmount, expectedSource = a.estimateExpectedFee(item)
		}

		if expectedAmount <= 0 {
			continue
		}

		overcharge := billedAmount - expectedAmount
		overchargePct := (overcharge / expectedAmount) * 100

		if overchargePct > overchargeThresholdPct {
			discrepancy := map[string]interface{}{
				"item_id":           item.ID,
				"tracking_number":   item.TrackingNumber,
				"order_no":          item.OrderNo,
				"provider":          item.ProviderName,
				"channel":           item.ChannelName,
				"destination":       item.DestinationCountry,
				"billed_amount":     billedAmount,
				"expected_amount":   round2(expectedAmount),
				"overcharge":        round2(overcharge),
				"overcharge_pct":    round2(overchargePct),
				"billed_weight_kg":  safeFloat(item.BilledWeightKg),
				"expected_source":   expectedSource,
				"severity":          overchargeSeverity(overchargePct),
			}
			discrepancies = append(discrepancies, discrepancy)
			totalOvercharge += overcharge
		} else {
			matchedCount++
		}
	}

	totalAudited := auditedCount
	confidence = 0.85
	riskLevel = "low"
	if float64(len(discrepancies)) > float64(totalAudited)*0.3 {
		riskLevel = "high"
		confidence = 0.90
	} else if len(discrepancies) > 0 {
		riskLevel = "medium"
		confidence = 0.88
	}
	if totalAudited == 0 {
		confidence = 0.70
		riskLevel = "medium"
	}

	message := fmt.Sprintf("共审计 %d 条运单账单，发现 %d 条差异，总超额收费 ¥%.2f",
		totalAudited, len(discrepancies), round2(totalOvercharge))
	if len(discrepancies) == 0 {
		message = fmt.Sprintf("共审计 %d 条运单账单，全部在正常范围内，无超额收费", totalAudited)
	}

	output = map[string]interface{}{
		"decision_point":    "shipping_bill_audit",
		"status":            "completed",
		"message":           message,
		"total_audited":     totalAudited,
		"discrepancies":     discrepancies,
		"total_overcharge":  round2(totalOvercharge),
		"matched_count":     matchedCount,
		"audit_period_days": lookupInterval,
		"confidence":        confidence,
	}

	return output, confidence, riskLevel, nil
}

// estimateExpectedFee tries to compute the expected shipping fee for a bill
// item by matching against quote rules in the shipping catalogue.
func (a *LogisticsOpsAgent) estimateExpectedFee(item shipping.ShippingBillItem) (float64, string) {
	if item.BilledWeightKg == nil || *item.BilledWeightKg <= 0 {
		return 0, ""
	}
	if item.ProviderName == "" {
		return 0, ""
	}

	weightKg := *item.BilledWeightKg

	// Find the provider by name (case-insensitive).
	var provider shipping.ShippingProvider
	if err := a.db.Where("LOWER(name) = LOWER(?) AND status = ?", item.ProviderName, 1).First(&provider).Error; err != nil {
		return 0, ""
	}

	// Find matching channel.
	var channels []shipping.ShippingChannel
	q := a.db.Where("provider_id = ? AND status = ?", provider.ID, 1)
	if item.ChannelName != "" {
		q = q.Where("LOWER(name) = LOWER(?)", item.ChannelName)
	}
	if err := q.Order("sort_order ASC").Limit(1).Find(&channels).Error; err != nil || len(channels) == 0 {
		return 0, ""
	}
	ch := channels[0]

	// Find zone for destination country.
	var zone shipping.ShippingZone
	if item.DestinationCountry != "" {
		if err := a.db.Where("channel_id = ? AND country_code = ? AND status = ?",
			ch.ID, item.DestinationCountry, 1).First(&zone).Error; err != nil {
			// Fallback: global zone.
			if err := a.db.Where("channel_id = ? AND (country_code = '' OR country_code IS NULL) AND status = ?",
				ch.ID, 1).First(&zone).Error; err != nil {
				return 0, ""
			}
		}
	}

	// Find best matching quote rule.
	var rules []shipping.ShippingQuoteRule
	ruleQ := a.db.Where("channel_id = ? AND status = ?", ch.ID, 1)
	if zone.ID > 0 {
		ruleQ = ruleQ.Where("(zone_id = ? OR zone_id IS NULL)", zone.ID)
	}
	if err := ruleQ.Order("priority ASC").Find(&rules).Error; err != nil || len(rules) == 0 {
		return 0, ""
	}

	bestCost := math.MaxFloat64
	bestDetail := ""
	for _, rule := range rules {
		cost, detail := estimateShippingCost(weightKg, 0, ch, rule)
		if cost >= 0 && cost < bestCost {
			bestCost = cost
			bestDetail = detail
		}
	}

	if bestCost >= math.MaxFloat64 {
		return 0, ""
	}
	return bestCost, bestDetail
}

// overchargeSeverity returns a Chinese severity label.
func overchargeSeverity(pct float64) string {
	switch {
	case pct > 20:
		return "严重"
	case pct > 10:
		return "中等"
	default:
		return "轻微"
	}
}

// ========================================================================
// 3. carrier_performance — Score carriers on on-time rate (40%),
//    damage rate (30%), and cost (30%).
// ========================================================================

func (a *LogisticsOpsAgent) carrierPerformance(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return map[string]interface{}{
			"decision_point":      "carrier_performance",
			"status":              "no_database",
			"message":             "数据库不可用，无法分析承运商绩效",
			"carriers":            []map[string]interface{}{},
			"recommended_changes": []string{},
			"confidence":          0.0,
		}, 0.0, "high", nil
	}

	// Query carrier performance from the shipping snapshots joined with orders.
	type perfRow struct {
		ProviderName string
		TotalCount   int
		AvgCost      float64
		AvgDelDays   *float64
		OnTimeRate   *float64
	}

	var rows []perfRow
	since := time.Now().AddDate(0, -3, 0) // last 3 months

	rawSQL := `
		SELECT
			s.provider_name,
			COUNT(*)                                                                   AS total_count,
			COALESCE(AVG(s.total_shipping_fee), 0)                                     AS avg_cost,
			AVG(EXTRACT(EPOCH FROM (o.delivered_at - o.shipped_at)) / 86400.0)         AS avg_del_days,
			COUNT(CASE WHEN o.delivered_at IS NOT NULL THEN 1 END) * 1.0 / COUNT(*)    AS on_time_rate
		FROM sales_order_shipping_snapshot s
		JOIN sales_order o ON s.order_id = o.id
		WHERE o.shipped_at IS NOT NULL
		  AND s.created_at >= ?
		GROUP BY s.provider_id, s.provider_name
		ORDER BY avg_cost ASC
	`

	if err := a.db.Raw(rawSQL, since).Scan(&rows).Error; err != nil {
		a.logger.Warn("carrier_performance: query failed", zap.Error(err))
		return map[string]interface{}{
			"decision_point":      "carrier_performance",
			"status":              "query_failed",
			"message":             "查询承运商业绩数据失败",
			"carriers":            []map[string]interface{}{},
			"recommended_changes": []string{},
			"confidence":          0.0,
		}, 0.0, "high", nil
	}

	if len(rows) == 0 {
		return map[string]interface{}{
			"decision_point":      "carrier_performance",
			"status":              "no_data",
			"message":             "近3个月无发货记录，无法评估承运商绩效",
			"carriers":            []map[string]interface{}{},
			"recommended_changes": []string{},
			"confidence":          0.45,
		}, 0.45, "medium", nil
	}

	// Build carrier scores.
	var scores []carrierScore
	for _, row := range rows {
		onTimeRate := 0.0
		if row.OnTimeRate != nil {
			onTimeRate = *row.OnTimeRate
		}
		avgDelDays := 0.0
		if row.AvgDelDays != nil {
			avgDelDays = *row.AvgDelDays
		}

		// Damage rate is not tracked in current schema; use ~0.02 as baseline.
		damageRate := 0.02
		if v, ok := ctx["assumed_damage_rate"]; ok {
			if f := safeFloat(v); f > 0 {
				damageRate = f
			}
		}

		scores = append(scores, carrierScore{
			ProviderName:    row.ProviderName,
			TotalShipments:  row.TotalCount,
			OnTimeRate:      onTimeRate,
			AvgDeliveryDays: avgDelDays,
			AvgCost:         row.AvgCost,
			DamageRate:      damageRate,
		})
	}

	// Normalize and score.
	minCost := math.MaxFloat64
	maxOnTime := 0.0
	for _, s := range scores {
		if s.AvgCost > 0 && s.AvgCost < minCost {
			minCost = s.AvgCost
		}
		if s.OnTimeRate > maxOnTime {
			maxOnTime = s.OnTimeRate
		}
	}
	if minCost == math.MaxFloat64 {
		minCost = 1.0
	}
	if maxOnTime <= 0 {
		maxOnTime = 1.0
	}

	type scoredCarrier struct {
		ProviderName    string  `json:"provider"`
		TotalShipments  int     `json:"total_shipments"`
		OnTimeRate      float64 `json:"on_time_rate"`
		AvgDeliveryDays float64 `json:"avg_delivery_days"`
		AvgCost         float64 `json:"avg_cost"`
		DamageRate      float64 `json:"damage_rate"`
		Score           float64 `json:"score"`
	}

	var scored []scoredCarrier
	for _, s := range scores {
		// On-time score (40%): normalized against best on-time rate.
		onTimeScore := 0.0
		if maxOnTime > 0 {
			onTimeScore = (s.OnTimeRate / maxOnTime) * carrierOnTimeWeight
		}

		// Damage score (30%): lower damage rate is better; max score = 30.
		damageScore := carrierDamageWeight * (1.0 - s.DamageRate)
		if damageScore < 0 {
			damageScore = 0
		}

		// Cost score (30%): inversely proportional to cost; lowest cost = 30.
		costScore := 0.0
		if minCost > 0 && s.AvgCost > 0 {
			costScore = carrierCostWeight * (minCost / s.AvgCost)
		} else if s.AvgCost <= 0 {
			costScore = carrierCostWeight
		}
		if costScore > carrierCostWeight {
			costScore = 30
		}

		totalScore := round2(onTimeScore + damageScore + costScore)

		scored = append(scored, scoredCarrier{
			ProviderName:    s.ProviderName,
			TotalShipments:  s.TotalShipments,
			OnTimeRate:      round2(s.OnTimeRate),
			AvgDeliveryDays: round2(s.AvgDeliveryDays),
			AvgCost:         round2(s.AvgCost),
			DamageRate:      round2(s.DamageRate),
			Score:           totalScore,
		})
	}

	// Sort by score descending.
	sort.Slice(scored, func(i, j int) bool {
		return scored[j].Score < scored[i].Score
	})
	// Generate recommendations.
	recommendedChanges := []string{}
	if len(scored) >= 2 {
		best := scored[0]
		worst := scored[len(scored)-1]
		if best.Score-worst.Score > 20 {
			recommendedChanges = append(recommendedChanges,
				fmt.Sprintf("建议增加 %s 份额，其综合评分(%.1f)高于 %s(%.1f)",
					best.ProviderName, best.Score, worst.ProviderName, worst.Score))
		}
		if best.AvgCost > 0 {
			for _, s := range scored[1:] {
				if s.AvgCost > best.AvgCost*costComparisonRatio && s.OnTimeRate < best.OnTimeRate*onTimeComparisonRatio {
					recommendedChanges = append(recommendedChanges,
						fmt.Sprintf("%s 成本偏高(¥%.2f)且时效较慢，建议减少份额",
							s.ProviderName, s.AvgCost))
				}
			}
		}
	}
	if len(recommendedChanges) == 0 {
		recommendedChanges = append(recommendedChanges, "当前承运商配置合理，无需调整")
	}

	hasDamageData := false
	if _, ok := ctx["assumed_damage_rate"]; ok {
		hasDamageData = true
	}

	confidence = 0.85
	if !hasDamageData {
		confidence = 0.70
	}
	if len(scored) == 0 {
		confidence = 0.45
	}
	if len(scored) < 3 {
		confidence = math.Min(confidence, 0.75)
	}

	message := fmt.Sprintf("共评估 %d 家承运商业绩，最高分 %.1f 分(%s)",
		len(scored), scored[0].Score, scored[0].ProviderName)
	if !hasDamageData {
		message += "（破损率使用默认估算值，建议接入实际破损数据以提高精度）"
	}

	output = map[string]interface{}{
		"decision_point":      "carrier_performance",
		"status":              "completed",
		"message":             message,
		"carriers":            scored,
		"scoring_method":      "on_time(40%) + damage_rate(30%) + cost(30%)",
		"recommended_changes": recommendedChanges,
		"evaluation_period":   "近3个月",
		"confidence":          confidence,
	}

	riskLevel = "low"
	if len(recommendedChanges) > 0 && recommendedChanges[0] != "当前承运商配置合理，无需调整" {
		riskLevel = "medium"
	}

	return output, confidence, riskLevel, nil
}

// ========================================================================
// 4. logistics_route_opt — Analyze current logistics route split and
//    recommend adjustments based on destination distribution.
// ========================================================================

func (a *LogisticsOpsAgent) logisticsRouteOpt(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return map[string]interface{}{
			"decision_point":     "logistics_route_opt",
			"status":             "no_database",
			"message":            "数据库不可用，无法分析物流路线",
			"current_split":      map[string]interface{}{},
			"recommended_split":  map[string]interface{}{},
			"expected_savings":   0.0,
			"confidence":         0.0,
		}, 0.0, "high", nil
	}

	// Analyze current fulfillment distribution from shipping snapshots.
	type routeRow struct {
		PackageSource string
		TotalCount    int
		AvgCost       float64
		TotalCost     float64
	}
	var routeRows []routeRow

	routeSQL := `
		SELECT
			COALESCE(NULLIF(package_source, ''), '直发')  AS package_source,
			COUNT(*)                                          AS total_count,
			COALESCE(AVG(total_shipping_fee), 0)              AS avg_cost,
			COALESCE(SUM(total_shipping_fee), 0)              AS total_cost
		FROM sales_order_shipping_snapshot
		WHERE created_at >= ?
		GROUP BY package_source
		ORDER BY total_count DESC
	`
	since := time.Now().AddDate(0, -6, 0) // last 6 months
	if err := a.db.Raw(routeSQL, since).Scan(&routeRows).Error; err != nil {
		a.logger.Warn("logistics_route_opt: route query failed", zap.Error(err))
		return map[string]interface{}{
			"decision_point":    "logistics_route_opt",
			"status":            "query_failed",
			"message":           "查询物流路线数据失败",
			"current_split":     map[string]interface{}{},
			"recommended_split": map[string]interface{}{},
			"expected_savings":  0.0,
			"confidence":        0.0,
		}, 0.0, "high", nil
	}

	if len(routeRows) == 0 {
		return map[string]interface{}{
			"decision_point":     "logistics_route_opt",
			"status":             "no_data",
			"message":            "近6个月无发货记录，无法分析物流路线",
			"current_split":      map[string]interface{}{},
			"recommended_split":  map[string]interface{}{},
			"expected_savings":   0.0,
			"confidence":         0.45,
		}, 0.45, "medium", nil
	}

	totalShipments := 0
	for _, r := range routeRows {
		totalShipments += r.TotalCount
	}

	// Build current split with percentages.
	currentSplit := make(map[string]interface{})
	for _, r := range routeRows {
		pct := 0.0
		if totalShipments > 0 {
			pct = round2(float64(r.TotalCount) / float64(totalShipments) * 100)
		}
		currentSplit[r.PackageSource] = map[string]interface{}{
			"shipments":  r.TotalCount,
			"percentage": pct,
			"avg_cost":   round2(r.AvgCost),
			"total_cost": round2(r.TotalCost),
		}
	}

	// Analyze destination distribution.
	type destRow struct {
		DestinationCountry string
		TotalCount         int
	}
	var destRows []destRow
	destSQL := `
		SELECT destination_country, COUNT(*) AS total_count
		FROM sales_order_shipping_snapshot
		WHERE created_at >= ?
		GROUP BY destination_country
		ORDER BY total_count DESC
	`
	if err := a.db.Raw(destSQL, since).Scan(&destRows).Error; err != nil {
		a.logger.Warn("logistics_route_opt: dest query failed", zap.Error(err))
	}

	// Build recommendations based on destination and route analysis.
	recommendedSplit := map[string]interface{}{}
	savings := 0.0
	recommendations := []string{}

	// Count FBA vs overseas warehouse vs direct.
	fbaCount := 0
	owCount := 0
	directCount := 0

	fbaKeywords := []string{"fba", "亚马逊", "amazon"}
	owKeywords := []string{"海外仓", "overseas", "warehouse"}
	directKeywords := []string{"直发", "direct", "国内直发"}

	for _, r := range routeRows {
		sourceLower := strings.ToLower(r.PackageSource)
		isFBA := false
		isOW := false
		for _, kw := range fbaKeywords {
			if strings.Contains(sourceLower, kw) {
				isFBA = true
				break
			}
		}
		if isFBA {
			fbaCount += r.TotalCount
			continue
		}
		for _, kw := range owKeywords {
			if strings.Contains(sourceLower, kw) {
				isOW = true
				break
			}
		}
		if isOW {
			owCount += r.TotalCount
		} else {
			isDirect := false
			for _, kw := range directKeywords {
				if strings.Contains(sourceLower, kw) {
					isDirect = true
					break
				}
			}
			if isDirect {
				directCount += r.TotalCount
			} else {
				// Unrecognized source — classify as direct.
				directCount += r.TotalCount
			}
		}
	}

	// Recompute as percentage for recommendation.
	totalForClass := fbaCount + owCount + directCount
	if totalForClass == 0 {
		totalForClass = totalShipments
	}
	fbaPct := 0.0
	owPct := 0.0
	directPct := 0.0
	if totalForClass > 0 {
		fbaPct = round2(float64(fbaCount) / float64(totalForClass) * 100)
		owPct = round2(float64(owCount) / float64(totalForClass) * 100)
		directPct = round2(float64(directCount) / float64(totalForClass) * 100)
	}

	// Check if any destination has high volume suggesting local warehouse.
	topDestCount := 0
	topDest := ""
	if len(destRows) > 0 {
		topDest = destRows[0].DestinationCountry
		topDestCount = destRows[0].TotalCount
	}
	pctToTopDest := 0.0
	if totalShipments > 0 {
		pctToTopDest = float64(topDestCount) / float64(totalShipments) * 100
	}

	// Generate recommendations.
	if directPct > 60 && pctToTopDest > 30 {
		recFbaPct := round2(fbaPct + 5)
		recOwPct := round2(owPct + 10)
		recDirectPct := round2(directPct - 15)
		recommendedSplit["FBA"] = map[string]interface{}{
			"percentage": recFbaPct,
			"note":       fmt.Sprintf("建议增加FBA库存比例至%.0f%%以缩短 %s 配送时效", recFbaPct, topDest),
		}
		recommendedSplit["海外仓"] = map[string]interface{}{
			"percentage": recOwPct,
			"note":       fmt.Sprintf("在 %s 设立海外仓可覆盖 %.0f%% 订单", topDest, pctToTopDest),
		}
		recommendedSplit["直发"] = map[string]interface{}{
			"percentage": recDirectPct,
			"note":       "直发仅用于新品测试和小批量补货",
		}
		recommendations = append(recommendations,
			fmt.Sprintf("%s 订单占比 %.0f%%，建议设立海外仓以降低物流成本", topDest, pctToTopDest))
		// Rough estimate: 8% savings on ~$5 avg per shipment.
		savings = round2(float64(totalShipments) * 0.08 * 5.0)
	} else if fbaPct < 30 && owPct > 50 {
		recFbaPct := round2(fbaPct + 15)
		recOwPct := round2(owPct - 10)
		recDirectPct := round2(directPct - 5)
		recommendedSplit["FBA"] = map[string]interface{}{
			"percentage": recFbaPct,
			"note":       "热销品增加FBA比例可提升转化率",
		}
		recommendedSplit["海外仓"] = map[string]interface{}{
			"percentage": recOwPct,
			"note":       "慢周转品可保留在海外仓",
		}
		recommendedSplit["直发"] = map[string]interface{}{
			"percentage": recDirectPct,
			"note":       "低价值小件可直发",
		}
		recommendations = append(recommendations,
			fmt.Sprintf("热销品FBA比例(%.0f%%)偏低，建议提升至45%%以上以提升转化", fbaPct))
		savings = round2(float64(totalShipments) * 0.03 * 3.0)
	} else {
		// Already fairly balanced.
		if fbaPct == 0 {
			fbaPct = 30
		}
		if owPct == 0 {
			owPct = 40
		}
		if directPct == 0 {
			directPct = 30
		}
		recommendedSplit["FBA"] = map[string]interface{}{
			"percentage": round2(fbaPct),
			"note":       "维持当前FBA比例",
		}
		recommendedSplit["海外仓"] = map[string]interface{}{
			"percentage": round2(owPct),
			"note":       "维持当前海外仓比例",
		}
		recommendedSplit["直发"] = map[string]interface{}{
			"percentage": round2(directPct),
			"note":       "维持当前直发比例",
		}
		recommendations = append(recommendations, "当前物流路线配置合理，建议持续监控")
		savings = 0
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前物流路线配置合理，建议持续监控")
	}

	confidence = 0.85
	if len(routeRows) < 2 {
		confidence = 0.70
	}
	if totalShipments < 10 {
		confidence = 0.45
	}

	message := fmt.Sprintf("近6个月共 %d 笔发货，当前物流路线配置已分析", totalShipments)

	output = map[string]interface{}{
		"decision_point":      "logistics_route_opt",
		"status":              "completed",
		"message":             message,
		"current_split":       currentSplit,
		"recommended_split":   recommendedSplit,
		"expected_savings":    savings,
		"recommendations":     recommendations,
		"top_destination":     topDest,
		"top_destination_pct": round2(pctToTopDest),
		"analysis_period":     "近6个月",
		"confidence":          confidence,
	}

	riskLevel = "low"
	if savings > 0 {
		riskLevel = "medium"
	}

	return output, confidence, riskLevel, nil
}
