package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func ShippingTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "shipping.carrier.compare",
			Version:     "1.0.0",
			Description: "承运商比价——根据发货地址和包裹信息，比较不同承运商的运费和时效",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "比价参数",
				Properties: map[string]*toolregistry.Schema{
					"from_country":  {Type: "string", Description: "发货国家"},
					"to_country":    {Type: "string", Description: "收货国家"},
					"to_city":       {Type: "string", Description: "收货城市"},
					"weight_kg":     {Type: "number", Description: "包裹重量（kg）"},
					"length_cm":     {Type: "number", Description: "包裹长度（cm，可选）"},
					"width_cm":      {Type: "number", Description: "包裹宽度（cm，可选）"},
					"height_cm":     {Type: "number", Description: "包裹高度（cm，可选）"},
					"declared_value": {Type: "number", Description: "申报价值（USD，可选）"},
				},
				Required: []string{"from_country", "to_country", "to_city", "weight_kg"},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "承运商报价列表，按总费用升序排列，包含承运商名称、运输方式、费用、预计时效等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"shipping:read:carrier"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         15 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":      "ok",
					"message":     "stub: 承运商比价功能尚在实现中，将在后续版本上线",
					"from_country": input["from_country"],
					"to_country":   input["to_country"],
				}, nil
			},
		},
		{
			Name:        "shipping.track",
			Version:     "1.0.0",
			Description: "物流查询——根据运单号查询物流轨迹信息",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "物流查询参数",
				Properties: map[string]*toolregistry.Schema{
					"tracking_no": {Type: "string", Description: "运单号"},
					"carrier":     {Type: "string", Description: "承运商（可选，如CDEK/Boxberry等）"},
				},
				Required: []string{"tracking_no"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "物流轨迹信息，包含运单号、承运商、当前状态、轨迹节点列表（时间、地点、状态描述）",
			},
			RequiredPermissions: []string{"shipping:read:track"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":      "ok",
					"message":     "stub: 物流轨迹查询功能尚在实现中，将在后续版本上线",
					"tracking_no": input["tracking_no"],
				}, nil
			},
		},
		{
			Name:        "shipping.bill.audit",
			Version:     "1.0.0",
			Description: "运费审计——审计物流账单，比对实际运费与预估费用，识别异常收费",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "运费审计参数",
				Properties: map[string]*toolregistry.Schema{
					"bill_id":    {Type: "string", Description: "物流账单ID"},
					"carrier":    {Type: "string", Description: "承运商（可选）"},
					"start_date": {Type: "string", Format: "date", Description: "账单开始日期（YYYY-MM-DD）"},
					"end_date":   {Type: "string", Format: "date", Description: "账单结束日期（YYYY-MM-DD）"},
				},
				Required: []string{"bill_id"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "审计结果，包含总费用、预估费用、差异金额、异常明细列表等",
			},
			RequiredPermissions: []string{"shipping:write:bill"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         30 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":  "ok",
					"message": "stub: 运费审计功能尚在实现中，将在后续版本上线",
					"bill_id": input["bill_id"],
				}, nil
			},
		},
	}
}
