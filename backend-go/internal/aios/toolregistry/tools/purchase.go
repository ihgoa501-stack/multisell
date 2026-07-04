package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/purchase"
	"gorm.io/gorm"
)

// ── Package-level state for purchase tool handlers ──────────────────

var purchaseDB *gorm.DB

// SetPurchaseDB sets the database connection for purchase tool handlers.
// Must be called during server initialization.
func SetPurchaseDB(db *gorm.DB) {
	purchaseDB = db
}

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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if purchaseDB == nil {
					return uninitializedResponse(), nil
				}

				// Return existing purchase suggestions from the database.
				q := purchaseDB.Model(&purchase.PurchaseSuggestion{})

				if skusRaw := input["skus"]; skusRaw != nil {
					if skusList, ok := skusRaw.([]interface{}); ok && len(skusList) > 0 {
						skuIDs := make([]int64, 0, len(skusList))
						for _, s := range skusList {
							if id, err := strconv.ParseInt(fmt.Sprint(s), 10, 64); err == nil {
								skuIDs = append(skuIDs, id)
							}
						}
						if len(skuIDs) > 0 {
							q = q.Where("sku_id IN ?", skuIDs)
						}
					}
				}

				var suggestions []purchase.PurchaseSuggestion
				if err := q.Order("id DESC").Limit(50).Find(&suggestions).Error; err != nil {
					return nil, fmt.Errorf("purchase_order.suggest: %w", err)
				}

				items := make([]map[string]interface{}, len(suggestions))
				for i, s := range suggestions {
					items[i] = map[string]interface{}{
						"id":            s.ID,
						"sku_id":        s.SkuID,
						"suggested_qty": s.SuggestedQty,
						"reason":        s.Reason,
						"status":        s.Status,
						"generated_at":  s.GeneratedAt.Format(time.RFC3339),
					}
				}

				return map[string]interface{}{
					"status": "success",
					"total":  len(items),
					"items":  items,
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if purchaseDB == nil {
					return uninitializedResponse(), nil
				}

				supplierStr := safeString(input["supplier_id"])
				if supplierStr == "" {
					return nil, fmt.Errorf("purchase_order.create: supplier_id is required")
				}
				supplierID, err := strconv.ParseInt(supplierStr, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("purchase_order.create: invalid supplier_id: %s", supplierStr)
				}

				itemsRaw := input["items"]
				itemsList, ok := itemsRaw.([]interface{})
				if !ok || len(itemsList) == 0 {
					return nil, fmt.Errorf("purchase_order.create: items must be a non-empty array")
				}

				expectedDelivery := safeString(input["expected_delivery"])
				remark := safeString(input["remark"])

				now := time.Now()
				orderNo := fmt.Sprintf("PO-%d", now.UnixMilli())

				order := purchase.PurchaseOrder{
					OrderNo:          orderNo,
					SupplierID:       supplierID,
					Status:           purchase.StatusDraft,
					ExpectedDelivery: &expectedDelivery,
					Remark:           remark,
				}

				var orderItems []purchase.PurchaseOrderItem
				var totalAmount float64

				for _, itemRaw := range itemsList {
					item, ok := itemRaw.(map[string]interface{})
					if !ok {
						continue
					}
					skuStr := safeString(item["sku"])
					qty := int(safeFloat(item["quantity"], 0))
					unitPrice := safeFloat(item["unit_price"], 0)
					if skuStr == "" || qty <= 0 {
						continue
					}
					skuID, err := strconv.ParseInt(skuStr, 10, 64)
					if err != nil {
						continue
					}
					subtotal := float64(qty) * unitPrice
					totalAmount += subtotal
					orderItems = append(orderItems, purchase.PurchaseOrderItem{
						SkuID:     skuID,
						Quantity:  qty,
						UnitPrice: unitPrice,
						Subtotal:  subtotal,
					})
				}

				if len(orderItems) == 0 {
					return nil, fmt.Errorf("purchase_order.create: no valid items provided")
				}

				order.TotalAmount = totalAmount

				tx := purchaseDB.Begin()
				if err := tx.Create(&order).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("purchase_order.create: %w", err)
				}
				for i := range orderItems {
					orderItems[i].PurchaseOrderID = order.ID
				}
				if err := tx.Create(&orderItems).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("purchase_order.create: items failed: %w", err)
				}
				tx.Commit()

				itemMaps := make([]map[string]interface{}, len(orderItems))
				for i, oi := range orderItems {
					itemMaps[i] = map[string]interface{}{
						"sku_id":     oi.SkuID,
						"quantity":   oi.Quantity,
						"unit_price": oi.UnitPrice,
						"subtotal":   oi.Subtotal,
					}
				}

				return map[string]interface{}{
					"status":            "created",
					"order_id":          order.ID,
					"order_no":          order.OrderNo,
					"supplier_id":       order.SupplierID,
					"order_status":     order.Status,
					"total_amount":      order.TotalAmount,
					"expected_delivery": expectedDelivery,
					"remark":            order.Remark,
					"items":             itemMaps,
					"message":           fmt.Sprintf("采购订单 %s 已创建，共 %.2f", orderNo, totalAmount),
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if purchaseDB == nil {
					return uninitializedResponse(), nil
				}

				orderStr := safeString(input["order_id"])
				action := safeString(input["action"])
				remark := safeString(input["remark"])

				if orderStr == "" {
					return nil, fmt.Errorf("purchase_order.approve: order_id is required")
				}
				if action != "approve" && action != "reject" {
					return nil, fmt.Errorf("purchase_order.approve: action must be 'approve' or 'reject'")
				}

				orderID, err := strconv.ParseInt(orderStr, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("purchase_order.approve: invalid order_id: %s", orderStr)
				}

				var order purchase.PurchaseOrder
				if err := purchaseDB.First(&order, orderID).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						return notFoundResponse("purchase_order", orderStr), nil
					}
					return nil, fmt.Errorf("purchase_order.approve: %w", err)
				}

				newStatus := purchase.StatusApproved
				if action == "reject" {
					newStatus = purchase.StatusCancelled
				}

				if err := purchaseDB.Model(&order).Update("status", newStatus).Error; err != nil {
					return nil, fmt.Errorf("purchase_order.approve: %w", err)
				}

				return map[string]interface{}{
					"status":     "completed",
					"order_id":   order.ID,
					"order_no":   order.OrderNo,
					"action":     action,
					"new_status": newStatus,
					"remark":     remark,
					"message":    fmt.Sprintf("采购订单 %s 已%s", order.OrderNo, map[string]string{"approve": "批准", "reject": "驳回"}[action]),
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if purchaseDB == nil {
					return uninitializedResponse(), nil
				}

				q := purchaseDB.Model(&purchase.PurchaseOrder{})

				if status := safeString(input["status"]); status != "" {
					q = q.Where("status = ?", status)
				}
				if supplierStr := safeString(input["supplier_id"]); supplierStr != "" {
					if supplierID, err := strconv.ParseInt(supplierStr, 10, 64); err == nil {
						q = q.Where("supplier_id = ?", supplierID)
					}
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
					return nil, fmt.Errorf("purchase_order.list: count failed: %w", err)
				}

				var orders []purchase.PurchaseOrder
				if err := q.Offset(offset).Limit(size).Order("id DESC").Preload("Items").Find(&orders).Error; err != nil {
					return nil, fmt.Errorf("purchase_order.list: %w", err)
				}

				items := make([]map[string]interface{}, len(orders))
				for i, o := range orders {
					itemList := make([]map[string]interface{}, len(o.Items))
					for j, oi := range o.Items {
						itemList[j] = map[string]interface{}{
							"sku_id":     oi.SkuID,
							"quantity":   oi.Quantity,
							"received_qty": oi.ReceivedQty,
							"unit_price": oi.UnitPrice,
							"subtotal":   oi.Subtotal,
						}
					}
					m := map[string]interface{}{
						"id":          o.ID,
						"order_no":    o.OrderNo,
						"supplier_id": o.SupplierID,
						"status":      o.Status,
						"total_amount": o.TotalAmount,
						"remark":      o.Remark,
						"created_at":  o.CreatedAt.Format(time.RFC3339),
						"updated_at":  o.UpdatedAt.Format(time.RFC3339),
						"items":       itemList,
					}
					if o.ExpectedDelivery != nil {
						m["expected_delivery"] = *o.ExpectedDelivery
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
