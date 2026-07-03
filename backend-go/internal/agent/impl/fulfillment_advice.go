package impl

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// === Fulfillment Intelligence OS - A10 Structured Advice ===
// Phase 1: Adds bill_discrepancy and channel_performance advice types.

func (a *LogisticsOpsAgent) registerFulfillmentAdvice() {
	// Register bill_discrepancy_advice tool
	a.toolReg.Register(&toolregistry.Tool{
		Name:        "bill_discrepancy_advice",
		Version:     "1.0.0",
		Description: "账单差异建议——基于物流账单对账结果，输出可解释的复核和处理建议",
		Squad:       "logistics",
		RiskLevel:   toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, err := a.billDiscrepancyAdvice(input)
			if err != nil {
				return nil, err
			}
			return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
		},
	})

	// Register channel_performance_advice tool
	a.toolReg.Register(&toolregistry.Tool{
		Name:        "channel_performance_advice",
		Version:     "1.0.0",
		Description: "渠道表现建议——基于渠道时效、账单偏差和异常率，输出渠道切换或优化的可解释建议",
		Squad:       "logistics",
		RiskLevel:   toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, err := a.channelPerformanceAdvice(input)
			if err != nil {
				return nil, err
			}
			return &logisticsToolResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
		},
	})
}

// billDiscrepancyAdvice generates structured advice based on bill reconciliation results.
// Input: batch_id or list of discrepancies from shipping bill audit.
// Output: structured advice with reason, data basis, risk level, suggested action, and approval requirement.
func (a *LogisticsOpsAgent) billDiscrepancyAdvice(input map[string]interface{}) (map[string]interface{}, float64, string, error) {
	if a.db == nil {
		return map[string]interface{}{
			"advice_type":  "bill_discrepancy",
			"reason":       "数据库不可用，无法生成账单差异建议",
			"data_basis":   "无",
			"risk_level":   "medium",
			"suggested_action": "检查数据库连接后重新生成",
			"needs_approval": false,
			"confidence":   0.0,
			"advices":      []map[string]interface{}{},
		}, 0.0, "medium", nil
	}

	var advices []map[string]interface{}

	// If specific batch_id is given, look at that batch's anomalies
	if batchID, ok := input["batch_id"]; ok {
		bid := toInt64(batchID)
		if bid > 0 {
			var items []struct {
				ID                 int64
				ProviderName       string
				ChannelName        string
				OrderNo           string
				TotalActualFee    *float64
				SnapshotShippingFee *float64
				VarianceAmount    *float64
				VariancePct       *float64
				AnomalyType       string
				DestinationCountry string
			}
			a.db.Table("shipping_bill_item").
				Where("batch_id = ? AND anomaly_type != ''", bid).
				Find(&items)

			for _, item := range items {
				variance := 0.0
				if item.VarianceAmount != nil {
					variance = *item.VarianceAmount
				}
				variancePct := 0.0
				if item.VariancePct != nil {
					variancePct = *item.VariancePct
				}

				risk := "medium"
				needsApproval := false
				if math.Abs(variancePct) > 20 {
					risk = "high"
					needsApproval = true
				} else if math.Abs(variancePct) > 10 {
					risk = "medium"
				} else {
					risk = "low"
				}

				reason := fmt.Sprintf("订单 %s 使用 %s/%s 的账单运费与预估差异 %.1f%% (¥%.2f)",
					item.OrderNo, item.ProviderName, item.ChannelName, variancePct, variance)
				dataBasis := fmt.Sprintf("实际运费 ¥%.2f, 预估运费 ¥%.2f, 差异绝对值 ¥%.2f, 渠道 %s/%s, 目的国 %s",
					safeFloat(item.TotalActualFee), safeFloat(item.SnapshotShippingFee),
					math.Abs(variance), item.ProviderName, item.ChannelName, item.DestinationCountry)

				suggestedAction := "标记为待复核，联系物流商确认收费明细"
				if item.AnomalyType == "undercharge" {
					suggestedAction = "标记为待关注，确认是否漏收后续补扣"
					risk = "low"
				}

				advices = append(advices, map[string]interface{}{
					"advice_type":     "bill_discrepancy",
					"reason":          reason,
					"data_basis":      dataBasis,
					"risk_level":      risk,
					"suggested_action": suggestedAction,
					"needs_approval":  needsApproval,
					"order_no":        item.OrderNo,
					"provider":        item.ProviderName,
					"channel":         item.ChannelName,
					"variance_pct":    round2(variancePct),
					"variance_amount": round2(variance),
					"confidence":      0.85,
				})
			}
		}
	}

	// If no batch_id, scan recent unmatched bills
	if len(advices) == 0 {
		var recentItems []struct {
			ID                 int64
			ProviderName       string
			ChannelName        string
			OrderNo           string
			TotalActualFee    *float64
			SnapshotShippingFee *float64
			VarianceAmount    *float64
			VariancePct       *float64
			AnomalyType       string
			ReconciliationStatus string
		}
		since := time.Now().AddDate(0, 0, -30)
		a.db.Table("shipping_bill_item").
			Where("(anomaly_type != '' OR reconciliation_status = 'unmatched_order') AND created_at >= ?", since).
			Order("created_at DESC").Limit(20).Find(&recentItems)

		for _, item := range recentItems {
			if item.ReconciliationStatus == "unmatched_order" {
				advices = append(advices, map[string]interface{}{
					"advice_type":     "bill_discrepancy",
					"reason":          fmt.Sprintf("订单 %s 的账单无法匹配到运费快照", item.OrderNo),
					"data_basis":      fmt.Sprintf("物流商 %s, 渠道 %s, 账单金额 ¥%.2f, 无法匹配订单快照",
						item.ProviderName, item.ChannelName, safeFloat(item.TotalActualFee)),
					"risk_level":      "medium",
					"suggested_action": "手动确认订单号和物流商信息，补充运费快照后重新对账",
					"needs_approval":  false,
					"order_no":        item.OrderNo,
					"provider":        item.ProviderName,
					"channel":         item.ChannelName,
					"confidence":      0.70,
				})
			}
		}
	}

	if len(advices) == 0 {
		advices = []map[string]interface{}{}
	}

	output := map[string]interface{}{
		"advice_type":  "bill_discrepancy",
		"reason":       fmt.Sprintf("发现 %d 条账单差异建议", len(advices)),
		"data_basis":   "shipping_bill_item 对账结果",
		"risk_level":   "medium",
		"suggested_action": "按风险等级排序处理",
		"needs_approval": false,
		"advices":      advices,
		"total_advices": len(advices),
		"confidence":   0.85,
	}

	return output, 0.85, "low", nil
}

