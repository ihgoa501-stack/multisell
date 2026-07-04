package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func PlatformTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "platform_fee.list",
			Version:     "1.0.0",
			Description: "平台费用查询——查询各平台的佣金、手续费、广告费等费用明细和费率",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "费用查询参数",
				Properties: map[string]*toolregistry.Schema{
					"platform":    {Type: "string", Description: "平台名称（ozon/shopee等）"},
					"fee_type":    {Type: "string", Description: "费用类型（commission/listing/advertising/fulfillment，可选）"},
					"start_date":  {Type: "string", Format: "date", Description: "开始日期（YYYY-MM-DD，可选）"},
					"end_date":    {Type: "string", Format: "date", Description: "结束日期（YYYY-MM-DD，可选）"},
				},
				Required: []string{"platform"},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "费用明细列表，包含费用类型、金额、费率、计费周期、关联订单等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"platform:read:fee"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":   "ok",
					"message":  "stub: 平台费用查询功能尚在实现中，将在后续版本上线",
					"platform": input["platform"],
				}, nil
			},
		},
	}
}
