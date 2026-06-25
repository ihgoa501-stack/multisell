package observability

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// dpObs is an internal observation of a single decision-point execution.
type dpObs struct {
	decisionPoint string
	agentID       string
	confidence    float64
	latencyMs     int
	success       bool
	failureType   string
	timestamp     time.Time
}

// Collector is an in-memory metrics collector for agent observability.
//
// It records AgentMetrics snapshots and per-decision-point observations, then
// provides aggregation queries, cost breakdowns, and anomaly detection.
type Collector struct {
	events []AgentMetrics
	dpObs  []dpObs
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewCollector creates a new Collector with the given logger.
func NewCollector(logger *zap.Logger) *Collector {
	return &Collector{
		events: make([]AgentMetrics, 0),
		dpObs:  make([]dpObs, 0),
		logger: logger,
	}
}

// Record stores an AgentMetrics snapshot in the collector.
func (c *Collector) Record(metrics AgentMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, metrics)
	if c.logger != nil {
		c.logger.Debug("recorded metrics",
			zap.String("agent_id", metrics.AgentID),
			zap.Int("decisions", metrics.DecisionsMade),
			zap.Float64("confidence", metrics.AverageConfidence),
		)
	}
}

// recordDecisionObservation stores a per-decision-point observation.
//
// This is an internal helper used by tests and by adapters that can
// instrument individual decision executions. The method is unexported
// because external callers route through Record; decision-point-level
// instrumentation is added by the decision pipeline layer.
func (c *Collector) recordDecisionObservation(dp, agentID string, confidence float64, latencyMs int, success bool, failureType string, ts time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dpObs = append(c.dpObs, dpObs{
		decisionPoint: dp,
		agentID:       agentID,
		confidence:    confidence,
		latencyMs:     latencyMs,
		success:       success,
		failureType:   failureType,
		timestamp:     ts,
	})
}

// Query returns aggregated AgentMetrics for a single agent since the given
// time. All volume fields are summed, and quality/cost fields are averaged.
// Returns nil if no events match.
func (c *Collector) Query(agentID string, since time.Time) *AgentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var filtered []AgentMetrics
	for _, e := range c.events {
		if e.AgentID == agentID && !e.PeriodEnd.Before(since) {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return aggregate(filtered)
}

// QueryAll returns aggregated AgentMetrics for every agent that has records
// since the given time, grouped by AgentID.
func (c *Collector) QueryAll(since time.Time) []AgentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	byAgent := make(map[string][]AgentMetrics)
	for _, e := range c.events {
		if !e.PeriodEnd.Before(since) {
			byAgent[e.AgentID] = append(byAgent[e.AgentID], e)
		}
	}

	result := make([]AgentMetrics, 0, len(byAgent))
	for _, group := range byAgent {
		result = append(result, *aggregate(group))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AgentID < result[j].AgentID
	})
	return result
}

// QueryDecisionPoint returns aggregated stats for all observations of a
// named decision point since the given time. Returns nil if no observations
// match.
func (c *Collector) QueryDecisionPoint(dp string, since time.Time) *DecisionPointStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var filtered []dpObs
	for _, o := range c.dpObs {
		if o.decisionPoint == dp && !o.timestamp.Before(since) {
			filtered = append(filtered, o)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	var totalConf float64
	var totalLatency int
	var successCount int
	failureBreakdown := make(map[string]int)

	for _, o := range filtered {
		totalConf += o.confidence
		totalLatency += o.latencyMs
		if o.success {
			successCount++
		} else {
			failureBreakdown[o.failureType]++
		}
	}

	n := len(filtered)
	return &DecisionPointStats{
		DecisionPoint:     dp,
		TotalExecutions:   n,
		AverageConfidence: totalConf / float64(n),
		AverageLatencyMs:  totalLatency / n,
		SuccessRate:       float64(successCount) / float64(n),
		FailureBreakdown:  failureBreakdown,
	}
}

// QueryCost returns cost breakdown rows grouped by the given dimension since
// the given time.
//
// Supported groupBy values: "agent", "decision_point".
// For "agent", cost data comes from AgentMetrics.
// For "decision_point", cost data comes from decision-point observations
// (derived from tool calls per observation).
// Unknown groupBy values default to "agent".
func (c *Collector) QueryCost(groupBy string, since time.Time) []CostRow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch groupBy {
	case "decision_point":
		return c.queryCostByDPSince(since)
	default:
		return c.queryCostByAgentSince(since)
	}
}

func (c *Collector) queryCostByAgentSince(since time.Time) []CostRow {
	byAgent := make(map[string]*CostRow)
	var agentOrder []string

	for _, e := range c.events {
		if e.PeriodEnd.Before(since) {
			continue
		}
		cr, ok := byAgent[e.AgentID]
		if !ok {
			cr = &CostRow{Dimension: e.AgentID}
			agentOrder = append(agentOrder, e.AgentID)
			byAgent[e.AgentID] = cr
		}
		cr.TotalCost += e.EstimatedCostUsd
		cr.TotalTokens += e.TokensUsed
		cr.CallCount += e.ToolCallsMade
	}

	result := make([]CostRow, 0, len(byAgent))
	for _, id := range agentOrder {
		result = append(result, *byAgent[id])
	}
	return result
}

