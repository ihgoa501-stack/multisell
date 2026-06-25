// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ========================================================================
// refund_decision
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
