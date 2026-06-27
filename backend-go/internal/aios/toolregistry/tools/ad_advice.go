package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// AdAdviceTools returns the tool definitions for the A3 AdAdviceAgent domain.
// These tools provide ACoS analysis (risk grading, CPC bid suggestions, budget alerts)
// and ad optimization (negative keyword detection, bid reduction, budget scaling).
func AdAdviceTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "acos.analyze",
			Version:     "1.0.0",
			Description: "ACoS分析——分析广告花费/销售比率(ACoS)，检测异常(毛利率超限/目标超限)，生成CPC出价建议、预算消耗预警、库存风险提示和低CTR/CVR告警",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "ACoS分析参数",
				Properties: map[string]*toolregistry.Schema{
					"campaign_id": {Type: "string", Description: "广告活动ID（必填）"},
					"spend":       {Type: "number", Description: "广告花费（必填）"},
					"sales":       {Type: "number", Description: "广告带来的销售额（必填）"},
					"clicks":      {Type: "number", Description: "点击次数（可选）"},
					"impressions": {Type: "number", Description: "展示次数（可选）"},
					"conversions": {Type: "number", Description: "转化次数（可选）"},
					"budget":      {Type: "number", Description: "广告预算（可选）"},
					"inventory_status": {
						Type:        "string",
						Description: "库存状态（可选，normal/low/out_of_stock）",
						Enum:        []string{"normal", "low", "out_of_stock"},
					},
					"gross_margin": {Type: "number", Description: "毛利率百分比（可选）"},
					"target_acos":  {Type: "number", Description: "目标ACoS百分比（可选，默认30）"},
				},
				Required: []string{"campaign_id", "spend", "sales"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "ACoS分析结果，包含整体状态、指标(ACoS/CTR/CVR/CPC/预算使用率)、告警列表、建议列表、CPC出价建议",
			},
			RequiredPermissions: []string{"ad:read:acos"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         5 * time.Second,
		},
		{
			Name:        "ad.optimize",
			Version:     "1.0.0",
			Description: "广告优化建议——分析搜索词表现，生成否定关键词、出价下调、预算增加等优化建议",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "广告优化参数",
				Properties: map[string]*toolregistry.Schema{
					"campaign_id": {Type: "string", Description: "广告活动ID（必填）"},
					"spend":       {Type: "number", Description: "广告花费（可选）"},
					"sales":       {Type: "number", Description: "广告带来的销售额（可选）"},
					"clicks":      {Type: "number", Description: "点击次数（可选）"},
					"budget":      {Type: "number", Description: "广告预算（可选）"},
					"target_acos": {Type: "number", Description: "目标ACoS百分比（可选，默认30）"},
					"search_terms": {
						Type:        "array",
						Description: "搜索词表现数据列表",
						Items: &toolregistry.Schema{
							Type: "object",
							Properties: map[string]*toolregistry.Schema{
								"keyword": {Type: "string", Description: "搜索词"},
								"spend":   {Type: "number", Description: "花费"},
								"sales":   {Type: "number", Description: "销售额"},
								"clicks":  {Type: "number", Description: "点击次数"},
							},
						},
					},
				},
				Required: []string{"campaign_id"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "广告优化结果，包含优化项列表(否定关键词、出价下调、预算增加)和当前ACoS",
			},
			RequiredPermissions: []string{"ad:write:optimize"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         5 * time.Second,
		},
	}
}
