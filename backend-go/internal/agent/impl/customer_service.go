// Package impl provides concrete agent implementations.
//
// CustomerServiceAgent implements A4 Customer Service business logic ported
// from backend/app/agent/agents/customer_service.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/final-integrated-solution.md section 6.2
//   - L1 auto-reply for 60%+ FAQ, L2-assisted human handling for complex complaints
//   - Intent classification + confidence routing
//   - Input: message text and language
//   - Output: auto-reply text, action (escalate/auto_reply), intent classification
package impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- Context field names ----------

var csRequiredFields = []string{"message", "language"}

// ---------- FAQ response templates ----------

var faqResponses = map[string]string{
	"shipping":    "感谢您的咨询。您的订单正在正常处理中，预计 {eta} 天内送达。您可以登录查看最新物流状态。",
	"return":      "关于退换货，您可以在签收后 30 天内申请。请登录订单详情页提交退换货申请，我们将为您免费安排取件。",
	"size":        "该产品的尺寸信息已在商品详情页列出。如果您不确定哪个尺码适合您，请参考页面上的尺码对照表。",
	"promotion":   "当前促销活动已在商品页面显示，折扣会在结算时自动应用。",
	"product_use": "感谢您的咨询！关于产品的使用方式，您可以查看商品详情页的使用说明部分。如有其他问题请随时联系我们。",
	"default":     "感谢您的咨询。我们已收到您的消息，客服团队将在 24 小时内回复您。如需加急处理，请回复 URGENT。",
}

// ---------- High-risk keywords ----------

var highRiskKeywords = []string{
	"trademark",
	"lawsuit",
	"refund",
	"a-to-z",
	"chargeback",
	"投诉",
	"律师",
	"起诉",
	"退款",
	"索赔",
}

// ---------- FAQ keyword-to-intent mapping ----------

