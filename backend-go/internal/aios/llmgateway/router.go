package llmgateway

import (
	"context"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// ModelTarget
// ---------------------------------------------------------------------------

// ModelTarget specifies which model and strategy to use for a request.
type ModelTarget struct {
	// Model name, e.g. "claude-haiku-4", "claude-sonnet-4", "claude-opus-4".
	Model string `json:"model"`
	// Priority: 0 = primary, 1 = first fallback, 2 = second fallback.
	Priority int `json:"priority"`
	// MaxRetries is the number of retry attempts before moving to the next fallback.
	MaxRetries int `json:"max_retries"`
	// Timeout is the per-attempt timeout for this model.
	Timeout time.Duration `json:"timeout"`
	// CostWeight is a factor for cost optimization scoring (higher = more expensive).
	CostWeight float64 `json:"cost_weight"`
	// Reason explains why this model was selected.
	Reason string `json:"reason,omitempty"`
}

// ---------------------------------------------------------------------------
// Router interface
// ---------------------------------------------------------------------------

// Router decides which model to use for a given request.
type Router interface {
	// Select returns the optimal ModelTarget for the request.
	Select(ctx context.Context, req *Request) ModelTarget
}

// ---------------------------------------------------------------------------
// DefaultRouter
// ---------------------------------------------------------------------------

// DefaultRouter implements the standard routing strategy:
//
//  1. If MaxLatency < 3s → Haiku (fastest)
//  2. If Sensitive (financial/risk) → Opus (most thorough)
//  3. Analysis, reasoning, advice → Sonnet (balanced)
//  4. Classification, extraction → Haiku (cheap)
//  5. Complex decisions, negotiation → Opus (deep)
//  6. Otherwise → Sonnet (default for general-purpose)
//
// Every routing decision respects req.MinModel as a floor — if the selected
// model is below the caller's minimum requirement, it is upgraded.
type DefaultRouter struct{}

// NewDefaultRouter creates a DefaultRouter.
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{}
}

// Select implements Router.Select with the documented routing rules.
func (r *DefaultRouter) Select(_ context.Context, req *Request) ModelTarget {
	selected, reason := r.routeByRules(req)

	// Enforce MinModel as a floor.
	if req.MinModel != "" && modelTier(selected) < modelTier(req.MinModel) {
		upgraded := maxModel(selected, req.MinModel)
		reason += "; upgraded to " + upgraded + " (min_model=" + req.MinModel + ")"
		selected = upgraded
	}

	return ModelTarget{
		Model:      selected,
		Priority:   0,
		MaxRetries: 2,
		Timeout:    30 * time.Second,
		CostWeight: modelCostWeight(selected),
		Reason:     reason,
	}
}

// routeByRules applies the routing decision rules in priority order.
func (r *DefaultRouter) routeByRules(req *Request) (string, string) {
	// Rule 1: Low latency requirement → Haiku
	if req.MaxLatency > 0 && req.MaxLatency < 3*time.Second {
		return "claude-haiku-4", "max_latency < 3s"
	}

	// Rule 2: Sensitive (financial/risk) → Opus
	if req.Sensitive {
		return "claude-opus-4", "sensitive request (financial/risk)"
	}

	// Rule 3-6: Content-based routing
	content := analyseRequestContent(req)
	switch content {
	case "analysis", "reasoning", "advice":
		return "claude-sonnet-4", "content: analysis/reasoning/advice"
	case "classification", "extraction":
		return "claude-haiku-4", "content: classification/extraction"
	case "complex_decision", "negotiation":
		return "claude-opus-4", "content: complex decision/negotiation"
	default:
		return "claude-sonnet-4", "default (general purpose)"
	}
}

// ---------------------------------------------------------------------------
// Content analysis
// ---------------------------------------------------------------------------

// contentCategory classifies the request content for routing decisions.
type contentCategory string

const (
	contentGeneral        contentCategory = "general"
	contentAnalysis       contentCategory = "analysis"
	contentReasoning      contentCategory = "reasoning"
	contentAdvice         contentCategory = "advice"
	contentClassification contentCategory = "classification"
	contentExtraction     contentCategory = "extraction"
	contentComplex        contentCategory = "complex_decision"
	contentNegotiation    contentCategory = "negotiation"
)

// analyseRequestContent examines the system prompt and user messages to
// classify the request type for routing. Uses keyword heuristics that are
// safe for production — they never leak prompt content into logs.
func analyseRequestContent(req *Request) string {
	text := req.System + " " + lastMessage(req.Messages)
	text = strings.ToLower(text)

	// Check for classification keywords first (cheapest model).
	if hasAnyKeywords(text, classKeywords) {
		return string(contentClassification)
	}
	if hasAnyKeywords(text, extractKeywords) {
		return string(contentExtraction)
	}

	// Complex/high-stakes decisions.
	if hasAnyKeywords(text, complexKeywords) {
		return string(contentComplex)
	}
	if hasAnyKeywords(text, negotiationKeywords) {
		return string(contentNegotiation)
	}

	// Mid-tier reasoning.
	if hasAnyKeywords(text, analysisKeywords) {
		return string(contentAnalysis)
	}
	if hasAnyKeywords(text, reasoningKeywords) {
		return string(contentReasoning)
	}
	if hasAnyKeywords(text, adviceKeywords) {
		return string(contentAdvice)
	}

	return string(contentGeneral)
}

func hasAnyKeywords(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// Keyword sets for content classification.
var (
	classKeywords = []string{
		"classify", "categorize", "categorisation", "tag", "label",
		"assign category", "sort into", "group by",
	}
	extractKeywords = []string{
		"extract", "parse", "structured data", "json output",
		"pull out", "scrape", "collect fields",
	}
	analysisKeywords = []string{
		"analyze", "analyse", "analysis", "examine", "evaluate",
		"assess", "break down", "comparison", "trend",
	}
	reasoningKeywords = []string{
		"reason", "reasoning", "why", "explain", "infer",
		"deduce", "logical", "therefore", "because", "cause",
		"root cause", "conclude", "conclusion",
	}
	adviceKeywords = []string{
		"suggest", "recommend", "advise", "advice", "propose",
		"what should", "best approach", "strategy", "optimize",
		"improve", "actionable",
	}
	complexKeywords = []string{
		"complex decision", "trade-off", "multi-factor", "weighted",
		"priority matrix", "critical", "high stakes", "significant impact",
		"strategic decision", "long-term",
	}
	negotiationKeywords = []string{
		"negotiate", "negotiation", "bargain", "counter", "vendor terms",
		"contract", "price agreement", "dispute", "settle",
	}
)

// modelCostWeight returns a relative cost weight for a model name.
// Higher values indicate more expensive models.
func modelCostWeight(model string) float64 {
	switch {
	case strings.Contains(model, "opus"):
		return 15.0
	case strings.Contains(model, "sonnet"):
		return 3.0
	case strings.Contains(model, "haiku"):
		return 1.0
	default:
		return 1.0
	}
}
