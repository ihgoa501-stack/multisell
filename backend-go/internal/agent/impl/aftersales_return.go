// Package impl provides concrete agent implementations.
package impl

import (
	"math"
	"sort"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"go.uber.org/zap"
)

// reasonRow is the intermediate struct for reason grouping.
type reasonRow struct {
	Reason string
	Cnt    int64
}

// ========================================================================
// return_analysis
// ========================================================================

// returnAnalysis analyzes aftersales returns grouped by reason, SKU, and time
// to surface return rates, top reasons, problem SKUs, and trend direction.
func (a *AftersalesMgmtAgent) returnAnalysis(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	period := safeString(ctx["period"], "30d")
	days := parsePeriodDays(period)
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	prevStart := start.AddDate(0, 0, -days)

	// Graceful degradation when DB is nil.
	if a.db == nil {
		return degradedReturnAnalysis(period), 0.3, "medium", nil
	}

	// Total returns (current period)
	totalReturns, err := a.queryCount(a.db.Model(&aftersales.AfterSalesOrder{}).
		Where("created_at >= ? AND created_at < ?", start, now))
	if err != nil {
		a.logger.Warn("return_analysis: total_returns query failed", zap.Error(err))
		return degradedReturnAnalysis(period), 0.3, "medium", nil
	}

	// Total orders (current period)
	totalOrders, err := a.queryCount(a.db.Model(&order.Order{}).
		Where("created_at >= ? AND created_at < ?", start, now))
	if err != nil {
		a.logger.Warn("return_analysis: total_orders query failed", zap.Error(err))
		totalOrders = 0
	}

	// Previous period returns (for trend)
	prevReturns, _ := a.queryCount(a.db.Model(&aftersales.AfterSalesOrder{}).
		Where("created_at >= ? AND created_at < ?", prevStart, start))

	// Return rate
	returnRate := 0.0
	if totalOrders > 0 {
		returnRate = math.Round(float64(totalReturns)/float64(totalOrders)*10000) / 100
	}

	// Group by reason
	var reasonRows []reasonRow
	if err := a.db.Model(&aftersales.AfterSalesOrder{}).
		Select("reason, COUNT(*) AS cnt").
		Where("created_at >= ? AND created_at < ?", start, now).
		Group("reason").
		Order("cnt DESC").
		Scan(&reasonRows).Error; err != nil {
		a.logger.Warn("return_analysis: reason grouping failed", zap.Error(err))
		reasonRows = nil
	}
	topReasons := buildTopReasons(reasonRows, totalReturns)

	// Problem SKUs (return_rate > 15%)
	problemSKUs := a.findProblemSKUs(start, now)

	// Trend
	trend := calculateTrend(totalReturns, prevReturns)

	confidence = 0.85
	riskLevel = "low"
	if returnRate > 15 {
		riskLevel = "high"
	} else if returnRate > 8 {
		riskLevel = "medium"
	}

	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "return_analysis",
		"period":         period,
		"total_returns":  totalReturns,
		"total_orders":   totalOrders,
		"return_rate":    returnRate,
		"top_reasons":    topReasons,
		"problem_skus":   problemSKUs,
		"trend":          trend,
		"risk_level":     riskLevel,
		"confidence":     confidence,
	}
	return output, confidence, riskLevel, nil
}

// buildTopReasons categorises individual return reasons into standard buckets
// and returns the top categories with counts and percentages.
func buildTopReasons(rows []reasonRow, totalReturns int64) []map[string]interface{} {
	cats := map[string]int64{
		"product_quality":  0,
		"logistics_damage": 0,
		"buyer_remorse":    0,
		"wrong_item":       0,
		"other":            0,
	}
	for _, r := range rows {
		cat := categorizeReason(r.Reason)
		cats[cat] += r.Cnt
	}

	var top []map[string]interface{}
	for cat, cnt := range cats {
		if cnt == 0 {
			continue
		}
		pct := 0.0
		if totalReturns > 0 {
			pct = math.Round(float64(cnt)/float64(totalReturns)*10000) / 100
		}
		label := reasonCategoryLabel(cat)
		top = append(top, map[string]interface{}{
			"category":   cat,
			"label":      label,
			"count":      cnt,
			"percentage": pct,
		})
	}
	// Sort by count descending.
	sort.Slice(top, func(i, j int) bool {
		ci := int64(safeFloat(top[i]["count"]))
		cj := int64(safeFloat(top[j]["count"]))
		return cj > ci
	})
	return top
}

