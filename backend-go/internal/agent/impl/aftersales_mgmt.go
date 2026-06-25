// Package impl provides concrete agent implementations.
//
// AftersalesMgmtAgent implements A11 Aftersales Management business logic.
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.6
//   - return_analysis: analyze return reasons, rates, and problem SKUs
//   - refund_decision: auto-approve or escalate standard/risky refunds
//   - dispute_manage: platform dispute monitoring, SLA tracking, evidence checklist
//   - aftersales_report: aggregated KPI reporting with anomaly alerts
package impl

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- AftersalesMgmtAgent ----------

// AftersalesMgmtAgent implements A11 Aftersales Management logic.
// It handles return analysis, refund decision automation, dispute management,
// and aggregated reporting for aftersales operations.
type AftersalesMgmtAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAftersalesMgmtAgent creates a new AftersalesMgmtAgent.
func NewAftersalesMgmtAgent(db *gorm.DB, logger *zap.Logger) *AftersalesMgmtAgent {
	return &AftersalesMgmtAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "return_analysis"   — aggregated return reason analysis and problem SKU detection
//   - "refund_decision"   — evaluate a single refund or scan pending refunds
//   - "dispute_manage"    — platform dispute monitoring and response recommendation
//   - "aftersales_report" — KPI aggregation with trend and anomaly alerts
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *AftersalesMgmtAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "return_analysis":
		return a.returnAnalysis(ctx)
	case "refund_decision":
		return a.refundDecision(ctx)
	case "dispute_manage":
		return a.disputeManage(ctx)
	case "aftersales_report":
		return a.aftersalesReport(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("未知决策点: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ========================================================================
// 1. return_analysis
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

// reasonRow is the intermediate struct for reason grouping.
type reasonRow struct {
	Reason string
	Cnt    int64
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

// ========================================================================
// 2. refund_decision
// ========================================================================

// refundDecision evaluates a refund request (by after_sales_id) or scans all
// pending refunds, applying rules to auto-approve or escalate.
func (a *AftersalesMgmtAgent) refundDecision(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// If a specific aftersales ID is provided, evaluate that one.
	idVal, hasID := ctx["after_sales_id"]
	if hasID {
		asoID := int64(safeFloat(idVal))
		if asoID > 0 {
			return a.evaluateSingleRefund(asoID)
		}
	}
	// Otherwise scan all pending refund records.
	return a.scanPendingRefunds(ctx)
}

// evaluateSingleRefund evaluates one aftersales order for refund decision.
func (a *AftersalesMgmtAgent) evaluateSingleRefund(asoID int64) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return degradedRefundDecision("数据库不可用"), 0.3, "high", nil
	}

	var aso aftersales.AfterSalesOrder
	if err := a.db.First(&aso, asoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return degradedRefundDecision(fmt.Sprintf("售后单 %d 不存在", asoID)), 0.3, "high", nil
		}
		a.logger.Warn("evaluateSingleRefund: query failed", zap.Int64("aso_id", asoID), zap.Error(err))
		return degradedRefundDecision("数据库查询失败"), 0.3, "high", nil
	}

	// Only evaluate pending or approved-unrefunded records.
	if aso.Status == "refunded" || aso.Status == "rejected" {
		return map[string]interface{}{
			"status":         "completed",
			"decision_point": "refund_decision",
			"after_sales_id": aso.ID,
			"decision":       "no_action_needed",
			"reasoning":      fmt.Sprintf("售后单 %d 已处理(状态: %s)，无需再次处理", aso.ID, aso.Status),
			"risk_level":     "low",
			"confidence":     1.0,
		}, 1.0, "low", nil
	}

	// Determine risk factors.
	riskFactors := make([]string, 0)
	isRisky := false

	// 1. Check refund amount (threshold ~ $50 USD = ~360 CNY).
	if aso.RefundAmount > 360 {
		riskFactors = append(riskFactors, fmt.Sprintf("退款金额 ¥%.2f 超限", aso.RefundAmount))
		isRisky = true
	}

	// 2. Check frequent returner — same order has multiple aftersales.
	var sameOrderCount int64
	if err := a.db.Model(&aftersales.AfterSalesOrder{}).
		Where("order_id = ? AND id != ?", aso.OrderID, aso.ID).
		Count(&sameOrderCount).Error; err == nil && sameOrderCount > 0 {
		riskFactors = append(riskFactors, fmt.Sprintf("同一订单存在 %d 条售后申请", sameOrderCount+1))
		isRisky = true
	}

	// 3. Check order value vs refund amount.
	var ord order.Order
	if err := a.db.First(&ord, aso.OrderID).Error; err == nil {
		if ord.PayAmount > 0 && aso.RefundAmount > ord.PayAmount*0.8 {
			riskFactors = append(riskFactors, fmt.Sprintf("退款金额(¥%.2f)超过订单金额(¥%.2f)的80%%", aso.RefundAmount, ord.PayAmount))
			isRisky = true
		}
	}

	// 4. Check dispute-related reason.
	lowerReason := strings.ToLower(aso.Reason)
	if strings.Contains(lowerReason, "dispute") || strings.Contains(lowerReason, "争议") ||
		strings.Contains(lowerReason, "claim") || strings.Contains(lowerReason, "atoz") ||
		strings.Contains(lowerReason, "投诉") {
		riskFactors = append(riskFactors, "售后原因包含争议关键词，需人工介入")
		isRisky = true
	}

	// 5. Recent return rate for this SKU.
	if aso.SkuID != nil {
		var skuReturnCount int64
		_ = a.db.Model(&aftersales.AfterSalesOrder{}).
			Where("sku_id = ? AND created_at >= ?", *aso.SkuID, time.Now().AddDate(0, -1, 0)).
			Count(&skuReturnCount)
		if skuReturnCount > 5 {
			riskFactors = append(riskFactors, fmt.Sprintf("该SKU近30天已有 %d 笔退货", skuReturnCount))
			isRisky = true
		}
	}

	decision := "auto_approve"
	reasoning := "标准退款案例，自动批准"
	conf := 0.90
	riskLvl := "low"

	if isRisky {
		decision = "escalate_to_human"
		reasoning = "高风险退款案例，需人工审核: " + strings.Join(riskFactors, "; ")
		conf = 0.75
		riskLvl = "high"
	} else if aso.RefundAmount > 0 && aso.RefundAmount < 70 { // < $10 USD
		decision = "auto_approve"
		reasoning = fmt.Sprintf("小额退款(¥%.2f)，自动批准", aso.RefundAmount)
		conf = 0.95
		riskLvl = "low"
	}

	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "refund_decision",
		"after_sales_id": aso.ID,
		"order_id":       aso.OrderID,
		"refund_amount":  aso.RefundAmount,
		"reason":         aso.Reason,
		"decision":       decision,
		"reasoning":      reasoning,
		"risk_factors":   riskFactors,
		"risk_level":     riskLvl,
		"confidence":     conf,
	}
	return output, conf, riskLvl, nil
}