func (c *Collector) queryCostByDPSince(since time.Time) []CostRow {
	byDP := make(map[string]*CostRow)
	var dpOrder []string

	for _, o := range c.dpObs {
		if o.timestamp.Before(since) {
			continue
		}
		cr, ok := byDP[o.decisionPoint]
		if !ok {
			cr = &CostRow{Dimension: o.decisionPoint}
			dpOrder = append(dpOrder, o.decisionPoint)
			byDP[o.decisionPoint] = cr
		}
		cr.CallCount++
	}

	result := make([]CostRow, 0, len(byDP))
	for _, dp := range dpOrder {
		result = append(result, *byDP[dp])
	}
	return result
}

// ScanAnomalies scans all recorded events for anomalous agent behavior.
//
// It uses z-score analysis on AverageConfidence and HighRiskActions against
// the full event set. An event is flagged when its z-score magnitude exceeds
// the given threshold. Supported anomaly types:
//   - "confidence_drop": average confidence is threshold std below the mean
//   - "risk_spike": high-risk action count is threshold std above the mean
//
// With fewer than 3 events, no anomalies are reported (insufficient data to
// establish a baseline).
func (c *Collector) ScanAnomalies(threshold float64) []AnomalyReport {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.events) < 3 {
		return nil
	}

	// Compute mean and std dev for confidence and high-risk actions.
	meanConf, stdConf := meanStdDevConfidence(c.events)
	meanRisk, stdRisk := meanStdDevHighRisk(c.events)

	var reports []AnomalyReport
	now := time.Now()

	for _, e := range c.events {
		// Confidence drop: confidence is threshold std below mean.
		if stdConf > 0 {
			z := (meanConf - e.AverageConfidence) / stdConf
			if z > threshold {
				severity := "warning"
				if z > threshold*2 {
					severity = "critical"
				}
				reports = append(reports, AnomalyReport{
					AgentID:     e.AgentID,
					Type:        "confidence_drop",
					Severity:    severity,
					TriggeredAt: now,
					Details:     fmt.Sprintf("Confidence %.2f is %.1f std below mean %.2f", e.AverageConfidence, z, meanConf),
				})
			}
		}

		// Risk spike: high-risk actions is threshold std above mean.
		if stdRisk > 0 {
			z := (float64(e.HighRiskActions) - meanRisk) / stdRisk
			if z > threshold {
				severity := "warning"
				if z > threshold*2 {
					severity = "critical"
				}
				reports = append(reports, AnomalyReport{
					AgentID:     e.AgentID,
					Type:        "risk_spike",
					Severity:    severity,
					TriggeredAt: now,
					Details:     fmt.Sprintf("High-risk actions %d is %.1f std above mean %.0f", e.HighRiskActions, z, meanRisk),
				})
			}
		}
	}

	return reports
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

// aggregate combines multiple AgentMetrics into one, summing volume fields
// and averaging quality/cost fields.
func aggregate(events []AgentMetrics) *AgentMetrics {
	if len(events) == 0 {
		return nil
	}

	result := AgentMetrics{
		AgentID:     events[0].AgentID,
		PeriodStart: events[0].PeriodStart,
		PeriodEnd:   events[len(events)-1].PeriodEnd,
	}

	var totalConf float64
	var totalLatency int
	var totalRate float64
	var totalCost float64

	for _, e := range events {
		result.DecisionsMade += e.DecisionsMade
		result.ActionsCreated += e.ActionsCreated
		result.ActionsApproved += e.ActionsApproved
		result.ActionsRejected += e.ActionsRejected
		result.ActionsExecuted += e.ActionsExecuted
		result.HighRiskActions += e.HighRiskActions
		result.TokensUsed += e.TokensUsed
		result.ToolCallsMade += e.ToolCallsMade

		totalConf += e.AverageConfidence
		totalLatency += e.AvgLatencyMs
		totalRate += e.SuccessRate
		totalCost += e.EstimatedCostUsd

		if e.PeriodStart.Before(result.PeriodStart) {
			result.PeriodStart = e.PeriodStart
		}
	}

	n := float64(len(events))
	result.AverageConfidence = totalConf / n
	result.AvgLatencyMs = int(math.Round(float64(totalLatency) / n))
	result.SuccessRate = totalRate / n
	result.EstimatedCostUsd = totalCost

	return &result
}

func meanStdDevConfidence(events []AgentMetrics) (mean, std float64) {
	n := float64(len(events))
	if n == 0 {
		return 0, 0
	}
	for _, e := range events {
		mean += e.AverageConfidence
	}
	mean /= n
	if n < 2 {
		return mean, 0
	}
	for _, e := range events {
		d := e.AverageConfidence - mean
		std += d * d
	}
	std = math.Sqrt(std / (n - 1))
	return mean, std
}

func meanStdDevHighRisk(events []AgentMetrics) (mean, std float64) {
	n := float64(len(events))
	if n == 0 {
		return 0, 0
	}
	for _, e := range events {
		mean += float64(e.HighRiskActions)
	}
	mean /= n
	if n < 2 {
		return mean, 0
	}
	for _, e := range events {
		d := float64(e.HighRiskActions) - mean
		std += d * d
	}
	std = math.Sqrt(std / (n - 1))
	return mean, std
}
