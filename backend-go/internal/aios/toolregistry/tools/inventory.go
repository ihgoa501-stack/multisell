package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"gorm.io/gorm"
)

// ── Package-level state for inventory tool handlers ──────────────────

var inventoryDB *gorm.DB
var rollbackGuard *guardrails.RollbackGuard

// SetInventoryDB sets the database connection for inventory tool handlers.
// Must be called during server initialization.
func SetInventoryDB(db *gorm.DB) {
	inventoryDB = db
}

// SetRollbackGuard sets the rollback guard for inventory tool handlers.
// Tool handlers that perform mutating operations can record rollback entries
// for compensatable actions. Must be called during server initialization.
func SetRollbackGuard(rg *guardrails.RollbackGuard) {
	rollbackGuard = rg
}

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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if inventoryDB == nil {
					return uninitializedResponse(), nil
				}
				q := inventoryDB.Model(&inventory.Inventory{})
				if skuStr := safeString(input["sku"]); skuStr != "" {
					if skuID, err := strconv.ParseInt(skuStr, 10, 64); err == nil {
						q = q.Where("sku_id = ?", skuID)
					}
				}
				if warehouse := safeString(input["warehouse_id"]); warehouse != "" {
					q = q.Where("warehouse = ?", warehouse)
				}
				var items []inventory.Inventory
				if err := q.Find(&items).Error; err != nil {
					return nil, fmt.Errorf("inventory.read: %w", err)
				}
				results := make([]map[string]interface{}, len(items))
				for i, item := range items {
					results[i] = map[string]interface{}{
						"id":              item.ID,
						"sku_id":          item.SkuID,
						"warehouse":       item.Warehouse,
						"location":        item.Location,
						"quantity":        item.Quantity,
						"locked_quantity": item.LockedQuantity,
						"safety_stock":    item.SafetyStock,
						"created_at":      item.CreatedAt.Format(time.RFC3339),
						"updated_at":      item.UpdatedAt.Format(time.RFC3339),
					}
				}
				return map[string]interface{}{
					"status": "success",
					"total":  len(results),
					"items":  results,
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if inventoryDB == nil {
					return uninitializedResponse(), nil
				}
				q := inventoryDB.Model(&inventory.InventoryAlertRule{})

				if skuStr := safeString(input["sku"]); skuStr != "" {
					if skuID, err := strconv.ParseInt(skuStr, 10, 64); err == nil {
						q = q.Where("sku_id = ?", skuID)
					}
				}

				page := int(safeFloat(input["page"], 1))
				size := int(safeFloat(input["size"], 20))
				if page < 1 {
					page = 1
				}
				if size < 1 || size > 100 {
					size = 20
				}
				offset := (page - 1) * size

				var total int64
				if err := q.Count(&total).Error; err != nil {
					return nil, fmt.Errorf("inventory.alert.list: count failed: %w", err)
				}

				var rules []inventory.InventoryAlertRule
				if err := q.Offset(offset).Limit(size).Order("id DESC").Find(&rules).Error; err != nil {
					return nil, fmt.Errorf("inventory.alert.list: %w", err)
				}

				items := make([]map[string]interface{}, len(rules))
				for i, r := range rules {
					items[i] = map[string]interface{}{
						"id":             r.ID,
						"sku_id":         r.SkuID,
						"min_level":      r.MinLevel,
						"max_level":      r.MaxLevel,
						"lead_time_days": r.LeadTimeDays,
						"enabled":        r.Enabled,
						"created_at":     r.CreatedAt.Format(time.RFC3339),
						"updated_at":     r.UpdatedAt.Format(time.RFC3339),
					}
				}

				return map[string]interface{}{
					"status": "success",
					"total":  total,
					"page":   page,
					"size":   size,
					"items":  items,
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if inventoryDB == nil {
					return uninitializedResponse(), nil
				}

				skuStr := safeString(input["sku"])
				if skuStr == "" {
					return nil, fmt.Errorf("inventory.alert.create: sku is required")
				}
				skuID, err := strconv.ParseInt(skuStr, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("inventory.alert.create: invalid sku: %s", skuStr)
				}

				rule := inventory.InventoryAlertRule{
					SkuID:    skuID,
					MinLevel: int(safeFloat(input["min_stock"], 0)),
					MaxLevel: int(safeFloat(input["max_stock"], 0)),
					Enabled:  true,
				}

				if err := inventoryDB.Create(&rule).Error; err != nil {
					return nil, fmt.Errorf("inventory.alert.create: %w", err)
				}

				return map[string]interface{}{
					"status":    "created",
					"rule_id":   rule.ID,
					"sku_id":    rule.SkuID,
					"min_level": rule.MinLevel,
					"max_level": rule.MaxLevel,
					"enabled":   rule.Enabled,
					"message":   fmt.Sprintf("SKU %d 的库存预警规则已创建", skuID),
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if inventoryDB == nil {
					return uninitializedResponse(), nil
				}

				from := safeString(input["from_warehouse"])
				to := safeString(input["to_warehouse"])
				if from == "" || to == "" {
					return nil, fmt.Errorf("inventory.transfer.create: from_warehouse and to_warehouse are required")
				}

				// items is an array, each item has sku and quantity.
				// The model maps to a single-sku transfer, so we create one record per item or take the first.
				itemsRaw := input["items"]
				itemsList, ok := itemsRaw.([]interface{})
				if !ok || len(itemsList) == 0 {
					return nil, fmt.Errorf("inventory.transfer.create: items must be a non-empty array")
				}

				created := make([]map[string]interface{}, 0, len(itemsList))
				for idx, itemRaw := range itemsList {
					item, ok := itemRaw.(map[string]interface{})
					if !ok {
						continue
					}
					skuStr := safeString(item["sku"])
					qty := int(safeFloat(item["quantity"], 0))
					if skuStr == "" || qty <= 0 {
						continue
					}
					skuID, err := strconv.ParseInt(skuStr, 10, 64)
					if err != nil {
						continue
					}

					t := inventory.InventoryTransfer{
						FromWarehouse: from,
						ToWarehouse:   to,
						SkuID:         skuID,
						Quantity:      qty,
						Status:        "draft",
					}
					if err := inventoryDB.Create(&t).Error; err != nil {
						return nil, fmt.Errorf("inventory.transfer.create: item %d failed: %w", idx, err)
					}
					created = append(created, map[string]interface{}{
						"id":             t.ID,
						"sku_id":         t.SkuID,
						"quantity":       t.Quantity,
						"status":         t.Status,
						"from_warehouse": t.FromWarehouse,
						"to_warehouse":   t.ToWarehouse,
					})
				}

				// Register a compensatable rollback entry for the transfer.
				if rollbackGuard != nil {
					rollbackGuard.Record(guardrails.RollbackEntry{
						ActionID:   fmt.Sprintf("transfer-%d", time.Now().UnixNano()),
						ActionType: "inventory.transfer",
						OriginalState: map[string]interface{}{
							"from_warehouse": input["from_warehouse"],
							"to_warehouse":   input["to_warehouse"],
							"items":          input["items"],
						},
					})
				}

				return map[string]interface{}{
					"status":    "created",
					"count":     len(created),
					"transfers": created,
					"message":   fmt.Sprintf("已创建 %d 条调拨记录", len(created)),
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if inventoryDB == nil {
					return uninitializedResponse(), nil
				}

				q := inventoryDB.Model(&inventory.InventoryTransfer{})

				if status := safeString(input["status"]); status != "" {
					q = q.Where("status = ?", status)
				}
				if startDate := safeString(input["start_date"]); startDate != "" {
					q = q.Where("created_at >= ?", startDate)
				}
				if endDate := safeString(input["end_date"]); endDate != "" {
					q = q.Where("created_at <= ?", endDate+" 23:59:59")
				}

				page := int(safeFloat(input["page"], 1))
				size := int(safeFloat(input["size"], 20))
				if page < 1 {
					page = 1
				}
				if size < 1 || size > 100 {
					size = 20
				}
				offset := (page - 1) * size

				var total int64
				if err := q.Count(&total).Error; err != nil {
					return nil, fmt.Errorf("inventory.transfer.list: count failed: %w", err)
				}

				var ts []inventory.InventoryTransfer
				if err := q.Offset(offset).Limit(size).Order("id DESC").Find(&ts).Error; err != nil {
					return nil, fmt.Errorf("inventory.transfer.list: %w", err)
				}

				items := make([]map[string]interface{}, len(ts))
				for i, t := range ts {
					m := map[string]interface{}{
						"id":             t.ID,
						"from_warehouse": t.FromWarehouse,
						"to_warehouse":   t.ToWarehouse,
						"sku_id":         t.SkuID,
						"quantity":       t.Quantity,
						"status":         t.Status,
						"note":           t.Note,
						"created_at":     t.CreatedAt.Format(time.RFC3339),
						"updated_at":     t.UpdatedAt.Format(time.RFC3339),
					}
					if t.Carrier != "" {
						m["carrier"] = t.Carrier
					}
					if t.TrackingNo != "" {
						m["tracking_no"] = t.TrackingNo
					}
					if t.EstimatedArrival != nil {
						m["estimated_arrival"] = t.EstimatedArrival.Format(time.RFC3339)
					}
					if t.CompletedAt != nil {
						m["completed_at"] = t.CompletedAt.Format(time.RFC3339)
					}
					items[i] = m
				}

				return map[string]interface{}{
					"status": "success",
					"total":  total,
					"page":   page,
					"size":   size,
					"items":  items,
				}, nil
			},
		},
	}
}
