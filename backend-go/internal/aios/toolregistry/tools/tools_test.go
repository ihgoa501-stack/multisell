package tools

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Sourcing tool handler tests (direct handler invocation)
// ---------------------------------------------------------------------------

func TestSourcingRecommend_Viable(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url":  "https://detail.1688.com/offer/test.html",
		"price_1688":  50.0,
		"weight_kg":   0.1,
		"destination": "US",
		"markup_pct":  250.0,
	}
	result, err := SourcingTools()[0].Handler(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] != "viable" {
		t.Errorf("expected status 'viable', got %v", m["status"])
	}
	if m["action"] != "escalate_to_optimizer" {
		t.Errorf("expected action 'escalate_to_optimizer', got %v", m["action"])
	}
}

func TestSourcingRecommend_Marginal(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url":  "https://detail.1688.com/offer/test.html",
		"price_1688":  50.0,
		"weight_kg":   0.1,
		"destination": "US",
		"markup_pct":  140.0,
	}
	result, err := SourcingTools()[0].Handler(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] != "marginal" {
		t.Errorf("expected status 'marginal', got %v", m["status"])
	}
	if m["action"] != "review" {
		t.Errorf("expected action 'review', got %v", m["action"])
	}
}

func TestSourcingRecommend_Unviable(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url":  "https://detail.1688.com/offer/test.html",
		"price_1688":  50.0,
		"weight_kg":   0.1,
		"destination": "US",
		"markup_pct":  130.0,
	}
	result, err := SourcingTools()[0].Handler(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] != "unviable" {
		t.Errorf("expected status 'unviable', got %v", m["status"])
	}
	if m["action"] != "discard" {
		t.Errorf("expected action 'discard', got %v", m["action"])
	}
}

func TestSourcingRecommend_ZeroPrice(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url":  "https://detail.1688.com/offer/test.html",
		"price_1688":  0.0,
		"weight_kg":   0.5,
		"destination": "US",
	}
	_, err := SourcingTools()[0].Handler(ctx, input)
	if err == nil {
		t.Fatal("expected error for zero price, got nil")
	}
}

func TestSourcingRecommend_DestinationDefaults(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url": "https://detail.1688.com/offer/test.html",
		"price_1688": 50.0,
		"weight_kg":  0.1,
	}
	result, err := SourcingTools()[0].Handler(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] == "" {
		t.Error("expected non-empty status with default destination")
	}
}

// ---------------------------------------------------------------------------
// Sourcing recommend via ToolRegistry integration
// ---------------------------------------------------------------------------

func TestSourcingRecommend_ViaRegistry(t *testing.T) {
	ctx := context.Background()
	input := map[string]interface{}{
		"source_url":  "https://detail.1688.com/offer/test.html",
		"price_1688":  50.0,
		"weight_kg":   0.1,
		"destination": "US",
		"markup_pct":  250.0,
	}
	result, err := callToolViaRegistry(ctx, "sourcing.recommend", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "viable" {
		t.Errorf("expected status 'viable', got %v", result["status"])
	}
}

// ---------------------------------------------------------------------------
// Customer service: classifyIntent tests
// ---------------------------------------------------------------------------

func TestClassifyIntent_HighRiskEnglish(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "I want a refund for my order",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "high_risk" {
		t.Errorf("expected 'high_risk', got %v", m["intent"])
	}
	if m["action"] != "escalate_human" {
		t.Errorf("expected 'escalate_human', got %v", m["action"])
	}
}

func TestClassifyIntent_HighRiskChinese(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "我要投诉你们的产品质量问题",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "high_risk" {
		t.Errorf("expected 'high_risk', got %v", m["intent"])
	}
}

func TestClassifyIntent_AllHighRiskKeywords(t *testing.T) {
	keywords := []string{"trademark", "lawsuit", "refund", "a-to-z", "chargeback"}
	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			result, err := classifyIntent(context.Background(), map[string]interface{}{
				"message": "this is about " + kw,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			m := result.(map[string]interface{})
			if m["intent"] != "high_risk" {
				t.Errorf("for keyword %q: expected 'high_risk', got %v", kw, m["intent"])
			}
		})
	}
}

func TestClassifyIntent_FAQShipping(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "where is my order",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "shipping" {
		t.Errorf("expected 'shipping', got %v", m["intent"])
	}
	if m["action"] != "auto_reply" {
		t.Errorf("expected 'auto_reply', got %v", m["action"])
	}
}

func TestClassifyIntent_FAQReturn(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "I want to return my item",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "return" {
		t.Errorf("expected 'return', got %v", m["intent"])
	}
}

func TestClassifyIntent_FAQChinese(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "怎么查物流",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "shipping" {
		t.Errorf("expected 'shipping', got %v", m["intent"])
	}
}

func TestClassifyIntent_Unknown(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "hello world this is a test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "unknown" {
		t.Errorf("expected 'unknown', got %v", m["intent"])
	}
	if m["action"] != "escalate_human" {
		t.Errorf("expected 'escalate_human', got %v", m["action"])
	}
}

func TestClassifyIntent_EmptyMessage(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "unknown" {
		t.Errorf("expected 'unknown' for empty message, got %v", m["intent"])
	}
}

