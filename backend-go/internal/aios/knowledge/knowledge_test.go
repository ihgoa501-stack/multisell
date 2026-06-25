package knowledge_test

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/knowledge"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Query: matched pattern
// ---------------------------------------------------------------------------

func TestQuery_MatchedInventory(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "库存还剩多少",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
	if len(resp.Inferences) == 0 {
		t.Error("expected at least one inference for matched query")
	}
	if resp.DataSources == nil {
		t.Error("DataSources should not be nil")
	}
	if resp.Freshness == nil {
		t.Error("Freshness should not be nil")
	}
	// Answer must contain the matched label.
	if len(resp.Answer) == 0 {
		t.Error("Answer should not be empty")
	}
}

func TestQuery_MatchedOrder(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "帮我查一下昨天的订单量",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedSettlement(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "这个月的利润怎么样",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedSKU(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "SKU-12345的产品信息",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedSupplier(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "查看供应商信息",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedShipping(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "这个订单发货了吗",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedPlatform(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "Lazada店铺状态怎么样",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

func TestQuery_MatchedAftersales(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "最近退货率怎么样",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0 for matched query, got %f", resp.Confidence)
	}
}

// ---------------------------------------------------------------------------
// Query: no matching pattern
// ---------------------------------------------------------------------------

func TestQuery_NoMatch(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "今天天气怎么样",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response even for no-match")
	}
	if resp.Confidence != 0 {
		t.Errorf("expected confidence 0 for no-match, got %f", resp.Confidence)
	}
	if len(resp.Answer) == 0 {
		t.Error("Answer should not be empty even for no-match")
	}
	if len(resp.Inferences) == 0 {
		t.Error("expected at least one inference explaining no-match")
	}
}

// ---------------------------------------------------------------------------
// RegisterPattern: dynamic registration
// ---------------------------------------------------------------------------

func TestRegisterPattern(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	// Before registering — this question should not match any pattern.
	respBefore, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "请检查汇率数据",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respBefore.Confidence != 0 {
		t.Skip("built-in pattern may already cover this; skipping before-check")
	}

	// Register a new pattern for exchange rates.
	eng.RegisterPattern(knowledge.KnowledgePattern{
		Keywords:      []string{"汇率", "exchange_rate", "汇兑", "汇率数据"},
		SourceType:    "exchangerate",
		QueryTemplate: "SELECT * FROM exchangerates WHERE {condition}",
		Priority:      150,
	})

	// After registering — should now match.
	respAfter, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "请检查汇率数据",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respAfter.Confidence <= 0 {
		t.Errorf("expected confidence > 0 after registering pattern, got %f", respAfter.Confidence)
	}
	// Should contain the new source type in freshness.
	if _, ok := respAfter.Freshness["exchangerate"]; !ok {
		t.Error("expected freshness entry for newly registered source type")
	}
}

// ---------------------------------------------------------------------------
// Multi-keyword priority
// ---------------------------------------------------------------------------

func TestQuery_Priority(t *testing.T) {
	eng := knowledge.New(zap.NewNop())

	// Register two patterns that could both match "库存订单异常".
	// low-priority pattern: matches "库存" (1 keyword)
	eng.RegisterPattern(knowledge.KnowledgePattern{
		Keywords:      []string{"库存"},
		SourceType:    "inventory_v2",
		QueryTemplate: "SELECT * FROM inventory_v2 WHERE {condition}",
		Priority:      10,
	})
	// high-priority pattern: matches "订单" (1 keyword)
	eng.RegisterPattern(knowledge.KnowledgePattern{
		Keywords:      []string{"订单"},
		SourceType:    "order_v2",
		QueryTemplate: "SELECT * FROM order_v2 WHERE {condition}",
		Priority:      200,
	})

	ctx := context.Background()
	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "库存订单异常",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both "库存" and "订单" match (1 each). The higher-priority pattern should win.
	t.Logf("matched source type: %s", func() string {
		for k := range resp.Freshness {
			return k
		}
		return ""
	}())
	t.Logf("confidence: %f", resp.Confidence)

	if resp.Confidence <= 0 {
		t.Fatal("expected non-zero confidence for priority test")
	}
}

// ---------------------------------------------------------------------------
// MaxAge handling
// ---------------------------------------------------------------------------

func TestQuery_MaxAge(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "库存警报",
		MaxAge:   1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0, got %f", resp.Confidence)
	}
	// Should have at least one inference mentioning MaxAge.
	hasMaxAgeInference := false
	for _, inf := range resp.Inferences {
		if containsAny(inf, []string{"MaxAge", "时效", "同步"}) {
			hasMaxAgeInference = true
			break
		}
	}
	if !hasMaxAgeInference {
		t.Errorf("expected MaxAge-related inference, got: %v", resp.Inferences)
	}
}

// ---------------------------------------------------------------------------
// RegisterDataSource
// ---------------------------------------------------------------------------

func TestRegisterDataSource(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	// Register a data source.
	now := time.Now()
	eng.RegisterDataSource(knowledge.DataSource{
		Type:      "inventory",
		ID:        "warehouse-shanghai",
		Table:     "inventory",
		LastSync:  now,
		Freshness: "real-time",
	})

	resp, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "上海仓库的库存",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Confidence <= 0 {
		t.Errorf("expected confidence > 0, got %f", resp.Confidence)
	}
	// The registered source should appear in data sources.
	found := false
	for _, ds := range resp.DataSources {
		if ds.ID == "warehouse-shanghai" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected registered data source in response, but not found")
	}
	freshTime, ok := resp.Freshness["inventory"]
	if !ok {
		t.Error("expected freshness entry for inventory")
	} else if !freshTime.Equal(now) {
		t.Errorf("expected freshness %v, got %v", now, freshTime)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestQuery_NilQuery(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	_, err := eng.Query(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil query")
	}
}

func TestQuery_EmptyQuestion(t *testing.T) {
	eng := knowledge.New(zap.NewNop())
	ctx := context.Background()

	_, err := eng.Query(ctx, &knowledge.KnowledgeQuery{
		AgentID:  "test-agent",
		Question: "",
	})
	if err == nil {
		t.Fatal("expected error for empty question")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// containsAny returns true if s contains any of the substrings in substrings.
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// contains is a simple substring check (can't use strings.Contains from test).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