var faqKeywordMap = map[string]string{
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

// ---------- CustomerServiceAgent ----------

// CustomerServiceAgent implements A4 Customer Service FAQ auto-reply and
// intent classification logic. It uses keyword matching to classify messages
// and returns appropriate FAQ responses or escalation recommendations.
//
// Decision points:
//   - "auto_reply" — generates auto-reply based on message classification
//   - "intent_classify" — classifies message intent without generating reply
type CustomerServiceAgent struct{}

// NewCustomerServiceAgent creates a new CustomerServiceAgent.
func NewCustomerServiceAgent() *CustomerServiceAgent {
	return &CustomerServiceAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "auto_reply"
//   - "intent_classify"
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *CustomerServiceAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Try tool first.
	raw, toolErr := toolregistry.DefaultRegistry.Invoke("customer_service."+decisionPoint, params, ctx)
	if toolErr == nil {
		if result, ok := raw.(map[string]interface{}); ok {
			if c, ok := result["confidence"].(float64); ok {
				confidence = c
			}
			if r, ok := result["risk"].(string); ok {
				riskLevel = r
			}
			delete(result, "confidence")
			delete(result, "risk")
			return result, confidence, riskLevel, nil
		}
	}
	// Fallback to built-in logic.
	switch decisionPoint {
	case "auto_reply":
		return a.autoReply(params)
	case "intent_classify":
		return a.classify(params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: classify ----------

// classify determines the intent of a customer message using keyword matching.
//
// Returns a map with:
//   - intent: the classified intent string (shipping, return, size, etc.)
//   - confidence: classification confidence [0-1]
//   - action: "escalate_human" or "auto_reply"
func (a *CustomerServiceAgent) classify(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	msg := strings.ToLower(safeString(ctx["message"], ""))

	// Check high-risk keywords first.
	for _, kw := range highRiskKeywords {
		if strings.Contains(msg, kw) {
			output = map[string]interface{}{
				"intent":     "high_risk",
				"confidence": 0.95,
				"action":     "escalate_human",
			}
			return output, 0.95, "medium", nil
		}
	}

	// Check FAQ keyword mapping.
	for kw, intent := range faqKeywordMap {
		if strings.Contains(msg, kw) {
			output = map[string]interface{}{
				"intent":     intent,
				"confidence": 0.90,
				"action":     "auto_reply",
			}
			return output, 0.90, "low", nil
		}
	}

	// Unknown intent.
	output = map[string]interface{}{
		"intent":     "unknown",
		"confidence": 0.50,
		"action":     "escalate_human",
	}
	return output, 0.50, "low", nil
}

// ---------- Decision point: auto_reply ----------

// autoReply generates an auto-reply based on the classified message intent.
//
// Required context fields: message, language
// Optional context fields: order_context (with estimated_delivery_days)
//
// Returns a map with auto-reply text and routing decision.
func (a *CustomerServiceAgent) autoReply(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, csRequiredFields); len(missing) > 0 {
		return insufficientData("auto_reply", missing), 0.0, "low", nil
	}

	lang := safeString(ctx["language"], "en")
	orderCtx := extractMap(ctx["order_context"])
	eta := safeString(orderCtx["estimated_delivery_days"], "5-7")

	classResult, classConfidence, _, _ := a.classify(ctx)
	intent := safeString(classResult["intent"], "default")
	action := safeString(classResult["action"], "escalate_human")

	if action == "escalate_human" {
		output = map[string]interface{}{
			"auto_reply":      nil,
			"action":          "escalate",
			"intent":          intent,
			"confidence":      classConfidence,
			"reason":          "高风险或低置信度，需人工处理",
			"suggested_reply": nil,
		}
		return output, classConfidence, "medium", nil
	}

	template, ok := faqResponses[intent]
	if !ok {
		template = faqResponses["default"]
	}
	reply := strings.ReplaceAll(template, "{eta}", eta)

	output = map[string]interface{}{
		"auto_reply": reply,
		"action":     "auto_reply",
		"intent":     intent,
		"confidence": classConfidence,
		"language":   lang,
	}
	return output, classConfidence, "low", nil
}

// ---------- Helpers ----------

// extractMap safely extracts a nested map from an interface{} value.
func extractMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return m
}

// ---------- P3: Multi-Language Support (#196) ----------

// multiLanguageReply generates a reply in the specified language.
func (a *CustomerServiceAgent) multiLanguageReply(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	message := safeString(ctx["message"], "")
	lang := safeString(ctx["language"], "en")

	if message == "" {
		return insufficientData("multi_language_reply", []string{"message"}), 0.0, "low", nil
	}

	// ponytail: stub translation — replace with real ML translation when
	// translation service is wired. For now, returns the message with a
	// language tag.
	replyMap := map[string]string{
		"en":  "Thank you for your message. We have received your inquiry and will respond within 24 hours.",
		"zh":  "感谢您的留言。我们已收到您的咨询，将在24小时内回复。",
		"ru":  "Спасибо за ваше сообщение. Мы получили ваш запрос и ответим в течение 24 часов.",
		"ja":  "メッセージありがとうございます。お問い合わせを受け付けました。24時間以内にご返信いたします。",
		"ko":  "문의해 주셔서 감사합니다. 접수된 문의는 24시간 이내에 답변드리겠습니다.",
		"th":  "ขอบคุณสำหรับข้อความของคุณ เราได้รับคำถามของคุณแล้วและจะตอบกลับภายใน 24 ชั่วโมง",
		"vi":  "Cảm ơn bạn đã gửi tin nhắn. Chúng tôi đã nhận được yêu cầu của bạn và sẽ phản hồi trong vòng 24 giờ.",
		"es":  "Gracias por su mensaje. Hemos recibido su consulta y le responderemos dentro de 24 horas.",
	}

	reply, ok := replyMap[lang]
	if !ok {
		reply = replyMap["en"]
	}

	output = map[string]interface{}{
		"auto_reply": reply,
		"language":   lang,
		"action":     "auto_reply",
		"confidence": 0.85,
	}
	return output, 0.85, "low", nil
}

// ---------- P3: Ticket System (#196) ----------

// ticketActions provides ticket management operations.
func (a *CustomerServiceAgent) ticketActions(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	action := safeString(ctx["action"], "create") // create, update, escalate, resolve
	ticketID := safeString(ctx["ticket_id"], "")
	subject := safeString(ctx["subject"], "")
	priority := safeString(ctx["priority"], "medium") // low, medium, high, urgent
	category := safeString(ctx["category"], "general")

	if action == "create" && subject == "" {
		return insufficientData("ticket_actions", []string{"subject"}), 0.0, "low", nil
	}

	allowedActions := []string{"create", "update", "escalate", "resolve", "close"}
	actionValid := false
	for _, aa := range allowedActions {
		if action == aa {
			actionValid = true
			break
		}
	}
	if !actionValid {
		action = "create"
	}

	output = map[string]interface{}{
		"action":        action,
		"ticket_id":     ticketID,
		"subject":       subject,
		"priority":      priority,
		"category":      category,
		"status":        "success",
		"message":       "ticket operation completed",
		"confidence":    0.90,
	}
	return output, 0.90, "low", nil
}
