package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ProductScoutTools returns the tool definitions for A1 ProductScoutAgent.
func ProductScoutTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "product_scout",
			Version:     "1.0.0",
			Description: "选品打分——多维度（需求/竞争/利润/趋势）对候选商品打分排序，返回 Top-20 结果",
			Squad:       "growth",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "选品打分参数",
				Properties: map[string]*toolregistry.Schema{
					"category":    {Type: "string", Description: "商品类目"},
					"marketplace": {Type: "string", Description: "目标市场（如 US/JP/EU）"},
					"candidates": {
						Type:        "array",
						Description: "候选商品列表",
						Items: &toolregistry.Schema{
							Type:       "object",
							Properties: map[string]*toolregistry.Schema{
								"name":          {Type: "string", Description: "商品名称"},
								"price":         {Type: "number", Description: "售价"},
								"cost":          {Type: "number", Description: "成本"},
								"search_volume": {Type: "number", Description: "搜索量"},
								"trend_growth":  {Type: "number", Description: "趋势增长率(%)"},
								"review_count":  {Type: "number", Description: "竞品评论数"},
							},
						},
					},
				},
				Required: []string{"category", "marketplace", "candidates"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "选品排序结果，包含 Top-20 商品及其多维度评分和风险标记",
			},
			RequiredPermissions: []string{"growth:read:product_scout"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":      "ok",
					"message":     "stub: 选品评分功能尚在实现中，将在后续版本上线",
					"category":    input["category"],
					"marketplace": input["marketplace"],
				}, nil
			},
		},
		{
			Name:        "market_analysis",
			Version:     "1.0.0",
			Description: "市场分析——快速评估品类市场概况（市场规模、趋势方向、置信度）",
			Squad:       "growth",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "市场分析参数",
				Properties: map[string]*toolregistry.Schema{
					"category":    {Type: "string", Description: "商品类目"},
					"marketplace": {Type: "string", Description: "目标市场（如 US/JP/EU，默认 US）"},
					"trend":       {Type: "string", Description: "趋势方向（可选，如 stable/rising/declining）"},
				},
				Required: []string{"category"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "市场分析概要，包含类目、市场规模评估、趋势方向和置信度",
			},
			RequiredPermissions: []string{"growth:read:market_analysis"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":   "ok",
					"message":  "stub: 市场分析功能尚在实现中，将在后续版本上线",
					"category": input["category"],
				}, nil
			},
		},
	}
}
