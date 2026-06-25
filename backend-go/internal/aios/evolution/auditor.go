package evolution

import (
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// DecisionRecord is a single agent decision suitable for audit analysis.
type DecisionRecord struct {
	// ID is the unique identifier for this decision.
	ID string `json:"id"`

	// AgentID identifies the agent that made the decision.
	AgentID string `json:"agent_id"`

	// DecisionPoint is the named decision context (e.g. "stock_alert").
	DecisionPoint string `json:"decision_point"`

	// WasAdopted is true if the user accepted/adopted the decision.
	WasAdopted bool `json:"was_adopted"`

	// WasCorrect is true if the decision outcome was correct.
	WasCorrect bool `json:"was_correct"`

	// RejectionReason describes why the decision was rejected (empty if adopted).
	RejectionReason string `json:"rejection_reason,omitempty"`

	// Confidence is the agent's confidence level for this decision.
	Confidence float64 `json:"confidence"`

	// Timestamp is when the decision was made.
	Timestamp time.Time `json:"timestamp"`

	// FailurePattern is a categorised failure mode, if the decision was incorrect.
	FailurePattern string `json:"failure_pattern,omitempty"`
}

// AuditReport summarises agent behavior over a specific period.
type AuditReport struct {
	// AgentID identifies the audited agent.
	AgentID string `json:"agent_id"`

	// PeriodStart is the beginning of the audit window.
	PeriodStart time.Time `json:"period_start"`

	// PeriodEnd is the end of the audit window.
	PeriodEnd time.Time `json:"period_end"`

	// DecisionSummary maps decision point names to decision counts.
	DecisionSummary map[string]int `json:"decision_summary"`

	// AdoptionRate is the fraction of decisions adopted by the user (0.0–1.0).
	AdoptionRate float64 `json:"adoption_rate"`

	// RejectionReasons maps each rejection reason to its occurrence count.
	RejectionReasons map[string]int `json:"rejection_reasons"`

	// TopFailures lists the most common failure patterns, sorted by frequency.
	TopFailures []FailurePattern `json:"top_failures"`

	// Recommendations are actionable suggestions derived from the audit.
	Recommendations []string `json:"recommendations"`
}

// FailurePattern describes a recurring failure mode found during audit.
type FailurePattern struct {
	Pattern    string   `json:"pattern"`
	Count      int      `json:"count"`
	ExampleIDs []string `json:"example_ids"`
}

// BehaviorAuditor analyses agent decision records and generates structured
// audit reports that highlight adoption trends, rejection patterns, and
// recurring failure modes.
type BehaviorAuditor struct {
	logger *zap.Logger
}

// NewBehaviorAuditor creates a BehaviorAuditor.
func NewBehaviorAuditor(logger *zap.Logger) *BehaviorAuditor {
	return &BehaviorAuditor{logger: logger}
}

// GenerateReport analyses the provided decision records and produces an
// AuditReport covering the specified time period. The records slice is
// assumed to fall within the period; no filtering by time is performed here.
func (a *BehaviorAuditor) GenerateReport(agentID string, periodStart, periodEnd time.Time, records []DecisionRecord) *AuditReport {
	report := &AuditReport{
		AgentID:          agentID,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		DecisionSummary:  make(map[string]int),
		RejectionReasons: make(map[string]int),
	}

	if len(records) == 0 {
		report.Recommendations = []string{"No decisions recorded in the audit period."}
		a.logger.Warn("audit report: no records provided", zap.String("agent_id", agentID))
		return report
	}

	// Count decisions per decision point.
	adoptedCount := 0
	for _, r := range records {
		report.DecisionSummary[r.DecisionPoint]++

		if r.WasAdopted {
			adoptedCount++
		} else if r.RejectionReason != "" {
			report.RejectionReasons[r.RejectionReason]++
		}

		// Collect failure patterns for incorrect decisions.
		_ = r.FailurePattern // will use below after aggregation
	}

	// Adoption rate.
	report.AdoptionRate = float64(adoptedCount) / float64(len(records))

	// Aggregate failure patterns.
	patternMap := make(map[string]*FailurePattern)
	for _, r := range records {
		if !r.WasCorrect && r.FailurePattern != "" {
			if _, exists := patternMap[r.FailurePattern]; !exists {
				patternMap[r.FailurePattern] = &FailurePattern{
					Pattern: r.FailurePattern,
				}
			}
			patternMap[r.FailurePattern].Count++
			if len(patternMap[r.FailurePattern].ExampleIDs) < 5 {
				patternMap[r.FailurePattern].ExampleIDs = append(patternMap[r.FailurePattern].ExampleIDs, r.ID)
			}
		}
	}

	// Sort patterns by frequency descending.
	for _, fp := range patternMap {
		report.TopFailures = append(report.TopFailures, *fp)
	}
	sort.Slice(report.TopFailures, func(i, j int) bool {
		return report.TopFailures[i].Count > report.TopFailures[j].Count
	})

	// Generate recommendations.
	report.Recommendations = a.generateRecommendations(report, records)

	a.logger.Info("audit report generated",
		zap.String("agent_id", agentID),
		zap.Int("records", len(records)),
		zap.Float64("adoption_rate", report.AdoptionRate),
		zap.Int("top_failures", len(report.TopFailures)))

	return report
}

// generateRecommendations produces actionable suggestions based on audit data.
func (a *BehaviorAuditor) generateRecommendations(report *AuditReport, records []DecisionRecord) []string {
	var recs []string

	// Low adoption rate.
	if report.AdoptionRate < 0.5 {
		recs = append(recs, "Adoption rate is below 50% — consider reviewing agent prompts or lowering confidence thresholds")
	} else if report.AdoptionRate < 0.75 {
		recs = append(recs, "Adoption rate is moderate — investigate top rejection reasons for improvement opportunities")
	}

	// Frequent rejection reasons.
	if len(report.RejectionReasons) > 0 {
		topReason := ""
		topCount := 0
		for reason, count := range report.RejectionReasons {
			if count > topCount {
				topCount = count
				topReason = reason
			}
		}
		if topReason != "" {
			recs = append(recs, "Most common rejection reason: \""+topReason+"\" ("+formatCount(topCount)+" times)")
		}
	}

	// High failure count.
	totalFailures := 0
	for _, fp := range report.TopFailures {
		totalFailures += fp.Count
	}
	if totalFailures > len(records)/2 {
		recs = append(recs, "More than half of decisions have failures — consider a full prompt review")
	}

	// Specific failure patterns.
	for _, fp := range report.TopFailures {
		switch {
		case strings.Contains(fp.Pattern, "confidence") || strings.Contains(fp.Pattern, "overconfidence"):
			recs = append(recs, "Overconfidence pattern detected (\""+fp.Pattern+"\" "+formatCount(fp.Count)+" times) — adjust calibration or add confidence threshold guardrails")
		case strings.Contains(fp.Pattern, "timeout") || strings.Contains(fp.Pattern, "latency"):
			recs = append(recs, "Timeout/latency failures (\""+fp.Pattern+"\" "+formatCount(fp.Count)+" times) — consider model downgrade or caching")
		case strings.Contains(fp.Pattern, "parse") || strings.Contains(fp.Pattern, "schema"):
			recs = append(recs, "Output format failures (\""+fp.Pattern+"\" "+formatCount(fp.Count)+" times) — review output schema or add retry logic")
		}
	}

	// Low-confidence decisions that were correct.
	lowConfCorrect := 0
	samples := make([]DecisionSample, 0)
	for _, r := range records {
		if r.Confidence < 0.3 && r.WasCorrect {
			lowConfCorrect++
		}
		samples = append(samples, DecisionSample{
			PredictedConfidence: r.Confidence,
			WasCorrect:          r.WasCorrect,
			WasAdopted:          r.WasAdopted,
		})
	}
	if lowConfCorrect > 0 {
		recs = append(recs, "Found correct decisions with low confidence (<0.3) — run threshold optimizer to recalibrate")
	}

	if len(recs) == 0 {
		recs = append(recs, "No major issues detected — continue monitoring")
	}

	return recs
}

// formatCount returns a human-readable count.
func formatCount(n int) string {
	if n <= 1 {
		return "1 time"
	}
	return itoa(n) + " times"
}

// itoa is a small integer-to-string helper to avoid importing strconv for
// this one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
