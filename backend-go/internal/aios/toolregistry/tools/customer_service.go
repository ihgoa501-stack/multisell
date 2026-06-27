package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"gorm.io/gorm"
)

// ── Package-level state for aftersales tool handlers ────────────────

var aftersalesDB *gorm.DB

// SetAftersalesDB sets the database connection for aftersales tool handlers.
// Must be called during server initialization (e.g. from ai Setup).
func SetAftersalesDB(db *gorm.DB) {
	aftersalesDB = db
}

// ── Helpers ─────────────────────────────────────────────────────────

// uninitializedResponse returns a standard response when the DB is not set.
func uninitializedResponse() map[string]interface{} {
	return map[string]interface{}{
		"status":  "unavailable",
		"message": "售后模块未初始化，请稍后重试",
	}
}

// notFoundResponse returns a standard not-found envelope.
func notFoundResponse(entity, id string) map[string]interface{} {
	return map[string]interface{}{
		"status":  "not_found",
		"message": fmt.Sprintf("%s %s 未找到", entity, id),
	}
}

// decisionLabel returns a human-readable Chinese label for a tool decision.
func decisionLabel(d string) string {
	switch d {
	case "approve":
		return "批准退款"
	case "reject":
		return "驳回"
	case "partial":
		return "部分退款处理"
	default:
		return d
	}
}

// aftersalesOrderToMap converts an AfterSalesOrder to a flat map suitable
// for JSON tool responses.
func aftersalesOrderToMap(o *aftersales.AfterSalesOrder) map[string]interface{} {
	m := map[string]interface{}{
		"id":                o.ID,
		"order_id":          o.OrderID,
		"return_quantity":   o.ReturnQuantity,
		"reason":            o.Reason,
		"status":            o.Status,
		"refund_amount":     o.RefundAmount,
		"inspection_result": o.InspectionResult,
		"rejection_reason":  o.RejectionReason,
		"created_by":        o.CreatedBy,
		"approved_by":       o.ApprovedBy,
		"rejected_by":       o.RejectedBy,
		"received_by":       o.ReceivedBy,
		"refunded_by":       o.RefundedBy,
		"created_at":        o.CreatedAt.Format(time.RFC3339),
		"updated_at":        o.UpdatedAt.Format(time.RFC3339),
	}
	if o.ItemID != nil {
		m["item_id"] = *o.ItemID
	}
	if o.SkuID != nil {
		m["sku_id"] = *o.SkuID
	}
	if o.ApprovedAt != nil {
		m["approved_at"] = o.ApprovedAt.Format(time.RFC3339)
	}
	if o.RejectedAt != nil {
		m["rejected_at"] = o.RejectedAt.Format(time.RFC3339)
	}
	if o.ReceivedAt != nil {
		m["received_at"] = o.ReceivedAt.Format(time.RFC3339)
	}
	if o.RefundedAt != nil {
		m["refunded_at"] = o.RefundedAt.Format(time.RFC3339)
	}
	return m
}

// ── FAQ response templates ──────────────────────────────────────────

var csFaqResponses = map[string]string{
	"shipping":    "感谢您的咨询。您的订单正在正常处理中，预计 {eta} 天内送达。您可以登录查看最新物流状态。",
	"return":      "关于退换货，您可以在签收后 30 天内申请。请登录订单详情页提交退换货申请，我们将为您免费安排取件。",
	"size":        "该产品的尺寸信息已在商品详情页列出。如果您不确定哪个尺码适合您，请参考页面上的尺码对照表。",
	"promotion":   "当前促销活动已在商品页面显示，折扣会在结算时自动应用。",
	"product_use": "感谢您的咨询！关于产品的使用方式，您可以查看商品详情页的使用说明部分。如有其他问题请随时联系我们。",
	"default":     "感谢您的咨询。我们已收到您的消息，客服团队将在 24 小时内回复您。如需加急处理，请回复 URGENT。",
}