// scanPendingRefunds scans all pending aftersales and returns a summary
// of recommended actions for each candidate.
func (a *AftersalesMgmtAgent) scanPendingRefunds(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return degradedRefundDecision("数据库不可用"), 0.3, "high", nil
	}

	limit := int(safeFloat(ctx["scan_limit"], 20))
	var pendings []aftersales.AfterSalesOrder
	if err := a.db.Where("status IN ?", []string{"pending", "approved"}).
		Order("created_at ASC").
		Limit(limit).
		Find(&pendings).Error; err != nil {
		a.logger.Warn("scanPendingRefunds: query failed", zap.Error(err))
		return degradedRefundDecision("扫描待处理退款失败"), 0.3, "high", nil
	}

	candidates := make([]map[string]interface{}, 0, len(pendings))
	autoApproveCount := 0
	escalateCount := 0
	for _, p := range pendings {
		r, _, _, _ := a.evaluateSingleRefund(p.ID)
		if r == nil {
			continue
		}
		decision := safeString(r["decision"])
		candidates = append(candidates, map[string]interface{}{
			"after_sales_id": p.ID,
			"order_id":       p.OrderID,
			"refund_amount":  p.RefundAmount,
			"reason":         p.Reason,
			"status":         p.Status,
			"decision":       decision,
			"reasoning":      r["reasoning"],
			"risk_factors":   r["risk_factors"],
		})
		if decision == "auto_approve" {
			autoApproveCount++
		} else if decision == "escalate_to_human" {
			escalateCount++
		}
	}

	conf := 0.80
	riskLvl := "low"
	if escalateCount > 0 {
		riskLvl = "medium"
	}

	output = map[string]interface{}{
		"status":             "completed",
		"decision_point":     "refund_decision",
		"total_pending":      len(pendings),
		"auto_approve_count": autoApproveCount,
		"escalate_count":     escalateCount,
		"candidates":         candidates,
		"pending_note":       fmt.Sprintf("共扫描 %d 条待处理售后", len(pendings)),
		"risk_level":         riskLvl,
		"confidence":         conf,
	}
	return output, conf, riskLvl, nil
}

// degradedRefundDecision returns a fallback result when DB is unavailable.
func degradedRefundDecision(reason string) map[string]interface{} {
	return map[string]interface{}{
		"status":         "degraded",
		"decision_point": "refund_decision",
		"decision":       "escalate_to_human",
		"reasoning":      reason,
		"risk_level":     "high",
		"confidence":     0.3,
	}
}

// ========================================================================
// 3. dispute_manage
// ========================================================================

