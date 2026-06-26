package metabolism

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// LLM Scorer Tests
// ---------------------------------------------------------------------------

func TestParseScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected float64
		name     string
	}{
		{name: "simple score", input: "0.7", expected: 0.7},
		{name: "with leading text", input: "score: 0.42", expected: 0.42},
		{name: "with trailing text", input: "0.85 (high)", expected: 0.85},
		{name: "newline suffix", input: "0.3\n", expected: 0.3},
		{name: "clamp above 1", input: "1.5", expected: 1.0},
		{name: "clamp below 0", input: "-0.5", expected: 0.0},
		{name: "full int", input: "1", expected: 1.0},
		{name: "zero", input: "0", expected: 0.0},
		{name: "empty string", input: "", expected: 0.5},
		{name: "gibberish", input: "I think this is important", expected: 0.5},
		{name: "multiple numbers", input: "0.7 (fallback 0.3)", expected: 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScore(tt.input)
			if got != tt.expected {
				t.Errorf("parseScore(%q) = %.4f, want %.4f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLLMSemanticScorer_WithStubProvider(t *testing.T) {
	logger := dbtest.NewLogger(t)
	provider := ai.NewLLMProvider(logger) // returns StubProvider when LLM_PROVIDER is unset

	scorer := NewLLMSemanticScorer(provider, logger, WithLLMTimeout(5*time.Second))

	ev := ScorableEvent{
		ID:         1,
		Source:     "ozon",
		Topic:      "order.created",
		Priority:   2,
		Status:     "pending",
		CreatedAt:  time.Now().Add(-1 * time.Hour),
		OpLogCount: 3,
		RefCount:   1,
	}

	score, err := scorer.Score(ev)
	if err != nil {
		t.Fatalf("Score should not error with stub provider: %v", err)
	}
	if score < 0 || score > 1 {
		t.Errorf("Score out of range [0,1]: %.4f", score)
	}
	t.Logf("Stub provider returned score: %.4f", score)
}

// stubRecorderProvider records the last request for test inspection.
type stubRecorderProvider struct {
	ai.LLMProvider
	lastRequest *ai.LLMRequest
}

func (s *stubRecorderProvider) Chat(ctx context.Context, req *ai.LLMRequest) (*ai.LLMResponse, error) {
	s.lastRequest = req
	return s.LLMProvider.Chat(ctx, req)
}

func TestLLMSemanticScorer_PromptContainsEventInfo(t *testing.T) {
	logger := dbtest.NewLogger(t)
	inner := ai.NewLLMProvider(logger)
	recorder := &stubRecorderProvider{LLMProvider: inner}

	scorer := NewLLMSemanticScorer(recorder, logger)

	ev := ScorableEvent{
		ID:         42,
		Source:     "shopee",
		Topic:      "price.change",
		Priority:   1,
		Status:     "pending",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		OpLogCount: 5,
		RefCount:   3,
	}

	_, err := scorer.Score(ev)
	if err != nil {
		t.Fatalf("Score error: %v", err)
	}

	if recorder.lastRequest == nil {
		t.Fatal("no LLM request was made")
	}

	// Verify event details appear in the prompt.
	combined := ""
	for _, msg := range recorder.lastRequest.Messages {
		combined += msg.Content
	}
	for _, expected := range []string{"42", "shopee", "price.change", "5", "3"} {
		if !contains(combined, expected) {
			t.Errorf("prompt missing expected content: %q", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 &&
		(len(substr) == 0 || searchStr(s, substr)))
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