// High-risk keywords.
var csHighRiskKeywords = []string{
	"trademark", "lawsuit", "refund", "a-to-z", "chargeback",
	"投诉", "律师", "起诉", "退款", "索赔",
}

// FAQ keyword-to-intent mapping.
var csFaqKeywordMap = map[string]string{
	"where is my order": "shipping",
	"tracking":          "shipping",
	"物流":               "shipping",
	"return":            "return",
	"refund":            "return",
	"退货":               "return",
	"退款":               "return",
	"size":              "size",
	"尺码":               "size",
	"fit":               "size",
	"promotion":         "promotion",
	"coupon":            "promotion",
	"折扣":               "promotion",
	"how to use":        "product_use",
	"如何使用":             "product_use",
}

func classifyIntent(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	msg := strings.ToLower(safeString(input["message"], ""))

	for _, kw := range csHighRiskKeywords {
		if strings.Contains(msg, kw) {
			return map[string]interface{}{
				"intent":     "high_risk",
				"confidence": 0.95,
				"action":     "escalate_human",
			}, nil
		}
	}
	for kw, intent := range csFaqKeywordMap {
		if strings.Contains(msg, kw) {
			return map[string]interface{}{
				"intent":     intent,
				"confidence": 0.90,
				"action":     "auto_reply",
			}, nil
		}
	}
	return map[string]interface{}{
		"intent":     "unknown",
		"confidence": 0.50,
		"action":     "escalate_human",
	}, nil
}

func autoReplyCS(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	msg := strings.ToLower(safeString(input["message"], ""))
	lang := safeString(input["language"], "en")
	eta := safeString(extractNested(input, "order_context.estimated_delivery_days"), "5-7")

	if msg == "" || lang == "" {
		return nil, fmt.Errorf("customer_service.auto_reply: message and language are required")
	}

	classifyResult, err := classifyIntent(ctx, input)
	if err != nil {
		return nil, err
	}
	cr := classifyResult.(map[string]interface{})
	intent := safeString(cr["intent"], "default")
	action := safeString(cr["action"], "escalate_human")
	conf, _ := cr["confidence"].(float64)

	if action == "escalate_human" {
		return map[string]interface{}{
			"auto_reply":      nil,
			"action":          "escalate",
			"intent":          intent,
			"confidence":      conf,
			"reason":          "高风险或低置信度，需人工处理",
			"suggested_reply": nil,
		}, nil
	}

	template := csFaqResponses[intent]
	if template == "" {
		template = csFaqResponses["default"]
	}
	reply := strings.ReplaceAll(template, "{eta}", eta)

	return map[string]interface{}{
		"auto_reply": reply,
		"action":     "auto_reply",
		"intent":     intent,
		"confidence": conf,
		"language":   lang,
	}, nil
}

// extractNested reads "a.b.c" from a nested map.
func extractNested(m map[string]interface{}, path string) string {
	if m == nil {
		return ""
	}
	parts := strings.SplitN(path, ".", 2)
	v := m[parts[0]]
	if v == nil {
		return ""
	}
	if len(parts) == 1 {
		s, _ := v.(string)
		return s
	}
	if sub, ok := v.(map[string]interface{}); ok {
		return extractNested(sub, parts[1])
	}
	return ""
}

