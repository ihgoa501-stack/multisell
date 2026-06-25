// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"go.uber.org/zap"
)

// ========================================================================
// aftersales_report
// ========================================================================

// aftersalesReport aggregates aftersales KPIs for a given period, compares
// against the previous period, and flags anomalies.
func (a *AftersalesMgmtAgent) aftersalesReport(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	period := safeString(ctx["period"], "30d")
	days := parsePeriodDays(period)
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	prevStart := start.AddDate(0, 0, -days)

	if a.db == nil {
		return degradedAftersalesReport(period), 0.3, "medium", nil
	}

	// Current and previous period KPI
	currentKPIs := a.computeKPI(start, now)
	prevKPIs := a.computeKPI(prevStart, start)

	// KPI trends
	returnRateChange := safeDeltaPct(safeFloat(currentKPIs["return_rate"]), safeFloat(prevKPIs["return_rate"]))
	refundRateChange := safeDeltaPct(safeFloat(currentKPIs["refund_rate"]), safeFloat(prevKPIs["refund_rate"]))
	avgRefundChange := safeDeltaPct(safeFloat(currentKPIs["avg_refund_amount"]), safeFloat(prevKPIs["avg_refund_amount"]))
	disputeWinChange := safeDeltaPct(safeFloat(currentKPIs["dispute_win_rate"]), safeFloat(prevKPIs["dispute_win_rate"]))

	trends := map[string]interface{}{
		"return_rate": map[string]interface{}{
			"current":    currentKPIs["return_rate"],
			"previous":   prevKPIs["return_rate"],
			"change_pct": returnRateChange,
		},
		"refund_rate": map[string]interface{}{
			"current":    currentKPIs["refund_rate"],
			"previous":   prevKPIs["refund_rate"],
			"change_pct": refundRateChange,
		},
		"avg_refund_amount": map[string]interface{}{
			"current":    currentKPIs["avg_refund_amount"],
			"previous":   prevKPIs["avg_refund_amount"],
			"change_pct": avgRefundChange,
		},
		"dispute_win_rate": map[string]interface{}{
			"current":    currentKPIs["dispute_win_rate"],
			"previous":   prevKPIs["dispute_win_rate"],
			"change_pct": disputeWinChange,
		},
	}

	// Anomaly alerts
	alerts := a.detectAnomalies(currentKPIs, prevKPIs, trends)

	riskLvl := "low"
	conf := 0.85
	if len(alerts) > 2 {
		riskLvl = "high"
		conf = 0.90
	} else if len(alerts) > 0 {
		riskLvl = "medium"
	}

	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "aftersales_report",
		"period":         period,
		"kpis":           currentKPIs,
		"trends":         trends,
		"alerts":         alerts,
		"risk_level":     riskLvl,
		"confidence":     conf,
	}
	return output, conf, riskLvl, nil
}

