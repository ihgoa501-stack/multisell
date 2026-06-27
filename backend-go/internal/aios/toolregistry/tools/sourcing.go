package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/sourcing"
)

// SourcingTools returns the tool definitions for the sourcing domain (A8).
func SourcingTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "sourcing.recommend",
			Version:     "1.0.0",
			Description: "AI选品推荐——基于1688商品价格、重量、目的地计算利润，给出选品建议并推送至刊登优化流程",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "选品推荐参数",
				Properties: map[string]*toolregistry.Schema{
					"source_url":  {Type: "string", Description: "1688商品链接"},
					"price_1688":  {Type: "number", Description: "1688供货价格(CNY)"},
					"weight_kg":   {Type: "number", Description: "预估包裹重量(kg)"},
					"destination": {Type: "string", Description: "目标市场(US/EU/JP/RU/BR/AU，默认US)"},
					"product_name": {Type: "string", Description: "商品名称(可选)"},
					"markup_pct":  {Type: "number", Description: "期望加价率百分比(默认250，即2.5倍)"},
				},
				Required: []string{"source_url", "price_1688", "weight_kg", "destination"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "选品分析结果，包含利润率、利润金额、可行性评估和建议操作",
			},
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				price := safeFloat(input["price_1688"], 0)
				weight := safeFloat(input["weight_kg"], 0)
				dest := safeString(input["destination"], "US")
				markup := safeFloat(input["markup_pct"], 250.0)

				if price <= 0 {
					return nil, fmt.Errorf("sourcing.recommend: price_1688 must be positive")
				}

				profit := sourcing.CalculateProfit(&sourcing.ProfitInput{
					SourcePriceCNY: price,
					WeightKg:       weight,
					Destination:    dest,
					MarkupPct:      markup,
				})

				status := "viable"
				action := "escalate_to_optimizer"
				var message string

				switch {
				case profit.MarginPct >= 15:
					status = "viable"
					action = "escalate_to_optimizer"
					message = fmt.Sprintf(
						"该商品毛利率 %.1f%%，利润 ¥%.2f，利润率良好，建议推送至 A2 开启刊登优化流程。",
						profit.MarginPct, profit.ProfitCNY,
					)
				case profit.MarginPct >= 5:
					status = "marginal"
					action = "review"
					message = fmt.Sprintf(
						"该商品毛利率 %.1f%%，利润 ¥%.2f，利润率偏低，建议人工评估后决定是否上架。",
						profit.MarginPct, profit.ProfitCNY,
					)
				default:
					status = "unviable"
					action = "discard"
					message = fmt.Sprintf(
						"该商品毛利率 %.1f%%，利润 ¥%.2f，不满足最低利润率要求(15%%)，建议放弃该选品。",
						profit.MarginPct, profit.ProfitCNY,
					)
				}

				return map[string]interface{}{
					"status":           status,
					"action":           action,
					"profit_breakdown": profit,
					"message":          message,
				}, nil
			},
			RequiredPermissions: []string{"sourcing:write:recommend"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         10 * time.Second,
		},
	}
}

// --- Helpers for tool handlers ---

// safeFloat extracts a float64 from an interface{} value.
func safeFloat(v interface{}, defaultVal ...float64) float64 {
	def := 0.0
	if len(defaultVal) > 0 {
		def = defaultVal[0]
	}
	if v == nil {
		return def
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	}
	return def
}

// safeString extracts a string from an interface{} value.
func safeString(v interface{}, defaultVal ...string) string {
	def := ""
	if len(defaultVal) > 0 {
		def = defaultVal[0]
	}
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}
