package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func PurchaseTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "purchase_order.suggest",
			Version:     "1.0.0",
			Description: "生成采购建议——基于库存水平、销售预测和补货策略，自动生成采购建议单",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "采购建议参数",
				Properties: map[string]*toolregistry.Schema{
					"skus": {
						Type:        "array",
						Description: "需要生成采购建议的SKU列表（可选，不传则分析全部SKU）",
						Items:       &toolregistry.Schema{Type: "string"},
					},
					"strategy": {
						Type:        "string",
						Description: "补货策略：auto（自动）/manual（手动指定），默认auto",
						Enum:        []string{"auto", "manual"},
					},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "采购建议列表，每条包含SKU、建议采购数量、预计到货时间、优先级等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"purchase:read:suggest"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         30 * time.Second,
		},
		{
			Name:        "purchase_order.create",
			Version:     "1.0.0",
			Description: "创建采购订单——根据采购建议或手动指定，创建采购订单并提交给供应商",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "采购订单参数",
				Properties: map[string]*toolregistry.Schema{
					"supplier_id":      {Type: "string", Description: "供应商ID"},
					"items": {
						Type:        "array",
						Description: "采购商品列表",
						Items: &toolregistry.Schema{
							Type:        "object",
							Description: "采购商品明细",
							Properties: map[string]*toolregistry.Schema{
								"sku":        {Type: "string", Description: "SKU编码"},
								"quantity":   {Type: "integer", Description: "采购数量"},
								"unit_price": {Type: "number", Description: "单价（可选，默认取供应商报价）"},
							},
						},
					},
					"expected_delivery": {Type: "string", Format: "date", Description: "预计到货日期"},
					"remark":            {Type: "string", Description: "备注（可选）"},
				},
				Required: []string{"supplier_id", "items"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "创建的采购订单详情，包含订单ID、状态、商品明细和时间信息",
			},
			RequiredPermissions: []string{"purchase:write:order"},
			RiskLevel:           toolregistry.RiskHigh,
			MaxDuration:         30 * time.Second,
		},
		{
			Name:        "purchase_order.approve",
			Version:     "1.0.0",
			Description: "审批采购订单——审批待处理的采购订单，批准后可进入采购执行流程",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "审批参数",
				Properties: map[string]*toolregistry.Schema{
					"order_id": {Type: "string", Description: "采购订单ID"},
					"action": {
						Type:        "string",
						Description: "审批动作：approve（批准）/reject（拒绝）",
						Enum:        []string{"approve", "reject"},
					},
					"remark": {Type: "string", Description: "审批意见（可选）"},
				},
				Required: []string{"order_id", "action"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "审批结果，包含订单ID、更新后的状态和时间",
			},
			RequiredPermissions: []string{"purchase:write:approve"},
			RiskLevel:           toolregistry.RiskCritical,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "purchase_order.list",
			Version:     "1.0.0",
			Description: "查询采购订单列表——按状态、供应商、日期等条件查询采购订单",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "采购订单查询参数",
				Properties: map[string]*toolregistry.Schema{
					"status":       {Type: "string", Description: "订单状态（draft/pending/approved/shipped/received/cancelled，可选）"},
					"supplier_id":  {Type: "string", Description: "供应商ID（可选）"},
					"start_date":   {Type: "string", Format: "date", Description: "开始日期（YYYY-MM-DD）"},
					"end_date":     {Type: "string", Format: "date", Description: "结束日期（YYYY-MM-DD）"},
					"page":         {Type: "integer", Description: "页码（默认1）"},
					"size":         {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "分页的采购订单列表，包含订单ID、供应商、状态、金额、日期等",
			},
			RequiredPermissions: []string{"purchase:read:order"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
	}
}
