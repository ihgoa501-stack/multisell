// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ========================================================================
// logistics_route_opt — Analyze current logistics route split and
// recommend adjustments based on destination distribution.
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