// disputeManage monitors platform disputes, checks SLA deadlines,
// and generates an evidence checklist based on dispute type.
func (a *AftersalesMgmtAgent) disputeManage(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.db == nil {
		return degradedDisputeManage("数据库不可用，请手动检查争议状态"), 0.3, "high", nil
	}

	// Scan for dispute-related aftersales.
	now := time.Now()
	disputeKeywords := []string{"dispute", "争议", "AtoZ", "claim", "投诉", "纠纷"}
	likeClauses := make([]string, 0, len(disputeKeywords))
	args := make([]interface{}, 0)
	for _, kw := range disputeKeywords {
		likeClauses = append(likeClauses, "LOWER(reason) LIKE LOWER(?)")
		args = append(args, "%"+kw+"%")
	}
	whereOr := strings.Join(likeClauses, " OR ")

	type disputeRow struct {
		ID           int64
		OrderID      int64
		Reason       string
		Status       string
		RefundAmount float64
		CreatedAt    time.Time
	}
	var disputes []disputeRow
	if err := a.db.Table("after_sales_order").
		Select("id, order_id, reason, status, refund_amount, created_at").
		Where("("+whereOr+") OR status = ?", append(args, "dispute")...).
		Order("created_at DESC").
		Limit(50).
		Scan(&disputes).Error; err != nil {
		a.logger.Warn("disputeManage: query failed", zap.Error(err))
		return degradedDisputeManage("争议查询失败"), 0.3, "high", nil
	}

	// Filter to unresolved (not refunded/rejected).
	activeDisputes := make([]map[string]interface{}, 0)
	slaBreachCount := 0
	slaThreshold := 48 * time.Hour

	for _, d := range disputes {
		if d.Status == "refunded" || d.Status == "rejected" {
			continue
		}
		age := now.Sub(d.CreatedAt)
		slaRemaining := slaThreshold - age
		slaBreach := slaRemaining <= 0
		if slaBreach {
			slaBreachCount++
		}

		dType := classifyDisputeType(d.Reason)
		checklist := disputeEvidenceChecklist(dType)

		activeDisputes = append(activeDisputes, map[string]interface{}{
			"id":                    d.ID,
			"order_id":              d.OrderID,
			"reason":                d.Reason,
			"dispute_type":          dType,
			"created_at":            d.CreatedAt.Format("2006-01-02 15:04"),
			"age_hours":             math.Round(age.Hours()),
			"sla_remaining_hours":   math.Round(slaRemaining.Hours()),
			"sla_breach":            slaBreach,
			"evidence_checklist":    checklist,
		})
	}

	activeCount := len(activeDisputes)
	slaRisk := "low"
	recommended := "继续监控，暂无需紧急处理"
	riskLvl := "low"
	conf := 0.85

	if activeCount > 0 {
		if slaBreachCount > 0 {
			slaRisk = "high"
			recommended = fmt.Sprintf("有 %d 个争议已超SLA时限，请立即响应", slaBreachCount)
			riskLvl = "high"
			conf = 0.90
		} else {
			slaRisk = "medium"
			recommended = fmt.Sprintf("有 %d 个争议待处理，请在 %d 小时内完成响应", activeCount, int(slaThreshold.Hours()))
			riskLvl = "medium"
		}
	}

	commonChecklist := disputeEvidenceChecklist("general")

	output = map[string]interface{}{
		"status":                "completed",
		"decision_point":        "dispute_manage",
		"active_disputes":       activeCount,
		"sla_breach_count":      slaBreachCount,
		"sla_breach_risk":       slaRisk,
		"disputes":              activeDisputes,
		"evidence_checklist":    commonChecklist,
		"recommended_response":  recommended,
		"sla_threshold_hours":   int(slaThreshold.Hours()),
		"risk_level":            riskLvl,
		"confidence":            conf,
	}
	return output, conf, riskLvl, nil
}

// degradedDisputeManage returns a fallback when DB is unavailable.
func degradedDisputeManage(msg string) map[string]interface{} {
	return map[string]interface{}{
		"status":                "degraded",
		"decision_point":        "dispute_manage",
		"active_disputes":       0,
		"sla_breach_risk":       "unknown",
		"evidence_checklist":    []string{},
		"recommended_response":  msg,
		"risk_level":            "high",
		"confidence":            0.3,
	}
}

// classifyDisputeType determines the dispute category from the reason text.
func classifyDisputeType(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "not received") || strings.Contains(lower, "未收到"):
		return "item_not_received"
	case strings.Contains(lower, "damage") || strings.Contains(lower, "破损") || strings.Contains(lower, "损坏"):
		return "damaged"
	case strings.Contains(lower, "wrong") || strings.Contains(lower, "错发") || strings.Contains(lower, "不对"):
		return "wrong_item"
	case strings.Contains(lower, "quality") || strings.Contains(lower, "质量") || strings.Contains(lower, "瑕疵"):
		return "quality_issue"
	case strings.Contains(lower, "refund") || strings.Contains(lower, "退款"):
		return "refund_dispute"
	default:
		return "other"
	}
}

