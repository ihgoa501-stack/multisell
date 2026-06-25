package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func InventoryTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "inventory.read",
			Version:     "1.0.0",
			Description: "查询库存——根据SKU编码或仓库查询当前库存数量、在途数量、可用库存等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "库存查询参数",
				Properties: map[string]*toolregistry.Schema{
					"sku":          {Type: "string", Description: "SKU编码（可选，不传则查全部）"},
					"warehouse_id": {Type: "string", Description: "仓库ID（可选）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "库存列表，每条包含SKU、仓库、在库数量、在途数量、可用数量",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"inventory:read:inventory"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "inventory.alert.list",
			Version:     "1.0.0",
			Description: "列出库存预警——查询已配置的安全库存预警规则",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "预警查询参数",
				Properties: map[string]*toolregistry.Schema{
					"sku":  {Type: "string", Description: "SKU编码（可选）"},
					"page": {Type: "integer", Description: "页码（默认1）"},
					"size": {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "分页的预警规则列表，包含规则ID、SKU、安全库存阈值、当前库存等",
			},
			RequiredPermissions: []string{"inventory:read:alert"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "inventory.alert.create",
			Version:     "1.0.0",
			Description: "创建库存预警规则——为指定SKU设置安全库存阈值，低于最低值或高于最高值时触发预警",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "预警规则参数",
				Properties: map[string]*toolregistry.Schema{
					"sku":       {Type: "string", Description: "SKU编码"},
					"min_stock": {Type: "integer", Description: "最低库存阈值"},
					"max_stock": {Type: "integer", Description: "最高库存阈值（可选，不设则表示仅监控下限）"},
				},
				Required: []string{"sku", "min_stock"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "创建的预警规则详情，包含规则ID和配置参数",
			},
			RequiredPermissions: []string{"inventory:write:alert"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "inventory.transfer.create",
			Version:     "1.0.0",
			Description: "创建调拨单——在不同仓库之间发起库存调拨，需指定源仓库、目标仓库和商品明细",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "调拨单参数",
				Properties: map[string]*toolregistry.Schema{
					"from_warehouse": {Type: "string", Description: "源仓库ID"},
					"to_warehouse":   {Type: "string", Description: "目标仓库ID"},
					"items": {
						Type:        "array",
						Description: "调拨商品列表",
						Items: &toolregistry.Schema{
							Type:        "object",
							Description: "调拨商品明细",
							Properties: map[string]*toolregistry.Schema{
								"sku":      {Type: "string", Description: "SKU编码"},
								"quantity": {Type: "integer", Description: "调拨数量"},
							},
						},
					},
				},
				Required: []string{"from_warehouse", "to_warehouse", "items"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "创建的调拨单详情，包含调拨单ID、状态和商品明细",
			},
			RequiredPermissions: []string{"inventory:write:transfer"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         30 * time.Second,
		},
		{
			Name:        "inventory.transfer.list",
			Version:     "1.0.0",
			Description: "查询调拨记录——按状态、日期范围等条件查询历史调拨单",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "调拨查询参数",
				Properties: map[string]*toolregistry.Schema{
					"status":     {Type: "string", Description: "调拨单状态（pending/approved/shipped/received/cancelled，可选）"},
					"start_date": {Type: "string", Format: "date", Description: "开始日期（YYYY-MM-DD）"},
					"end_date":   {Type: "string", Format: "date", Description: "结束日期（YYYY-MM-DD）"},
					"page":       {Type: "integer", Description: "页码（默认1）"},
					"size":       {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "分页的调拨单列表，包含调拨单ID、状态、仓库信息和商品明细",
			},
			RequiredPermissions: []string{"inventory:read:transfer"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
	}
}
