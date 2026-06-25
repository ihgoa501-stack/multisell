package tools

import (
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

func ListingTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "listing.read",
			Version:     "1.0.0",
			Description: "查询Listing——查询商品Listing详情，包括标题、描述、图片、价格、规格等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "Listing查询参数",
				Properties: map[string]*toolregistry.Schema{
					"listing_id": {Type: "string", Description: "Listing ID"},
					"sku":        {Type: "string", Description: "SKU编码（与listing_id二选一）"},
					"platform":   {Type: "string", Description: "平台（可选，如ozon/shopee）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "Listing详情，包含标题、描述、图片URL、价格、库存、规格参数等",
			},
			RequiredPermissions: []string{"listing:read:listing"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "listing.optimize_suggest",
			Version:     "1.0.0",
			Description: "Listing优化建议——根据平台最佳实践和竞品数据，生成Listing标题、描述、关键词等优化建议",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "优化建议参数",
				Properties: map[string]*toolregistry.Schema{
					"listing_id": {Type: "string", Description: "Listing ID"},
					"aspects": {
						Type:        "array",
						Description: "需要优化的方面（可选，不传则全部分析）",
						Items:       &toolregistry.Schema{Type: "string", Description: "title/description/keywords/images"},
					},
				},
				Required: []string{"listing_id"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "优化建议，包含各优化方面的当前状态、建议改进项和预期效果",
			},
			RequiredPermissions: []string{"listing:read:optimize"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         30 * time.Second,
		},
		{
			Name:        "sku.read",
			Version:     "1.0.0",
			Description: "SKU查询——查询SKU信息，包括规格、供应商、成本价、条码等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "SKU查询参数",
				Properties: map[string]*toolregistry.Schema{
					"sku": {Type: "string", Description: "SKU编码"},
				},
				Required: []string{"sku"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "SKU详情，包含SKU编码、名称、规格、成本价、供应商、条码等",
			},
			RequiredPermissions: []string{"listing:read:sku"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
		{
			Name:        "category.list",
			Version:     "1.0.0",
			Description: "分类列表——查询平台商品分类树，支持按父分类逐级查找",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "分类查询参数",
				Properties: map[string]*toolregistry.Schema{
					"platform":      {Type: "string", Description: "平台名称（ozon/shopee等）"},
					"parent_id":     {Type: "string", Description: "父分类ID（可选，不传则返回顶级分类）"},
					"keyword":       {Type: "string", Description: "关键词搜索（可选）"},
				},
				Required: []string{"platform"},
			},
			Returns: &toolregistry.Schema{
				Type:        "array",
				Description: "分类列表，包含分类ID、名称、父分类ID、是否叶子节点、子分类数等",
				Items:       &toolregistry.Schema{Type: "object"},
			},
			RequiredPermissions: []string{"listing:read:category"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
		},
	}
}