// channelPerformanceAdvice generates advice about channel performance.
func (a *LogisticsOpsAgent) channelPerformanceAdvice(input map[string]interface{}) (map[string]interface{}, float64, string, error) {
	if a.db == nil {
		return map[string]interface{}{
			"advice_type":  "channel_performance",
			"reason":       "数据库不可用，无法生成渠道表现建议",
			"data_basis":   "无",
			"risk_level":   "medium",
			"suggested_action": "检查数据库连接",
			"needs_approval": false,
			"advices":      []map[string]interface{}{},
		}, 0.0, "medium", nil
	}

	var advices []map[string]interface{}

	// Query channels with the most anomalies in the last 90 days
	type channelStat struct {
		ProviderName   string
		ChannelName    string
		TotalItems     int
		AnomalyItems   int
		TotalVariance  float64
		AvgVariancePct float64
	}
	var stats []channelStat
	a.db.Table("shipping_bill_item").
		Select(`provider_name, channel_name,
			COUNT(*) as total_items,
			SUM(CASE WHEN anomaly_type != '' THEN 1 ELSE 0 END) as anomaly_items,
			COALESCE(SUM(variance_amount), 0) as total_variance,
			COALESCE(AVG(ABS(variance_pct)), 0) as avg_variance_pct`).
		Where("created_at >= ?", time.Now().AddDate(0, 0, -90)).
		Group("provider_name, channel_name").
		Having("COUNT(*) > 0").
		Order("anomaly_items DESC").
		Limit(10).Scan(&stats)

	for _, st := range stats {
		if st.AnomalyItems == 0 {
			continue
		}
		anomalyRate := float64(st.AnomalyItems) / float64(st.TotalItems) * 100
		risk := "medium"
		needsApproval := true
		suggestedAction := fmt.Sprintf("建议复核 %s/%s 的费率规则和计费逻辑，近90天异常率 %.1f%%",
			st.ProviderName, st.ChannelName, anomalyRate)

		if anomalyRate > 30 {
			risk = "high"
		} else if anomalyRate < 10 {
			risk = "low"
			needsApproval = false
		}

		advices = append(advices, map[string]interface{}{
			"advice_type":     "channel_performance",
			"reason":          fmt.Sprintf("%s/%s 近90天账单异常率 %.1f%% (%d/%d)，总差异 ¥%.2f",
				st.ProviderName, st.ChannelName, anomalyRate, st.AnomalyItems, st.TotalItems, st.TotalVariance),
			"data_basis":      fmt.Sprintf("近90天共 %d 条账单记录，其中 %d 条异常，平均差异比例 %.1f%%",
				st.TotalItems, st.AnomalyItems, st.AvgVariancePct),
			"risk_level":      risk,
			"suggested_action": suggestedAction,
			"needs_approval":  needsApproval,
			"provider":        st.ProviderName,
			"channel":         st.ChannelName,
			"anomaly_rate_pct": round2(anomalyRate),
			"total_variance":  round2(st.TotalVariance),
			"confidence":      0.80,
		})
	}

	if len(advices) == 0 {
		advices = []map[string]interface{}{}
	}

	output := map[string]interface{}{
		"advice_type":  "channel_performance",
		"reason":       fmt.Sprintf("发现 %d 条渠道表现建议", len(advices)),
		"data_basis":   "shipping_bill_item 近90天统计",
		"risk_level":   "medium",
		"suggested_action": "按异常率排序处理",
		"needs_approval": false,
		"advices":      advices,
		"total_advices": len(advices),
		"confidence":   0.80,
	}

	return output, 0.80, "low", nil
}

// toInt64 safely converts an interface{} to int64.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case uint64:
		return int64(val)
	default:
		return 0
	}
}
