package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ClassificationResult holds the AI-suggested classification for a submission.
type ClassificationResult struct {
	FeedbackType string   `json:"feedback_type"`
	Severity     string   `json:"severity"`
	CategoryName string   `json:"category_name"`
	Tags         []string `json:"tags"`
	Summary      string   `json:"summary"`
	Confidence   float64  `json:"confidence"`
}

// Classifier defines the interface for AI-powered classification.
type Classifier interface {
	// ClassifySubmission analyzes feedback text and suggests classification.
	ClassifySubmission(ctx context.Context, title, description string) (*ClassificationResult, error)
}

// PromptBuilder builds the system prompt for classification.
const classificationSystemPrompt = `You are a UX feedback classifier. Analyze the user's feedback and return a JSON object with:
- feedback_type: one of "bug", "feature", "improvement", "question", "other"
- severity: one of "critical", "major", "minor", "trivial" (empty string for questions/other)
- category_name: a short category name like "UI", "Performance", "Checkout", "Dashboard", etc.
- tags: up to 3 relevant tags as an array of strings
- summary: one-sentence summary of the core issue or request
- confidence: float 0-1 indicating how confident you are in this classification

Return ONLY valid JSON, no markdown, no explanation.`

// LLMChatFunc is the function signature for calling an LLM.
type LLMChatFunc func(ctx context.Context, systemPrompt string, userMessage string) (string, error)

// aiClassifier implements Classifier using an LLM.
type aiClassifier struct {
	chat   LLMChatFunc
	logger logger
}

// logger is a minimal logger interface for the classifier.
type logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
}

// NewAIClassifier creates a new AI classifier with the given LLM chat function.
func NewAIClassifier(chat LLMChatFunc, log logger) Classifier {
	return &aiClassifier{chat: chat, logger: log}
}

// ClassifySubmission implements Classifier.
func (c *aiClassifier) ClassifySubmission(ctx context.Context, title, description string) (*ClassificationResult, error) {
	userMsg := fmt.Sprintf("Title: %s\nDescription: %s", title, description)

	start := time.Now()
	answer, err := c.chat(ctx, classificationSystemPrompt, userMsg)
	latency := time.Since(start)

	if err != nil {
		c.logger.Warnw("AI classification failed", "error", err, "latency_ms", latency.Milliseconds())
		return c.fallback(title, description), nil
	}

	// Parse JSON response (strip markdown code fences if present)
	cleaned := strings.TrimSpace(answer)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result ClassificationResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		c.logger.Warnw("Failed to parse AI classification response",
			"error", err, "raw", answer)
		return c.fallback(title, description), nil
	}

	// Validate and clean the result
	result = c.sanitize(result)

	c.logger.Infow("AI classification complete",
		"type", result.FeedbackType,
		"severity", result.Severity,
		"confidence", result.Confidence,
		"latency_ms", latency.Milliseconds())

	return &result, nil
}

// sanitize validates and defaults the classification result fields.
func (c *aiClassifier) sanitize(r ClassificationResult) ClassificationResult {
	validTypes := map[string]bool{"bug": true, "feature": true, "improvement": true, "question": true, "other": true}
	validSeverities := map[string]bool{"critical": true, "major": true, "minor": true, "trivial": true}

	if !validTypes[r.FeedbackType] {
		r.FeedbackType = "feature"
	}
	if r.Severity != "" && !validSeverities[r.Severity] {
		r.Severity = ""
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		r.Confidence = 0
	}
	if len(r.Tags) > 5 {
		r.Tags = r.Tags[:5]
	}
	return r
}

// fallback returns a safe default when AI classification fails.
func (c *aiClassifier) fallback(title, description string) *ClassificationResult {
	// Simple keyword-based fallback
	titleLower := strings.ToLower(title)
	descLower := strings.ToLower(description)
	combined := titleLower + " " + descLower

	result := &ClassificationResult{
		FeedbackType: "feature",
		Severity:     "",
		Confidence:   0,
	}

	switch {
	case containsAny(combined, "bug", "error", "crash", "broken", "not working", "failed", "exception"):
		result.FeedbackType = "bug"
		result.Severity = "major"
	case containsAny(combined, "slow", "performance", "lag", "loading", "timeout"):
		result.FeedbackType = "improvement"
		result.Severity = "minor"
		result.Tags = []string{"performance"}
	case containsAny(combined, "confus", "hard to", "difficult", "unclear", "where is"):
		result.FeedbackType = "improvement"
		result.Severity = "minor"
	case containsAny(combined, "would like", "wish", "hope", "suggest", "idea", "feature", "add"):
		result.FeedbackType = "feature"
	case containsAny(combined, "how to", "how do", "question", "what is", "where"):
		result.FeedbackType = "question"
	}

	if result.Severity == "" {
		result.Severity = "minor"
	}

	c.logger.Infow("Fallback classification used (no AI)", "type", result.FeedbackType)
	return result
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
