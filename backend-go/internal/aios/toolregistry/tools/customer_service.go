package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// FAQ response templates.
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