// disputeEvidenceChecklist returns a list of required evidence items based on dispute type.
func disputeEvidenceChecklist(disputeType string) []string {
	base := []string{"订单详情截图", "物流追踪记录"}
	switch disputeType {
	case "item_not_received":
		return append(base, "物流签收截图", "与物流商沟通记录", "客户沟通记录")
	case "damaged":
		return append(base, "商品破损照片/视频", "外包装照片", "开箱视频/照片")
	case "wrong_item":
		return append(base, "客户收到的商品照片", "客户订单截图", "仓库发货记录", "拣货出库照片")
	case "quality_issue":
		return append(base, "质量问题照片/视频", "同批次质检报告", "产品说明/规格文档")
	case "refund_dispute":
		return append(base, "退款申请记录", "与客户沟通记录", "平台政策截图")
	default:
		return append(base, "客户申诉内容", "商品信息截图", "历史沟通记录")
	}
}

// ========================================================================
// 4. aftersales_report
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

// safeMapFloat extracts a nested float from a map-of-maps chain.
func safeMapFloat(v interface{}, key string) float64 {
	if v == nil {
		return 0
	}
	if m, ok := v.(map[string]interface{}); ok {
		return safeFloat(m[key])
	}
	return 0
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

// ========================================================================
// Internal helpers
// ========================================================================

// queryCount is a nil-safe shortcut for GORM Count().
func (a *AftersalesMgmtAgent) queryCount(tx *gorm.DB) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	var cnt int64
	if err := tx.Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// parsePeriodDays converts a period string like "7d", "30d", "week", "month"
// into a number of days (defaults to 30).
func parsePeriodDays(period string) int {
	switch {
	case strings.HasSuffix(period, "d"):
		var n int
		if _, err := fmt.Sscanf(period, "%dd", &n); err == nil && n > 0 {
			return n
		}
	case strings.EqualFold(period, "week") || strings.EqualFold(period, "7d"):
		return 7
	case strings.EqualFold(period, "month") || strings.EqualFold(period, "30d"):
		return 30
	case strings.EqualFold(period, "quarter") || strings.EqualFold(period, "90d"):
		return 90
	}
	return 30
}

// calculateTrend compares two counts to determine direction.
func calculateTrend(current, previous int64) string {
	if previous == 0 {
		if current == 0 {
			return "stable"
		}
		return "increasing"
	}
	ratio := float64(current) / float64(previous)
	switch {
	case ratio > 1.1:
		return "increasing"
	case ratio < 0.9:
		return "decreasing"
	default:
		return "stable"
	}
}

// categorizeReason maps a raw reason string to a standard category.
func categorizeReason(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "quality") || strings.Contains(lower, "质量") ||
		strings.Contains(lower, "defect") || strings.Contains(lower, "瑕疵") ||
		strings.Contains(lower, "malfunction") || strings.Contains(lower, "故障") ||
		strings.Contains(lower, "broken") || strings.Contains(lower, "坏了"):
		return "product_quality"
	case strings.Contains(lower, "物流") || strings.Contains(lower, "运输") ||
		(strings.Contains(lower, "damage") && !strings.Contains(lower, "defect")) ||
		strings.Contains(lower, "shipping") || strings.Contains(lower, "delivery") ||
		strings.Contains(lower, "外包装"):
		return "logistics_damage"
	case strings.Contains(lower, "not want") || strings.Contains(lower, "不想要") ||
		strings.Contains(lower, "change mind") || strings.Contains(lower, "改变主意") ||
		strings.Contains(lower, "不需要") || strings.Contains(lower, "no longer") ||
		strings.Contains(lower, "ordered wrong") || strings.Contains(lower, "买错"):
		return "buyer_remorse"
	case strings.Contains(lower, "wrong") || strings.Contains(lower, "错发") ||
		strings.Contains(lower, "不对") || strings.Contains(lower, "incorrect") ||
		strings.Contains(lower, "not match") || strings.Contains(lower, "不匹配"):
		return "wrong_item"
	default:
		return "other"
	}
}

// reasonCategoryLabel returns a Chinese label for a reason category.
func reasonCategoryLabel(cat string) string {
	switch cat {
	case "product_quality":
		return "产品质量问题"
	case "logistics_damage":
		return "物流运输损坏"
	case "buyer_remorse":
		return "买家原因(改变主意/不需要)"
	case "wrong_item":
		return "错发/漏发"
	case "other":
		return "其他原因"
	default:
		return cat
	}
}

// safeDeltaPct computes the percentage change between two KPI values.
func safeDeltaPct(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return math.Round((current-previous)/previous*10000) / 100
}
