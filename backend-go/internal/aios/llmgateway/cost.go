package llmgateway

import (
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// ModelCostRecord
// ---------------------------------------------------------------------------

// ModelCostRecord captures cost data for a single LLM call.
type ModelCostRecord struct {
	Model     string    `json:"model"`
	TokensIn  int       `json:"tokens_in"`
	TokensOut int       `json:"tokens_out"`
	CostUSD   float64   `json:"cost_usd"`
	Timestamp time.Time `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// CostSummary
// ---------------------------------------------------------------------------

// CostSummary aggregates cost tracking statistics.
type CostSummary struct {
	TotalCalls     int            `json:"total_calls"`
	TotalTokensIn  int            `json:"total_tokens_in"`
	TotalTokensOut int            `json:"total_tokens_out"`
	TotalCostUSD   float64        `json:"total_cost_usd"`
	CallsByModel   map[string]int `json:"calls_by_model"`
	TokensByModel  map[string]int `json:"tokens_by_model"`
}

// ---------------------------------------------------------------------------
// CostTracker
// ---------------------------------------------------------------------------

// CostTracker records per-call token usage and model choices.
// All methods are safe for concurrent use.
type CostTracker struct {
	mu      sync.RWMutex
	records []ModelCostRecord
}

// NewCostTracker creates an empty CostTracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		records: make([]ModelCostRecord, 0),
	}
}

// Record adds a cost record for a single LLM call.
func (ct *CostTracker) Record(model string, tokensIn, tokensOut int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.records = append(ct.records, ModelCostRecord{
		Model:     model,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   EstimateCost(model, tokensIn, tokensOut),
		Timestamp: time.Now(),
	})
}

// Summary returns aggregated cost statistics.
func (ct *CostTracker) Summary() CostSummary {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	s := CostSummary{
		TotalCalls:    len(ct.records),
		CallsByModel:  make(map[string]int),
		TokensByModel: make(map[string]int),
	}
	for _, r := range ct.records {
		s.TotalTokensIn += r.TokensIn
		s.TotalTokensOut += r.TokensOut
		s.TotalCostUSD += r.CostUSD
		s.CallsByModel[r.Model]++
		s.TokensByModel[r.Model] += r.TokensIn + r.TokensOut
	}
	return s
}

// Reset clears all recorded cost data.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.records = make([]ModelCostRecord, 0)
}

// Records returns a copy of all recorded entries (for inspection in tests).
func (ct *CostTracker) Records() []ModelCostRecord {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	out := make([]ModelCostRecord, len(ct.records))
	copy(out, ct.records)
	return out
}

// ---------------------------------------------------------------------------
// Cost estimation
// ---------------------------------------------------------------------------

// costPerToken maps model name fragments to approximate cost per 1K tokens.
// Rates reflect approximate per-1K-token pricing; update as provider pricing changes.
var costPerToken = map[string]struct {
	Input  float64
	Output float64
}{
	"opus":   {15.00, 75.00},
	"sonnet": {3.00, 15.00},
	"haiku":  {0.25, 1.25},
}

// EstimateCost computes an approximate cost in USD for an LLM call.
// Uses model name heuristics; returns Haiku-level estimate for unknown models.
func EstimateCost(model string, tokensIn, tokensOut int) float64 {
	var rates struct{ Input, Output float64 }
	base := strings.ToLower(model)
	for _, prefix := range []string{"claude-", "gpt-", "gemini-"} {
		base = strings.TrimPrefix(base, prefix)
	}
	switch {
	case strings.Contains(base, "opus"):
		rates = costPerToken["opus"]
	case strings.Contains(base, "sonnet"):
		rates = costPerToken["sonnet"]
	case strings.Contains(base, "haiku"):
		rates = costPerToken["haiku"]
	default:
		rates = costPerToken["haiku"]
	}
	return float64(tokensIn)/1000*rates.Input + float64(tokensOut)/1000*rates.Output
}