// computeKPI aggregates aftersales metrics for a given time range.
func (a *AftersalesMgmtAgent) computeKPI(tStart, tEnd time.Time) map[string]interface{} {
	totalReturns, _ := a.queryCount(a.db.Model(&aftersales.AfterSalesOrder{}).
		Where("created_at >= ? AND created_at < ?", tStart, tEnd))

	totalOrders, _ := a.queryCount(a.db.Model(&order.Order{}).
		Where("created_at >= ? AND created_at < ?", tStart, tEnd))

	type refundAgg struct {
		Cnt   int64
		Total float64
	}
	var ra refundAgg
	if err := a.db.Model(&aftersales.AfterSalesOrder{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(refund_amount),0) AS total").
		Where("created_at >= ? AND created_at < ? AND status = ?", tStart, tEnd, "refunded").
		Scan(&ra).Error; err != nil {
		a.logger.Debug("aftersales_report: refund aggregate failed", zap.Error(err))
	}

	// Dispute count and won count.
	type disputeAgg struct {
		Total int64
		Won   int64
	}
	var da disputeAgg
	disputeKeywords := []string{"dispute", "争议", "AtoZ", "claim", "投诉", "纠纷"}
	likeClauses := make([]string, 0, len(disputeKeywords))
	dargs := make([]interface{}, 0)
	for _, kw := range disputeKeywords {
		likeClauses = append(likeClauses, "LOWER(reason) LIKE LOWER(?)")
		dargs = append(dargs, "%"+kw+"%")
	}
	whereOr := strings.Join(likeClauses, " OR ")
	dargs = append(dargs, tStart, tEnd)
	if err := a.db.Table("after_sales_order").
		Select(`
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0) AS won
		`).
		Where("("+whereOr+") AND created_at >= ? AND created_at < ?", dargs...).
		Scan(&da).Error; err != nil {
		a.logger.Debug("aftersales_report: dispute aggregate failed", zap.Error(err))
	}

	returnRate := 0.0
	if totalOrders > 0 {
		returnRate = math.Round(float64(totalReturns)/float64(totalOrders)*10000) / 100
	}
	refundRate := 0.0
	if totalReturns > 0 {
		refundRate = math.Round(float64(ra.Cnt)/float64(totalReturns)*10000) / 100
	}
	avgRefund := 0.0
	if ra.Cnt > 0 {
		avgRefund = math.Round(ra.Total/float64(ra.Cnt)*100) / 100
	}
	disputeWinRate := 0.0
	if da.Total > 0 {
		disputeWinRate = math.Round(float64(da.Won)/float64(da.Total)*10000) / 100
	}

	return map[string]interface{}{
		"total_returns":     totalReturns,
		"total_orders":      totalOrders,
		"total_refunded":    ra.Cnt,
		"total_refund_amt":  ra.Total,
		"total_disputes":    da.Total,
		"disputes_won":      da.Won,
		"return_rate":       returnRate,
		"refund_rate":       refundRate,
		"avg_refund_amount": avgRefund,
		"dispute_win_rate":  disputeWinRate,
	}
}

// detectAnomalies flags significant deviations between periods.
func (a *AftersalesMgmtAgent) detectAnomalies(current, previous, trends map[string]interface{}) []map[string]interface{} {
	alerts := make([]map[string]interface{}, 0)

	returnT := safeDeltaPct(
		safeFloat(trends["return_rate"]),
		safeFloat(trends["return_rate"]),
	)
	if rtr, ok := trends["return_rate"]; ok {
		if rtrMap, ok := rtr.(map[string]interface{}); ok {
			returnT = safeFloat(rtrMap["change_pct"])
		}
	}

	// Return rate spike > 20% week-over-week.
	if returnT > 20 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "return_rate_spike",
			"severity": "high",
			"message":  fmt.Sprintf("退货率环比上升 %.1f%%，需关注原因", returnT),
			"current":  safeFloat(current["return_rate"]),
			"previous": safeFloat(previous["return_rate"]),
		})
	}
	// Return rate decrease (positive signal).
	if returnT < -20 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "return_rate_improvement",
			"severity": "info",
			"message":  fmt.Sprintf("退货率环比下降 %.1f%%，售后改善明显", -returnT),
			"current":  safeFloat(current["return_rate"]),
			"previous": safeFloat(previous["return_rate"]),
		})
	}

	// Refund rate spike.
	refundT := safeMapFloat(trends["refund_rate"], "change_pct")
	if refundT > 30 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "refund_rate_spike",
			"severity": "high",
			"message":  fmt.Sprintf("退款率环比上升 %.1f%%，退款金额异常增长", refundT),
			"current":  safeFloat(current["refund_rate"]),
			"previous": safeFloat(previous["refund_rate"]),
		})
	}

	// Average refund amount increase.
	avgT := safeMapFloat(trends["avg_refund_amount"], "change_pct")
	if avgT > 50 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "avg_refund_increase",
			"severity": "medium",
			"message":  fmt.Sprintf("平均退款金额环比增加 %.1f%%，存在高价值退款增长趋势", avgT),
			"current":  safeFloat(current["avg_refund_amount"]),
			"previous": safeFloat(previous["avg_refund_amount"]),
		})
	}

	// Dispute win rate drop.
	winT := safeMapFloat(trends["dispute_win_rate"], "change_pct")
	if winT < -20 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "dispute_win_rate_drop",
			"severity": "high",
			"message":  fmt.Sprintf("争议胜诉率环比下降 %.1f%%，申诉证据或策略需优化", -winT),
			"current":  safeFloat(current["dispute_win_rate"]),
			"previous": safeFloat(previous["dispute_win_rate"]),
		})
	}

	// Raw volume check.
	curReturns := safeFloat(current["total_returns"])
	prevReturns := safeFloat(previous["total_returns"])
	if prevReturns > 0 {
		volChange := (curReturns - prevReturns) / prevReturns * 100
		if volChange > 50 {
			alerts = append(alerts, map[string]interface{}{
				"type":     "return_volume_spike",
				"severity": "high",
				"message":  fmt.Sprintf("售后单量环比增长 %.0f%%，需排查是否存在系统性质量问题", volChange),
				"current":  curReturns,
				"previous": prevReturns,
			})
		}
	}

	// Return rate threshold.
	curRate := safeFloat(current["return_rate"])
	if curRate > 15 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "high_return_rate",
			"severity": "high",
			"message":  fmt.Sprintf("当前退货率 %.2f%% 超过15%%警戒线", curRate),
			"current":  curRate,
			"previous": safeFloat(previous["return_rate"]),
		})
	}

	return alerts
}

// degradedAftersalesReport returns a graceful fallback for report.
func degradedAftersalesReport(period string) map[string]interface{} {
	return map[string]interface{}{
		"status":         "degraded",
		"decision_point": "aftersales_report",
		"period":         period,
		"kpis": map[string]interface{}{
			"return_rate":      0.0,
			"refund_rate":      0.0,
			"avg_refund_amount": 0.0,
			"dispute_win_rate": 0.0,
			"total_returns":    0,
			"total_orders":     0,
		},
		"trends":       map[string]interface{}{},
		"alerts":       []map[string]interface{}{},
		"message":      "数据库不可用，无法生成售后报告",
		"confidence":   0.3,
		"risk_level":   "medium",
	}
}
