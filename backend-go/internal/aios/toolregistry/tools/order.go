package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func OrderTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "order.read",
			Version:     "1.0.0",
			Description: "查询订单——根据订单ID或平台订单号查询订单详情，包括商品、金额、物流、状态等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "订单查询参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id":          {Type: "string", Description: "系统订单ID（与platform_order_no二选一）"},
					"platform_order_no": {Type: "string", Description: "平台订单号（与order_id二选一）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "订单详情，包含订单ID、商品列表、金额、物流信息、状态等",
			},
			RequiredPermissions: []string{"order:read:order"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "order.modify_address",
			Version:     "1.0.0",
			Description: "修改订单地址——修改已下单但未发货的订单收货地址",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "地址修改参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id":  {Type: "string", Description: "订单ID"},
					"recipient": {Type: "string", Description: "收件人姓名"},
					"phone":     {Type: "string", Description: "收件人电话"},
					"province":  {Type: "string", Description: "省"},
					"city":      {Type: "string", Description: "市"},
					"district":  {Type: "string", Description: "区/县"},
					"address":   {Type: "string", Description: "详细地址"},
					"zip_code":  {Type: "string", Description: "邮编（可选）"},
				},
				Required: []string{"order_id", "recipient", "phone", "province", "city", "address"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "修改结果，包含订单ID、更新后的地址信息",
			},
			RequiredPermissions: []string{"order:write:address"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "order.modify_logistics",
			Version:     "1.0.0",
			Description: "修改物流方式——修改已下单但未发货订单的物流方式和承运商",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "物流修改参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id":        {Type: "string", Description: "订单ID"},
					"carrier":         {Type: "string", Description: "承运商（如CDEK、Boxberry等）"},
					"shipping_method": {Type: "string", Description: "物流方式（standard/express/economy）"},
				},
				Required: []string{"order_id", "carrier"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "修改结果，包含订单ID、更新后的物流信息",
			},
			RequiredPermissions: []string{"order:write:logistics"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "order.split",
			Version:     "1.0.0",
			Description: "拆分订单——将一笔订单按商品拆分为多笔子订单，用于分仓发货或部分处理",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "拆分参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id": {Type: "string", Description: "订单ID"},
					"items": {
						Type:        "array",
						Description: "拆分方案：将指定SKU拆出形成新订单",
						Items: &toolregistry.Schema{
							Type:        "object",
							Description: "拆分项",
							Properties: map[string]*toolregistry.Schema{
								"sku":      {Type: "string", Description: "要拆出的SKU"},
								"quantity": {Type: "integer", Description: "拆出数量"},
							},
						},
					},
				},
				Required: []string{"order_id", "items"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "拆分结果，包含原订单ID和新创建的子订单ID列表",
			},
			RequiredPermissions: []string{"order:write:split"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         15 * time.Second,
		},
		{
			Name:        "order.merge",
			Version:     "1.0.0",
			Description: "合并订单——将同一收件人的多笔待处理订单合并为一笔，减少发货成本",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "合并参数",
				Properties: map[string]*toolregistry.Schema{
					"order_ids": {
						Type:        "array",
						Description: "要合并的订单ID列表（2-10笔）",
						Items:       &toolregistry.Schema{Type: "string"},
					},
				},
				Required: []string{"order_ids"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "合并结果，包含合并后的新订单ID和被合并的原始订单ID列表",
			},
			RequiredPermissions: []string{"order:write:merge"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         15 * time.Second,
		},
	}
}
