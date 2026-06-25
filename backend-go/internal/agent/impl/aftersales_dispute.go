// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ========================================================================
// dispute_manage
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