// CustomerServiceTools returns tools for the customer-service / aftersales domain.
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if aftersalesDB == nil {
					return uninitializedResponse(), nil
				}
				asoIDStr := safeString(input["aftersale_id"])
				orderIDStr := safeString(input["order_id"])
				if asoIDStr == "" && orderIDStr == "" {
					return nil, fmt.Errorf("aftersales.order.read: aftersale_id or order_id is required")
				}

				if asoIDStr != "" {
					id, err := strconv.ParseInt(asoIDStr, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("aftersales.order.read: invalid aftersale_id: %s", asoIDStr)
					}
					var result aftersales.AfterSalesOrder
					if err := aftersalesDB.First(&result, id).Error; err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							return notFoundResponse("aftersales_order", asoIDStr), nil
						}
						return nil, fmt.Errorf("aftersales.order.read: %w", err)
					}
					return map[string]interface{}{
						"status": "success",
						"item":   aftersalesOrderToMap(&result),
					}, nil
				}

				// Query by order_id (may return multiple).
				oid, err := strconv.ParseInt(orderIDStr, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("aftersales.order.read: invalid order_id: %s", orderIDStr)
				}
				var results []aftersales.AfterSalesOrder
				if err := aftersalesDB.Where("order_id = ?", oid).Find(&results).Error; err != nil {
					return nil, fmt.Errorf("aftersales.order.read: %w", err)
				}
				items := make([]map[string]interface{}, len(results))
				for i := range results {
					items[i] = aftersalesOrderToMap(&results[i])
				}
				return map[string]interface{}{
					"status": "success",
					"count":  len(items),
					"items":  items,
				}, nil
			},
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
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if aftersalesDB == nil {
					return uninitializedResponse(), nil
				}
				asoIDStr := safeString(input["aftersale_id"])
				decision := safeString(input["decision"])
				if asoIDStr == "" {
					return nil, fmt.Errorf("aftersales.refund.decide: aftersale_id is required")
				}
				if decision == "" {
					return nil, fmt.Errorf("aftersales.refund.decide: decision is required")
				}

				asoID, err := strconv.ParseInt(asoIDStr, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("aftersales.refund.decide: invalid aftersale_id: %s", asoIDStr)
				}

				var aso aftersales.AfterSalesOrder
				if err := aftersalesDB.First(&aso, asoID).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return notFoundResponse("aftersales_order", asoIDStr), nil
					}
					return nil, fmt.Errorf("aftersales.refund.decide: %w", err)
				}

				// Prevent action on terminal statuses.
				if aso.Status == "refunded" || aso.Status == "rejected" {
					return map[string]interface{}{
						"status":       "terminal",
						"aftersale_id": aso.ID,
						"decision":     decision,
						"current_status": aso.Status,
						"message":      fmt.Sprintf("售后单 %d 已是终态(%s)，无法处理", aso.ID, aso.Status),
					}, nil
				}

				now := time.Now()

				switch decision {
				case "approve":
					updates := map[string]interface{}{
						"status":            "approved",
						"approved_by":       "system",
						"approved_at":       &now,
						"inspection_result": safeString(input["reason"], "AI自动审核通过"),
					}
					if err := aftersalesDB.Model(&aso).Updates(updates).Error; err != nil {
						return nil, fmt.Errorf("aftersales.refund.decide: approve failed: %w", err)
					}
				case "reject":
					updates := map[string]interface{}{
						"status":           "rejected",
						"rejected_by":      "system",
						"rejected_at":      &now,
						"rejection_reason": safeString(input["reason"], "AI自动审核驳回"),
					}
					if err := aftersalesDB.Model(&aso).Updates(updates).Error; err != nil {
						return nil, fmt.Errorf("aftersales.refund.decide: reject failed: %w", err)
					}
				case "partial":
					refundAmt := safeFloat(input["refund_amount"], 0)
					updates := map[string]interface{}{
						"refund_amount": refundAmt,
					}
					if err := aftersalesDB.Model(&aso).Updates(updates).Error; err != nil {
						return nil, fmt.Errorf("aftersales.refund.decide: partial update failed: %w", err)
					}
				default:
					return nil, fmt.Errorf("aftersales.refund.decide: unknown decision %q (must be approve/reject/partial)", decision)
				}

				// Reload after update.
				aftersalesDB.First(&aso, asoID)

				return map[string]interface{}{
					"status":        "completed",
					"aftersale_id":  aso.ID,
					"decision":      decision,
					"new_status":    aso.Status,
					"refund_amount": aso.RefundAmount,
					"message":       fmt.Sprintf("售后单 %d 已%s", aso.ID, decisionLabel(decision)),
				}, nil
			},
		},
		{
			Name:        "aftersales.dispute.query",
			Version:     "1.0.0",
			Description: "纠纷查询——查询平台纠纷/争议记录，包括纠纷类型、状态、举证信息等",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "纠纷查询参数",
				Properties: map[string]*toolregistry.Schema{
					"dispute_id": {Type: "string", Description: "纠纷ID（可选）"},
					"order_id":   {Type: "string", Description: "关联订单ID（可选）"},
					"status":     {Type: "string", Description: "纠纷状态（可选）"},
					"page":       {Type: "integer", Description: "页码（默认1）"},
					"size":       {Type: "integer", Description: "每页数量（默认20）"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "纠纷列表（或详情），包含纠纷ID、类型、状态、关联订单、金额、时间等",
			},
			RequiredPermissions: []string{"aftersales:read:dispute"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				if aftersalesDB == nil {
					return uninitializedResponse(), nil
				}

				q := aftersalesDB.Model(&aftersales.AfterSalesOrder{})

				if didStr := safeString(input["dispute_id"]); didStr != "" {
					if did, err := strconv.ParseInt(didStr, 10, 64); err == nil {
						q = q.Where("id = ?", did)
					}
				}
				if oidStr := safeString(input["order_id"]); oidStr != "" {
					if oid, err := strconv.ParseInt(oidStr, 10, 64); err == nil {
						q = q.Where("order_id = ?", oid)
					}
				}
				if status := safeString(input["status"]); status != "" {
					q = q.Where("status = ?", status)
				}

				// Filter by dispute-related keywords in the reason field.
				disputeKeywords := []string{"dispute", "争议", "AtoZ", "claim", "投诉", "纠纷"}
				likeClauses := make([]string, 0, len(disputeKeywords))
				args := make([]interface{}, 0)
				for _, kw := range disputeKeywords {
					likeClauses = append(likeClauses, "LOWER(reason) LIKE LOWER(?)")
					args = append(args, "%"+kw+"%")
				}
				whereOr := strings.Join(likeClauses, " OR ")
				q = q.Where("("+whereOr+") OR status = ?", append(args, "dispute")...)

				// Pagination.
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
					return nil, fmt.Errorf("aftersales.dispute.query: count failed: %w", err)
				}

				var results []aftersales.AfterSalesOrder
				if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&results).Error; err != nil {
					return nil, fmt.Errorf("aftersales.dispute.query: query failed: %w", err)
				}

				items := make([]map[string]interface{}, len(results))
				for i := range results {
					items[i] = aftersalesOrderToMap(&results[i])
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
			Name:        "customer_service.classify",
			Version:     "1.0.0",
			Description: "客服消息意图分类——通过关键词匹配识别消息意图",
			Parameters: &toolregistry.Schema{
				Type: "object",
				Properties: map[string]*toolregistry.Schema{
					"message":  {Type: "string", Description: "客户消息内容"},
					"language": {Type: "string", Description: "语言代码（zh/en）"},
				},
				Required: []string{"message"},
			},
			Handler:             classifyIntent,
			RequiredPermissions: []string{"customer_service:read:classify"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         3 * time.Second,
		},
		{
			Name:        "customer_service.auto_reply",
			Version:     "1.0.0",
			Description: "客服自动回复——分类消息意图后生成合适的自动回复文本或升级建议",
			Parameters: &toolregistry.Schema{
				Type: "object",
				Properties: map[string]*toolregistry.Schema{
					"message":      {Type: "string", Description: "客户消息内容"},
					"language":     {Type: "string", Description: "语言代码（zh/en）"},
					"order_context": {Type: "object", Description: "订单上下文（可选，含estimated_delivery_days）"},
				},
				Required: []string{"message", "language"},
			},
			Handler:             autoReplyCS,
			RequiredPermissions: []string{"customer_service:read:auto_reply"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         5 * time.Second,
		},
	}
}
