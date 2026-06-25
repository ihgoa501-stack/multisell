package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func CustomerServiceTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "aftersales.order.read",
			Version:     "1.0.0",
			Description: "售后单查询——查询售后/退货单详情，包括原因、状态、商品、退款金额等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "售后单查询参数",
				Properties: map[string]*toolregistry.Schema{
					"aftersale_id": {Type: "string", Description: "售后单ID"},
					"order_id":     {Type: "string", Description: "关联订单ID（可选，与aftersale_id二选一）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "售后单详情，包含售后单ID、关联订单、商品、原因、状态、退款金额、时间线等",
			},
			RequiredPermissions: []string{"aftersales:read:order"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "aftersales.refund.decide",
			Version:     "1.0.0",
			Description: "退款决策——对售后申请做出退款或驳回决策，需指定处理意见",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "退款决策参数",
				Properties: map[string]*toolregistry.Schema{
					"aftersale_id": {Type: "string", Description: "售后单ID"},
					"decision": {
						Type:        "string",
						Description: "决策：approve（同意退款）/reject（驳回）/partial（部分退款）",
						Enum:        []string{"approve", "reject", "partial"},
					},
					"refund_amount": {Type: "number", Description: "退款金额（partial时必填）"},
					"reason":        {Type: "string", Description: "决策原因/备注"},
				},
				Required: []string{"aftersale_id", "decision"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "决策结果，包含售后单ID、处理状态、退款金额和时间",
			},
			RequiredPermissions: []string{"aftersales:write:refund"},
			RiskLevel:           toolregistry.RiskCritical,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "aftersales.dispute.query",
			Version:     "1.0.0",
			Description: "纠纷查询——查询平台纠纷/争议记录，包括纠纷类型、状态、举证信息等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "纠纷查询参数",
				Properties: map[string]*toolregistry.Schema{
					"dispute_id":  {Type: "string", Description: "纠纷ID（可选）"},
					"order_id":    {Type: "string", Description: "关联订单ID（可选）"},
					"status":      {Type: "string", Description: "纠纷状态（可选）"},
					"page":        {Type: "integer", Description: "页码（默认1）"},
					"size":        {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "纠纷列表（或详情），包含纠纷ID、类型、状态、关联订单、金额、时间等",
			},
			RequiredPermissions: []string{"aftersales:read:dispute"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
	}
}
