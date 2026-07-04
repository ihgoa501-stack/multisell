package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func SupplierTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "supplier.query",
			Version:     "1.0.0",
			Description: "查询供应商——根据名称、ID或品类查询供应商信息，包括联系方式、合作状态、评分等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "供应商查询参数",
				Properties: map[string]*toolregistry.Schema{
					"supplier_id": {Type: "string", Description: "供应商ID（可选）"},
					"keyword":     {Type: "string", Description: "供应商名称关键词（可选）"},
					"category":    {Type: "string", Description: "主营品类（可选）"},
					"status":      {Type: "string", Description: "合作状态（active/inactive/blacklisted，可选）"},
					"page":        {Type: "integer", Description: "页码（默认1）"},
					"size":        {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "供应商列表或详情，包含供应商ID、名称、联系方式、合作状态、评分、主营品类等",
			},
			RequiredPermissions: []string{"supplier:read:supplier"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":      "ok",
					"message":     "stub: 供应商查询功能尚在实现中，将在后续版本上线",
					"supplier_id": input["supplier_id"],
				}, nil
			},
		},
		{
			Name:        "supplier.quote.compare",
			Version:     "1.0.0",
			Description: "比价——对同一SKU或商品在不同供应商之间的报价进行对比",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "比价参数",
				Properties: map[string]*toolregistry.Schema{
					"sku": {Type: "string", Description: "SKU编码"},
				},
				Required: []string{"sku"},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "供应商报价对比列表，按价格升序排列，包含供应商名称、报价、起订量、交货周期、评分等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"supplier:read:quote"},
			RiskLevel:           toolregistry.RiskMedium,
			MaxDuration:         15 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"status":  "ok",
					"message": "stub: 供应商比价功能尚在实现中，将在后续版本上线",
					"sku":     input["sku"],
				}, nil
			},
		},
	}
}