// findProblemSKUs queries SKUs with return rates exceeding 15%.
func (a *AftersalesMgmtAgent) findProblemSKUs(start time.Time, now time.Time) []map[string]interface{} {
	type skuReturnRow struct {
		SkuID    int64
		SkuCode  string
		RetCount int64
	}
	var skuRows []skuReturnRow
	if err := a.db.Table("after_sales_order").
		Select(`
			COALESCE(aso.sku_id, 0) AS sku_id,
			COALESCE(s.code, '') AS sku_code,
			COUNT(*) AS ret_count
		`).
		Joins("LEFT JOIN sku s ON s.id = aso.sku_id").
		Where("aso.created_at >= ? AND aso.created_at < ?", start, now).
		Group("aso.sku_id, s.code").
		Order("ret_count DESC").
		Limit(20).
		Scan(&skuRows).Error; err != nil {
		a.logger.Warn("findProblemSKUs: query failed", zap.Error(err))
		return nil
	}
	if len(skuRows) == 0 {
		return nil
	}

	// Get total order count per SKU from sales_order_item.
	skuIDs := make([]int64, 0, len(skuRows))
	for _, r := range skuRows {
		if r.SkuID > 0 {
			skuIDs = append(skuIDs, r.SkuID)
		}
	}
	type skuOrderRow struct {
		SkuID int64
		Cnt   int64
	}
	var orderCounts []skuOrderRow
	if len(skuIDs) > 0 {
		if err := a.db.Table("sales_order_item").
			Select("sku_id, COUNT(DISTINCT order_id) AS cnt").
			Where("sku_id IN ? AND created_at >= ? AND created_at < ?", skuIDs, start, now).
			Group("sku_id").
			Scan(&orderCounts).Error; err != nil {
			a.logger.Warn("findProblemSKUs: order count query failed", zap.Error(err))
		}
	}
	orderMap := make(map[int64]int64, len(orderCounts))
	for _, oc := range orderCounts {
		orderMap[oc.SkuID] = oc.Cnt
	}

	problemSKUs := make([]map[string]interface{}, 0)
	for _, r := range skuRows {
		orderCnt := orderMap[r.SkuID]
		skuRate := 0.0
		if orderCnt > 0 {
			skuRate = math.Round(float64(r.RetCount)/float64(orderCnt)*10000) / 100
		}
		if skuRate > 15 || r.SkuID == 0 {
			riskLabel := "high"
			if skuRate <= 15 {
				riskLabel = "medium"
			}
			problemSKUs = append(problemSKUs, map[string]interface{}{
				"sku_id":       r.SkuID,
				"sku_code":     r.SkuCode,
				"return_count": r.RetCount,
				"order_count":  orderCnt,
				"return_rate":  skuRate,
				"risk_level":   riskLabel,
			})
		}
	}
	return problemSKUs
}

// degradedReturnAnalysis returns a best-effort result when DB is unavailable.
func degradedReturnAnalysis(period string) map[string]interface{} {
	return map[string]interface{}{
		"status":         "degraded",
		"decision_point": "return_analysis",
		"period":         period,
		"total_returns":  0,
		"total_orders":   0,
		"return_rate":    0.0,
		"top_reasons":    []map[string]interface{}{},
		"problem_skus":   []map[string]interface{}{},
		"trend":          "unknown",
		"message":        "数据库不可用，无法获取售后数据",
		"confidence":     0.3,
	}
}
