// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
)

// Package-level constants for carrier performance scoring weights.
const (
	carrierOnTimeWeight   = 40.0 // on-time performance weight (%)
	carrierDamageWeight   = 30.0 // damage rate weight (%)
	carrierCostWeight     = 30.0 // cost weight (%)
	costComparisonRatio   = 1.3  // max acceptable cost ratio vs best
	onTimeComparisonRatio = 0.9  // min acceptable on-time ratio vs best
)

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

// ========================================================================
// carrier_performance — Score carriers on on-time rate (40%),
// damage rate (30%), and cost (30%).
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
