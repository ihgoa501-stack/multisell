// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"go.uber.org/zap"
)

// Package-level constants for bill audit thresholds.
const (
	overchargeThresholdPct = 5.0 // overcharge percentage threshold
	billAuditLookbackDays  = 90 // days to look back for bill audit
)

// ========================================================================
// shipping_bill_audit — Audit shipping bills against expected costs
// and flag discrepancies >5%.
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
