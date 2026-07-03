package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func FinanceTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "finance.profit.calculate",
			Version:     "1.0.0",
			Description: "计算订单利润——根据订单收入、成本、平台费用等计算单笔订单的利润和利润率",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "利润计算参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id": {Type: "string", Description: "订单ID"},
				},
				Required: []string{"order_id"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "订单利润详情，包含收入、成本、费用、利润、利润率等",
			},
			RequiredPermissions: []string{"finance:read:profit"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         15 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":   "ok",
					"message":  "stub: 单笔利润计算功能尚在实现中，将在后续版本上线",
					"order_id": input["order_id"],
				}, nil
			},
		},
		{
			Name:        "finance.profit.summary",
			Version:     "1.0.0",
			Description: "利润汇总——按时间段、平台、店铺等维度汇总利润数据",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "利润汇总参数",
				Properties: map[string]*toolregistry.Schema{
					"start_date": {Type: "string", Format: "date", Description: "开始日期（YYYY-MM-DD）"},
					"end_date":   {Type: "string", Format: "date", Description: "结束日期（YYYY-MM-DD）"},
					"platform":   {Type: "string", Description: "平台（可选，如ozon/shopee）"},
					"group_by":   {Type: "string", Description: "汇总维度：day/week/month/platform/sku，默认day", Enum: []string{"day", "week", "month", "platform", "sku"}},
				},
				Required: []string{"start_date", "end_date"},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "利润汇总数据列表，按指定维度聚合的总收入、总成本、总利润、平均利润率等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"finance:read:profit"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         15 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":     "ok",
					"message":    "stub: 利润汇总功能尚在实现中，将在后续版本上线",
					"start_date": input["start_date"],
					"end_date":   input["end_date"],
				}, nil
			},
		},
		{
			Name:        "finance.settlement.import",
			Version:     "1.0.0",
			Description: "导入结算单——导入平台结算单数据，用于对账和利润计算",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "结算单导入参数",
				Properties: map[string]*toolregistry.Schema{
					"platform": {
						Type:        "string",
						Description: "平台名称（ozon/shopee等）",
						Enum:        []string{"ozon", "shopee"},
					},
					"file_url":       {Type: "string", Description: "结算单文件URL"},
					"period":         {Type: "string", Description: "结算周期（如2026-W26）"},
					"auto_reconcile": {Type: "boolean", Description: "导入后自动对账（默认true）"},
				},
				Required: []string{"platform", "file_url"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "导入结果，包含导入ID、匹配订单数、金额差异列表等",
			},
			RequiredPermissions: []string{"finance:write:settlement"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         60 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":   "ok",
					"message":  "stub: 结算单导入功能尚在实现中，将在后续版本上线",
					"platform": input["platform"],
				}, nil
			},
		},
		{
			Name:        "finance.settlement.reconcile",
			Version:     "1.0.0",
			Description: "结算对账——将平台结算单与系统订单数据进行逐笔匹配对账，识别差异",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "对账参数",
				Properties: map[string]*toolregistry.Schema{
					"settlement_id": {Type: "string", Description: "结算单ID"},
					"threshold":     {Type: "number", Description: "差异容忍阈值（金额，默认0）"},
				},
				Required: []string{"settlement_id"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "对账结果，包含已匹配数、未匹配数、差异明细列表、差异总金额等",
			},
			RequiredPermissions: []string{"finance:write:settlement"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         60 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":        "ok",
					"message":       "stub: 结算对账功能尚在实现中，将在后续版本上线",
					"settlement_id": input["settlement_id"],
				}, nil
			},
		},
	}
}
