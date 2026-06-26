package metabolism

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
)

// LLMSemanticScorer implements SemanticScorer by calling the AI module's
// LLMProvider to evaluate the semantic importance of scorable events.
//
// Events in the gray zone [0.40, 0.75) trigger the LLM call. The scorer
// constructs a prompt describing the event and asks the LLM for a 0-1
// importance score, which is then blended into the final combined score.
type LLMSemanticScorer struct {
	provider ai.LLMProvider
	logger   *zap.Logger
	model    string
	timeout  time.Duration
}

// LLMScorerOption configures an LLMSemanticScorer.
type LLMScorerOption func(*LLMSemanticScorer)

// WithLLMModel sets the model name for semantic scoring.
func WithLLMModel(model string) LLMScorerOption {
	return func(s *LLMSemanticScorer) {
		s.model = model
	}
}

// WithLLMTimeout sets the timeout for LLM calls.
func WithLLMTimeout(timeout time.Duration) LLMScorerOption {
	return func(s *LLMSemanticScorer) {
		s.timeout = timeout
	}
}

// NewLLMSemanticScorer creates a new LLM-backed semantic scorer.
// Uses the default LLM provider model. Pass options to customize.
func NewLLMSemanticScorer(provider ai.LLMProvider, logger *zap.Logger, opts ...LLMScorerOption) *LLMSemanticScorer {
	s := &LLMSemanticScorer{
		provider: provider,
		logger:   logger.Named("semantic_scorer"),
		model:    "",
		timeout:  10 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Score evaluates the semantic importance of a single ScorableEvent.
// Returns a float64 in [0, 1] where higher = more important / worth retaining.
func (s *LLMSemanticScorer) Score(ev ScorableEvent) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	prompt := s.buildPrompt(ev)

	req := &ai.LLMRequest{
		Model:       s.model,
		Temperature: 0.1, // low temp for consistent scoring
		MaxTokens:   50,
		Messages: []ai.LLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		s.logger.Warn("semantic scoring LLM call failed",
			zap.Int64("event_id", ev.ID),
			zap.Error(err),
		)
		return 0.5, nil // neutral fallback on error
	}

	score := parseScore(resp.Answer)
	s.logger.Debug("semantic score",
		zap.Int64("event_id", ev.ID),
		zap.Float64("score", score),
		zap.String("raw", resp.Answer),
	)
	return score, nil
}

// systemPrompt instructs the LLM how to score events.
const systemPrompt = `You are a data lifecycle scoring system for an AI AgentOS.
Your job is to rate the SEMANTIC IMPORTANCE of system events on a scale of 0.0 to 1.0.

Rules:
- 0.0–0.3: Low importance. Routine noise, repeated health checks, debug-level events.
- 0.4–0.6: Medium importance. Status transitions, periodic sync completions.
- 0.7–1.0: High importance. Business-significant events: orders, price changes,
  inventory alerts, compliance flags, customer interactions, anomalies.

Respond with ONLY a single number between 0.0 and 1.0. No explanation. No markdown.
Default to 0.5 when uncertain.`

// buildPrompt constructs a user prompt describing the event for the LLM.
func (s *LLMSemanticScorer) buildPrompt(ev ScorableEvent) string {
	age := time.Since(ev.CreatedAt).Truncate(time.Minute)
	return fmt.Sprintf(`Event:
- ID: %d
- Source: %s
- Topic: %s
- Priority: %d
- Status: %s
- Age: %s
- OpLogCount: %d
- RefCount: %d`,
		ev.ID, ev.Source, ev.Topic, ev.Priority, ev.Status, age, ev.OpLogCount, ev.RefCount)
}

// parseScore extracts a float64 score from the LLM's response text.
// Supports formats: "0.7", "0.70", "score: 0.7", "0.7 (medium)"
func parseScore(answer string) float64 {
	// Try to extract the first floating-point number.
	var val float64
	cleaned := ""
	for _, r := range answer {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			cleaned += string(r)
		} else if cleaned != "" {
			break // stop at first non-numeric after we started
		}
	}
	if cleaned == "" {
		return 0.5
	}
	if err := json.Unmarshal([]byte(cleaned), &val); err != nil {
		return 0.5
	}
	if val < 0 {
		return 0.0
	}
	if val > 1 {
		return 1.0
	}
	return val
}