func TestClassifyIntent_HighRiskTakesPrecedence(t *testing.T) {
	result, err := classifyIntent(context.Background(), map[string]interface{}{
		"message": "I want a refund for my order — where is my order",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["intent"] != "high_risk" {
		t.Errorf("expected 'high_risk' (precedence over 'shipping'), got %v", m["intent"])
	}
}

// ---------------------------------------------------------------------------
// Customer service: autoReplyCS tests
// ---------------------------------------------------------------------------

func TestAutoReplyCS_HighRisk(t *testing.T) {
	ctx := context.Background()
	result, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "I want a refund",
		"language": "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["action"] != "escalate" {
		t.Errorf("expected action 'escalate', got %v", m["action"])
	}
	if m["auto_reply"] != nil {
		t.Errorf("expected nil auto_reply for high-risk, got %v", m["auto_reply"])
	}
	if m["intent"] != "high_risk" {
		t.Errorf("expected intent 'high_risk', got %v", m["intent"])
	}
}

func TestAutoReplyCS_ShippingWithDefaultETA(t *testing.T) {
	ctx := context.Background()
	result, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "查询物流信息",
		"language": "zh",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["action"] != "auto_reply" {
		t.Errorf("expected action 'auto_reply', got %v", m["action"])
	}
	reply, ok := m["auto_reply"].(string)
	if !ok {
		t.Fatalf("expected string auto_reply, got %T", m["auto_reply"])
	}
	if reply == "" {
		t.Fatal("expected non-empty auto_reply")
	}
	if m["intent"] != "shipping" {
		t.Errorf("expected intent 'shipping', got %v", m["intent"])
	}
}

func TestAutoReplyCS_ShippingWithCustomETA(t *testing.T) {
	ctx := context.Background()
	result, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "where is my order",
		"language": "en",
		"order_context": map[string]interface{}{
			"estimated_delivery_days": "3-5",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	reply, ok := m["auto_reply"].(string)
	if !ok {
		t.Fatalf("expected string auto_reply, got %T", m["auto_reply"])
	}
	if m["intent"] != "shipping" {
		t.Errorf("expected intent 'shipping', got %v", m["intent"])
	}
	_ = reply
}

func TestAutoReplyCS_Return(t *testing.T) {
	ctx := context.Background()
	result, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "I need to return an item",
		"language": "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["action"] != "auto_reply" {
		t.Errorf("expected action 'auto_reply', got %v", m["action"])
	}
	reply, ok := m["auto_reply"].(string)
	if !ok {
		t.Fatalf("expected string auto_reply, got %T", m["auto_reply"])
	}
	if reply == "" {
		t.Fatal("expected non-empty auto_reply")
	}
}

func TestAutoReplyCS_Unknown(t *testing.T) {
	ctx := context.Background()
	result, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "I have a random question",
		"language": "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["action"] != "escalate" {
		t.Errorf("expected action 'escalate' for unknown intent, got %v", m["action"])
	}
}

func TestAutoReplyCS_EmptyMessage(t *testing.T) {
	ctx := context.Background()
	_, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "",
		"language": "en",
	})
	if err == nil {
		t.Fatal("expected error for empty message")
	}
}

func TestAutoReplyCS_EmptyLanguage(t *testing.T) {
	ctx := context.Background()
	_, err := autoReplyCS(ctx, map[string]interface{}{
		"message":  "hello",
		"language": "",
	})
	if err == nil {
		t.Fatal("expected error for empty language")
	}
}

// ---------------------------------------------------------------------------
// extractNested tests
// ---------------------------------------------------------------------------

func TestExtractNested_Basic(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
	}
	if got := extractNested(m, "key"); got != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

func TestExtractNested_Nested(t *testing.T) {
	m := map[string]interface{}{
		"order_context": map[string]interface{}{
			"estimated_delivery_days": "3-5",
		},
	}
	if got := extractNested(m, "order_context.estimated_delivery_days"); got != "3-5" {
		t.Errorf("expected '3-5', got %q", got)
	}
}

func TestExtractNested_NilMap(t *testing.T) {
	if got := extractNested(nil, "key"); got != "" {
		t.Errorf("expected empty string for nil map, got %q", got)
	}
}

func TestExtractNested_MissingKey(t *testing.T) {
	m := map[string]interface{}{"other": "value"}
	if got := extractNested(m, "nonexistent"); got != "" {
		t.Errorf("expected empty string for missing key, got %q", got)
	}
}

func TestExtractNested_DeepMissing(t *testing.T) {
	m := map[string]interface{}{
		"order_context": map[string]interface{}{
			"missing": "nope",
		},
	}
	if got := extractNested(m, "order_context.estimated_delivery_days"); got != "" {
		t.Errorf("expected empty string for deep missing key, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func callToolViaRegistry(ctx context.Context, name string, input map[string]interface{}) (map[string]interface{}, error) {
	reg := toolregistry.NewToolRegistry(zap.NewNop())
	for _, t := range SourcingTools() {
		reg.Register(&t)
	}
	result, err := reg.Call(ctx, name, input)
	if err != nil {
		return nil, err
	}
	out, ok := result.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	return out, nil
}
